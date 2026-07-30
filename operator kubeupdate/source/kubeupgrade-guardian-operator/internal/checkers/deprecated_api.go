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
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/snapshot"
)

const lastAppliedAnnotation = "kubectl.kubernetes.io/last-applied-configuration"

// DeprecatedAPI reports objects still written through an API version a target
// Kubernetes release no longer serves.
type DeprecatedAPI struct{}

func (DeprecatedAPI) Name() string { return "deprecated-apis" }

type removedAPI struct {
	APIVersion     string `json:"apiVersion"`
	Kind           string `json:"kind"`
	RemovedInMinor int    `json:"removedInMinor"`
}

type removedAPITable struct {
	RemovedAPIs []removedAPI `json:"removedApis"`
}

//go:embed data/removed_apis.yaml
var removedAPIsYAML []byte

// removedAPIs is the removal table, indexed by kind for lookup.
var removedAPIs = mustLoadRemovedAPIs(removedAPIsYAML)

func mustLoadRemovedAPIs(raw []byte) map[string][]removedAPI {
	var table removedAPITable
	if err := utilyaml.NewYAMLOrJSONDecoder(bytes.NewReader(raw), 4096).Decode(&table); err != nil {
		panic(fmt.Sprintf("checkers: cannot parse the embedded removed API table: %v", err))
	}

	byKind := map[string][]removedAPI{}
	for _, api := range table.RemovedAPIs {
		byKind[api.Kind] = append(byKind[api.Kind], api)
	}
	for kind := range byKind {
		entries := byKind[kind]
		sort.SliceStable(entries, func(i, j int) bool { return entries[i].APIVersion < entries[j].APIVersion })
	}
	return byKind
}

// Check inspects the objects served under their current API version. A removed
// version is not listable on a cluster that already dropped it, so listing those
// versions directly - as this checker used to - always came back empty and
// reported false confidence. The trace of a stale writer survives in
// metadata.managedFields and in the last-applied configuration instead.
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
	candidates := removedAPIs[kind]
	if len(candidates) == 0 {
		return nil
	}

	findings := make([]upgradev1alpha1.Finding, 0, len(candidates))
	for _, api := range candidates {
		writers := removedAPIWriters(meta, api)
		if len(writers) == 0 {
			continue
		}

		severity := upgradev1alpha1.RiskLevelHigh
		if api.RemovedInMinor <= targetMinor {
			severity = upgradev1alpha1.RiskLevelCritical
		}

		findings = append(findings, upgradev1alpha1.Finding{
			ID:       findingID(upgradev1alpha1.FindingTypeDeprecatedOrRemovedAPI, meta.Namespace, api.Kind, meta.Name, api.APIVersion),
			Type:     upgradev1alpha1.FindingTypeDeprecatedOrRemovedAPI,
			Severity: severity,
			Category: "DeprecatedAPI",
			Resource: resource(api.APIVersion, api.Kind, meta.Namespace, meta.Name),
			Message:  fmt.Sprintf("%s %s/%s is still written through %s, removed in Kubernetes 1.%d.", api.Kind, meta.Namespace, meta.Name, api.APIVersion, api.RemovedInMinor),
			Evidence: []upgradev1alpha1.Evidence{{
				ID:          evidenceID(upgradev1alpha1.FindingTypeDeprecatedOrRemovedAPI, meta.Namespace, api.Kind, meta.Name, api.APIVersion),
				Description: "Removed API version observed in managedFields or in the last-applied configuration.",
				Observed: map[string]string{
					"apiVersion":    api.APIVersion,
					"kind":          api.Kind,
					"removedIn":     "1." + strconv.Itoa(api.RemovedInMinor),
					"fieldManagers": strings.Join(writers, ","),
				},
			}},
			Recommendation: "Update the manifests and controllers writing this object to a served API version before upgrading.",
		})
	}
	return findings
}

// removedAPIWriters returns the field managers still writing through the removed
// version, plus "kubectl-last-applied" when the stored manifest uses it. An empty
// result means nothing observable points at the removed version.
func removedAPIWriters(meta metav1.ObjectMeta, api removedAPI) []string {
	var writers []string
	seen := map[string]struct{}{}
	add := func(manager string) {
		if _, ok := seen[manager]; ok {
			return
		}
		seen[manager] = struct{}{}
		writers = append(writers, manager)
	}

	for _, entry := range meta.ManagedFields {
		if entry.APIVersion != api.APIVersion {
			continue
		}
		manager := entry.Manager
		if manager == "" {
			manager = "unknown"
		}
		add(manager)
	}

	if usesDeprecatedSource(meta.Annotations, api) {
		add("kubectl-last-applied")
	}

	sort.Strings(writers)
	return writers
}

type appliedObjectHeader struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
}

func usesDeprecatedSource(annotations map[string]string, api removedAPI) bool {
	if len(annotations) == 0 {
		return false
	}
	value := strings.TrimSpace(annotations[lastAppliedAnnotation])
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
