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
	"sort"
	"strconv"

	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/snapshot"
)

// PDB detects disruption budgets that can block voluntary disruption.
type PDB struct{}

func (PDB) Name() string { return "pdb" }

type workloadReplicaRef struct {
	Kind      string
	Namespace string
	Name      string
	Labels    map[string]string
	Replicas  int32
}

func (PDB) Check(snap *snapshot.ClusterSnapshot, _ *upgradev1alpha1.UpgradeAssessment) []upgradev1alpha1.Finding {
	workloadsByNamespace := pdbWorkloads(snap)
	pdbsByNamespace := make(map[string][]policyv1.PodDisruptionBudget, len(snap.PDBs))
	for _, pdb := range snap.PDBs {
		pdbsByNamespace[pdb.Namespace] = append(pdbsByNamespace[pdb.Namespace], pdb)
	}

	var findings []upgradev1alpha1.Finding
	for _, namespace := range snap.Namespaces {
		workloads := workloadsByNamespace[namespace.Name]

		matchedWorkloads := map[string]struct{}{}
		for _, pdb := range pdbsByNamespace[namespace.Name] {
			pdbFindings, matched := evaluatePDB(pdb, workloads)
			findings = append(findings, pdbFindings...)
			for _, key := range matched {
				matchedWorkloads[key] = struct{}{}
			}
		}

		for _, workload := range workloads {
			if workload.Replicas < 2 {
				continue
			}
			if _, ok := matchedWorkloads[workloadKey(workload)]; ok {
				continue
			}
			findings = append(findings, workloadWithoutPDBFinding(workload))
		}
	}

	return findings
}

// pdbWorkloads indexes the assessed workloads by namespace: a PodDisruptionBudget
// only ever selects pods of its own namespace.
func pdbWorkloads(snap *snapshot.ClusterSnapshot) map[string][]workloadReplicaRef {
	byNamespace := map[string][]workloadReplicaRef{}

	for _, item := range snap.Deployments {
		byNamespace[item.Namespace] = append(byNamespace[item.Namespace], workloadReplicaRef{
			Kind:      "Deployment",
			Namespace: item.Namespace,
			Name:      item.Name,
			Labels:    item.Spec.Template.Labels,
			Replicas:  replicaCount(item.Spec.Replicas),
		})
	}

	for _, item := range snap.StatefulSets {
		byNamespace[item.Namespace] = append(byNamespace[item.Namespace], workloadReplicaRef{
			Kind:      "StatefulSet",
			Namespace: item.Namespace,
			Name:      item.Name,
			Labels:    item.Spec.Template.Labels,
			Replicas:  replicaCount(item.Spec.Replicas),
		})
	}

	for namespace := range byNamespace {
		workloads := byNamespace[namespace]
		sort.SliceStable(workloads, func(i, j int) bool {
			if workloads[i].Kind != workloads[j].Kind {
				return workloads[i].Kind < workloads[j].Kind
			}
			return workloads[i].Name < workloads[j].Name
		})
	}

	return byNamespace
}

func evaluatePDB(pdb policyv1.PodDisruptionBudget, workloads []workloadReplicaRef) ([]upgradev1alpha1.Finding, []string) {
	if pdb.Spec.Selector == nil {
		return []upgradev1alpha1.Finding{pdbWithoutMatchFinding(pdb, "missing selector")}, nil
	}

	selector, err := metav1LabelSelectorAsSelector(pdb)
	if err != nil {
		return []upgradev1alpha1.Finding{pdbWithoutMatchFinding(pdb, err.Error())}, nil
	}

	var findings []upgradev1alpha1.Finding
	matched := make([]string, 0, len(workloads))
	for _, workload := range workloads {
		if !selector.Matches(labels.Set(workload.Labels)) {
			continue
		}
		matched = append(matched, workloadKey(workload))
		findings = append(findings, pdbBlockingFindings(pdb, workload)...)
	}

	if len(matched) == 0 {
		findings = append(findings, pdbWithoutMatchFinding(pdb, selector.String()))
	}

	return findings, matched
}

func metav1LabelSelectorAsSelector(pdb policyv1.PodDisruptionBudget) (labels.Selector, error) {
	return metav1LabelSelectorAsSelectorFunc(pdb.Spec.Selector)
}

var metav1LabelSelectorAsSelectorFunc = func(selector *metav1.LabelSelector) (labels.Selector, error) {
	return metav1.LabelSelectorAsSelector(selector)
}

func pdbBlockingFindings(pdb policyv1.PodDisruptionBudget, workload workloadReplicaRef) []upgradev1alpha1.Finding {
	var findings []upgradev1alpha1.Finding
	if pdb.Spec.MinAvailable != nil {
		minAvailable := intOrStringValue(*pdb.Spec.MinAvailable, workload.Replicas)
		if minAvailable >= workload.Replicas {
			findings = append(findings, pdbBlockingFinding(pdb, workload, upgradev1alpha1.RiskLevelCritical, "minAvailable", minAvailable))
		}
	}
	if pdb.Spec.MaxUnavailable != nil {
		maxUnavailable := intOrStringValue(*pdb.Spec.MaxUnavailable, workload.Replicas)
		if maxUnavailable == 0 {
			findings = append(findings, pdbBlockingFinding(pdb, workload, upgradev1alpha1.RiskLevelHigh, "maxUnavailable", maxUnavailable))
		}
	}
	return findings
}

func intOrStringValue(value intstr.IntOrString, replicas int32) int32 {
	if value.Type == intstr.String {
		parsed, err := intstr.GetScaledValueFromIntOrPercent(&value, int(replicas), false)
		if err != nil {
			return 0
		}
		return int32(parsed)
	}
	return int32(value.IntValue())
}

func pdbBlockingFinding(pdb policyv1.PodDisruptionBudget, workload workloadReplicaRef, severity upgradev1alpha1.RiskLevel, field string, value int32) upgradev1alpha1.Finding {
	return upgradev1alpha1.Finding{
		ID:       findingID(upgradev1alpha1.FindingTypePDBBlockingRisk, pdb.Namespace, pdb.Name, workload.Kind, workload.Name, field),
		Type:     upgradev1alpha1.FindingTypePDBBlockingRisk,
		Severity: severity,
		Category: "PDB",
		Resource: resource("policy/v1", "PodDisruptionBudget", pdb.Namespace, pdb.Name),
		Message:  fmt.Sprintf("PDB %s/%s may block disruption for %s %s/%s.", pdb.Namespace, pdb.Name, workload.Kind, workload.Namespace, workload.Name),
		Evidence: []upgradev1alpha1.Evidence{{
			ID:          evidenceID(upgradev1alpha1.FindingTypePDBBlockingRisk, pdb.Namespace, pdb.Name, workload.Name),
			Description: "PDB and workload replica relationship.",
			Observed: map[string]string{
				"workloadKind": workload.Kind,
				"workloadName": workload.Name,
				"replicas":     strconv.Itoa(int(workload.Replicas)),
				field:          strconv.Itoa(int(value)),
				"selector":     labels.SelectorFromSet(pdb.Spec.Selector.MatchLabels).String(),
			},
		}},
		Recommendation: "Increase workload replicas or relax the PodDisruptionBudget before upgrade.",
	}
}

func pdbWithoutMatchFinding(pdb policyv1.PodDisruptionBudget, selector string) upgradev1alpha1.Finding {
	return upgradev1alpha1.Finding{
		ID:       findingID(upgradev1alpha1.FindingTypePDBBlockingRisk, pdb.Namespace, pdb.Name, "no-match"),
		Type:     upgradev1alpha1.FindingTypePDBBlockingRisk,
		Severity: upgradev1alpha1.RiskLevelHigh,
		Category: "PDB",
		Resource: resource("policy/v1", "PodDisruptionBudget", pdb.Namespace, pdb.Name),
		Message:  fmt.Sprintf("PDB %s/%s does not match any assessed workload.", pdb.Namespace, pdb.Name),
		Evidence: []upgradev1alpha1.Evidence{{
			ID:          evidenceID(upgradev1alpha1.FindingTypePDBBlockingRisk, pdb.Namespace, pdb.Name, "no-match"),
			Description: "No workload matched the PDB selector.",
			Observed: map[string]string{
				"selector": selector,
			},
		}},
		Recommendation: "Fix or remove stale PodDisruptionBudgets before upgrade validation.",
	}
}

func workloadWithoutPDBFinding(workload workloadReplicaRef) upgradev1alpha1.Finding {
	return upgradev1alpha1.Finding{
		ID:       findingID(upgradev1alpha1.FindingTypePDBBlockingRisk, workload.Namespace, workload.Kind, workload.Name, "missing"),
		Type:     upgradev1alpha1.FindingTypePDBBlockingRisk,
		Severity: upgradev1alpha1.RiskLevelLow,
		Category: "PDB",
		Resource: resource("apps/v1", workload.Kind, workload.Namespace, workload.Name),
		Message:  fmt.Sprintf("%s %s/%s has no matching PodDisruptionBudget.", workload.Kind, workload.Namespace, workload.Name),
		Evidence: []upgradev1alpha1.Evidence{{
			ID:          evidenceID(upgradev1alpha1.FindingTypePDBBlockingRisk, workload.Namespace, workload.Kind, workload.Name, "missing"),
			Description: "No PDB selector matched this workload.",
			Observed: map[string]string{
				"replicas": strconv.Itoa(int(workload.Replicas)),
			},
		}},
		Recommendation: "Consider adding a PodDisruptionBudget for critical workloads before upgrade.",
	}
}

func workloadKey(workload workloadReplicaRef) string {
	return workload.Namespace + "/" + workload.Kind + "/" + workload.Name
}
