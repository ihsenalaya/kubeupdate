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
	"fmt"
	"sort"
	"strings"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
)

const (
	AssessmentMarkdownKey = "assessment.md"
	PlanMarkdownKey       = "plan.md"
)

// ArtifactName returns the ConfigMap name used for human-readable output.
func ArtifactName(assessment *upgradev1alpha1.UpgradeAssessment) string {
	return assessment.Name + "-artifact"
}

// AssessmentMarkdown renders a human-readable assessment summary.
func AssessmentMarkdown(assessment *upgradev1alpha1.UpgradeAssessment) string {
	var b strings.Builder

	writeHeading(&b, 1, "Upgrade Assessment")
	writeKV(&b, "Name", namespacedName(assessment.Namespace, assessment.Name))
	writeKV(&b, "Source version", valueOr(assessment.Spec.SourceVersion, "current"))
	writeKV(&b, "Target version", assessment.Spec.TargetVersion)
	writeKV(&b, "Profile", valueOr(string(assessment.Spec.Profile), string(upgradev1alpha1.AssessmentProfileProduction)))
	writeKV(&b, "Phase", string(assessment.Status.Phase))
	writeKV(&b, "Decision risk level", string(assessment.Status.RiskLevel))
	writeKV(&b, "Score", fmt.Sprintf("%d", assessment.Status.Score))
	writeKV(&b, "Generated plan", planName(assessment))
	writeKV(&b, "Artifact", artifactName(assessment))

	writeHeading(&b, 2, "Effective Finding Summary")
	writeFindingSummary(&b, assessment.Status.Summary)

	writeHeading(&b, 2, "Raw Finding Summary")
	writeFindingSummary(&b, assessment.Status.RawSummary)

	writeHeading(&b, 2, "Classification Summary")
	writeClassificationSummary(&b, assessment.Status.ClassificationSummary)

	writeHeading(&b, 2, "Findings")
	if len(assessment.Status.Findings) == 0 {
		b.WriteString("No findings were published.\n")
		return b.String()
	}

	b.WriteString("| Classification | Severity | Category | Resource | Message | Recommendation |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	for _, finding := range assessment.Status.Findings {
		b.WriteString("| ")
		b.WriteString(escapeTable(classificationStatus(finding)))
		b.WriteString(" | ")
		b.WriteString(escapeTable(string(finding.Severity)))
		b.WriteString(" | ")
		b.WriteString(escapeTable(finding.Category))
		b.WriteString(" | ")
		b.WriteString(escapeTable(resourceName(finding.Resource)))
		b.WriteString(" | ")
		b.WriteString(escapeTable(finding.Message))
		b.WriteString(" | ")
		b.WriteString(escapeTable(finding.Recommendation))
		b.WriteString(" |\n")
	}

	return b.String()
}

// PlanMarkdown renders a structured, operator-grade upgrade plan in 7 sections:
// Executive Decision → Top Blockers → Remediation Plan → Go/No-Go Gates →
// Upgrade Chronology → Provider Managed Risks → Raw Findings Appendix.
func PlanMarkdown(plan *upgradev1alpha1.UpgradePlan) string {
	var b strings.Builder
	writePlanHeader(&b, plan)
	writePlanExecutiveDecision(&b, plan)
	writePlanTopBlockers(&b, plan)
	writePlanRemediation(&b, plan)
	writePlanGoNoGoGates(&b, plan)
	writePlanChronology(&b, plan)
	writePlanProviderManaged(&b, plan)
	writePlanAppendix(&b, plan)
	return b.String()
}

// --- Section 0: Header ---

func writePlanHeader(b *strings.Builder, plan *upgradev1alpha1.UpgradePlan) {
	writeHeading(b, 1, "Upgrade Plan")
	writeKV(b, "Name", namespacedName(plan.Namespace, plan.Name))
	writeKV(b, "Assessment", namespacedName(plan.Spec.AssessmentRef.Namespace, plan.Spec.AssessmentRef.Name))
	writeKV(b, "Source version", valueOr(plan.Spec.SourceVersion, "current"))
	writeKV(b, "Target version", plan.Spec.TargetVersion)
	writeKV(b, "Risk level", string(plan.Spec.RiskLevel))
	writeKV(b, "Score", fmt.Sprintf("%d", plan.Spec.Score))
	b.WriteString("\n")
}

// --- Section 1: Executive Decision ---

func writePlanExecutiveDecision(b *strings.Builder, plan *upgradev1alpha1.UpgradePlan) {
	writeHeading(b, 2, "Executive Decision")

	_, label := computeDecisionLabel(plan)
	b.WriteString(fmt.Sprintf("**Upgrade readiness: %s**\n\n", label))

	if plan.Spec.ClassificationSummary.Blocking > 0 {
		b.WriteString(fmt.Sprintf(
			"This cluster has **%d blocking findings** with %s risk level. Upgrade must not proceed until all P0 blockers are resolved.\n\n",
			plan.Spec.ClassificationSummary.Blocking, plan.Spec.RiskLevel))
	}

	counts := countFindingsByCategory(plan.Spec.BlockingFindings)
	if len(counts) > 0 {
		b.WriteString("**Main blockers:**\n\n")
		b.WriteString("| Category | Blocking Findings |\n")
		b.WriteString("| --- | --- |\n")
		for _, cc := range sortedCategoryCounts(counts) {
			b.WriteString(fmt.Sprintf("| %s | %d |\n", cc.category, cc.count))
		}
		b.WriteString(fmt.Sprintf("| **Total** | **%d** |\n", plan.Spec.ClassificationSummary.Blocking))
		b.WriteString("\n")
	}
}

// computeDecisionLabel derives the human-facing decision from spec data.
// A plan with blocking findings and Critical risk is BLOCKED regardless of the stored Decision value,
// since the controller may have emitted ProceedWithCaution before classification was complete.
func computeDecisionLabel(plan *upgradev1alpha1.UpgradePlan) (decision, label string) {
	if plan.Spec.ClassificationSummary.Blocking > 0 && plan.Spec.RiskLevel == upgradev1alpha1.RiskLevelCritical {
		return "BLOCKED", "NOT READY — BLOCKED"
	}
	for _, f := range plan.Spec.BlockingFindings {
		if f.Category == "Capacity" {
			return "BLOCKED", "NOT READY — BLOCKED"
		}
	}
	switch plan.Spec.Decision {
	case upgradev1alpha1.DecisionDoNotUpgrade:
		return "BLOCKED", "NOT READY — BLOCKED"
	case upgradev1alpha1.DecisionProceedWithCaution:
		return "PROCEED WITH CAUTION", "PROCEED WITH CAUTION — review findings before upgrade"
	case upgradev1alpha1.DecisionProceed:
		return "READY", "READY — no blocking findings"
	default:
		return string(plan.Spec.Decision), string(plan.Spec.Decision)
	}
}

// --- Section 2: Top Upgrade Blockers ---

func writePlanTopBlockers(b *strings.Builder, plan *upgradev1alpha1.UpgradePlan) {
	writeHeading(b, 2, "Top Upgrade Blockers")

	byCategory := groupFindingsByCategory(plan.Spec.BlockingFindings)
	if len(byCategory) == 0 {
		b.WriteString("No blocking findings.\n\n")
		return
	}

	priorityOrder := []string{"Capacity", "AdmissionWebhook", "WorkloadAvailability", "PolicyRisk", "ReadinessProbes"}
	rendered := map[string]bool{}

	for _, cat := range priorityOrder {
		findings, ok := byCategory[cat]
		if !ok {
			continue
		}
		rendered[cat] = true
		writeBlockerCategorySection(b, cat, findings)
	}

	for _, cat := range sortedCategoryKeys(byCategory) {
		if rendered[cat] {
			continue
		}
		writeBlockerCategorySection(b, cat, byCategory[cat])
	}
}

func writeBlockerCategorySection(b *strings.Builder, category string, findings []upgradev1alpha1.ClassifiedFindingRef) {
	writeHeading(b, 3, fmt.Sprintf("%s (%d finding%s)", category, len(findings), plural(len(findings))))

	switch category {
	case "AdmissionWebhook":
		b.WriteString("**Impact:** Webhooks with `failurePolicy: Fail` block all API admission if the webhook pod is unavailable during node drain, potentially stalling the upgrade mid-flight.\n\n")
		components := uniqueWebhookComponents(findings)
		if len(components) > 0 {
			b.WriteString("**Affected components:**\n")
			for _, c := range components {
				b.WriteString(fmt.Sprintf("- %s\n", c))
			}
			b.WriteString("\n")
		}
		b.WriteString("**Required actions:**\n")
		b.WriteString("- Set `failurePolicy: Ignore` during upgrade windows for non-critical webhooks\n")
		b.WriteString("- Add `namespaceSelector` to limit webhook scope where missing\n")
		b.WriteString("- Test webhook availability before each upgrade phase\n\n")

	case "WorkloadAvailability":
		b.WriteString("**Impact:** Single-replica controllers will be unavailable during node drain, causing downtime for critical platform services.\n\n")
		byNS := workloadsByNamespace(findings)
		if len(byNS) > 0 {
			b.WriteString("**Affected workloads by namespace:**\n\n")
			b.WriteString("| Namespace | Workloads |\n")
			b.WriteString("| --- | --- |\n")
			for _, ns := range sortedKeys(byNS) {
				b.WriteString(fmt.Sprintf("| %s | %s |\n", ns, strings.Join(byNS[ns], ", ")))
			}
			b.WriteString("\n")
		}
		b.WriteString("**Required action:** Scale to ≥ 2 replicas, or document and accept disruption before upgrade.\n\n")

	case "ReadinessProbes":
		b.WriteString("**Impact:** Without readiness probes, Kubernetes cannot reliably determine when a rescheduled pod is healthy, risking premature traffic routing after node drain.\n\n")
		resources := uniqueResourceNames(findings)
		if len(resources) > 0 {
			b.WriteString("**Affected workloads:**\n")
			for _, r := range resources {
				b.WriteString(fmt.Sprintf("- %s\n", r))
			}
			b.WriteString("\n")
		}
		b.WriteString("**Required action:** Add readinessProbe to affected containers, or document upstream chart constraints as accepted risk.\n\n")

	case "Capacity":
		b.WriteString("**Impact:** Insufficient CPU headroom means workloads may fail to reschedule after a node is drained during upgrade.\n\n")
		for _, f := range findings {
			b.WriteString(fmt.Sprintf("- %s\n", f.Message))
		}
		b.WriteString("\n")
		b.WriteString("**Required action:** Add node capacity or reduce pod resource requests before upgrade.\n\n")

	case "PolicyRisk":
		b.WriteString("**Impact:** Workloads violating namespace Pod Security policies may fail admission after the upgrade activates stricter enforcement.\n\n")
		for _, f := range findings {
			if f.Resource.Name != "" {
				b.WriteString(fmt.Sprintf("- %s %s/%s: %s\n", f.Resource.Kind, f.Resource.Namespace, f.Resource.Name, f.Message))
			} else {
				b.WriteString(fmt.Sprintf("- %s: %s\n", f.Resource.Namespace, f.Message))
			}
		}
		b.WriteString("\n")
		b.WriteString("**Required action:** Add `runAsNonRoot: true` or adjust namespace policy before upgrade.\n\n")

	default:
		for _, f := range findings {
			b.WriteString(fmt.Sprintf("- %s: %s\n", resourceValueName(f.Resource), f.Message))
		}
		b.WriteString("\n")
	}
}

// --- Section 3: Remediation Plan ---

func writePlanRemediation(b *strings.Builder, plan *upgradev1alpha1.UpgradePlan) {
	writeHeading(b, 2, "Remediation Plan")

	p0, p1, p2 := partitionByPriority(plan.Spec.BlockingFindings)

	writeHeading(b, 3, "P0 — Must Fix Before Upgrade")
	if len(p0) == 0 {
		b.WriteString("No P0 blockers.\n\n")
	} else {
		writeRemediationGroup(b, p0)
	}

	writeHeading(b, 3, "P1 — Strongly Recommended")
	if len(p1) == 0 {
		b.WriteString("No P1 findings.\n\n")
	} else {
		writeRemediationGroup(b, p1)
	}

	writeHeading(b, 3, "P2 — Validate or Document")
	if len(p2) == 0 {
		b.WriteString("No P2 findings.\n\n")
	} else {
		writeRemediationGroup(b, p2)
	}
}

func writeRemediationGroup(b *strings.Builder, findings []upgradev1alpha1.ClassifiedFindingRef) {
	byCategory := groupFindingsByCategory(findings)
	for _, cat := range sortedCategoryKeys(byCategory) {
		catFindings := byCategory[cat]
		b.WriteString(fmt.Sprintf("**%s (%d finding%s)**\n\n", cat, len(catFindings), plural(len(catFindings))))

		if cat == "AdmissionWebhook" {
			compCounts := map[string]int{}
			compRec := map[string]string{}
			for _, f := range catFindings {
				comp := webhookComponentFromName(f.Resource.Name)
				compCounts[comp]++
				if compRec[comp] == "" {
					compRec[comp] = f.Recommendation
				}
			}
			comps := sortedStringKeys(compCounts)
			for _, comp := range comps {
				b.WriteString(fmt.Sprintf("- %s (%d finding%s): %s\n", comp, compCounts[comp], plural(compCounts[comp]), compRec[comp]))
			}
		} else {
			for _, f := range catFindings {
				res := resourceValueName(f.Resource)
				if res == "cluster" {
					b.WriteString(fmt.Sprintf("- %s\n", f.Recommendation))
				} else {
					b.WriteString(fmt.Sprintf("- %s: %s\n", res, f.Recommendation))
				}
			}
		}
		b.WriteString("\n")
	}
}

// --- Section 4: Go/No-Go Gates ---

func writePlanGoNoGoGates(b *strings.Builder, plan *upgradev1alpha1.UpgradePlan) {
	writeHeading(b, 2, "Go/No-Go Gates")

	p0Count := countPriorityFindings(plan.Spec.BlockingFindings, "P0")
	p1Count := countPriorityFindings(plan.Spec.BlockingFindings, "P1")
	webhookFailCount := countWebhookFailFindings(plan.Spec.BlockingFindings)
	capacityCount := countCategoryFindings(plan.Spec.BlockingFindings, "Capacity")
	availCount := countCategoryFindings(plan.Spec.BlockingFindings, "WorkloadAvailability")

	// Gate 1
	writeHeading(b, 3, "Gate 1 — Platform Healthy (before upgrade start)")
	b.WriteString("| Check | Status | Details |\n")
	b.WriteString("| --- | --- | --- |\n")

	if plan.Spec.ClassificationSummary.Blocking > 0 {
		b.WriteString(fmt.Sprintf("| All P0 blockers resolved | FAIL | %d blocking findings remain (%d P0, %d P1) |\n",
			plan.Spec.ClassificationSummary.Blocking, p0Count, p1Count))
	} else {
		b.WriteString("| All P0 blockers resolved | PASS | No blocking findings |\n")
	}
	b.WriteString("| Argo CD: all apps Synced + Healthy | CHECK | Manual verification required |\n")
	b.WriteString("| No pods in Error / CrashLoopBackOff | CHECK | Manual verification required |\n")
	b.WriteString("| Operator available (≥ 2 replicas, PDB allows ≥ 1 disruption) | CHECK | Verify kubeupgrade-guardian-system |\n")

	if plan.Spec.ClassificationSummary.Blocking == 0 {
		b.WriteString("\nGate 1: **PASSED** — ready to proceed with upgrade.\n\n")
	} else {
		b.WriteString(fmt.Sprintf("\nGate 1: **NOT PASSED** — resolve %d P0 blocker%s before starting upgrade.\n\n", p0Count, plural(p0Count)))
	}

	// Gate 2
	writeHeading(b, 3, "Gate 2 — Upgrade Safe (before each node pool)")
	b.WriteString("| Check | Status | Details |\n")
	b.WriteString("| --- | --- | --- |\n")

	if webhookFailCount > 0 {
		b.WriteString(fmt.Sprintf("| No blocking webhooks (failurePolicy=Fail) | FAIL | %d webhook%s with failurePolicy=Fail |\n", webhookFailCount, plural(webhookFailCount)))
	} else {
		b.WriteString("| No blocking webhooks (failurePolicy=Fail) | PASS | No blocking webhook findings |\n")
	}

	if capacityCount > 0 {
		headroom := extractCapacityHeadroom(plan.Spec.BlockingFindings)
		details := "insufficient headroom for one-node drain"
		if headroom != "" {
			details = fmt.Sprintf("~%s headroom after one-node loss", headroom)
		}
		b.WriteString(fmt.Sprintf("| Node capacity for one-node drain | FAIL | %s |\n", details))
	} else {
		b.WriteString("| Node capacity for one-node drain | PASS | Sufficient node capacity headroom |\n")
	}

	if availCount > 0 {
		b.WriteString(fmt.Sprintf("| Critical controllers have ≥ 2 replicas | FAIL | %d single-replica workload%s |\n", availCount, plural(availCount)))
	} else {
		b.WriteString("| Critical controllers have ≥ 2 replicas | PASS | All critical controllers have ≥ 2 replicas |\n")
	}

	b.WriteString("| Argo CD sync stable for 15 minutes | CHECK | Manual verification required |\n")

	gate2Pass := webhookFailCount == 0 && capacityCount == 0 && availCount == 0
	if gate2Pass {
		b.WriteString("\nGate 2: **PASSED** — node pool upgrade can proceed.\n\n")
	} else {
		b.WriteString("\nGate 2: **NOT PASSED** — resolve P0 and P1 blockers before node pool upgrade.\n\n")
	}
}

// --- Section 5: Upgrade Chronology ---

func writePlanChronology(b *strings.Builder, plan *upgradev1alpha1.UpgradePlan) {
	writeHeading(b, 2, "Upgrade Chronology")

	if len(plan.Spec.UpgradePath) == 0 {
		b.WriteString("No upgrade path was generated.\n\n")
		return
	}

	hasWebhookBlockers := countCategoryFindings(plan.Spec.BlockingFindings, "AdmissionWebhook") > 0
	hasCapacityBlockers := countCategoryFindings(plan.Spec.BlockingFindings, "Capacity") > 0
	hasAvailBlockers := countCategoryFindings(plan.Spec.BlockingFindings, "WorkloadAvailability") > 0

	for _, step := range plan.Spec.UpgradePath {
		writeHeading(b, 3, fmt.Sprintf("%s -> %s", step.From, step.To))
		for i, phase := range step.Phases {
			b.WriteString(fmt.Sprintf("**Step %d — %s**\n\n", i+1, phase.Name))
			if phase.Description != "" {
				b.WriteString(phase.Description)
				b.WriteString("\n\n")
			}
			conditions := phaseConditions(phase.Name, hasWebhookBlockers, hasCapacityBlockers, hasAvailBlockers)
			if conditions != "" {
				b.WriteString(conditions)
				b.WriteString("\n")
			}
		}
	}
}

func phaseConditions(phase string, hasWebhook, hasCapacity, hasAvail bool) string {
	var lines []string
	switch phase {
	case "prechecks":
		lines = append(lines, "Required before continuing:")
		lines = append(lines, "- Gate 1 passed: all P0 blockers resolved")
		lines = append(lines, "- Cluster backup validated and restorable")
		lines = append(lines, "- Argo CD: all applications Synced + Healthy")
		if hasWebhook {
			lines = append(lines, "- Admission webhooks patched: failurePolicy=Ignore for upgrade window")
		}
		if hasCapacity {
			lines = append(lines, "- Node capacity increased or pod requests reduced")
		}
	case "control-plane-upgrade":
		lines = append(lines, "Continue only if:")
		lines = append(lines, "- Gate 1 passed")
		if hasWebhook {
			lines = append(lines, "- Webhooks responding to dry-run pod creation")
		}
		lines = append(lines, "- Monitoring dashboards showing no active critical alerts")
	case "post-control-plane-validation":
		lines = append(lines, "Continue only if:")
		lines = append(lines, "- API server healthy and responding")
		lines = append(lines, "- Admission behavior stable (dry-run test passes)")
		lines = append(lines, "- Argo CD sync state unchanged")
		lines = append(lines, "- Platform controllers showing Ready")
	case "system-nodepool-upgrade":
		lines = append(lines, "Continue only if:")
		lines = append(lines, "- Post-control-plane validation passed")
		lines = append(lines, "- Operator still available (≥ 1 replica running)")
		if hasAvail {
			lines = append(lines, "- Critical controllers scaled to ≥ 2 replicas or disruption documented")
		}
	case "user-nodepool-upgrade":
		lines = append(lines, "Continue only if:")
		lines = append(lines, "- Gate 2 passed")
		lines = append(lines, "- No new Unavailable pods since system nodepool upgrade")
		lines = append(lines, "- Argo CD sync stable for 15 minutes")
	case "application-validation":
		lines = append(lines, "Validation complete when:")
		lines = append(lines, "- All Argo CD apps Synced + Healthy")
		lines = append(lines, "- SLO dashboards within normal range")
		lines = append(lines, "- No alert firing above warning threshold")
		lines = append(lines, "- Rollback window elapsed without incident")
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

// --- Section 6: Provider Managed Risks ---

func writePlanProviderManaged(b *strings.Builder, plan *upgradev1alpha1.UpgradePlan) {
	writeHeading(b, 2, "Provider Managed Risks")
	if len(plan.Spec.ProviderManagedRisks) == 0 {
		b.WriteString("No provider-managed risks identified.\n\n")
		return
	}
	b.WriteString("These resources are owned by the cloud provider (AKS) and cannot be modified directly. They are excluded from the blocking score.\n\n")

	byResource := map[string][]string{}
	resourceOrder := []string{}
	seen := map[string]bool{}
	for _, f := range plan.Spec.ProviderManagedRisks {
		resName := resourceValueName(f.Resource)
		risk := extractRiskLabel(f.Message)
		if !seen[resName] {
			seen[resName] = true
			resourceOrder = append(resourceOrder, resName)
		}
		byResource[resName] = append(byResource[resName], risk)
	}

	b.WriteString("| Resource | Risks |\n")
	b.WriteString("| --- | --- |\n")
	for _, resName := range resourceOrder {
		risks := strings.Join(byResource[resName], "; ")
		b.WriteString(fmt.Sprintf("| %s | %s |\n", escapeTable(resName), escapeTable(risks)))
	}
	b.WriteString("\n")
}

// --- Section 7: Raw Findings Appendix ---

func writePlanAppendix(b *strings.Builder, plan *upgradev1alpha1.UpgradePlan) {
	writeAppendixSection(b, "Appendix — All Blocking Findings", plan.Spec.BlockingFindings)
	writeAppendixSection(b, "Appendix — Accepted Risks", plan.Spec.AcceptedRisks)
	writeAppendixSection(b, "Appendix — Informational Findings", plan.Spec.InformationalFindings)
}

func writeAppendixSection(b *strings.Builder, title string, findings []upgradev1alpha1.ClassifiedFindingRef) {
	writeHeading(b, 2, title)
	if len(findings) == 0 {
		b.WriteString("No entries.\n\n")
		return
	}
	b.WriteString("| Classification | Severity | Category | Resource | Message | Recommendation |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	for _, f := range findings {
		b.WriteString("| ")
		b.WriteString(escapeTable(string(f.Classification.Status)))
		b.WriteString(" | ")
		b.WriteString(escapeTable(string(f.Severity)))
		b.WriteString(" | ")
		b.WriteString(escapeTable(f.Category))
		b.WriteString(" | ")
		b.WriteString(escapeTable(resourceValueName(f.Resource)))
		b.WriteString(" | ")
		b.WriteString(escapeTable(f.Message))
		b.WriteString(" | ")
		b.WriteString(escapeTable(f.Recommendation))
		b.WriteString(" |\n")
	}
	b.WriteString("\n")
}

// --- Shared helpers ---

func writeHeading(b *strings.Builder, level int, title string) {
	b.WriteString(strings.Repeat("#", level))
	b.WriteString(" ")
	b.WriteString(title)
	b.WriteString("\n\n")
}

func writeKV(b *strings.Builder, key, value string) {
	b.WriteString("- ")
	b.WriteString(key)
	b.WriteString(": ")
	b.WriteString(valueOr(value, "not set"))
	b.WriteString("\n")
}

func writeFindingSummary(b *strings.Builder, summary upgradev1alpha1.FindingSummary) {
	b.WriteString(fmt.Sprintf("- Total: %d\n", summary.TotalFindings))
	b.WriteString(fmt.Sprintf("- Critical: %d\n", summary.Critical))
	b.WriteString(fmt.Sprintf("- High: %d\n", summary.High))
	b.WriteString(fmt.Sprintf("- Medium: %d\n", summary.Medium))
	b.WriteString(fmt.Sprintf("- Low: %d\n", summary.Low))
	b.WriteString(fmt.Sprintf("- Info: %d\n\n", summary.Info))
}

func writeClassificationSummary(b *strings.Builder, summary upgradev1alpha1.ClassificationSummary) {
	b.WriteString(fmt.Sprintf("- Total: %d\n", summary.Total))
	b.WriteString(fmt.Sprintf("- Blocking: %d\n", summary.Blocking))
	b.WriteString(fmt.Sprintf("- Accepted risk: %d\n", summary.AcceptedRisk))
	b.WriteString(fmt.Sprintf("- Provider managed: %d\n", summary.ProviderManaged))
	b.WriteString(fmt.Sprintf("- Informational: %d\n\n", summary.Informational))
}

func classificationStatus(finding upgradev1alpha1.Finding) string {
	if finding.Classification == nil {
		return "Unclassified"
	}
	return string(finding.Classification.Status)
}

func planName(assessment *upgradev1alpha1.UpgradeAssessment) string {
	if assessment.Status.GeneratedPlanRef == nil {
		return ""
	}
	return assessment.Status.GeneratedPlanRef.Name
}

func artifactName(assessment *upgradev1alpha1.UpgradeAssessment) string {
	if assessment.Status.ArtifactRef == nil {
		return ""
	}
	return assessment.Status.ArtifactRef.Name
}

func namespacedName(namespace, name string) string {
	if namespace == "" {
		return name
	}
	return namespace + "/" + name
}

func resourceName(ref *upgradev1alpha1.ResourceRef) string {
	if ref == nil {
		return "cluster"
	}
	return resourceValueName(*ref)
}

func resourceValueName(ref upgradev1alpha1.ResourceRef) string {
	name := ref.Name
	if name == "" {
		name = "cluster"
	}
	if ref.Namespace != "" {
		name = ref.Namespace + "/" + name
	}
	if ref.Kind != "" {
		name = ref.Kind + " " + name
	}
	return name
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func escapeTable(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	if value == "" {
		return " "
	}
	return value
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// --- Finding grouping and counting ---

type categoryCount struct {
	category string
	count    int
}

func countFindingsByCategory(findings []upgradev1alpha1.ClassifiedFindingRef) map[string]int {
	counts := make(map[string]int)
	for _, f := range findings {
		counts[f.Category]++
	}
	return counts
}

func sortedCategoryCounts(counts map[string]int) []categoryCount {
	result := make([]categoryCount, 0, len(counts))
	for cat, n := range counts {
		result = append(result, categoryCount{cat, n})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].count != result[j].count {
			return result[i].count > result[j].count
		}
		return result[i].category < result[j].category
	})
	return result
}

func groupFindingsByCategory(findings []upgradev1alpha1.ClassifiedFindingRef) map[string][]upgradev1alpha1.ClassifiedFindingRef {
	groups := make(map[string][]upgradev1alpha1.ClassifiedFindingRef)
	for _, f := range findings {
		groups[f.Category] = append(groups[f.Category], f)
	}
	return groups
}

func sortedCategoryKeys(groups map[string][]upgradev1alpha1.ClassifiedFindingRef) []string {
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedStringKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func countCategoryFindings(findings []upgradev1alpha1.ClassifiedFindingRef, category string) int {
	n := 0
	for _, f := range findings {
		if f.Category == category {
			n++
		}
	}
	return n
}

func countWebhookFailFindings(findings []upgradev1alpha1.ClassifiedFindingRef) int {
	n := 0
	for _, f := range findings {
		if f.Category == "AdmissionWebhook" && strings.Contains(f.Message, "failurePolicy=Fail") {
			n++
		}
	}
	return n
}

func countPriorityFindings(findings []upgradev1alpha1.ClassifiedFindingRef, priority string) int {
	n := 0
	for _, f := range findings {
		if findingPriority(f) == priority {
			n++
		}
	}
	return n
}

// findingPriority assigns P0/P1/P2 based on category and content.
// P0: mechanically blocks upgrade (capacity, webhook failurePolicy=Fail).
// P1: downtime risk or policy enforcement risk.
// P2: health visibility or documentation gap.
func findingPriority(f upgradev1alpha1.ClassifiedFindingRef) string {
	switch f.Category {
	case "Capacity":
		return "P0"
	case "AdmissionWebhook":
		if strings.Contains(f.Message, "failurePolicy=Fail") {
			return "P0"
		}
		return "P1"
	case "WorkloadAvailability":
		return "P1"
	case "PolicyRisk":
		if f.Severity == upgradev1alpha1.RiskLevelHigh {
			return "P1"
		}
		return "P2"
	default:
		return "P2"
	}
}

func partitionByPriority(findings []upgradev1alpha1.ClassifiedFindingRef) (p0, p1, p2 []upgradev1alpha1.ClassifiedFindingRef) {
	for _, f := range findings {
		switch findingPriority(f) {
		case "P0":
			p0 = append(p0, f)
		case "P1":
			p1 = append(p1, f)
		default:
			p2 = append(p2, f)
		}
	}
	return
}

// --- Webhook helpers ---

var webhookComponentPrefixes = []struct{ prefix, component string }{
	{"cert-manager", "cert-manager"},
	{"istiod", "istio"},
	{"istio", "istio"},
	{"kyverno", "kyverno"},
	{"externalsecret", "external-secrets"},
	{"secretstore", "external-secrets"},
	{"azure-wi", "azure-workload-identity"},
}

func webhookComponentFromName(name string) string {
	lower := strings.ToLower(name)
	for _, p := range webhookComponentPrefixes {
		if strings.HasPrefix(lower, p.prefix) {
			return p.component
		}
	}
	return name
}

func uniqueWebhookComponents(findings []upgradev1alpha1.ClassifiedFindingRef) []string {
	seen := map[string]bool{}
	var components []string
	for _, f := range findings {
		comp := webhookComponentFromName(f.Resource.Name)
		if !seen[comp] {
			seen[comp] = true
			components = append(components, comp)
		}
	}
	sort.Strings(components)
	return components
}

// --- Workload helpers ---

func workloadsByNamespace(findings []upgradev1alpha1.ClassifiedFindingRef) map[string][]string {
	byNS := make(map[string][]string)
	seen := map[string]bool{}
	for _, f := range findings {
		ns := valueOr(f.Resource.Namespace, "cluster")
		key := ns + "/" + f.Resource.Name
		if !seen[key] {
			seen[key] = true
			byNS[ns] = append(byNS[ns], f.Resource.Name)
		}
	}
	return byNS
}

func uniqueResourceNames(findings []upgradev1alpha1.ClassifiedFindingRef) []string {
	seen := map[string]bool{}
	var names []string
	for _, f := range findings {
		name := resourceValueName(f.Resource)
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// --- Message parsing helpers ---

func extractRiskLabel(message string) string {
	const marker = "has risk: "
	if idx := strings.Index(message, marker); idx >= 0 {
		return strings.TrimSuffix(message[idx+len(marker):], ".")
	}
	return message
}

func extractCapacityHeadroom(findings []upgradev1alpha1.ClassifiedFindingRef) string {
	const marker = "Estimated remaining capacity after one-node loss: "
	for _, f := range findings {
		if f.Category != "Capacity" {
			continue
		}
		if idx := strings.Index(f.Recommendation, marker); idx >= 0 {
			rest := f.Recommendation[idx+len(marker):]
			if end := strings.IndexAny(rest, ".\n"); end >= 0 {
				return rest[:end]
			}
			return rest
		}
	}
	return ""
}
