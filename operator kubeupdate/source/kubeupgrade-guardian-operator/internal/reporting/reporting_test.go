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

package reporting

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
)

func TestAssessmentMarkdownContainsHumanSections(t *testing.T) {
	assessment := &upgradev1alpha1.UpgradeAssessment{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "platform"},
		Spec: upgradev1alpha1.UpgradeAssessmentSpec{
			SourceVersion: "1.34",
			TargetVersion: "1.35",
			Profile:       upgradev1alpha1.AssessmentProfileProduction,
		},
		Status: upgradev1alpha1.UpgradeAssessmentStatus{
			Phase:     upgradev1alpha1.AssessmentPhaseCompleted,
			RiskLevel: upgradev1alpha1.RiskLevelHigh,
			Score:     10,
			Summary:   upgradev1alpha1.FindingSummary{TotalFindings: 1, High: 1},
			Findings: []upgradev1alpha1.Finding{{
				Severity: upgradev1alpha1.RiskLevelHigh,
				Category: "Capacity",
				Message:  "Cluster has limited headroom.",
				Classification: &upgradev1alpha1.FindingClassification{
					Status: upgradev1alpha1.FindingClassificationBlocking,
				},
			}},
		},
	}

	rendered := AssessmentMarkdown(assessment)
	for _, want := range []string{"# Upgrade Assessment", "Effective Finding Summary", "Classification Summary", "Cluster has limited headroom."} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected rendered assessment to contain %q:\n%s", want, rendered)
		}
	}
}

func TestPlanMarkdownContainsChronology(t *testing.T) {
	plan := &upgradev1alpha1.UpgradePlan{
		ObjectMeta: metav1.ObjectMeta{Name: "demo-plan", Namespace: "platform"},
		Spec: upgradev1alpha1.UpgradePlanSpec{
			Decision:      upgradev1alpha1.DecisionProceedWithCaution,
			RiskLevel:     upgradev1alpha1.RiskLevelHigh,
			SourceVersion: "1.34",
			TargetVersion: "1.35",
			UpgradePath: []upgradev1alpha1.UpgradePathStep{{
				From: "1.34",
				To:   "1.35",
				Phases: []upgradev1alpha1.UpgradePhase{{
					Name: "prechecks",
				}},
			}},
		},
	}

	rendered := PlanMarkdown(plan)
	for _, want := range []string{"# Upgrade Plan", "Upgrade Chronology", "1.34 -> 1.35", "prechecks"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected rendered plan to contain %q:\n%s", want, rendered)
		}
	}
}

func TestPlanMarkdownBlockedDecisionOnCriticalRisk(t *testing.T) {
	plan := &upgradev1alpha1.UpgradePlan{
		ObjectMeta: metav1.ObjectMeta{Name: "prod-plan", Namespace: "platform"},
		Spec: upgradev1alpha1.UpgradePlanSpec{
			Decision:  upgradev1alpha1.DecisionProceedWithCaution,
			RiskLevel: upgradev1alpha1.RiskLevelCritical,
			ClassificationSummary: upgradev1alpha1.ClassificationSummary{
				Total:    10,
				Blocking: 5,
			},
			BlockingFindings: []upgradev1alpha1.ClassifiedFindingRef{
				{
					Category: "AdmissionWebhook",
					Severity: upgradev1alpha1.RiskLevelHigh,
					Message:  "webhook has risk: failurePolicy=Fail.",
					Resource: upgradev1alpha1.ResourceRef{Kind: "MutatingWebhookConfiguration", Name: "cert-manager-webhook"},
					Classification: upgradev1alpha1.FindingClassification{
						Status: upgradev1alpha1.FindingClassificationBlocking,
					},
				},
			},
		},
	}

	rendered := PlanMarkdown(plan)
	if !strings.Contains(rendered, "NOT READY") {
		t.Fatalf("expected BLOCKED label for Critical risk with blocking findings:\n%s", rendered)
	}
	if !strings.Contains(rendered, "BLOCKED") {
		t.Fatalf("expected BLOCKED keyword in plan:\n%s", rendered)
	}
}

func TestPlanMarkdownExecutiveDecisionShowsBlockerCounts(t *testing.T) {
	plan := &upgradev1alpha1.UpgradePlan{
		ObjectMeta: metav1.ObjectMeta{Name: "plan", Namespace: "platform"},
		Spec: upgradev1alpha1.UpgradePlanSpec{
			Decision:  upgradev1alpha1.DecisionProceedWithCaution,
			RiskLevel: upgradev1alpha1.RiskLevelCritical,
			ClassificationSummary: upgradev1alpha1.ClassificationSummary{
				Total: 3, Blocking: 3,
			},
			BlockingFindings: []upgradev1alpha1.ClassifiedFindingRef{
				{Category: "AdmissionWebhook", Severity: upgradev1alpha1.RiskLevelHigh, Message: "failurePolicy=Fail.",
					Resource: upgradev1alpha1.ResourceRef{Kind: "MutatingWebhookConfiguration", Name: "cert-manager-webhook"},
					Classification: upgradev1alpha1.FindingClassification{Status: upgradev1alpha1.FindingClassificationBlocking}},
				{Category: "AdmissionWebhook", Severity: upgradev1alpha1.RiskLevelHigh, Message: "failurePolicy=Fail.",
					Resource: upgradev1alpha1.ResourceRef{Kind: "MutatingWebhookConfiguration", Name: "istio-sidecar-injector"},
					Classification: upgradev1alpha1.FindingClassification{Status: upgradev1alpha1.FindingClassificationBlocking}},
				{Category: "Capacity", Severity: upgradev1alpha1.RiskLevelHigh, Message: "Cluster may not have enough capacity.",
					Classification: upgradev1alpha1.FindingClassification{Status: upgradev1alpha1.FindingClassificationBlocking}},
			},
		},
	}

	rendered := PlanMarkdown(plan)
	if !strings.Contains(rendered, "Main blockers") {
		t.Fatalf("expected Main blockers section:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Executive Decision") {
		t.Fatalf("expected Executive Decision section:\n%s", rendered)
	}
}

func TestPlanMarkdownTopBlockersGroupsWebhooksByComponent(t *testing.T) {
	plan := &upgradev1alpha1.UpgradePlan{
		ObjectMeta: metav1.ObjectMeta{Name: "plan", Namespace: "platform"},
		Spec: upgradev1alpha1.UpgradePlanSpec{
			Decision:  upgradev1alpha1.DecisionProceedWithCaution,
			RiskLevel: upgradev1alpha1.RiskLevelHigh,
			BlockingFindings: []upgradev1alpha1.ClassifiedFindingRef{
				{Category: "AdmissionWebhook", Message: "failurePolicy=Fail.",
					Resource: upgradev1alpha1.ResourceRef{Kind: "MutatingWebhookConfiguration", Name: "cert-manager-webhook"},
					Classification: upgradev1alpha1.FindingClassification{Status: upgradev1alpha1.FindingClassificationBlocking}},
				{Category: "AdmissionWebhook", Message: "namespaceSelector is absent.",
					Resource: upgradev1alpha1.ResourceRef{Kind: "ValidatingWebhookConfiguration", Name: "cert-manager-webhook"},
					Classification: upgradev1alpha1.FindingClassification{Status: upgradev1alpha1.FindingClassificationBlocking}},
				{Category: "AdmissionWebhook", Message: "failurePolicy=Fail.",
					Resource: upgradev1alpha1.ResourceRef{Kind: "MutatingWebhookConfiguration", Name: "kyverno-policy-mutating-webhook-cfg"},
					Classification: upgradev1alpha1.FindingClassification{Status: upgradev1alpha1.FindingClassificationBlocking}},
			},
		},
	}

	rendered := PlanMarkdown(plan)
	if !strings.Contains(rendered, "Top Upgrade Blockers") {
		t.Fatalf("expected Top Upgrade Blockers section:\n%s", rendered)
	}
	// Component names should appear grouped, not each individual webhook config
	if !strings.Contains(rendered, "cert-manager") {
		t.Fatalf("expected cert-manager component in blockers:\n%s", rendered)
	}
	if !strings.Contains(rendered, "kyverno") {
		t.Fatalf("expected kyverno component in blockers:\n%s", rendered)
	}
}

func TestPlanMarkdownRemediationPriorities(t *testing.T) {
	plan := &upgradev1alpha1.UpgradePlan{
		ObjectMeta: metav1.ObjectMeta{Name: "plan", Namespace: "platform"},
		Spec: upgradev1alpha1.UpgradePlanSpec{
			Decision:  upgradev1alpha1.DecisionProceedWithCaution,
			RiskLevel: upgradev1alpha1.RiskLevelCritical,
			ClassificationSummary: upgradev1alpha1.ClassificationSummary{Blocking: 3},
			BlockingFindings: []upgradev1alpha1.ClassifiedFindingRef{
				{Category: "Capacity", Severity: upgradev1alpha1.RiskLevelHigh,
					Message: "Cluster may not have enough capacity.", Recommendation: "Add capacity.",
					Classification: upgradev1alpha1.FindingClassification{Status: upgradev1alpha1.FindingClassificationBlocking}},
				{Category: "WorkloadAvailability", Severity: upgradev1alpha1.RiskLevelHigh,
					Message: "Fewer than 2 replicas.", Recommendation: "Increase replicas.",
					Resource:       upgradev1alpha1.ResourceRef{Kind: "Deployment", Namespace: "cert-manager", Name: "cert-manager"},
					Classification: upgradev1alpha1.FindingClassification{Status: upgradev1alpha1.FindingClassificationBlocking}},
				{Category: "ReadinessProbes", Severity: upgradev1alpha1.RiskLevelMedium,
					Message: "No readinessProbe.", Recommendation: "Add a readinessProbe.",
					Resource:       upgradev1alpha1.ResourceRef{Kind: "Deployment", Namespace: "cert-manager", Name: "cert-manager-cainjector"},
					Classification: upgradev1alpha1.FindingClassification{Status: upgradev1alpha1.FindingClassificationBlocking}},
			},
		},
	}

	rendered := PlanMarkdown(plan)
	if !strings.Contains(rendered, "P0") {
		t.Fatalf("expected P0 section:\n%s", rendered)
	}
	if !strings.Contains(rendered, "P1") {
		t.Fatalf("expected P1 section:\n%s", rendered)
	}
	if !strings.Contains(rendered, "P2") {
		t.Fatalf("expected P2 section:\n%s", rendered)
	}
	// Capacity must appear in P0
	p0Start := strings.Index(rendered, "P0")
	p1Start := strings.Index(rendered, "P1")
	if p0Start < 0 || p1Start < 0 {
		t.Fatalf("missing P0 or P1 sections")
	}
	if !strings.Contains(rendered[p0Start:p1Start], "Capacity") {
		t.Fatalf("expected Capacity in P0 section")
	}
}

func TestPlanMarkdownGoNoGoGatesFailOnBlockers(t *testing.T) {
	plan := &upgradev1alpha1.UpgradePlan{
		ObjectMeta: metav1.ObjectMeta{Name: "plan", Namespace: "platform"},
		Spec: upgradev1alpha1.UpgradePlanSpec{
			Decision:  upgradev1alpha1.DecisionProceedWithCaution,
			RiskLevel: upgradev1alpha1.RiskLevelCritical,
			ClassificationSummary: upgradev1alpha1.ClassificationSummary{Blocking: 2},
			BlockingFindings: []upgradev1alpha1.ClassifiedFindingRef{
				{Category: "AdmissionWebhook", Message: "webhook has risk: failurePolicy=Fail.",
					Resource: upgradev1alpha1.ResourceRef{Kind: "MutatingWebhookConfiguration", Name: "cert-manager-webhook"},
					Classification: upgradev1alpha1.FindingClassification{Status: upgradev1alpha1.FindingClassificationBlocking}},
				{Category: "Capacity", Message: "Cluster may not have enough capacity.",
					Recommendation: "Add capacity. Estimated remaining capacity after one-node loss: 1900m CPU.",
					Classification: upgradev1alpha1.FindingClassification{Status: upgradev1alpha1.FindingClassificationBlocking}},
			},
		},
	}

	rendered := PlanMarkdown(plan)
	if !strings.Contains(rendered, "Go/No-Go Gates") {
		t.Fatalf("expected Go/No-Go Gates section:\n%s", rendered)
	}
	if !strings.Contains(rendered, "NOT PASSED") {
		t.Fatalf("expected NOT PASSED gate result:\n%s", rendered)
	}
	if !strings.Contains(rendered, "1900m CPU") {
		t.Fatalf("expected capacity headroom extracted in gate details:\n%s", rendered)
	}
}

func TestPlanMarkdownGoNoGoGatesPassWhenNoBlockers(t *testing.T) {
	plan := &upgradev1alpha1.UpgradePlan{
		ObjectMeta: metav1.ObjectMeta{Name: "plan", Namespace: "platform"},
		Spec: upgradev1alpha1.UpgradePlanSpec{
			Decision:  upgradev1alpha1.DecisionProceed,
			RiskLevel: upgradev1alpha1.RiskLevelLow,
		},
	}

	rendered := PlanMarkdown(plan)
	if !strings.Contains(rendered, "PASSED") {
		t.Fatalf("expected PASSED gate result when no blockers:\n%s", rendered)
	}
}

func TestPlanMarkdownChronologyIncludesConditions(t *testing.T) {
	plan := &upgradev1alpha1.UpgradePlan{
		ObjectMeta: metav1.ObjectMeta{Name: "plan", Namespace: "platform"},
		Spec: upgradev1alpha1.UpgradePlanSpec{
			Decision:  upgradev1alpha1.DecisionProceedWithCaution,
			RiskLevel: upgradev1alpha1.RiskLevelCritical,
			ClassificationSummary: upgradev1alpha1.ClassificationSummary{Blocking: 1},
			BlockingFindings: []upgradev1alpha1.ClassifiedFindingRef{
				{Category: "AdmissionWebhook", Message: "webhook has risk: failurePolicy=Fail.",
					Resource: upgradev1alpha1.ResourceRef{Kind: "MutatingWebhookConfiguration", Name: "cert-manager-webhook"},
					Classification: upgradev1alpha1.FindingClassification{Status: upgradev1alpha1.FindingClassificationBlocking}},
			},
			UpgradePath: []upgradev1alpha1.UpgradePathStep{{
				From: "1.34",
				To:   "1.35",
				Phases: []upgradev1alpha1.UpgradePhase{
					{Name: "prechecks", Description: "Verify blocking findings."},
					{Name: "control-plane-upgrade"},
				},
			}},
		},
	}

	rendered := PlanMarkdown(plan)
	// Webhook blocker conditions must appear in the chronology
	if !strings.Contains(rendered, "failurePolicy=Ignore") {
		t.Fatalf("expected failurePolicy=Ignore condition in chronology:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Gate 1 passed") {
		t.Fatalf("expected Gate 1 passed condition in chronology:\n%s", rendered)
	}
}

func TestPlanMarkdownProviderManagedGroupsByResource(t *testing.T) {
	plan := &upgradev1alpha1.UpgradePlan{
		ObjectMeta: metav1.ObjectMeta{Name: "plan", Namespace: "platform"},
		Spec: upgradev1alpha1.UpgradePlanSpec{
			Decision:  upgradev1alpha1.DecisionProceedWithCaution,
			RiskLevel: upgradev1alpha1.RiskLevelHigh,
			ProviderManagedRisks: []upgradev1alpha1.ClassifiedFindingRef{
				{Category: "AdmissionWebhook",
					Message:  "aks-node-mutating-webhook has risk: failurePolicy=Fail.",
					Resource: upgradev1alpha1.ResourceRef{Kind: "MutatingWebhookConfiguration", Name: "aks-node-mutating-webhook"},
					Classification: upgradev1alpha1.FindingClassification{
						Status: upgradev1alpha1.FindingClassificationProviderManaged,
					}},
				{Category: "AdmissionWebhook",
					Message:  "aks-node-mutating-webhook has risk: namespaceSelector is absent.",
					Resource: upgradev1alpha1.ResourceRef{Kind: "MutatingWebhookConfiguration", Name: "aks-node-mutating-webhook"},
					Classification: upgradev1alpha1.FindingClassification{
						Status: upgradev1alpha1.FindingClassificationProviderManaged,
					}},
			},
		},
	}

	rendered := PlanMarkdown(plan)
	if !strings.Contains(rendered, "Provider Managed Risks") {
		t.Fatalf("expected Provider Managed Risks section:\n%s", rendered)
	}
	// Both risks should be on a single row (grouped by resource name)
	if strings.Count(rendered, "aks-node-mutating-webhook") < 1 {
		t.Fatalf("expected aks-node-mutating-webhook in provider managed section:\n%s", rendered)
	}
}

func TestComputeDecisionLabel(t *testing.T) {
	cases := []struct {
		name      string
		plan      *upgradev1alpha1.UpgradePlan
		wantLabel string
	}{
		{
			name: "critical_with_blockers_overrides_proceed_with_caution",
			plan: &upgradev1alpha1.UpgradePlan{Spec: upgradev1alpha1.UpgradePlanSpec{
				Decision:              upgradev1alpha1.DecisionProceedWithCaution,
				RiskLevel:             upgradev1alpha1.RiskLevelCritical,
				ClassificationSummary: upgradev1alpha1.ClassificationSummary{Blocking: 5},
			}},
			wantLabel: "NOT READY — BLOCKED",
		},
		{
			name: "capacity_blocker_triggers_blocked",
			plan: &upgradev1alpha1.UpgradePlan{Spec: upgradev1alpha1.UpgradePlanSpec{
				Decision:  upgradev1alpha1.DecisionProceedWithCaution,
				RiskLevel: upgradev1alpha1.RiskLevelHigh,
				BlockingFindings: []upgradev1alpha1.ClassifiedFindingRef{
					{Category: "Capacity", Classification: upgradev1alpha1.FindingClassification{Status: upgradev1alpha1.FindingClassificationBlocking}},
				},
			}},
			wantLabel: "NOT READY — BLOCKED",
		},
		{
			name: "do_not_upgrade_maps_to_blocked",
			plan: &upgradev1alpha1.UpgradePlan{Spec: upgradev1alpha1.UpgradePlanSpec{
				Decision: upgradev1alpha1.DecisionDoNotUpgrade,
			}},
			wantLabel: "NOT READY — BLOCKED",
		},
		{
			name: "proceed_maps_to_ready",
			plan: &upgradev1alpha1.UpgradePlan{Spec: upgradev1alpha1.UpgradePlanSpec{
				Decision: upgradev1alpha1.DecisionProceed,
			}},
			wantLabel: "READY — no blocking findings",
		},
		{
			name: "proceed_with_caution_no_critical_risk",
			plan: &upgradev1alpha1.UpgradePlan{Spec: upgradev1alpha1.UpgradePlanSpec{
				Decision:  upgradev1alpha1.DecisionProceedWithCaution,
				RiskLevel: upgradev1alpha1.RiskLevelHigh,
			}},
			wantLabel: "PROCEED WITH CAUTION — review findings before upgrade",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, got := computeDecisionLabel(tc.plan)
			if got != tc.wantLabel {
				t.Errorf("computeDecisionLabel() = %q, want %q", got, tc.wantLabel)
			}
		})
	}
}

func TestFindingPriority(t *testing.T) {
	cases := []struct {
		category string
		message  string
		want     string
	}{
		{"Capacity", "not enough headroom", "P0"},
		{"AdmissionWebhook", "webhook has risk: failurePolicy=Fail.", "P0"},
		{"AdmissionWebhook", "webhook has risk: namespaceSelector is absent.", "P1"},
		{"WorkloadAvailability", "fewer than 2 replicas", "P1"},
		{"PolicyRisk", "runAsNonRoot absent", "P1"},
		{"ReadinessProbes", "no readinessProbe", "P2"},
	}
	for _, tc := range cases {
		f := upgradev1alpha1.ClassifiedFindingRef{
			Category: tc.category,
			Message:  tc.message,
			Severity: upgradev1alpha1.RiskLevelHigh,
		}
		if f.Category == "PolicyRisk" {
			f.Severity = upgradev1alpha1.RiskLevelHigh
		}
		if got := findingPriority(f); got != tc.want {
			t.Errorf("findingPriority(%q, %q) = %q, want %q", tc.category, tc.message, got, tc.want)
		}
	}
}

func TestWebhookComponentFromName(t *testing.T) {
	cases := []struct {
		name      string
		want      string
	}{
		{"cert-manager-webhook", "cert-manager"},
		{"istio-sidecar-injector", "istio"},
		{"istiod-default-validator", "istio"},
		{"kyverno-policy-mutating-webhook-cfg", "kyverno"},
		{"externalsecret-validate", "external-secrets"},
		{"secretstore-validate", "external-secrets"},
		{"azure-wi-webhook-mutating-webhook-configuration", "azure-workload-identity"},
		{"unknown-webhook", "unknown-webhook"},
	}
	for _, tc := range cases {
		if got := webhookComponentFromName(tc.name); got != tc.want {
			t.Errorf("webhookComponentFromName(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestExtractCapacityHeadroom(t *testing.T) {
	findings := []upgradev1alpha1.ClassifiedFindingRef{
		{
			Category:       "Capacity",
			Recommendation: "Add capacity or reduce requests before upgrade. Estimated remaining capacity after one-node loss: 1900m CPU.",
			Classification: upgradev1alpha1.FindingClassification{Status: upgradev1alpha1.FindingClassificationBlocking},
		},
	}
	got := extractCapacityHeadroom(findings)
	if got != "1900m CPU" {
		t.Errorf("extractCapacityHeadroom() = %q, want %q", got, "1900m CPU")
	}
}
