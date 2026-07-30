/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package checkers

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	resourceapi "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/snapshot"
)

func TestWorkloadAvailabilityDetectsDeploymentReplicasBelowTwo(t *testing.T) {
	snap := &snapshot.ClusterSnapshot{
		Namespaces:  []corev1.Namespace{namespace("production", nil)},
		Deployments: []appsv1.Deployment{deployment("production", "payment-api", 1)},
	}

	findings := WorkloadAvailability{}.Check(snap, assessment("production"))
	assertFinding(t, findings, upgradev1alpha1.FindingTypeWorkloadAvailability, upgradev1alpha1.RiskLevelHigh)
	if got := findings[0].Evidence[0].Observed["replicas"]; got != "1" {
		t.Fatalf("expected replicas evidence 1, got %q", got)
	}
}

func TestWorkloadAvailabilityIgnoresStandalonePodsOutsideScope(t *testing.T) {
	snap := &snapshot.ClusterSnapshot{
		Namespaces: []corev1.Namespace{namespace("production", nil)},
		Pods: []corev1.Pod{
			{ObjectMeta: metav1.ObjectMeta{Name: "debug", Namespace: "production"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "debug", Namespace: "sandbox"}},
		},
	}

	findings := WorkloadAvailability{}.Check(snap, assessment("production"))
	if len(findings) != 1 {
		t.Fatalf("expected only the in-scope standalone pod, got %#v", findings)
	}
	if findings[0].Resource.Namespace != "production" {
		t.Fatalf("expected the production pod, got %#v", findings[0].Resource)
	}
}

func TestReadinessProbeDetectsMissingProbe(t *testing.T) {
	snap := &snapshot.ClusterSnapshot{
		Namespaces: []corev1.Namespace{namespace("production", nil)},
		Deployments: []appsv1.Deployment{{
			ObjectMeta: metav1.ObjectMeta{Name: "payment-api", Namespace: "production"},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "api", Image: "example/api:latest"}},
					},
				},
			},
		}},
	}

	findings := ReadinessProbe{}.Check(snap, assessment("production"))
	assertFinding(t, findings, upgradev1alpha1.FindingTypeMissingReadinessProbe, upgradev1alpha1.RiskLevelMedium)
	if got := findings[0].Evidence[0].Observed["containerName"]; got != "api" {
		t.Fatalf("expected containerName evidence api, got %q", got)
	}
}

func TestPDBDetectsMinAvailableBlockingSingleReplica(t *testing.T) {
	minAvailable := intstr.FromInt32(1)
	workload := deployment("production", "payment-api", 1)
	workload.Spec.Template.Labels = map[string]string{"app": "payment-api"}

	snap := &snapshot.ClusterSnapshot{
		Namespaces:  []corev1.Namespace{namespace("production", nil)},
		Deployments: []appsv1.Deployment{workload},
		PDBs: []policyv1.PodDisruptionBudget{{
			ObjectMeta: metav1.ObjectMeta{Name: "payment-api", Namespace: "production"},
			Spec: policyv1.PodDisruptionBudgetSpec{
				MinAvailable: &minAvailable,
				Selector:     &metav1.LabelSelector{MatchLabels: map[string]string{"app": "payment-api"}},
			},
		}},
	}

	findings := PDB{}.Check(snap, assessment("production"))
	assertFinding(t, findings, upgradev1alpha1.FindingTypePDBBlockingRisk, upgradev1alpha1.RiskLevelCritical)
	if got := findings[0].Evidence[0].Observed["minAvailable"]; got != "1" {
		t.Fatalf("expected minAvailable evidence 1, got %q", got)
	}
}

func TestPDBDoesNotMatchWorkloadsOfAnotherNamespace(t *testing.T) {
	minAvailable := intstr.FromInt32(1)
	workload := deployment("staging", "payment-api", 1)
	workload.Spec.Template.Labels = map[string]string{"app": "payment-api"}

	snap := &snapshot.ClusterSnapshot{
		Namespaces:  []corev1.Namespace{namespace("production", nil), namespace("staging", nil)},
		Deployments: []appsv1.Deployment{workload},
		PDBs: []policyv1.PodDisruptionBudget{{
			ObjectMeta: metav1.ObjectMeta{Name: "payment-api", Namespace: "production"},
			Spec: policyv1.PodDisruptionBudgetSpec{
				MinAvailable: &minAvailable,
				Selector:     &metav1.LabelSelector{MatchLabels: map[string]string{"app": "payment-api"}},
			},
		}},
	}

	findings := PDB{}.Check(snap, assessment())
	for _, finding := range findings {
		if strings.Contains(finding.Message, "may block disruption") {
			t.Fatalf("PDB matched a workload of another namespace: %s", finding.Message)
		}
	}
	assertFinding(t, findings, upgradev1alpha1.FindingTypePDBBlockingRisk, upgradev1alpha1.RiskLevelHigh)
}

func TestPolicyRiskDetectsRestrictedNamespaceAndPrivilegedWorkload(t *testing.T) {
	privileged := true
	snap := &snapshot.ClusterSnapshot{
		Namespaces: []corev1.Namespace{namespace("production", map[string]string{"pod-security.kubernetes.io/enforce": "restricted"})},
		Deployments: []appsv1.Deployment{{
			ObjectMeta: metav1.ObjectMeta{Name: "payment-api", Namespace: "production"},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:            "api",
							Image:           "example/api:latest",
							SecurityContext: &corev1.SecurityContext{Privileged: &privileged},
						}},
					},
				},
			},
		}},
	}

	findings := PolicyRisk{}.Check(snap, assessment("production"))
	assertFinding(t, findings, upgradev1alpha1.FindingTypePolicyRisk, upgradev1alpha1.RiskLevelMedium)
	assertFinding(t, findings, upgradev1alpha1.FindingTypePolicyRisk, upgradev1alpha1.RiskLevelHigh)
}

func TestCapacityDetectsInsufficientOneNodeHeadroom(t *testing.T) {
	snap := &snapshot.ClusterSnapshot{
		Nodes: []corev1.Node{node("node-a", "2000m", "4Gi"), node("node-b", "2000m", "4Gi")},
		Pods: []corev1.Pod{{
			ObjectMeta: metav1.ObjectMeta{Name: "payment-api", Namespace: "production"},
			Spec: corev1.PodSpec{Containers: []corev1.Container{{
				Name: "api",
				Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resourceapi.MustParse("3500m"),
					corev1.ResourceMemory: resourceapi.MustParse("2Gi"),
				}},
			}}},
		}},
	}

	findings := Capacity{}.Check(snap, &upgradev1alpha1.UpgradeAssessment{})
	assertFinding(t, findings, upgradev1alpha1.FindingTypeCapacityDrainRisk, upgradev1alpha1.RiskLevelHigh)
}

func TestObservabilityDetectsMissingMonitoringNamespace(t *testing.T) {
	snap := &snapshot.ClusterSnapshot{
		AllNamespaces: []corev1.Namespace{namespace("production", nil)},
		Namespaces:    []corev1.Namespace{namespace("production", nil)},
		CRDNames:      map[string]struct{}{},
	}

	findings := Observability{}.Check(snap, &upgradev1alpha1.UpgradeAssessment{})
	assertFinding(t, findings, upgradev1alpha1.FindingTypeObservabilityGap, upgradev1alpha1.RiskLevelMedium)
}

func TestObservabilityUsesUnscopedNamespaces(t *testing.T) {
	snap := &snapshot.ClusterSnapshot{
		AllNamespaces: []corev1.Namespace{namespace("production", nil), namespace("monitoring", nil)},
		Namespaces:    []corev1.Namespace{namespace("production", nil)},
		CRDNames:      map[string]struct{}{},
	}

	findings := Observability{}.Check(snap, assessment("production"))
	for _, finding := range findings {
		if strings.Contains(finding.Message, "No monitoring, prometheus, or observability namespace") {
			t.Fatalf("monitoring namespace outside the assessed scope must still count as present")
		}
	}
}

// TestCheckersNeverReachTheAPIServer enforces the single-read-point rule: every
// cluster fact a checker uses has to come from the snapshot.
func TestCheckersNeverReachTheAPIServer(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}

	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(".", name), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range file.Imports {
			path := strings.Trim(imported.Path.Value, `"`)
			if strings.HasPrefix(path, "sigs.k8s.io/controller-runtime/pkg/client") {
				t.Errorf("%s imports %s: checkers must read the cluster through internal/snapshot only", name, path)
			}
		}
	}
}

func assessment(namespaces ...string) *upgradev1alpha1.UpgradeAssessment {
	return &upgradev1alpha1.UpgradeAssessment{
		Spec: upgradev1alpha1.UpgradeAssessmentSpec{
			TargetVersion: "1.32",
			Mode:          upgradev1alpha1.AssessmentModeReadOnly,
			Scope: upgradev1alpha1.AssessmentScope{
				Namespaces: upgradev1alpha1.NamespaceScope{Include: namespaces},
			},
		},
	}
}

func namespace(name string, labels map[string]string) corev1.Namespace {
	return corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels}}
}

func deployment(namespace, name string, replicas int32) appsv1.Deployment {
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": name}},
			},
		},
	}
}

func node(name, cpu, memory string) corev1.Node {
	return corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: corev1.NodeStatus{Allocatable: corev1.ResourceList{
			corev1.ResourceCPU:    resourceapi.MustParse(cpu),
			corev1.ResourceMemory: resourceapi.MustParse(memory),
		}},
	}
}

func assertFinding(t *testing.T, findings []upgradev1alpha1.Finding, findingType string, severity upgradev1alpha1.RiskLevel) {
	t.Helper()
	for _, finding := range findings {
		if finding.Type == findingType && finding.Severity == severity {
			return
		}
	}
	t.Fatalf("expected finding type=%s severity=%s in %#v", findingType, severity, findings)
}
