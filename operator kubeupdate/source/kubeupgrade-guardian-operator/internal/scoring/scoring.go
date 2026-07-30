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

package scoring

import upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"

// Score returns the weighted score and finding summary.
func Score(findings []upgradev1alpha1.Finding) (int, upgradev1alpha1.FindingSummary) {
	var score int
	var summary upgradev1alpha1.FindingSummary

	for _, finding := range findings {
		summary.TotalFindings++
		switch finding.Severity {
		case upgradev1alpha1.RiskLevelCritical:
			summary.Critical++
			score += 25
		case upgradev1alpha1.RiskLevelHigh:
			summary.High++
			score += 10
		case upgradev1alpha1.RiskLevelMedium:
			summary.Medium++
			score += 4
		case upgradev1alpha1.RiskLevelLow:
			summary.Low++
			score++
		case upgradev1alpha1.RiskLevelInfo:
			summary.Info++
		}
	}

	return score, summary
}

// RiskLevel maps the aggregate score to a global risk level.
func RiskLevel(score int) upgradev1alpha1.RiskLevel {
	switch {
	case score > 60:
		return upgradev1alpha1.RiskLevelCritical
	case score >= 31:
		return upgradev1alpha1.RiskLevelHigh
	case score >= 11:
		return upgradev1alpha1.RiskLevelMedium
	default:
		return upgradev1alpha1.RiskLevelLow
	}
}

// Decision returns the non-executing upgrade recommendation.
func Decision(score int, summary upgradev1alpha1.FindingSummary, findings []upgradev1alpha1.Finding) upgradev1alpha1.Decision {
	if summary.Critical > 0 {
		return upgradev1alpha1.DecisionDoNotUpgrade
	}
	if score > 30 {
		return upgradev1alpha1.DecisionProceedWithCaution
	}
	if hasCriticalRBACGap(findings) {
		return upgradev1alpha1.DecisionAssessmentIncomplete
	}
	return upgradev1alpha1.DecisionProceed
}

func hasCriticalRBACGap(findings []upgradev1alpha1.Finding) bool {
	for _, finding := range findings {
		if finding.Type == upgradev1alpha1.FindingTypeRBACAssessmentGap &&
			(finding.Severity == upgradev1alpha1.RiskLevelCritical || finding.Severity == upgradev1alpha1.RiskLevelHigh) {
			return true
		}
	}
	return false
}
