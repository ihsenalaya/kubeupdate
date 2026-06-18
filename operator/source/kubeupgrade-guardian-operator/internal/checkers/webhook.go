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

	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
)

// AdmissionWebhook detects upgrade risks introduced by admission webhooks.
type AdmissionWebhook struct{}

func (AdmissionWebhook) Name() string { return "admission-webhooks" }

func (a AdmissionWebhook) Check(ctx context.Context, c client.Client, _ *upgradev1alpha1.UpgradeAssessment) ([]upgradev1alpha1.Finding, error) {
	var findings []upgradev1alpha1.Finding

	var validating admissionv1.ValidatingWebhookConfigurationList
	if err := c.List(ctx, &validating); err != nil {
		if isRBACDenied(err) {
			return rbacGap(a.Name()+"/validating", err), nil
		}
		return nil, err
	}
	for _, item := range validating.Items {
		for _, webhook := range item.Webhooks {
			findings = append(findings, evaluateWebhook(ctx, c, "ValidatingWebhookConfiguration", item.Name, webhook.Name, webhook.FailurePolicy, webhook.ClientConfig.Service, isBroadNamespaceSelector(webhook.NamespaceSelector))...)
		}
	}

	var mutating admissionv1.MutatingWebhookConfigurationList
	if err := c.List(ctx, &mutating); err != nil {
		if isRBACDenied(err) {
			findings = append(findings, rbacGap(a.Name()+"/mutating", err)...)
			return findings, nil
		}
		return nil, err
	}
	for _, item := range mutating.Items {
		for _, webhook := range item.Webhooks {
			findings = append(findings, evaluateWebhook(ctx, c, "MutatingWebhookConfiguration", item.Name, webhook.Name, webhook.FailurePolicy, webhook.ClientConfig.Service, isBroadNamespaceSelector(webhook.NamespaceSelector))...)
		}
	}

	return findings, nil
}

func evaluateWebhook(ctx context.Context, c client.Client, kind, configName, webhookName string, failurePolicy *admissionv1.FailurePolicyType, service *admissionv1.ServiceReference, broadScope bool) []upgradev1alpha1.Finding {
	var findings []upgradev1alpha1.Finding

	if failurePolicy != nil && *failurePolicy == admissionv1.Fail {
		findings = append(findings, webhookFinding(kind, configName, webhookName, upgradev1alpha1.RiskLevelHigh, "failurePolicy=Fail", "Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical."))
	}

	if service != nil {
		var svc corev1.Service
		err := c.Get(ctx, client.ObjectKey{Namespace: service.Namespace, Name: service.Name}, &svc)
		if apierrors.IsNotFound(err) {
			findings = append(findings, webhookFinding(kind, configName, webhookName, upgradev1alpha1.RiskLevelCritical, "referenced service is absent", "Restore the webhook Service or remove the webhook configuration before upgrade."))
		} else if isRBACDenied(err) {
			findings = append(findings, rbacGap("admission-webhooks/service", err)...)
		}
	}

	if broadScope {
		findings = append(findings, webhookFinding(kind, configName, webhookName, upgradev1alpha1.RiskLevelHigh, "namespaceSelector is absent", "Constrain webhook scope with namespaceSelector/objectSelector where possible."))
	}

	return findings
}

func isBroadNamespaceSelector(selector *metav1.LabelSelector) bool {
	return selector == nil || (len(selector.MatchLabels) == 0 && len(selector.MatchExpressions) == 0)
}

func webhookFinding(kind, configName, webhookName string, severity upgradev1alpha1.RiskLevel, reason, recommendation string) upgradev1alpha1.Finding {
	return upgradev1alpha1.Finding{
		ID:       findingID(upgradev1alpha1.FindingTypeAdmissionWebhookRisk, kind, configName, webhookName, reason),
		Type:     upgradev1alpha1.FindingTypeAdmissionWebhookRisk,
		Severity: severity,
		Category: "AdmissionWebhook",
		Resource: resource("admissionregistration.k8s.io/v1", kind, "", configName),
		Message:  fmt.Sprintf("%s %s webhook %s has risk: %s.", kind, configName, webhookName, reason),
		Evidence: []upgradev1alpha1.Evidence{{
			ID:          evidenceID(upgradev1alpha1.FindingTypeAdmissionWebhookRisk, kind, configName, webhookName),
			Description: "Admission webhook configuration risk.",
			Observed: map[string]string{
				"webhookName": webhookName,
				"reason":      reason,
			},
		}},
		Recommendation: recommendation,
	}
}
