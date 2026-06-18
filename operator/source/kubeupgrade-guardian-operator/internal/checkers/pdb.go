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
	"context"
	"fmt"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
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

func (p PDB) Check(ctx context.Context, c client.Client, assessment *upgradev1alpha1.UpgradeAssessment) ([]upgradev1alpha1.Finding, error) {
	nsList, err := namespaces(ctx, c, assessment)
	if err != nil {
		if isRBACDenied(err) {
			return rbacGap(p.Name(), err), nil
		}
		return nil, err
	}

	var findings []upgradev1alpha1.Finding
	for _, ns := range nsList {
		workloads, err := pdbWorkloads(ctx, c, ns)
		if err != nil {
			if isRBACDenied(err) {
				findings = append(findings, rbacGap(p.Name()+"/workloads", err)...)
				continue
			}
			return nil, err
		}

		var pdbs policyv1.PodDisruptionBudgetList
		if err := c.List(ctx, &pdbs, client.InNamespace(ns)); err != nil {
			if isRBACDenied(err) {
				findings = append(findings, rbacGap(p.Name()+"/poddisruptionbudgets", err)...)
				continue
			}
			return nil, err
		}

		matchedWorkloads := map[string]struct{}{}
		for _, pdb := range pdbs.Items {
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

	return findings, nil
}

func pdbWorkloads(ctx context.Context, c client.Client, namespace string) ([]workloadReplicaRef, error) {
	var workloads []workloadReplicaRef

	var deployments appsv1.DeploymentList
	if err := c.List(ctx, &deployments, client.InNamespace(namespace)); err != nil {
		return nil, err
	}
	for _, item := range deployments.Items {
		replicas := int32(1)
		if item.Spec.Replicas != nil {
			replicas = *item.Spec.Replicas
		}
		workloads = append(workloads, workloadReplicaRef{
			Kind:      "Deployment",
			Namespace: item.Namespace,
			Name:      item.Name,
			Labels:    item.Spec.Template.Labels,
			Replicas:  replicas,
		})
	}

	var statefulSets appsv1.StatefulSetList
	if err := c.List(ctx, &statefulSets, client.InNamespace(namespace)); err != nil {
		return nil, err
	}
	for _, item := range statefulSets.Items {
		replicas := int32(1)
		if item.Spec.Replicas != nil {
			replicas = *item.Spec.Replicas
		}
		workloads = append(workloads, workloadReplicaRef{
			Kind:      "StatefulSet",
			Namespace: item.Namespace,
			Name:      item.Name,
			Labels:    item.Spec.Template.Labels,
			Replicas:  replicas,
		})
	}

	return workloads, nil
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
	var matched []string
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
