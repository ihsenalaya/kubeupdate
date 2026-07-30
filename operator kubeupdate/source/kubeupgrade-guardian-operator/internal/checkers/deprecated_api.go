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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/snapshot"
)

// DeprecatedAPI uses a static MVP table of APIs removed by target Kubernetes versions.
type DeprecatedAPI struct{}

func (DeprecatedAPI) Name() string { return "deprecated-apis" }

type removedAPI struct {
	APIVersion string
	Kind       string
	RemovedIn  int
}

var removedAPIs = []removedAPI{
	{APIVersion: "policy/v1beta1", Kind: "PodDisruptionBudget", RemovedIn: 25},
	{APIVersion: "policy/v1beta1", Kind: "PodSecurityPolicy", RemovedIn: 25},
	{APIVersion: "autoscaling/v2beta2", Kind: "HorizontalPodAutoscaler", RemovedIn: 26},
	{APIVersion: "batch/v1beta1", Kind: "CronJob", RemovedIn: 25},
}

// Check inspects the objects served under their current API version. A removed
// API version is never listable on a cluster that already dropped it, so the
// signal has to come from the objects that survived the migration.
func (DeprecatedAPI) Check(snap *snapshot.ClusterSnapshot, assessment *upgradev1alpha1.UpgradeAssessment) []upgradev1alpha1.Finding {
	target := targetMinor(assessment.Spec.TargetVersion)

	var findings []upgradev1alpha1.Finding
	for _, item := range snap.PDBs {
		findings = append(findings, deprecatedSourceFindings(item.ObjectMeta, "PodDisruptionBudget", target)...)
	}
	for _, item := range snap.HPAs {
		findings = append(findings, deprecatedSourceFindings(item.ObjectMeta, "HorizontalPodAutoscaler", target)...)
	}
	for _, item := range snap.CronJobs {
		findings = append(findings, deprecatedSourceFindings(item.ObjectMeta, "CronJob", target)...)
	}

	return findings
}

func deprecatedSourceFindings(meta metav1.ObjectMeta, kind string, targetMinor int) []upgradev1alpha1.Finding {
	findings := make([]upgradev1alpha1.Finding, 0, len(removedAPIs))
	for _, api := range removedAPIs {
		if api.Kind != kind || !usesDeprecatedSource(meta.Annotations, api) {
			continue
		}
		severity := upgradev1alpha1.RiskLevelHigh
		if targetMinor >= api.RemovedIn {
			severity = upgradev1alpha1.RiskLevelCritical
		}
		findings = append(findings, upgradev1alpha1.Finding{
			ID:       findingID(upgradev1alpha1.FindingTypeDeprecatedOrRemovedAPI, meta.Namespace, api.Kind, meta.Name),
			Type:     upgradev1alpha1.FindingTypeDeprecatedOrRemovedAPI,
			Severity: severity,
			Category: "DeprecatedAPI",
			Resource: resource(api.APIVersion, api.Kind, meta.Namespace, meta.Name),
			Message:  fmt.Sprintf("%s %s/%s uses %s removed in Kubernetes 1.%d.", api.Kind, meta.Namespace, meta.Name, api.APIVersion, api.RemovedIn),
			Evidence: []upgradev1alpha1.Evidence{{
				ID:          evidenceID(upgradev1alpha1.FindingTypeDeprecatedOrRemovedAPI, meta.Namespace, api.Kind, meta.Name),
				Description: "Deprecated or removed API version observed in last-applied configuration.",
				Observed: map[string]string{
					"apiVersion": api.APIVersion,
					"kind":       api.Kind,
					"removedIn":  "1." + strconv.Itoa(api.RemovedIn),
				},
			}},
			Recommendation: "Migrate this resource to a served API version before upgrading.",
		})
	}
	return findings
}

type appliedObjectHeader struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
}

func usesDeprecatedSource(annotations map[string]string, api removedAPI) bool {
	if len(annotations) == 0 {
		return false
	}
	value := strings.TrimSpace(annotations["kubectl.kubernetes.io/last-applied-configuration"])
	if value == "" {
		return false
	}
	var header appliedObjectHeader
	if err := json.Unmarshal([]byte(value), &header); err != nil {
		return false
	}
	return header.APIVersion == api.APIVersion && header.Kind == api.Kind
}

func targetMinor(version string) int {
	parts := strings.Split(strings.TrimPrefix(version, "v"), ".")
	if len(parts) < 2 {
		return 0
	}
	minor, _ := strconv.Atoi(parts[1])
	return minor
}
