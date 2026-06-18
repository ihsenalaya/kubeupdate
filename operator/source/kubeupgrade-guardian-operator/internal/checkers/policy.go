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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
)

// PolicyRisk detects obvious conflicts with admission policy engines and Pod Security Admission.
type PolicyRisk struct{}

func (PolicyRisk) Name() string { return "policy-risks" }

func (p PolicyRisk) Check(ctx context.Context, c client.Client, assessment *upgradev1alpha1.UpgradeAssessment) ([]upgradev1alpha1.Finding, error) {
	nsList, err := namespaces(ctx, c, assessment)
	if err != nil {
		if isRBACDenied(err) {
			return rbacGap(p.Name(), err), nil
		}
		return nil, err
	}

	restricted := map[string]corev1.Namespace{}
	for _, ns := range nsList {
		var namespace corev1.Namespace
		if err := c.Get(ctx, client.ObjectKey{Name: ns}, &namespace); err != nil {
			if isRBACDenied(err) {
				return rbacGap(p.Name()+"/namespaces", err), nil
			}
			return nil, err
		}
		if namespace.Labels["pod-security.kubernetes.io/enforce"] == "restricted" {
			restricted[ns] = namespace
		}
	}

	var findings []upgradev1alpha1.Finding
	for _, namespace := range restricted {
		findings = append(findings, restrictedNamespaceFinding(namespace))
		findings = append(findings, p.incompatibleWorkloadFindings(ctx, c, namespace.Name)...)
	}

	engineFindings, err := p.policyEngineFindings(ctx, c)
	if err != nil {
		if isRBACDenied(err) {
			findings = append(findings, rbacGap(p.Name()+"/policy-engines", err)...)
			return findings, nil
		}
		return nil, err
	}
	findings = append(findings, engineFindings...)

	return findings, nil
}

func (p PolicyRisk) incompatibleWorkloadFindings(ctx context.Context, c client.Client, namespace string) []upgradev1alpha1.Finding {
	var findings []upgradev1alpha1.Finding

	var deployments appsv1.DeploymentList
	if err := c.List(ctx, &deployments, client.InNamespace(namespace)); err != nil {
		if isRBACDenied(err) {
			return rbacGap(p.Name()+"/deployments", err)
		}
		return []upgradev1alpha1.Finding{checkerErrorFinding(p.Name(), err)}
	}
	for _, item := range deployments.Items {
		findings = append(findings, podSpecPolicyFindings("Deployment", item.Namespace, item.Name, item.Spec.Template.Spec)...)
	}

	var statefulSets appsv1.StatefulSetList
	if err := c.List(ctx, &statefulSets, client.InNamespace(namespace)); err != nil {
		if isRBACDenied(err) {
			return rbacGap(p.Name()+"/statefulsets", err)
		}
		return []upgradev1alpha1.Finding{checkerErrorFinding(p.Name(), err)}
	}
	for _, item := range statefulSets.Items {
		findings = append(findings, podSpecPolicyFindings("StatefulSet", item.Namespace, item.Name, item.Spec.Template.Spec)...)
	}

	var daemonSets appsv1.DaemonSetList
	if err := c.List(ctx, &daemonSets, client.InNamespace(namespace)); err != nil {
		if isRBACDenied(err) {
			return rbacGap(p.Name()+"/daemonsets", err)
		}
		return []upgradev1alpha1.Finding{checkerErrorFinding(p.Name(), err)}
	}
	for _, item := range daemonSets.Items {
		findings = append(findings, podSpecPolicyFindings("DaemonSet", item.Namespace, item.Name, item.Spec.Template.Spec)...)
	}

	return findings
}

func (PolicyRisk) policyEngineFindings(ctx context.Context, c client.Client) ([]upgradev1alpha1.Finding, error) {
	var crds apiextensionsv1.CustomResourceDefinitionList
	if err := c.List(ctx, &crds); err != nil {
		return nil, err
	}

	var findings []upgradev1alpha1.Finding
	for _, crd := range crds.Items {
		switch crd.Name {
		case "clusterpolicies.kyverno.io", "policies.kyverno.io":
			findings = append(findings, policyEngineDetectedFinding("Kyverno", crd.Name))
		case "constrainttemplates.templates.gatekeeper.sh", "constraints.gatekeeper.sh":
			findings = append(findings, policyEngineDetectedFinding("Gatekeeper", crd.Name))
		}
	}
	return findings, nil
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

func checkerErrorFinding(checker string, err error) upgradev1alpha1.Finding {
	return upgradev1alpha1.Finding{
		ID:             findingID(upgradev1alpha1.FindingTypeRBACAssessmentGap, checker, "error"),
		Type:           upgradev1alpha1.FindingTypeRBACAssessmentGap,
		Severity:       upgradev1alpha1.RiskLevelHigh,
		Category:       "Assessment",
		Message:        err.Error(),
		Recommendation: "Fix the read error and rerun the assessment.",
	}
}
