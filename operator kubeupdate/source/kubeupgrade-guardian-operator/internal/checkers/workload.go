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
	"strconv"

	corev1 "k8s.io/api/core/v1"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/snapshot"
)

// WorkloadAvailability detects workloads unlikely to remain available during an upgrade.
type WorkloadAvailability struct{}

func (WorkloadAvailability) Name() string { return "workload-availability" }

func (WorkloadAvailability) Check(snap *snapshot.ClusterSnapshot, _ *upgradev1alpha1.UpgradeAssessment) []upgradev1alpha1.Finding {
	var findings []upgradev1alpha1.Finding

	for _, deployment := range snap.Deployments {
		if replicas := replicaCount(deployment.Spec.Replicas); replicas < 2 {
			findings = append(findings, replicaFinding("Deployment", deployment.Namespace, deployment.Name, replicas))
		}
	}

	for _, statefulSet := range snap.StatefulSets {
		if replicas := replicaCount(statefulSet.Spec.Replicas); replicas < 2 {
			findings = append(findings, replicaFinding("StatefulSet", statefulSet.Namespace, statefulSet.Name, replicas))
		}
	}

	scoped := namespaceSet(snap.Namespaces)
	for _, pod := range snap.Pods {
		if _, ok := scoped[pod.Namespace]; !ok {
			continue
		}
		if isStandalonePod(pod) {
			findings = append(findings, standalonePodFinding(pod))
		}
	}

	return findings
}

func replicaFinding(kind, namespace, name string, replicas int32) upgradev1alpha1.Finding {
	id := findingID(upgradev1alpha1.FindingTypeWorkloadAvailability, namespace, kind, name)
	return upgradev1alpha1.Finding{
		ID:       id,
		Type:     upgradev1alpha1.FindingTypeWorkloadAvailability,
		Severity: upgradev1alpha1.RiskLevelHigh,
		Category: "WorkloadAvailability",
		Resource: resource("apps/v1", kind, namespace, name),
		Message:  fmt.Sprintf("%s %s/%s has fewer than 2 replicas.", kind, namespace, name),
		Evidence: []upgradev1alpha1.Evidence{{
			ID:          evidenceID(upgradev1alpha1.FindingTypeWorkloadAvailability, namespace, kind, name),
			Description: "Replica count observed on the workload spec.",
			Observed: map[string]string{
				"replicas": strconv.Itoa(int(replicas)),
			},
		}},
		Recommendation: "Increase replicas to at least 2 or document why this workload can tolerate disruption.",
	}
}

func isStandalonePod(pod corev1.Pod) bool {
	for _, owner := range pod.OwnerReferences {
		if owner.Controller != nil && *owner.Controller {
			switch owner.Kind {
			case "Deployment", "ReplicaSet", "StatefulSet", "DaemonSet", "Job":
				return false
			}
		}
	}
	return true
}

func standalonePodFinding(pod corev1.Pod) upgradev1alpha1.Finding {
	return upgradev1alpha1.Finding{
		ID:       findingID(upgradev1alpha1.FindingTypeWorkloadAvailability, pod.Namespace, "Pod", pod.Name),
		Type:     upgradev1alpha1.FindingTypeWorkloadAvailability,
		Severity: upgradev1alpha1.RiskLevelMedium,
		Category: "WorkloadAvailability",
		Resource: resource("v1", "Pod", pod.Namespace, pod.Name),
		Message:  fmt.Sprintf("Pod %s/%s is standalone and may be lost during node disruption.", pod.Namespace, pod.Name),
		Evidence: []upgradev1alpha1.Evidence{{
			ID:          evidenceID(upgradev1alpha1.FindingTypeWorkloadAvailability, pod.Namespace, "Pod", pod.Name),
			Description: "Pod has no recognized controller owner reference.",
			Observed: map[string]string{
				"ownerReferences": strconv.Itoa(len(pod.OwnerReferences)),
			},
		}},
		Recommendation: "Manage the pod through a controller such as Deployment, StatefulSet, DaemonSet, or Job before upgrade.",
	}
}
