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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
)

// ReadinessProbe detects containers that do not expose readiness information.
type ReadinessProbe struct{}

func (ReadinessProbe) Name() string { return "readiness-probes" }

func (r ReadinessProbe) Check(ctx context.Context, c client.Client, assessment *upgradev1alpha1.UpgradeAssessment) ([]upgradev1alpha1.Finding, error) {
	nsList, err := namespaces(ctx, c, assessment)
	if err != nil {
		if isRBACDenied(err) {
			return rbacGap(r.Name(), err), nil
		}
		return nil, err
	}

	var findings []upgradev1alpha1.Finding
	for _, ns := range nsList {
		var deployments appsv1.DeploymentList
		if err := c.List(ctx, &deployments, client.InNamespace(ns)); err != nil {
			if isRBACDenied(err) {
				findings = append(findings, rbacGap(r.Name()+"/deployments", err)...)
				continue
			}
			return nil, err
		}
		for _, item := range deployments.Items {
			findings = append(findings, missingReadinessFindings("Deployment", item.Namespace, item.Name, item.Spec.Template.Spec.Containers)...)
		}

		var statefulSets appsv1.StatefulSetList
		if err := c.List(ctx, &statefulSets, client.InNamespace(ns)); err != nil {
			if isRBACDenied(err) {
				findings = append(findings, rbacGap(r.Name()+"/statefulsets", err)...)
				continue
			}
			return nil, err
		}
		for _, item := range statefulSets.Items {
			findings = append(findings, missingReadinessFindings("StatefulSet", item.Namespace, item.Name, item.Spec.Template.Spec.Containers)...)
		}

		var daemonSets appsv1.DaemonSetList
		if err := c.List(ctx, &daemonSets, client.InNamespace(ns)); err != nil {
			if isRBACDenied(err) {
				findings = append(findings, rbacGap(r.Name()+"/daemonsets", err)...)
				continue
			}
			return nil, err
		}
		for _, item := range daemonSets.Items {
			findings = append(findings, missingReadinessFindings("DaemonSet", item.Namespace, item.Name, item.Spec.Template.Spec.Containers)...)
		}
	}

	return findings, nil
}

func missingReadinessFindings(kind, namespace, name string, containers []corev1.Container) []upgradev1alpha1.Finding {
	var findings []upgradev1alpha1.Finding
	for _, container := range containers {
		if container.ReadinessProbe != nil {
			continue
		}
		findings = append(findings, upgradev1alpha1.Finding{
			ID:       findingID(upgradev1alpha1.FindingTypeMissingReadinessProbe, namespace, kind, name, container.Name),
			Type:     upgradev1alpha1.FindingTypeMissingReadinessProbe,
			Severity: upgradev1alpha1.RiskLevelMedium,
			Category: "ReadinessProbes",
			Resource: resource("apps/v1", kind, namespace, name),
			Message:  fmt.Sprintf("Container %s in %s %s/%s has no readinessProbe.", container.Name, kind, namespace, name),
			Evidence: []upgradev1alpha1.Evidence{{
				ID:          evidenceID(upgradev1alpha1.FindingTypeMissingReadinessProbe, namespace, kind, name, container.Name),
				Description: "Container readinessProbe is absent.",
				Observed: map[string]string{
					"containerName": container.Name,
				},
			}},
			Recommendation: "Add a readinessProbe that reflects whether the container can safely receive traffic.",
		})
	}
	return findings
}
