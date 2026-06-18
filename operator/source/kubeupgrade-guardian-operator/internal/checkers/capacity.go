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

	corev1 "k8s.io/api/core/v1"
	resourceapi "k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
)

// Capacity performs a conservative one-node-loss capacity estimate.
type Capacity struct{}

func (Capacity) Name() string { return "capacity" }

func (cap Capacity) Check(ctx context.Context, c client.Client, _ *upgradev1alpha1.UpgradeAssessment) ([]upgradev1alpha1.Finding, error) {
	var nodes corev1.NodeList
	if err := c.List(ctx, &nodes); err != nil {
		if isRBACDenied(err) {
			return rbacGap(cap.Name()+"/nodes", err), nil
		}
		return nil, err
	}

	var pods corev1.PodList
	if err := c.List(ctx, &pods); err != nil {
		if isRBACDenied(err) {
			return rbacGap(cap.Name()+"/pods", err), nil
		}
		return nil, err
	}

	totalCPU, totalMemory, largestCPU, largestMemory := int64(0), int64(0), int64(0), int64(0)
	for _, node := range nodes.Items {
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

	requestedCPU, requestedMemory := podRequests(pods.Items)
	remainingCPU := totalCPU - largestCPU
	remainingMemory := totalMemory - largestMemory

	var findings []upgradev1alpha1.Finding
	if requestedCPU > remainingCPU || requestedMemory > remainingMemory {
		findings = append(findings, capacityFinding(upgradev1alpha1.RiskLevelHigh, totalCPU, totalMemory, requestedCPU, requestedMemory, remainingCPU, remainingMemory))
	} else if highUtilization(requestedCPU, remainingCPU) || highUtilization(requestedMemory, remainingMemory) {
		findings = append(findings, capacityFinding(upgradev1alpha1.RiskLevelMedium, totalCPU, totalMemory, requestedCPU, requestedMemory, remainingCPU, remainingMemory))
	}

	return findings, nil
}

func podRequests(pods []corev1.Pod) (int64, int64) {
	var cpu, memory int64
	for _, pod := range pods {
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			continue
		}
		for _, container := range pod.Spec.Containers {
			cpu += container.Resources.Requests.Cpu().MilliValue()
			memory += container.Resources.Requests.Memory().Value()
		}
	}
	return cpu, memory
}

func highUtilization(requested, capacity int64) bool {
	if capacity <= 0 {
		return requested > 0
	}
	return float64(requested)/float64(capacity) > 0.8
}

func capacityFinding(severity upgradev1alpha1.RiskLevel, totalCPU, totalMemory, requestedCPU, requestedMemory, remainingCPU, remainingMemory int64) upgradev1alpha1.Finding {
	return upgradev1alpha1.Finding{
		ID:       findingID(upgradev1alpha1.FindingTypeCapacityDrainRisk, "one-node-loss"),
		Type:     upgradev1alpha1.FindingTypeCapacityDrainRisk,
		Severity: severity,
		Category: "Capacity",
		Message:  "Cluster may not have enough requested capacity headroom to tolerate one worker node loss.",
		Evidence: []upgradev1alpha1.Evidence{{
			ID:          evidenceID(upgradev1alpha1.FindingTypeCapacityDrainRisk, "one-node-loss"),
			Description: "Conservative capacity estimate using node allocatable minus the largest node.",
			Observed: map[string]string{
				"totalCPU":        strconv.FormatInt(totalCPU, 10) + "m",
				"totalMemory":     resourceapi.NewQuantity(totalMemory, resourceapi.BinarySI).String(),
				"requestedCPU":    strconv.FormatInt(requestedCPU, 10) + "m",
				"requestedMemory": resourceapi.NewQuantity(requestedMemory, resourceapi.BinarySI).String(),
				"remainingCPU":    strconv.FormatInt(remainingCPU, 10) + "m",
				"remainingMemory": resourceapi.NewQuantity(remainingMemory, resourceapi.BinarySI).String(),
			},
		}},
		Recommendation: fmt.Sprintf("Add capacity or reduce requests before upgrade. Estimated remaining capacity after one-node loss: %dm CPU.", remainingCPU),
	}
}
