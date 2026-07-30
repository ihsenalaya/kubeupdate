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
	resourceapi "k8s.io/apimachinery/pkg/api/resource"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/snapshot"
)

// mirrorPodAnnotation marks a static pod managed by a node's kubelet.
const mirrorPodAnnotation = "kubernetes.io/config.mirror"

// Capacity performs a conservative one-node-loss capacity estimate.
type Capacity struct{}

func (Capacity) Name() string { return "capacity" }

// Check reads the cluster-wide node and pod inventory: rescheduling headroom after
// a drain is a cluster property and must not be narrowed by the assessment scope.
func (Capacity) Check(snap *snapshot.ClusterSnapshot, _ *upgradev1alpha1.UpgradeAssessment) []upgradev1alpha1.Finding {
	totalCPU, totalMemory, largestCPU, largestMemory := int64(0), int64(0), int64(0), int64(0)
	for _, node := range snap.Nodes {
		if node.Spec.Unschedulable {
			continue
		}
		cpu := node.Status.Allocatable.Cpu().MilliValue()
		memory := node.Status.Allocatable.Memory().Value()
		totalCPU += cpu
		totalMemory += memory
		if cpu > largestCPU {
			largestCPU = cpu
		}
		if memory > largestMemory {
			largestMemory = memory
		}
	}

	demand := reschedulableDemand(snap.Pods)
	remainingCPU := totalCPU - largestCPU
	remainingMemory := totalMemory - largestMemory

	var findings []upgradev1alpha1.Finding
	switch {
	case demand.CPU > remainingCPU || demand.Memory > remainingMemory:
		findings = append(findings, capacityFinding(upgradev1alpha1.RiskLevelHigh, totalCPU, totalMemory, demand, remainingCPU, remainingMemory))
	case highUtilization(demand.CPU, remainingCPU) || highUtilization(demand.Memory, remainingMemory):
		findings = append(findings, capacityFinding(upgradev1alpha1.RiskLevelMedium, totalCPU, totalMemory, demand, remainingCPU, remainingMemory))
	}

	return findings
}

// reschedulableRequests is the amount of requested capacity that has to land
// somewhere else when a node goes away.
type reschedulableRequests struct {
	CPU      int64
	Memory   int64
	Pods     int
	Excluded int
}

// reschedulableDemand sums the effective requests of the pods a drain actually
// has to reschedule. DaemonSet pods are excluded: they are recreated by the
// DaemonSet on the remaining nodes only if those nodes do not already run one, so
// counting them inflates the demand. Static (mirror) pods belong to their node
// and never move. Terminated pods hold no capacity.
func reschedulableDemand(pods []corev1.Pod) reschedulableRequests {
	demand := reschedulableRequests{}
	for _, pod := range pods {
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			continue
		}
		if isDaemonSetPod(pod) || isMirrorPod(pod) {
			demand.Excluded++
			continue
		}
		cpu, memory := effectivePodRequests(pod.Spec)
		demand.CPU += cpu
		demand.Memory += memory
		demand.Pods++
	}
	return demand
}

// effectivePodRequests mirrors how the scheduler sizes a pod: the larger of the
// sum of regular containers (sidecars included, since a restartPolicy=Always init
// container runs for the whole pod lifetime) and the largest plain init container,
// plus the pod overhead of its runtime class.
func effectivePodRequests(spec corev1.PodSpec) (cpu, memory int64) {
	for _, container := range spec.Containers {
		cpu += container.Resources.Requests.Cpu().MilliValue()
		memory += container.Resources.Requests.Memory().Value()
	}

	var initCPU, initMemory int64
	for _, container := range spec.InitContainers {
		containerCPU := container.Resources.Requests.Cpu().MilliValue()
		containerMemory := container.Resources.Requests.Memory().Value()
		if isSidecarContainer(container) {
			cpu += containerCPU
			memory += containerMemory
			continue
		}
		if containerCPU > initCPU {
			initCPU = containerCPU
		}
		if containerMemory > initMemory {
			initMemory = containerMemory
		}
	}

	if initCPU > cpu {
		cpu = initCPU
	}
	if initMemory > memory {
		memory = initMemory
	}

	if spec.Overhead != nil {
		cpu += spec.Overhead.Cpu().MilliValue()
		memory += spec.Overhead.Memory().Value()
	}

	return cpu, memory
}

func isSidecarContainer(container corev1.Container) bool {
	return container.RestartPolicy != nil && *container.RestartPolicy == corev1.ContainerRestartPolicyAlways
}

func isDaemonSetPod(pod corev1.Pod) bool {
	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "DaemonSet" && owner.Controller != nil && *owner.Controller {
			return true
		}
	}
	return false
}

func isMirrorPod(pod corev1.Pod) bool {
	_, ok := pod.Annotations[mirrorPodAnnotation]
	return ok
}

func highUtilization(requested, capacity int64) bool {
	if capacity <= 0 {
		return requested > 0
	}
	return float64(requested)/float64(capacity) > 0.8
}

func capacityFinding(severity upgradev1alpha1.RiskLevel, totalCPU, totalMemory int64, demand reschedulableRequests, remainingCPU, remainingMemory int64) upgradev1alpha1.Finding {
	return upgradev1alpha1.Finding{
		ID:       findingID(upgradev1alpha1.FindingTypeCapacityDrainRisk, "one-node-loss"),
		Type:     upgradev1alpha1.FindingTypeCapacityDrainRisk,
		Severity: severity,
		Category: "Capacity",
		Message:  "Cluster may not have enough requested capacity headroom to tolerate one worker node loss.",
		Evidence: []upgradev1alpha1.Evidence{{
			ID:          evidenceID(upgradev1alpha1.FindingTypeCapacityDrainRisk, "one-node-loss"),
			Description: "Conservative capacity estimate using node allocatable minus the largest node, against the requests of the pods a drain has to reschedule.",
			Observed: map[string]string{
				"totalCPU":             strconv.FormatInt(totalCPU, 10) + "m",
				"totalMemory":          resourceapi.NewQuantity(totalMemory, resourceapi.BinarySI).String(),
				"requestedCPU":         strconv.FormatInt(demand.CPU, 10) + "m",
				"requestedMemory":      resourceapi.NewQuantity(demand.Memory, resourceapi.BinarySI).String(),
				"remainingCPU":         strconv.FormatInt(remainingCPU, 10) + "m",
				"remainingMemory":      resourceapi.NewQuantity(remainingMemory, resourceapi.BinarySI).String(),
				"reschedulablePods":    strconv.Itoa(demand.Pods),
				"nonReschedulablePods": strconv.Itoa(demand.Excluded),
			},
		}},
		Recommendation: fmt.Sprintf("Add capacity or reduce requests before upgrade. Estimated remaining capacity after one-node loss: %dm CPU.", remainingCPU),
	}
}
