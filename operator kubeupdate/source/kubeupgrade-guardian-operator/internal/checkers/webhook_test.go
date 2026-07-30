package checkers

import (
	"strings"
	"testing"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/snapshot"
)

const webhookServiceNamespace = "cert-manager"

func TestIsBroadNamespaceSelectorTreatsNilAndEmptyAsBroad(t *testing.T) {
	tests := []struct {
		name     string
		selector *metav1.LabelSelector
		want     bool
	}{
		{name: "nil", selector: nil, want: true},
		{name: "empty", selector: &metav1.LabelSelector{}, want: true},
		{name: "match labels", selector: &metav1.LabelSelector{MatchLabels: map[string]string{"team": "platform"}}, want: false},
		{
			name: "match expressions",
			selector: &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{{
				Key:      "team",
				Operator: metav1.LabelSelectorOpExists,
			}}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBroadNamespaceSelector(tt.selector); got != tt.want {
				t.Fatalf("isBroadNamespaceSelector() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInterceptsDrainOnlyForCoreGroupPodResources(t *testing.T) {
	tests := []struct {
		name  string
		rules []admissionv1.RuleWithOperations
		want  bool
	}{
		{name: "no rules", rules: nil, want: false},
		{name: "pods", rules: rules([]string{""}, []string{"pods"}), want: true},
		{name: "pod eviction", rules: rules([]string{""}, []string{"pods/eviction"}), want: true},
		{name: "core wildcard", rules: rules([]string{""}, []string{"*"}), want: true},
		{name: "group wildcard", rules: rules([]string{"*"}, []string{"*"}), want: true},
		{name: "configmaps only", rules: rules([]string{""}, []string{"configmaps"}), want: false},
		{name: "wildcard on another group", rules: rules([]string{"cert-manager.io"}, []string{"*"}), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := interceptsDrain(tt.rules); got != tt.want {
				t.Fatalf("interceptsDrain() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdmissionWebhookGradesFailurePolicyByBlastRadius(t *testing.T) {
	drainBlocking := webhookSnapshot(webhookOptions{
		failurePolicy: admissionv1.Fail,
		rules:         rules([]string{""}, []string{"pods"}),
		serviceReady:  true,
		selector:      &metav1.LabelSelector{MatchLabels: map[string]string{"team": "platform"}},
	})
	findings := AdmissionWebhook{}.Check(drainBlocking, &upgradev1alpha1.UpgradeAssessment{})
	assertFinding(t, findings, upgradev1alpha1.FindingTypeAdmissionWebhookRisk, upgradev1alpha1.RiskLevelHigh)

	certificatesOnly := webhookSnapshot(webhookOptions{
		failurePolicy: admissionv1.Fail,
		rules:         rules([]string{"cert-manager.io"}, []string{"certificates"}),
		serviceReady:  true,
		selector:      &metav1.LabelSelector{MatchLabels: map[string]string{"team": "platform"}},
	})
	findings = AdmissionWebhook{}.Check(certificatesOnly, &upgradev1alpha1.UpgradeAssessment{})
	assertFinding(t, findings, upgradev1alpha1.FindingTypeAdmissionWebhookRisk, upgradev1alpha1.RiskLevelMedium)
	for _, finding := range findings {
		if finding.Severity == upgradev1alpha1.RiskLevelHigh {
			t.Fatalf("a webhook that does not intercept pods cannot block a drain: %s", finding.Message)
		}
	}
}

func TestAdmissionWebhookReportsSlowFailingWebhookAndTimeoutEvidence(t *testing.T) {
	timeout := int32(30)
	snap := webhookSnapshot(webhookOptions{
		failurePolicy:  admissionv1.Fail,
		rules:          rules([]string{""}, []string{"pods"}),
		serviceReady:   true,
		selector:       &metav1.LabelSelector{MatchLabels: map[string]string{"team": "platform"}},
		timeoutSeconds: &timeout,
	})

	findings := AdmissionWebhook{}.Check(snap, &upgradev1alpha1.UpgradeAssessment{})
	finding := findingWithMessage(t, findings, "failurePolicy=Fail")
	if !strings.Contains(finding.Message, "timeoutSeconds=30") {
		t.Fatalf("a slow failing webhook must say so in the message, got %q", finding.Message)
	}
	if got := finding.Evidence[0].Observed["timeoutSeconds"]; got != "30" {
		t.Fatalf("expected timeoutSeconds evidence 30, got %q", got)
	}
}

func TestAdmissionWebhookDefaultsTimeoutEvidenceToTheAPIServerDefault(t *testing.T) {
	snap := webhookSnapshot(webhookOptions{
		failurePolicy: admissionv1.Fail,
		rules:         rules([]string{""}, []string{"pods"}),
		serviceReady:  true,
		selector:      &metav1.LabelSelector{MatchLabels: map[string]string{"team": "platform"}},
	})

	findings := AdmissionWebhook{}.Check(snap, &upgradev1alpha1.UpgradeAssessment{})
	finding := findingWithMessage(t, findings, "failurePolicy=Fail")
	if got := finding.Evidence[0].Observed["timeoutSeconds"]; got != "10" {
		t.Fatalf("expected the default timeout of 10s in evidence, got %q", got)
	}
	if strings.Contains(finding.Message, "timeoutSeconds") {
		t.Fatalf("a webhook within the default timeout must not mention it, got %q", finding.Message)
	}
}

func TestAdmissionWebhookDetectsMissingService(t *testing.T) {
	snap := webhookSnapshot(webhookOptions{
		failurePolicy: admissionv1.Ignore,
		serviceExists: false,
		selector:      &metav1.LabelSelector{MatchLabels: map[string]string{"team": "platform"}},
	})

	findings := AdmissionWebhook{}.Check(snap, &upgradev1alpha1.UpgradeAssessment{})
	finding := findingWithMessage(t, findings, "referenced service is absent")
	if finding.Severity != upgradev1alpha1.RiskLevelCritical {
		t.Fatalf("expected Critical for an absent backend, got %s", finding.Severity)
	}
}

func TestAdmissionWebhookDetectsBackendWithoutReadyEndpoints(t *testing.T) {
	snap := webhookSnapshot(webhookOptions{
		failurePolicy: admissionv1.Ignore,
		serviceExists: true,
		serviceReady:  false,
		selector:      &metav1.LabelSelector{MatchLabels: map[string]string{"team": "platform"}},
	})

	findings := AdmissionWebhook{}.Check(snap, &upgradev1alpha1.UpgradeAssessment{})
	finding := findingWithMessage(t, findings, "webhook backend has no ready endpoints")
	if finding.Severity != upgradev1alpha1.RiskLevelCritical {
		t.Fatalf("expected Critical for a backend without ready endpoints, got %s", finding.Severity)
	}
}

func TestAdmissionWebhookStaysSilentOnReadinessWhenEndpointSlicesAreDenied(t *testing.T) {
	snap := webhookSnapshot(webhookOptions{
		failurePolicy: admissionv1.Ignore,
		serviceExists: true,
		serviceReady:  false,
		selector:      &metav1.LabelSelector{MatchLabels: map[string]string{"team": "platform"}},
	})
	snap.Gaps = []upgradev1alpha1.Finding{{
		ID:   "RBAC_ASSESSMENT_GAP_SNAPSHOT_ENDPOINTSLICES",
		Type: upgradev1alpha1.FindingTypeRBACAssessmentGap,
		Evidence: []upgradev1alpha1.Evidence{{
			Observed: map[string]string{"resource": snapshot.ResourceEndpointSlices},
		}},
	}}

	for _, finding := range (AdmissionWebhook{}).Check(snap, &upgradev1alpha1.UpgradeAssessment{}) {
		if strings.Contains(finding.Message, "no ready endpoints") {
			t.Fatal("endpoint readiness must not be judged when the operator was not allowed to read EndpointSlices")
		}
	}
}

func TestAdmissionWebhookDowngradesBroadNamespaceSelectorToInfo(t *testing.T) {
	snap := webhookSnapshot(webhookOptions{
		failurePolicy: admissionv1.Ignore,
		serviceExists: true,
		serviceReady:  true,
	})

	findings := AdmissionWebhook{}.Check(snap, &upgradev1alpha1.UpgradeAssessment{})
	finding := findingWithMessage(t, findings, "namespaceSelector is absent")
	if finding.Severity != upgradev1alpha1.RiskLevelInfo {
		t.Fatalf("a missing namespaceSelector is too common to be actionable, expected Info, got %s", finding.Severity)
	}
}

func TestAdmissionWebhookAcceptsHealthyBackendOutsideTheAssessedScope(t *testing.T) {
	snap := webhookSnapshot(webhookOptions{
		failurePolicy: admissionv1.Ignore,
		serviceExists: true,
		serviceReady:  true,
		selector:      &metav1.LabelSelector{MatchLabels: map[string]string{"team": "platform"}},
	})
	snap.Namespaces = []corev1.Namespace{namespace("production", nil)}

	if findings := (AdmissionWebhook{}).Check(snap, assessment("production")); len(findings) != 0 {
		t.Fatalf("a healthy webhook backend outside the scope must produce no finding, got %#v", findings)
	}
}

type webhookOptions struct {
	failurePolicy  admissionv1.FailurePolicyType
	rules          []admissionv1.RuleWithOperations
	selector       *metav1.LabelSelector
	timeoutSeconds *int32
	serviceExists  bool
	serviceReady   bool
}

func webhookSnapshot(options webhookOptions) *snapshot.ClusterSnapshot {
	failurePolicy := options.failurePolicy
	snap := &snapshot.ClusterSnapshot{
		ValidatingWebhooks: []admissionv1.ValidatingWebhookConfiguration{{
			ObjectMeta: metav1.ObjectMeta{Name: "cert-manager-webhook"},
			Webhooks: []admissionv1.ValidatingWebhook{{
				Name:          "webhook.cert-manager.io",
				FailurePolicy: &failurePolicy,
				ClientConfig: admissionv1.WebhookClientConfig{
					Service: &admissionv1.ServiceReference{Namespace: webhookServiceNamespace, Name: "webhook"},
				},
				NamespaceSelector: options.selector,
				Rules:             options.rules,
				TimeoutSeconds:    options.timeoutSeconds,
			}},
		}},
	}

	if options.serviceExists || options.serviceReady {
		snap.Services = []corev1.Service{{
			ObjectMeta: metav1.ObjectMeta{Namespace: webhookServiceNamespace, Name: "webhook"},
		}}
	}
	if options.serviceReady {
		ready := true
		snap.EndpointSlices = []discoveryv1.EndpointSlice{{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: webhookServiceNamespace,
				Name:      "webhook-abcde",
				Labels:    map[string]string{discoveryv1.LabelServiceName: "webhook"},
			},
			Endpoints: []discoveryv1.Endpoint{{
				Conditions: discoveryv1.EndpointConditions{Ready: &ready},
			}},
		}}
	}

	return snap
}

func rules(apiGroups, resources []string) []admissionv1.RuleWithOperations {
	return []admissionv1.RuleWithOperations{{
		Operations: []admissionv1.OperationType{admissionv1.Create},
		Rule: admissionv1.Rule{
			APIGroups:   apiGroups,
			APIVersions: []string{"*"},
			Resources:   resources,
		},
	}}
}

func findingWithMessage(t *testing.T, findings []upgradev1alpha1.Finding, fragment string) upgradev1alpha1.Finding {
	t.Helper()
	for _, finding := range findings {
		if strings.Contains(finding.Message, fragment) {
			return finding
		}
	}
	t.Fatalf("expected a finding mentioning %q in %#v", fragment, findings)
	return upgradev1alpha1.Finding{}
}
