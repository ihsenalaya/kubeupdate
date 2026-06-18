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
	"strings"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
)

// Observability detects whether monitoring signals needed for upgrade validation are present.
type Observability struct{}

func (Observability) Name() string { return "observability" }

func (o Observability) Check(ctx context.Context, c client.Client, _ *upgradev1alpha1.UpgradeAssessment) ([]upgradev1alpha1.Finding, error) {
	var findings []upgradev1alpha1.Finding

	var namespaces corev1.NamespaceList
	if err := c.List(ctx, &namespaces); err != nil {
		if isRBACDenied(err) {
			return rbacGap(o.Name()+"/namespaces", err), nil
		}
		return nil, err
	}

	hasObservabilityNamespace := false
	for _, ns := range namespaces.Items {
		switch ns.Name {
		case "monitoring", "prometheus", "observability":
			hasObservabilityNamespace = true
		}
	}
	if !hasObservabilityNamespace {
		findings = append(findings, observabilityGapFinding("namespace", "No monitoring, prometheus, or observability namespace found.", upgradev1alpha1.RiskLevelMedium))
	}

	var crds apiextensionsv1.CustomResourceDefinitionList
	if err := c.List(ctx, &crds); err != nil {
		if isRBACDenied(err) {
			findings = append(findings, rbacGap(o.Name()+"/crds", err)...)
			return findings, nil
		}
		return nil, err
	}

	crdNames := map[string]struct{}{}
	for _, crd := range crds.Items {
		crdNames[crd.Name] = struct{}{}
	}

	for _, name := range []string{"servicemonitors.monitoring.coreos.com", "podmonitors.monitoring.coreos.com", "prometheuses.monitoring.coreos.com"} {
		if _, ok := crdNames[name]; ok {
			findings = append(findings, observabilityCapabilityFinding(name))
		}
	}
	if _, ok := crdNames["prometheuses.monitoring.coreos.com"]; !ok {
		findings = append(findings, observabilityGapFinding("prometheus-crd", "Prometheus CRD not detected; upgrade observability validation may be incomplete.", upgradev1alpha1.RiskLevelLow))
	}

	return findings, nil
}

func observabilityGapFinding(id, message string, severity upgradev1alpha1.RiskLevel) upgradev1alpha1.Finding {
	return upgradev1alpha1.Finding{
		ID:             findingID(upgradev1alpha1.FindingTypeObservabilityGap, id),
		Type:           upgradev1alpha1.FindingTypeObservabilityGap,
		Severity:       severity,
		Category:       "Observability",
		Message:        message,
		Recommendation: "Install or validate monitoring coverage before upgrade.",
	}
}

func observabilityCapabilityFinding(crd string) upgradev1alpha1.Finding {
	return upgradev1alpha1.Finding{
		ID:       findingID(upgradev1alpha1.FindingTypeObservabilityCapability, crd),
		Type:     upgradev1alpha1.FindingTypeObservabilityCapability,
		Severity: upgradev1alpha1.RiskLevelInfo,
		Category: "Observability",
		Resource: resource("apiextensions.k8s.io/v1", "CustomResourceDefinition", "", crd),
		Message:  "Observability CRD detected: " + crd,
		Evidence: []upgradev1alpha1.Evidence{{
			ID:          evidenceID(upgradev1alpha1.FindingTypeObservabilityCapability, crd),
			Description: "Monitoring capability CRD exists.",
			Observed: map[string]string{
				"crd":      crd,
				"category": strings.Split(crd, ".")[0],
			},
		}},
		Recommendation: "Use existing observability to validate workload health before and after upgrade.",
	}
}
