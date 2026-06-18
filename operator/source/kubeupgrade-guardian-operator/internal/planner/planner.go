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
	"strings"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
)

var defaultRecommendedOrder = []string{
	"fix-critical-blockers",
	"fix-high-risk-workloads",
	"validate-admission-policies",
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
	findings []upgradev1alpha1.Finding,
) upgradev1alpha1.UpgradePlanSpec {
	actions := make([]upgradev1alpha1.RequiredAction, 0, len(findings))
	for _, finding := range findings {
		if finding.Severity == upgradev1alpha1.RiskLevelInfo {
			continue
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
		Decision:         decision,
		RiskLevel:        riskLevel,
		Score:            score,
		Summary:          summary,
		RequiredActions:  actions,
		RecommendedOrder: append([]string{}, defaultRecommendedOrder...),
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
