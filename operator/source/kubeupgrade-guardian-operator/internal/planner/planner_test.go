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

package planner

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
)

func TestBuildSpecGeneratesRequiredActions(t *testing.T) {
	assessment := &upgradev1alpha1.UpgradeAssessment{
		ObjectMeta: metav1.ObjectMeta{Name: "prod-upgrade-assessment", Namespace: "platform"},
		Spec: upgradev1alpha1.UpgradeAssessmentSpec{
			SourceVersion: "1.34",
			TargetVersion: "1.35",
		},
	}
	findings := []upgradev1alpha1.Finding{
		{
			ID:             "PDB_BLOCKING_001",
			Type:           upgradev1alpha1.FindingTypePDBBlockingRisk,
			Severity:       upgradev1alpha1.RiskLevelCritical,
			Category:       "PDB",
			Resource:       &upgradev1alpha1.ResourceRef{APIVersion: "apps/v1", Kind: "Deployment", Namespace: "production", Name: "payment-api"},
			Evidence:       []upgradev1alpha1.Evidence{{ID: "PDB_BLOCKING_001_EVIDENCE"}},
			Recommendation: "Increase replicas or relax the PodDisruptionBudget before upgrade.",
		},
		{
			ID:       "OBSERVABILITY_DETECTED",
			Type:     upgradev1alpha1.FindingTypeObservabilityCapability,
			Severity: upgradev1alpha1.RiskLevelInfo,
			Category: "Observability",
		},
	}

	spec := BuildSpec(
		assessment,
		upgradev1alpha1.DecisionDoNotUpgrade,
		upgradev1alpha1.RiskLevelHigh,
		25,
		upgradev1alpha1.FindingSummary{TotalFindings: 2, Critical: 1, Info: 1},
		upgradev1alpha1.FindingSummary{TotalFindings: 2, Critical: 1, Info: 1},
		upgradev1alpha1.ClassificationSummary{Total: 2, Blocking: 1, Informational: 1},
		findings,
	)

	if spec.AssessmentRef.Name != "prod-upgrade-assessment" || spec.AssessmentRef.Namespace != "platform" {
		t.Fatalf("unexpected assessment ref: %#v", spec.AssessmentRef)
	}
	if spec.Decision != upgradev1alpha1.DecisionDoNotUpgrade || spec.RiskLevel != upgradev1alpha1.RiskLevelHigh || spec.Score != 25 {
		t.Fatalf("unexpected plan header: %#v", spec)
	}
	if len(spec.RequiredActions) != 1 {
		t.Fatalf("expected one non-info action, got %d", len(spec.RequiredActions))
	}
	action := spec.RequiredActions[0]
	if action.ID != "remediate-pdb-blocking-001" || action.Action != findings[0].Recommendation {
		t.Fatalf("unexpected action: %#v", action)
	}
	if len(action.EvidenceRefs) != 1 || action.EvidenceRefs[0] != "PDB_BLOCKING_001_EVIDENCE" {
		t.Fatalf("unexpected evidence refs: %#v", action.EvidenceRefs)
	}
	if len(spec.RecommendedOrder) != len(defaultRecommendedOrder) {
		t.Fatalf("unexpected recommended order: %#v", spec.RecommendedOrder)
	}
	if spec.SourceVersion != "1.34" || spec.TargetVersion != "1.35" {
		t.Fatalf("unexpected version fields: %#v", spec)
	}
	if len(spec.UpgradePath) != 1 || spec.UpgradePath[0].From != "1.34" || spec.UpgradePath[0].To != "1.35" {
		t.Fatalf("unexpected upgrade path: %#v", spec.UpgradePath)
	}
}

func TestBuildUpgradePathCreatesMinorVersionHops(t *testing.T) {
	path := BuildUpgradePath("1.30", "1.32")
	if len(path) != 2 {
		t.Fatalf("expected two version hops, got %#v", path)
	}
	if path[0].From != "1.30" || path[0].To != "1.31" || path[1].From != "1.31" || path[1].To != "1.32" {
		t.Fatalf("unexpected path: %#v", path)
	}
	if len(path[0].Phases) == 0 {
		t.Fatalf("expected phases in path: %#v", path)
	}
}
