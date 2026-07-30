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

	admissionv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/snapshot"
)

// AdmissionWebhook detects upgrade risks introduced by admission webhooks.
type AdmissionWebhook struct{}

func (AdmissionWebhook) Name() string { return "admission-webhooks" }

func (AdmissionWebhook) Check(snap *snapshot.ClusterSnapshot, _ *upgradev1alpha1.UpgradeAssessment) []upgradev1alpha1.Finding {
	services := serviceSet(snap)

	var findings []upgradev1alpha1.Finding
	for _, item := range snap.ValidatingWebhooks {
		for _, webhook := range item.Webhooks {
			findings = append(findings, evaluateWebhook(services, "ValidatingWebhookConfiguration", item.Name, webhook.Name, webhook.FailurePolicy, webhook.ClientConfig.Service, isBroadNamespaceSelector(webhook.NamespaceSelector))...)
		}
	}
	for _, item := range snap.MutatingWebhooks {
		for _, webhook := range item.Webhooks {
			findings = append(findings, evaluateWebhook(services, "MutatingWebhookConfiguration", item.Name, webhook.Name, webhook.FailurePolicy, webhook.ClientConfig.Service, isBroadNamespaceSelector(webhook.NamespaceSelector))...)
		}
	}

	return findings
}

// serviceSet indexes webhook backends by namespace/name. Webhook services are
// looked up cluster-wide: a webhook backend commonly lives outside the assessed
// namespaces, and treating it as absent would be a false Critical.
func serviceSet(snap *snapshot.ClusterSnapshot) map[string]struct{} {
	services := make(map[string]struct{}, len(snap.Services))
	for _, service := range snap.Services {
		services[service.Namespace+"/"+service.Name] = struct{}{}
	}
	return services
}

func evaluateWebhook(services map[string]struct{}, kind, configName, webhookName string, failurePolicy *admissionv1.FailurePolicyType, service *admissionv1.ServiceReference, broadScope bool) []upgradev1alpha1.Finding {
	var findings []upgradev1alpha1.Finding

	if failurePolicy != nil && *failurePolicy == admissionv1.Fail {
		findings = append(findings, webhookFinding(kind, configName, webhookName, upgradev1alpha1.RiskLevelHigh, "failurePolicy=Fail", "Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical."))
	}

	if service != nil {
		if _, ok := services[service.Namespace+"/"+service.Name]; !ok {
			findings = append(findings, webhookFinding(kind, configName, webhookName, upgradev1alpha1.RiskLevelCritical, "referenced service is absent", "Restore the webhook Service or remove the webhook configuration before upgrade."))
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
