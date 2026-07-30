package checkers

import (
	"strings"
	"testing"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/snapshot"
)

func TestRemovedAPITableIsSeededFromTheEmbeddedFile(t *testing.T) {
	want := []removedAPI{
		{APIVersion: "policy/v1beta1", Kind: "PodDisruptionBudget", RemovedInMinor: 25},
		{APIVersion: "policy/v1beta1", Kind: "PodSecurityPolicy", RemovedInMinor: 25},
		{APIVersion: "batch/v1beta1", Kind: "CronJob", RemovedInMinor: 25},
		{APIVersion: "autoscaling/v2beta1", Kind: "HorizontalPodAutoscaler", RemovedInMinor: 25},
		{APIVersion: "autoscaling/v2beta2", Kind: "HorizontalPodAutoscaler", RemovedInMinor: 26},
		{APIVersion: "flowcontrol.apiserver.k8s.io/v1beta1", Kind: "FlowSchema", RemovedInMinor: 26},
		{APIVersion: "flowcontrol.apiserver.k8s.io/v1beta1", Kind: "PriorityLevelConfiguration", RemovedInMinor: 26},
		{APIVersion: "storage.k8s.io/v1beta1", Kind: "CSIStorageCapacity", RemovedInMinor: 27},
		{APIVersion: "flowcontrol.apiserver.k8s.io/v1beta2", Kind: "FlowSchema", RemovedInMinor: 29},
		{APIVersion: "flowcontrol.apiserver.k8s.io/v1beta3", Kind: "FlowSchema", RemovedInMinor: 32},
	}

	for _, entry := range want {
		if !tableContains(entry) {
			t.Errorf("removal table is missing %s %s (removed in 1.%d)", entry.APIVersion, entry.Kind, entry.RemovedInMinor)
		}
	}
}

func tableContains(want removedAPI) bool {
	for _, api := range removedAPIs[want.Kind] {
		if api == want {
			return true
		}
	}
	return false
}

func TestDeprecatedAPIDetectsRemovedVersionInManagedFields(t *testing.T) {
	snap := &snapshot.ClusterSnapshot{
		PDBs: []policyv1.PodDisruptionBudget{{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "production",
				Name:      "payment-api",
				ManagedFields: []metav1.ManagedFieldsEntry{
					{Manager: "kubectl-client-side-apply", APIVersion: "policy/v1beta1"},
					{Manager: "kube-controller-manager", APIVersion: "policy/v1"},
				},
			},
		}},
	}

	findings := DeprecatedAPI{}.Check(snap, assessmentTargeting("1.32"))
	if len(findings) != 1 {
		t.Fatalf("expected one deprecated API finding, got %#v", findings)
	}
	finding := findings[0]
	if finding.Severity != upgradev1alpha1.RiskLevelCritical {
		t.Fatalf("policy/v1beta1 is already gone at 1.32, expected Critical, got %s", finding.Severity)
	}
	observed := finding.Evidence[0].Observed
	if observed["apiVersion"] != "policy/v1beta1" {
		t.Fatalf("expected the removed API version in evidence, got %q", observed["apiVersion"])
	}
	if observed["fieldManagers"] != "kubectl-client-side-apply" {
		t.Fatalf("expected the offending field manager in evidence, got %q", observed["fieldManagers"])
	}
}

func TestDeprecatedAPIIsHighWhenRemovalIsStillAhead(t *testing.T) {
	snap := &snapshot.ClusterSnapshot{
		HPAs: []autoscalingv2.HorizontalPodAutoscaler{{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:     "production",
				Name:          "payment-api",
				ManagedFields: []metav1.ManagedFieldsEntry{{Manager: "helm", APIVersion: "autoscaling/v2beta2"}},
			},
		}},
	}

	findings := DeprecatedAPI{}.Check(snap, assessmentTargeting("1.25"))
	if len(findings) != 1 {
		t.Fatalf("expected one deprecated API finding, got %#v", findings)
	}
	if findings[0].Severity != upgradev1alpha1.RiskLevelHigh {
		t.Fatalf("autoscaling/v2beta2 survives 1.25, expected High, got %s", findings[0].Severity)
	}
}

func TestDeprecatedAPIDetectsRemovedVersionInLastAppliedConfiguration(t *testing.T) {
	snap := &snapshot.ClusterSnapshot{
		CronJobs: []batchv1.CronJob{{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "production",
				Name:      "nightly",
				Annotations: map[string]string{
					lastAppliedAnnotation: `{"apiVersion":"batch/v1beta1","kind":"CronJob"}`,
				},
			},
		}},
	}

	findings := DeprecatedAPI{}.Check(snap, assessmentTargeting("1.32"))
	if len(findings) != 1 {
		t.Fatalf("expected one deprecated API finding, got %#v", findings)
	}
	if got := findings[0].Evidence[0].Observed["fieldManagers"]; got != "kubectl-last-applied" {
		t.Fatalf("expected the last-applied writer in evidence, got %q", got)
	}
}

func TestDeprecatedAPIStaysSilentOnMigratedObjects(t *testing.T) {
	snap := &snapshot.ClusterSnapshot{
		PDBs: []policyv1.PodDisruptionBudget{{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:     "production",
				Name:          "payment-api",
				ManagedFields: []metav1.ManagedFieldsEntry{{Manager: "helm", APIVersion: "policy/v1"}},
				Annotations: map[string]string{
					lastAppliedAnnotation: `{"apiVersion":"policy/v1","kind":"PodDisruptionBudget"}`,
				},
			},
		}},
		CronJobs: []batchv1.CronJob{{
			ObjectMeta: metav1.ObjectMeta{Namespace: "production", Name: "nightly"},
		}},
	}

	if findings := (DeprecatedAPI{}).Check(snap, assessmentTargeting("1.32")); len(findings) != 0 {
		t.Fatalf("objects written through a served version must not be reported, got %#v", findings)
	}
}

func TestDeprecatedAPIIgnoresKindsAbsentFromTheSnapshot(t *testing.T) {
	// PodSecurityPolicy and FlowSchema are in the table but have no served
	// counterpart in the snapshot; they must simply never match.
	if findings := (DeprecatedAPI{}).Check(&snapshot.ClusterSnapshot{}, assessmentTargeting("1.32")); len(findings) != 0 {
		t.Fatalf("an empty snapshot must produce no finding, got %#v", findings)
	}
}

func TestUsesDeprecatedSourceRequiresLastAppliedDeprecatedAPIVersion(t *testing.T) {
	api := removedAPI{APIVersion: "policy/v1beta1", Kind: "PodDisruptionBudget", RemovedInMinor: 25}

	tests := []struct {
		name       string
		annotation string
		want       bool
	}{
		{
			name:       "legacy source manifest",
			annotation: `{"apiVersion":"policy/v1beta1","kind":"PodDisruptionBudget","metadata":{"name":"legacy"}}`,
			want:       true,
		},
		{
			name:       "modern source manifest served through legacy endpoint",
			annotation: `{"apiVersion":"policy/v1","kind":"PodDisruptionBudget","metadata":{"name":"modern"}}`,
			want:       false,
		},
		{
			name:       "missing source annotation",
			annotation: "",
			want:       false,
		},
		{
			name:       "invalid source annotation",
			annotation: `not-json`,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var annotations map[string]string
			if tt.annotation != "" {
				annotations = map[string]string{lastAppliedAnnotation: tt.annotation}
			}
			if got := usesDeprecatedSource(annotations, api); got != tt.want {
				t.Fatalf("usesDeprecatedSource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTargetMinorParsesTheMinorRelease(t *testing.T) {
	tests := map[string]int{"1.32": 32, "v1.29": 29, "1.9": 9, "": 0, "garbage": 0}
	for version, want := range tests {
		if got := targetMinor(version); got != want {
			t.Errorf("targetMinor(%q) = %d, want %d", version, got, want)
		}
	}
}

func assessmentTargeting(version string) *upgradev1alpha1.UpgradeAssessment {
	return &upgradev1alpha1.UpgradeAssessment{
		Spec: upgradev1alpha1.UpgradeAssessmentSpec{TargetVersion: version},
	}
}

func TestDeprecatedAPIMessageNamesTheRemovalRelease(t *testing.T) {
	snap := &snapshot.ClusterSnapshot{
		PDBs: []policyv1.PodDisruptionBudget{{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:     "production",
				Name:          "payment-api",
				ManagedFields: []metav1.ManagedFieldsEntry{{Manager: "helm", APIVersion: "policy/v1beta1"}},
			},
		}},
	}

	findings := DeprecatedAPI{}.Check(snap, assessmentTargeting("1.32"))
	if !strings.Contains(findings[0].Message, "removed in Kubernetes 1.25") {
		t.Fatalf("expected the removal release in the message, got %q", findings[0].Message)
	}
}
