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

package classifier

import (
	"strconv"
	"strings"
	"time"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
)

const (
	ruleAcceptedRisk           = "ACCEPTED_RISK"
	ruleProviderManagedWebhook = "PROVIDER_MANAGED_WEBHOOK"
	ruleInformationalSeverity  = "INFORMATIONAL_SEVERITY"
	ruleLabNonBlockingSeverity = "LAB_NON_BLOCKING_SEVERITY"
	ruleStagingLowSeverity     = "STAGING_LOW_SEVERITY"
	rulePDBDoesNotBlockDrain   = "PDB_DOES_NOT_BLOCK_DRAIN"
	ruleDefaultBlocking        = "DEFAULT_BLOCKING"
)

// Result contains classified findings and the subset used for scoring.
type Result struct {
	Findings []upgradev1alpha1.Finding
	Summary  upgradev1alpha1.ClassificationSummary
}

// BlockingFindings returns only findings that should influence score and decision.
func (r Result) BlockingFindings() []upgradev1alpha1.Finding {
	return r.FindingsByStatus(upgradev1alpha1.FindingClassificationBlocking)
}

// FindingsByStatus returns classified findings matching a status.
func (r Result) FindingsByStatus(status upgradev1alpha1.FindingClassificationStatus) []upgradev1alpha1.Finding {
	var out []upgradev1alpha1.Finding
	for _, finding := range r.Findings {
		if finding.Classification == nil || finding.Classification.Status != status {
			continue
		}
		out = append(out, finding)
	}
	return out
}

// Classify applies deterministic decisioning rules without hiding raw findings.
func Classify(findings []upgradev1alpha1.Finding, spec upgradev1alpha1.UpgradeAssessmentSpec, now time.Time) Result {
	classified := make([]upgradev1alpha1.Finding, 0, len(findings))
	var summary upgradev1alpha1.ClassificationSummary

	for _, finding := range findings {
		next := finding
		classification := classifyFinding(finding, spec, now)
		next.Classification = &classification
		classified = append(classified, next)
		summary.Total++
		switch classification.Status {
		case upgradev1alpha1.FindingClassificationBlocking:
			summary.Blocking++
		case upgradev1alpha1.FindingClassificationAcceptedRisk:
			summary.AcceptedRisk++
		case upgradev1alpha1.FindingClassificationProviderManaged:
			summary.ProviderManaged++
		case upgradev1alpha1.FindingClassificationInformational:
			summary.Informational++
		}
	}

	return Result{Findings: classified, Summary: summary}
}

func classifyFinding(finding upgradev1alpha1.Finding, spec upgradev1alpha1.UpgradeAssessmentSpec, now time.Time) upgradev1alpha1.FindingClassification {
	if acceptedRisk, ok := matchingAcceptedRisk(finding, spec.AcceptedRisks, now); ok {
		return upgradev1alpha1.FindingClassification{
			Status:      upgradev1alpha1.FindingClassificationAcceptedRisk,
			MatchedRule: ruleAcceptedRisk,
			Reason:      "Risk accepted by " + acceptedRisk.ApprovedBy + ": " + acceptedRisk.Reason,
		}
	}

	if finding.Severity == upgradev1alpha1.RiskLevelInfo {
		return upgradev1alpha1.FindingClassification{
			Status:      upgradev1alpha1.FindingClassificationInformational,
			MatchedRule: ruleInformationalSeverity,
			Reason:      "Info findings are useful context but do not block upgrades.",
		}
	}

	if isProviderManagedWebhook(finding) {
		return upgradev1alpha1.FindingClassification{
			Status:      upgradev1alpha1.FindingClassificationProviderManaged,
			MatchedRule: ruleProviderManagedWebhook,
			Reason:      "Admission webhook appears to be managed by the Kubernetes provider.",
		}
	}

	if isPDBFindingThatAllowsDrain(finding) {
		return upgradev1alpha1.FindingClassification{
			Status:      upgradev1alpha1.FindingClassificationInformational,
			MatchedRule: rulePDBDoesNotBlockDrain,
			Reason:      "The observed PDB still allows at least one voluntary disruption.",
		}
	}

	profile := spec.Profile
	if profile == "" {
		profile = upgradev1alpha1.AssessmentProfileProduction
	}
	if profile == upgradev1alpha1.AssessmentProfileLab &&
		(finding.Severity == upgradev1alpha1.RiskLevelMedium || finding.Severity == upgradev1alpha1.RiskLevelLow) {
		return upgradev1alpha1.FindingClassification{
			Status:      upgradev1alpha1.FindingClassificationInformational,
			MatchedRule: ruleLabNonBlockingSeverity,
			Reason:      "Medium and low findings are tracked but not blocking in lab profile.",
		}
	}
	if profile == upgradev1alpha1.AssessmentProfileStaging && finding.Severity == upgradev1alpha1.RiskLevelLow {
		return upgradev1alpha1.FindingClassification{
			Status:      upgradev1alpha1.FindingClassificationInformational,
			MatchedRule: ruleStagingLowSeverity,
			Reason:      "Low findings are tracked but not blocking in staging profile.",
		}
	}

	return upgradev1alpha1.FindingClassification{
		Status:      upgradev1alpha1.FindingClassificationBlocking,
		MatchedRule: ruleDefaultBlocking,
		Reason:      "Finding can affect upgrade safety and has not been accepted or classified as provider-managed.",
	}
}

func matchingAcceptedRisk(finding upgradev1alpha1.Finding, acceptedRisks []upgradev1alpha1.AcceptedRisk, now time.Time) (upgradev1alpha1.AcceptedRisk, bool) {
	for _, acceptedRisk := range acceptedRisks {
		if acceptedRisk.ExpiresAt != nil && !acceptedRisk.ExpiresAt.Time.After(now) {
			continue
		}
		if acceptedRisk.Reason == "" || acceptedRisk.ApprovedBy == "" {
			continue
		}
		if acceptedRisk.FindingID != "" && acceptedRisk.FindingID != finding.ID {
			continue
		}
		if acceptedRisk.Type != "" && acceptedRisk.Type != finding.Type {
			continue
		}
		if acceptedRisk.Resource != nil && !resourceMatches(*acceptedRisk.Resource, finding.Resource) {
			continue
		}
		if acceptedRisk.FindingID == "" && acceptedRisk.Type == "" && acceptedRisk.Resource == nil {
			continue
		}
		return acceptedRisk, true
	}
	return upgradev1alpha1.AcceptedRisk{}, false
}

func resourceMatches(want upgradev1alpha1.ResourceRef, got *upgradev1alpha1.ResourceRef) bool {
	if got == nil {
		return false
	}
	return (want.APIVersion == "" || want.APIVersion == got.APIVersion) &&
		(want.Kind == "" || want.Kind == got.Kind) &&
		(want.Namespace == "" || want.Namespace == got.Namespace) &&
		(want.Name == "" || want.Name == got.Name)
}

func isProviderManagedWebhook(finding upgradev1alpha1.Finding) bool {
	if finding.Type != upgradev1alpha1.FindingTypeAdmissionWebhookRisk || finding.Resource == nil {
		return false
	}
	kind := finding.Resource.Kind
	if kind != "MutatingWebhookConfiguration" && kind != "ValidatingWebhookConfiguration" {
		return false
	}
	name := strings.ToLower(finding.Resource.Name)
	if strings.HasPrefix(name, "aks-") {
		return true
	}
	for _, evidence := range finding.Evidence {
		webhookName := strings.ToLower(evidence.Observed["webhookName"])
		if strings.Contains(webhookName, ".azmk8s.io") {
			return true
		}
	}
	return false
}

func isPDBFindingThatAllowsDrain(finding upgradev1alpha1.Finding) bool {
	if finding.Type != upgradev1alpha1.FindingTypePDBBlockingRisk {
		return false
	}
	for _, evidence := range finding.Evidence {
		replicas, ok := intObserved(evidence.Observed, "replicas")
		if !ok || replicas == 0 {
			continue
		}
		minAvailable, ok := intObserved(evidence.Observed, "minAvailable")
		if !ok {
			continue
		}
		return minAvailable < replicas
	}
	return false
}

func intObserved(observed map[string]string, key string) (int, bool) {
	if observed == nil {
		return 0, false
	}
	value, ok := observed[key]
	if !ok {
		return 0, false
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return parsed, true
}
