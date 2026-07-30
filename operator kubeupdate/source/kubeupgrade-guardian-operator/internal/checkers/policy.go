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

	corev1 "k8s.io/api/core/v1"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/snapshot"
)

// PolicyRisk detects obvious conflicts with admission policy engines and Pod Security Admission.
type PolicyRisk struct{}

func (PolicyRisk) Name() string { return "policy-risks" }

func (p PolicyRisk) Check(snap *snapshot.ClusterSnapshot, _ *upgradev1alpha1.UpgradeAssessment) []upgradev1alpha1.Finding {
	var restricted []corev1.Namespace
	for _, namespace := range snap.Namespaces {
		if namespace.Labels["pod-security.kubernetes.io/enforce"] == "restricted" {
			restricted = append(restricted, namespace)
		}
	}

	findings := make([]upgradev1alpha1.Finding, 0, len(restricted))
	for _, namespace := range restricted {
		findings = append(findings, restrictedNamespaceFinding(namespace))
		findings = append(findings, incompatibleWorkloadFindings(snap, namespace.Name)...)
	}

	return append(findings, policyEngineFindings(snap)...)
}

func incompatibleWorkloadFindings(snap *snapshot.ClusterSnapshot, namespace string) []upgradev1alpha1.Finding {
	var findings []upgradev1alpha1.Finding

	for _, item := range snap.Deployments {
		if item.Namespace == namespace {
			findings = append(findings, podSpecPolicyFindings("Deployment", item.Namespace, item.Name, item.Spec.Template.Spec)...)
		}
	}
	for _, item := range snap.StatefulSets {
		if item.Namespace == namespace {
			findings = append(findings, podSpecPolicyFindings("StatefulSet", item.Namespace, item.Name, item.Spec.Template.Spec)...)
		}
	}
	for _, item := range snap.DaemonSets {
		if item.Namespace == namespace {
			findings = append(findings, podSpecPolicyFindings("DaemonSet", item.Namespace, item.Name, item.Spec.Template.Spec)...)
		}
	}

	return findings
}

func policyEngineFindings(snap *snapshot.ClusterSnapshot) []upgradev1alpha1.Finding {
	names := make([]string, 0, len(snap.CRDNames))
	for name := range snap.CRDNames {
		names = append(names, name)
	}
	sort.Strings(names)

	var findings []upgradev1alpha1.Finding
	for _, name := range names {
		switch name {
		case "clusterpolicies.kyverno.io", "policies.kyverno.io":
			findings = append(findings, policyEngineDetectedFinding("Kyverno", name))
		case "constrainttemplates.templates.gatekeeper.sh", "constraints.gatekeeper.sh":
			findings = append(findings, policyEngineDetectedFinding("Gatekeeper", name))
		}
	}
	return findings
}

func restrictedNamespaceFinding(namespace corev1.Namespace) upgradev1alpha1.Finding {
	return upgradev1alpha1.Finding{
		ID:       findingID(upgradev1alpha1.FindingTypePolicyRisk, namespace.Name, "restricted"),
		Type:     upgradev1alpha1.FindingTypePolicyRisk,
		Severity: upgradev1alpha1.RiskLevelMedium,
		Category: "PolicyRisk",
		Resource: resource("v1", "Namespace", "", namespace.Name),
		Message:  fmt.Sprintf("Namespace %s enforces Pod Security restricted.", namespace.Name),
		Evidence: []upgradev1alpha1.Evidence{{
			ID:          evidenceID(upgradev1alpha1.FindingTypePolicyRisk, namespace.Name, "restricted"),
			Description: "Pod Security Admission namespace label.",
			Observed: map[string]string{
				"pod-security.kubernetes.io/enforce": "restricted",
			},
		}},
		Recommendation: "Validate all workloads in this namespace against the restricted Pod Security profile before upgrade.",
	}
}

func podSpecPolicyFindings(kind, namespace, name string, spec corev1.PodSpec) []upgradev1alpha1.Finding {
	var findings []upgradev1alpha1.Finding
	for _, volume := range spec.Volumes {
		if volume.HostPath != nil {
			findings = append(findings, policyFinding(kind, namespace, name, "hostPath volume", volume.Name))
		}
	}
	for _, container := range spec.Containers {
		if container.SecurityContext == nil {
			findings = append(findings, policyFinding(kind, namespace, name, "missing securityContext", container.Name))
			continue
		}
		if container.SecurityContext.Privileged != nil && *container.SecurityContext.Privileged {
			findings = append(findings, policyFinding(kind, namespace, name, "privileged=true", container.Name))
		}
		if container.SecurityContext.RunAsNonRoot == nil {
			findings = append(findings, policyFinding(kind, namespace, name, "runAsNonRoot absent", container.Name))
		}
		if container.SecurityContext.AllowPrivilegeEscalation != nil && *container.SecurityContext.AllowPrivilegeEscalation {
			findings = append(findings, policyFinding(kind, namespace, name, "allowPrivilegeEscalation=true", container.Name))
		}
	}
	return findings
}

func policyFinding(kind, namespace, name, reason, subject string) upgradev1alpha1.Finding {
	return upgradev1alpha1.Finding{
		ID:       findingID(upgradev1alpha1.FindingTypePolicyRisk, namespace, kind, name, reason, subject),
		Type:     upgradev1alpha1.FindingTypePolicyRisk,
		Severity: upgradev1alpha1.RiskLevelHigh,
		Category: "PolicyRisk",
		Resource: resource("apps/v1", kind, namespace, name),
		Message:  fmt.Sprintf("%s %s/%s may violate restricted policy: %s.", kind, namespace, name, reason),
		Evidence: []upgradev1alpha1.Evidence{{
			ID:          evidenceID(upgradev1alpha1.FindingTypePolicyRisk, namespace, kind, name, subject),
			Description: "Workload pod template has a restricted policy incompatibility.",
			Observed: map[string]string{
				"reason":  reason,
				"subject": subject,
			},
		}},
		Recommendation: "Adjust the pod security context or namespace policy before upgrade.",
	}
}

func policyEngineDetectedFinding(engine, crdName string) upgradev1alpha1.Finding {
	return upgradev1alpha1.Finding{
		ID:       findingID(upgradev1alpha1.FindingTypePolicyEngineDetected, engine, crdName),
		Type:     upgradev1alpha1.FindingTypePolicyEngineDetected,
		Severity: upgradev1alpha1.RiskLevelInfo,
		Category: "PolicyRisk",
		Resource: resource("apiextensions.k8s.io/v1", "CustomResourceDefinition", "", crdName),
		Message:  fmt.Sprintf("%s policy engine CRD detected.", engine),
		Evidence: []upgradev1alpha1.Evidence{{
			ID:          evidenceID(upgradev1alpha1.FindingTypePolicyEngineDetected, engine, crdName),
			Description: "Policy engine CRD exists in the cluster.",
			Observed: map[string]string{
				"engine": engine,
				"crd":    crdName,
			},
		}},
		Recommendation: "Review policy reports and admission behavior before upgrade.",
	}
}
