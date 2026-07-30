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
	"fmt"
	"strconv"
	"strings"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
)

var defaultRecommendedOrder = []string{
	"fix-critical-blockers",
	"review-accepted-risks",
	"review-provider-managed-risks",
	"execute-version-upgrade-path",
	"validate-observability",
	"proceed-with-controlled-upgrade",
}

// BuildSpec generates an idempotent UpgradePlan spec from assessment results.
func BuildSpec(
	assessment *upgradev1alpha1.UpgradeAssessment,
	decision upgradev1alpha1.Decision,
	riskLevel upgradev1alpha1.RiskLevel,
	score int,
	summary upgradev1alpha1.FindingSummary,
	rawSummary upgradev1alpha1.FindingSummary,
	classificationSummary upgradev1alpha1.ClassificationSummary,
	findings []upgradev1alpha1.Finding,
) upgradev1alpha1.UpgradePlanSpec {
	actions := make([]upgradev1alpha1.RequiredAction, 0, len(findings))
	blockingFindings := make([]upgradev1alpha1.ClassifiedFindingRef, 0, len(findings))
	acceptedRisks := make([]upgradev1alpha1.ClassifiedFindingRef, 0, len(findings))
	providerManagedRisks := make([]upgradev1alpha1.ClassifiedFindingRef, 0, len(findings))
	informationalFindings := make([]upgradev1alpha1.ClassifiedFindingRef, 0, len(findings))
	for _, finding := range findings {
		classification := classificationValue(finding)
		ref := classifiedFindingRef(finding, classification)
		switch classification.Status {
		case upgradev1alpha1.FindingClassificationAcceptedRisk:
			acceptedRisks = append(acceptedRisks, ref)
			continue
		case upgradev1alpha1.FindingClassificationProviderManaged:
			providerManagedRisks = append(providerManagedRisks, ref)
			continue
		case upgradev1alpha1.FindingClassificationInformational:
			informationalFindings = append(informationalFindings, ref)
			continue
		default:
			blockingFindings = append(blockingFindings, ref)
		}

		action := finding.Recommendation
		if action == "" {
			action = fmt.Sprintf("Review and remediate %s before upgrading.", strings.ToLower(finding.Category))
		}

		actions = append(actions, upgradev1alpha1.RequiredAction{
			ID:           actionID(finding),
			Severity:     finding.Severity,
			Category:     finding.Category,
			Resource:     resourceValue(finding.Resource),
			Action:       action,
			EvidenceRefs: evidenceRefs(finding.Evidence),
		})
	}

	return upgradev1alpha1.UpgradePlanSpec{
		AssessmentRef: upgradev1alpha1.AssessmentReference{
			Name:      assessment.Name,
			Namespace: assessment.Namespace,
		},
		Decision:              decision,
		RiskLevel:             riskLevel,
		Score:                 score,
		Summary:               summary,
		RawSummary:            rawSummary,
		ClassificationSummary: classificationSummary,
		SourceVersion:         sourceVersion(assessment.Spec.SourceVersion),
		TargetVersion:         assessment.Spec.TargetVersion,
		UpgradePath:           BuildUpgradePath(assessment.Spec.SourceVersion, assessment.Spec.TargetVersion),
		RequiredActions:       actions,
		BlockingFindings:      blockingFindings,
		AcceptedRisks:         acceptedRisks,
		ProviderManagedRisks:  providerManagedRisks,
		InformationalFindings: informationalFindings,
		RecommendedOrder:      append([]string{}, defaultRecommendedOrder...),
	}
}

// BuildUpgradePath returns the non-executing Kubernetes minor-version chronology.
func BuildUpgradePath(source, target string) []upgradev1alpha1.UpgradePathStep {
	source = sourceVersion(source)
	if source == "current" {
		return []upgradev1alpha1.UpgradePathStep{{
			From:   source,
			To:     target,
			Phases: defaultUpgradePhases(),
		}}
	}

	sourceMinor, sourceOK := parseMinor(source)
	targetMinor, targetOK := parseMinor(target)
	if !sourceOK || !targetOK || targetMinor <= sourceMinor {
		return []upgradev1alpha1.UpgradePathStep{{
			From:   source,
			To:     target,
			Phases: defaultUpgradePhases(),
		}}
	}

	steps := make([]upgradev1alpha1.UpgradePathStep, 0, targetMinor-sourceMinor)
	for minor := sourceMinor; minor < targetMinor; minor++ {
		steps = append(steps, upgradev1alpha1.UpgradePathStep{
			From:   fmt.Sprintf("1.%d", minor),
			To:     fmt.Sprintf("1.%d", minor+1),
			Phases: defaultUpgradePhases(),
		})
	}
	return steps
}

func defaultUpgradePhases() []upgradev1alpha1.UpgradePhase {
	return []upgradev1alpha1.UpgradePhase{
		{Name: "prechecks", Description: "Verify blocking findings, accepted risks, provider-managed risks, capacity, observability, and backups before this version hop."},
		{Name: "control-plane-upgrade", Description: "Upgrade the Kubernetes control plane first."},
		{Name: "post-control-plane-validation", Description: "Validate API server health, admission behavior, Argo CD sync state, and core platform controllers."},
		{Name: "system-nodepool-upgrade", Description: "Upgrade system node pools with controlled drain and workload validation."},
		{Name: "user-nodepool-upgrade", Description: "Upgrade application node pools by wave, starting with the lowest-risk workloads."},
		{Name: "application-validation", Description: "Validate application readiness, SLO dashboards, alerts, and rollback criteria before continuing."},
	}
}

func parseMinor(version string) (int, bool) {
	parts := strings.Split(strings.TrimPrefix(version, "v"), ".")
	if len(parts) < 2 || parts[0] != "1" {
		return 0, false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, false
	}
	return minor, true
}

func sourceVersion(source string) string {
	if source == "" {
		return "current"
	}
	return strings.TrimPrefix(source, "v")
}

func classificationValue(finding upgradev1alpha1.Finding) upgradev1alpha1.FindingClassification {
	if finding.Classification != nil {
		return *finding.Classification
	}
	if finding.Severity == upgradev1alpha1.RiskLevelInfo {
		return upgradev1alpha1.FindingClassification{
			Status:      upgradev1alpha1.FindingClassificationInformational,
			MatchedRule: "LEGACY_INFO",
			Reason:      "Info findings are useful context but do not block upgrades.",
		}
	}
	return upgradev1alpha1.FindingClassification{
		Status:      upgradev1alpha1.FindingClassificationBlocking,
		MatchedRule: "LEGACY_UNCLASSIFIED",
		Reason:      "Finding was not classified; treating it as blocking.",
	}
}

func classifiedFindingRef(finding upgradev1alpha1.Finding, classification upgradev1alpha1.FindingClassification) upgradev1alpha1.ClassifiedFindingRef {
	return upgradev1alpha1.ClassifiedFindingRef{
		ID:             finding.ID,
		Type:           finding.Type,
		Severity:       finding.Severity,
		Category:       finding.Category,
		Resource:       resourceValue(finding.Resource),
		Message:        finding.Message,
		Recommendation: finding.Recommendation,
		Classification: classification,
	}
}

func actionID(finding upgradev1alpha1.Finding) string {
	if finding.ID != "" {
		return "remediate-" + strings.ToLower(strings.ReplaceAll(finding.ID, "_", "-"))
	}
	return "remediate-" + strings.ToLower(strings.ReplaceAll(finding.Type, "_", "-"))
}

func resourceValue(ref *upgradev1alpha1.ResourceRef) upgradev1alpha1.ResourceRef {
	if ref == nil {
		return upgradev1alpha1.ResourceRef{}
	}
	return *ref
}

func evidenceRefs(evidence []upgradev1alpha1.Evidence) []string {
	refs := make([]string, 0, len(evidence))
	for _, item := range evidence {
		if item.ID != "" {
			refs = append(refs, item.ID)
		}
	}
	return refs
}
