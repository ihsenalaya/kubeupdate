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

package export

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/reporting"
)

func TestDefaultRenderersPublishTheThreeArtifactKeys(t *testing.T) {
	want := map[string]bool{
		reporting.AssessmentMarkdownKey: false,
		reporting.PlanMarkdownKey:       false,
		JSONKey:                         false,
	}

	for _, renderer := range Default() {
		if _, ok := want[renderer.Key()]; !ok {
			t.Fatalf("unexpected artifact key %q", renderer.Key())
		}
		want[renderer.Key()] = true
	}

	for key, seen := range want {
		if !seen {
			t.Fatalf("artifact key %q is not published", key)
		}
	}
}

func TestMarkdownRenderersMatchTheReportingPackage(t *testing.T) {
	assessment, plan := fixture()

	rendered, err := AssessmentMarkdown{}.Render(assessment, plan)
	if err != nil {
		t.Fatal(err)
	}
	if string(rendered) != reporting.AssessmentMarkdown(assessment) {
		t.Fatal("assessment markdown must be byte-identical to the reporting output")
	}

	rendered, err = PlanMarkdown{}.Render(assessment, plan)
	if err != nil {
		t.Fatal(err)
	}
	if string(rendered) != reporting.PlanMarkdown(plan) {
		t.Fatal("plan markdown must be byte-identical to the reporting output")
	}
}

func TestJSONRenderCarriesTheDecisionAndItsEvidence(t *testing.T) {
	assessment, plan := fixture()

	rendered, err := JSON{}.Render(assessment, plan)
	if err != nil {
		t.Fatal(err)
	}

	var document map[string]any
	if err := json.Unmarshal(rendered, &document); err != nil {
		t.Fatalf("export is not valid JSON: %v", err)
	}

	assertField(t, document, "exportVersion", JSONVersion)
	assertField(t, document, "takenAt", "2026-07-30T08:00:00Z")
	assertField(t, document, "targetVersion", "1.32")
	assertField(t, document, "sourceVersion", "1.31")
	assertField(t, document, "profile", "production")
	assertField(t, document, "decision", string(upgradev1alpha1.DecisionDoNotUpgrade))
	assertField(t, document, "riskLevel", string(upgradev1alpha1.RiskLevelCritical))

	if score, ok := document["score"].(float64); !ok || int(score) != 42 {
		t.Fatalf("expected score 42, got %v", document["score"])
	}

	assessmentRef, ok := document["assessment"].(map[string]any)
	if !ok || assessmentRef["name"] != "prod-upgrade" || assessmentRef["namespace"] != "guardian" {
		t.Fatalf("unexpected assessment reference %v", document["assessment"])
	}
	planRef, ok := document["plan"].(map[string]any)
	if !ok || planRef["name"] != "prod-upgrade-plan" {
		t.Fatalf("unexpected plan reference %v", document["plan"])
	}

	findings, ok := document["findings"].([]any)
	if !ok || len(findings) != 1 {
		t.Fatalf("expected one published finding, got %v", document["findings"])
	}
	finding, _ := findings[0].(map[string]any)
	classification, ok := finding["classification"].(map[string]any)
	if !ok || classification["status"] != string(upgradev1alpha1.FindingClassificationBlocking) {
		t.Fatalf("published findings must carry their full classification, got %v", finding["classification"])
	}
	if evidence, ok := finding["evidence"].([]any); !ok || len(evidence) != 1 {
		t.Fatalf("published findings must keep their evidence, got %v", finding["evidence"])
	}

	for _, key := range []string{"summary", "rawSummary", "classificationSummary"} {
		if _, ok := document[key].(map[string]any); !ok {
			t.Fatalf("expected %s in the export, got %v", key, document[key])
		}
	}
	if actions, ok := document["requiredActions"].([]any); !ok || len(actions) != 1 {
		t.Fatalf("expected the plan actions in the export, got %v", document["requiredActions"])
	}
	if path, ok := document["upgradePath"].([]any); !ok || len(path) != 1 {
		t.Fatalf("expected the upgrade path in the export, got %v", document["upgradePath"])
	}
	if order, ok := document["recommendedOrder"].([]any); !ok || len(order) != 2 {
		t.Fatalf("expected the recommended order in the export, got %v", document["recommendedOrder"])
	}
}

func TestJSONRenderEmitsEmptyCollectionsRatherThanNull(t *testing.T) {
	rendered, err := JSON{}.Render(&upgradev1alpha1.UpgradeAssessment{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	var document map[string]any
	if err := json.Unmarshal(rendered, &document); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"findings", "requiredActions", "upgradePath", "recommendedOrder"} {
		if _, ok := document[key].([]any); !ok {
			t.Fatalf("expected %s to be an array, got %v", key, document[key])
		}
	}
	if _, ok := document["takenAt"]; ok {
		t.Fatal("takenAt must be omitted when no snapshot time was recorded")
	}
}

func TestJSONRenderIsDeterministic(t *testing.T) {
	assessment, plan := fixture()

	first, err := JSON{}.Render(assessment, plan)
	if err != nil {
		t.Fatal(err)
	}
	second, err := JSON{}.Render(assessment, plan)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("the JSON export must be byte-stable across renders")
	}
}

func fixture() (*upgradev1alpha1.UpgradeAssessment, *upgradev1alpha1.UpgradePlan) {
	takenAt := metav1.NewTime(time.Date(2026, 7, 30, 8, 0, 0, 0, time.UTC))

	assessment := &upgradev1alpha1.UpgradeAssessment{
		ObjectMeta: metav1.ObjectMeta{Name: "prod-upgrade", Namespace: "guardian"},
		Spec: upgradev1alpha1.UpgradeAssessmentSpec{
			SourceVersion: "1.31",
			TargetVersion: "1.32",
			Profile:       upgradev1alpha1.AssessmentProfileProduction,
		},
		Status: upgradev1alpha1.UpgradeAssessmentStatus{
			Phase:            upgradev1alpha1.AssessmentPhaseCompleted,
			RiskLevel:        upgradev1alpha1.RiskLevelCritical,
			Score:            42,
			LastAssessedTime: &takenAt,
			Summary:          upgradev1alpha1.FindingSummary{TotalFindings: 1, Critical: 1},
			RawSummary:       upgradev1alpha1.FindingSummary{TotalFindings: 2, Critical: 1, Info: 1},
			ClassificationSummary: upgradev1alpha1.ClassificationSummary{
				Total: 2, Blocking: 1, Informational: 1,
			},
			Findings: []upgradev1alpha1.Finding{{
				ID:       "PDB_BLOCKING_RISK_PRODUCTION_PAYMENT_API",
				Type:     upgradev1alpha1.FindingTypePDBBlockingRisk,
				Severity: upgradev1alpha1.RiskLevelCritical,
				Category: "PDB",
				Resource: &upgradev1alpha1.ResourceRef{
					APIVersion: "policy/v1", Kind: "PodDisruptionBudget",
					Namespace: "production", Name: "payment-api",
				},
				Message: "PDB production/payment-api may block disruption.",
				Evidence: []upgradev1alpha1.Evidence{{
					ID:          "PDB_BLOCKING_RISK_EVIDENCE",
					Description: "PDB and workload replica relationship.",
					Observed:    map[string]string{"replicas": "1", "minAvailable": "1"},
				}},
				Recommendation: "Increase workload replicas or relax the PodDisruptionBudget.",
				Classification: &upgradev1alpha1.FindingClassification{
					Status:      upgradev1alpha1.FindingClassificationBlocking,
					Reason:      "Universal upgrade blocker.",
					MatchedRule: "universal",
				},
			}},
		},
	}

	plan := &upgradev1alpha1.UpgradePlan{
		ObjectMeta: metav1.ObjectMeta{Name: "prod-upgrade-plan", Namespace: "guardian"},
		Spec: upgradev1alpha1.UpgradePlanSpec{
			AssessmentRef: upgradev1alpha1.AssessmentReference{Name: "prod-upgrade", Namespace: "guardian"},
			Decision:      upgradev1alpha1.DecisionDoNotUpgrade,
			RiskLevel:     upgradev1alpha1.RiskLevelCritical,
			Score:         42,
			SourceVersion: "1.31",
			TargetVersion: "1.32",
			RequiredActions: []upgradev1alpha1.RequiredAction{{
				ID:       "PDB_BLOCKING_RISK_PRODUCTION_PAYMENT_API",
				Severity: upgradev1alpha1.RiskLevelCritical,
				Category: "PDB",
				Action:   "Increase workload replicas or relax the PodDisruptionBudget.",
			}},
			UpgradePath: []upgradev1alpha1.UpgradePathStep{{
				From:   "1.31",
				To:     "1.32",
				Phases: []upgradev1alpha1.UpgradePhase{{Name: "prechecks"}},
			}},
			RecommendedOrder: []string{"prechecks", "control-plane-upgrade"},
		},
	}

	return assessment, plan
}

func assertField(t *testing.T, document map[string]any, key, want string) {
	t.Helper()
	if got, _ := document[key].(string); got != want {
		t.Fatalf("expected %s=%q, got %q", key, want, got)
	}
}
