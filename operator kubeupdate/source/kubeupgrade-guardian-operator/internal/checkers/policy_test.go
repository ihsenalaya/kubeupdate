package checkers

import (
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/snapshot"
)

func TestPolicyRiskAcceptsSecurityContextDefinedOnlyAtPodLevel(t *testing.T) {
	nonRoot := true
	noEscalation := false
	spec := corev1.PodSpec{
		SecurityContext: &corev1.PodSecurityContext{
			RunAsNonRoot:   &nonRoot,
			SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
		},
		Containers: []corev1.Container{{
			Name: "api",
			SecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: &noEscalation,
				Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
			},
		}},
	}

	findings := PolicyRisk{}.Check(restrictedSnapshot(spec), assessment("production"))
	for _, finding := range findings {
		if strings.Contains(finding.Message, "runAsNonRoot") {
			t.Fatalf("runAsNonRoot set at pod level must satisfy the container: %s", finding.Message)
		}
		if strings.Contains(finding.Message, "seccompProfile") {
			t.Fatalf("seccompProfile set at pod level must satisfy the container: %s", finding.Message)
		}
	}
	assertNoSeverity(t, findings, upgradev1alpha1.RiskLevelHigh)
}

func TestPolicyRiskLetsTheContainerOverrideThePodSecurityContext(t *testing.T) {
	nonRoot := true
	root := false
	spec := corev1.PodSpec{
		SecurityContext: &corev1.PodSecurityContext{
			RunAsNonRoot:   &nonRoot,
			SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
		},
		Containers: []corev1.Container{{
			Name: "api",
			SecurityContext: &corev1.SecurityContext{
				RunAsNonRoot: &root,
				Capabilities: &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
			},
		}},
	}

	findings := PolicyRisk{}.Check(restrictedSnapshot(spec), assessment("production"))
	assertMessage(t, findings, "runAsNonRoot=false")
}

func TestPolicyRiskDetectsMissingRestrictedSeccompAndCapabilities(t *testing.T) {
	nonRoot := true
	spec := corev1.PodSpec{
		Containers: []corev1.Container{{
			Name:            "api",
			SecurityContext: &corev1.SecurityContext{RunAsNonRoot: &nonRoot},
		}},
	}

	findings := PolicyRisk{}.Check(restrictedSnapshot(spec), assessment("production"))
	assertMessage(t, findings, "seccompProfile is not RuntimeDefault or Localhost")
	assertMessage(t, findings, "capabilities.drop does not contain ALL")
}

func TestPolicyRiskAcceptsLocalhostSeccompProfile(t *testing.T) {
	nonRoot := true
	profile := "operator/profile.json"
	spec := corev1.PodSpec{
		Containers: []corev1.Container{{
			Name: "api",
			SecurityContext: &corev1.SecurityContext{
				RunAsNonRoot: &nonRoot,
				SeccompProfile: &corev1.SeccompProfile{
					Type:             corev1.SeccompProfileTypeLocalhost,
					LocalhostProfile: &profile,
				},
				Capabilities: &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
			},
		}},
	}

	findings := PolicyRisk{}.Check(restrictedSnapshot(spec), assessment("production"))
	assertNoSeverity(t, findings, upgradev1alpha1.RiskLevelHigh)
}

func TestPolicyRiskReportsAContainerWithoutAnySecurityContextOnce(t *testing.T) {
	spec := corev1.PodSpec{Containers: []corev1.Container{{Name: "api"}}}

	findings := PolicyRisk{}.Check(restrictedSnapshot(spec), assessment("production"))
	high := 0
	for _, finding := range findings {
		if finding.Severity == upgradev1alpha1.RiskLevelHigh {
			high++
		}
	}
	if high != 1 {
		t.Fatalf("a container with no security context at all should yield a single finding, got %#v", findings)
	}
	assertMessage(t, findings, "missing securityContext")
}

func TestPolicyRiskKeepsHostPathAndPrivilegedSignals(t *testing.T) {
	privileged := true
	escalate := true
	nonRoot := true
	spec := corev1.PodSpec{
		Volumes: []corev1.Volume{{
			Name:         "host",
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/var/run"}},
		}},
		Containers: []corev1.Container{{
			Name: "api",
			SecurityContext: &corev1.SecurityContext{
				Privileged:               &privileged,
				AllowPrivilegeEscalation: &escalate,
				RunAsNonRoot:             &nonRoot,
				SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
				Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
			},
		}},
	}

	findings := PolicyRisk{}.Check(restrictedSnapshot(spec), assessment("production"))
	assertMessage(t, findings, "hostPath volume")
	assertMessage(t, findings, "privileged=true")
	assertMessage(t, findings, "allowPrivilegeEscalation=true")
}

func restrictedSnapshot(spec corev1.PodSpec) *snapshot.ClusterSnapshot {
	return &snapshot.ClusterSnapshot{
		Namespaces: []corev1.Namespace{
			namespace("production", map[string]string{"pod-security.kubernetes.io/enforce": "restricted"}),
		},
		Deployments: []appsv1.Deployment{{
			ObjectMeta: metav1.ObjectMeta{Name: "payment-api", Namespace: "production"},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{Spec: spec},
			},
		}},
	}
}

func assertMessage(t *testing.T, findings []upgradev1alpha1.Finding, fragment string) {
	t.Helper()
	for _, finding := range findings {
		if strings.Contains(finding.Message, fragment) {
			return
		}
	}
	t.Fatalf("expected a finding mentioning %q in %#v", fragment, findings)
}

func assertNoSeverity(t *testing.T, findings []upgradev1alpha1.Finding, severity upgradev1alpha1.RiskLevel) {
	t.Helper()
	for _, finding := range findings {
		if finding.Severity == severity {
			t.Fatalf("expected no %s finding, got %q", severity, finding.Message)
		}
	}
}
