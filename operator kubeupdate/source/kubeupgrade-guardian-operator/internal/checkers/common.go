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
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/snapshot"
)

// Checker assesses one read-only upgrade-readiness risk category. Checkers are
// pure functions over a ClusterSnapshot: they never talk to the API server, so
// every checker of a given assessment observes the same cluster state.
type Checker interface {
	Name() string
	Check(snap *snapshot.ClusterSnapshot, assessment *upgradev1alpha1.UpgradeAssessment) []upgradev1alpha1.Finding
}

// Default returns checkers enabled by the assessment. If no check is explicitly enabled,
// all MVP checkers run.
func Default(assessment *upgradev1alpha1.UpgradeAssessment) []Checker {
	checks := assessment.Spec.Checks
	runAll := !checks.DeprecatedAPIs &&
		!checks.WorkloadAvailability &&
		!checks.PDB &&
		!checks.ReadinessProbes &&
		!checks.AdmissionWebhooks &&
		!checks.PolicyRisks &&
		!checks.Capacity &&
		!checks.Observability

	var selected []Checker
	add := func(enabled bool, checker Checker) {
		if runAll || enabled {
			selected = append(selected, checker)
		}
	}

	add(checks.WorkloadAvailability, WorkloadAvailability{})
	add(checks.ReadinessProbes, ReadinessProbe{})
	add(checks.PDB, PDB{})
	add(checks.AdmissionWebhooks, AdmissionWebhook{})
	add(checks.PolicyRisks, PolicyRisk{})
	add(checks.DeprecatedAPIs, DeprecatedAPI{})
	add(checks.Capacity, Capacity{})
	add(checks.Observability, Observability{})

	return selected
}

// AssessmentErrorFinding reports that a checker did not complete. It keeps the
// assessment auditable: the caller publishes the findings it does have and this
// finding says which check is missing and why.
func AssessmentErrorFinding(checker string, err error) upgradev1alpha1.Finding {
	id := findingID(upgradev1alpha1.FindingTypeAssessmentError, checker)
	return upgradev1alpha1.Finding{
		ID:       id,
		Type:     upgradev1alpha1.FindingTypeAssessmentError,
		Severity: upgradev1alpha1.RiskLevelHigh,
		Category: "Assessment",
		Message:  fmt.Sprintf("Checker %s did not complete: %v", checker, err),
		Evidence: []upgradev1alpha1.Evidence{{
			ID:          evidenceID(upgradev1alpha1.FindingTypeAssessmentError, checker),
			Description: "A checker failed while evaluating the cluster snapshot.",
			Observed: map[string]string{
				"checker": checker,
				"error":   err.Error(),
			},
		}},
		Recommendation: "Report this failure: the corresponding risk category was not assessed, so the result is incomplete.",
	}
}

// namespaceSet indexes the in-scope namespaces so checkers can narrow the
// cluster-wide collections of a snapshot.
func namespaceSet(namespaces []corev1.Namespace) map[string]struct{} {
	set := make(map[string]struct{}, len(namespaces))
	for _, namespace := range namespaces {
		set[namespace.Name] = struct{}{}
	}
	return set
}

func replicaCount(replicas *int32) int32 {
	if replicas == nil {
		return 1
	}
	return *replicas
}

func resource(apiVersion, kind, namespace, name string) *upgradev1alpha1.ResourceRef {
	return &upgradev1alpha1.ResourceRef{
		APIVersion: apiVersion,
		Kind:       kind,
		Namespace:  namespace,
		Name:       name,
	}
}

func findingID(findingType string, parts ...string) string {
	values := append([]string{findingType}, parts...)
	return strings.ToUpper(sanitizeID(strings.Join(values, "_")))
}

func evidenceID(findingType string, parts ...string) string {
	return findingID(findingType, append([]string{"EVIDENCE"}, parts...)...)
}

func sanitizeID(value string) string {
	replacer := strings.NewReplacer("/", "_", ".", "_", "-", "_", " ", "_", ":", "_")
	return replacer.Replace(value)
}
