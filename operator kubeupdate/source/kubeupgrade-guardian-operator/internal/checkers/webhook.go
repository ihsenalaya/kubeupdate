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

	admissionv1 "k8s.io/api/admissionregistration/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/snapshot"
)

// slowWebhookTimeoutSeconds is the timeout above which a failing webhook stalls a
// drain long enough to matter during an upgrade window.
const slowWebhookTimeoutSeconds = 15

// AdmissionWebhook detects upgrade risks introduced by admission webhooks.
type AdmissionWebhook struct{}

func (AdmissionWebhook) Name() string { return "admission-webhooks" }

// webhookBackends is the cluster-wide view of webhook backends: which services
// exist, which of them have a ready endpoint, and whether endpoint readiness can
// be judged at all.
type webhookBackends struct {
	services       map[string]struct{}
	readyServices  map[string]struct{}
	readinessKnown bool
}

func (AdmissionWebhook) Check(snap *snapshot.ClusterSnapshot, _ *upgradev1alpha1.UpgradeAssessment) []upgradev1alpha1.Finding {
	backends := collectWebhookBackends(snap)

	var findings []upgradev1alpha1.Finding
	for _, item := range snap.ValidatingWebhooks {
		for _, webhook := range item.Webhooks {
			findings = append(findings, evaluateWebhook(backends, webhookSubject{
				Kind:              "ValidatingWebhookConfiguration",
				ConfigName:        item.Name,
				WebhookName:       webhook.Name,
				FailurePolicy:     webhook.FailurePolicy,
				Service:           webhook.ClientConfig.Service,
				NamespaceSelector: webhook.NamespaceSelector,
				Rules:             webhook.Rules,
				TimeoutSeconds:    webhook.TimeoutSeconds,
			})...)
		}
	}
	for _, item := range snap.MutatingWebhooks {
		for _, webhook := range item.Webhooks {
			findings = append(findings, evaluateWebhook(backends, webhookSubject{
				Kind:              "MutatingWebhookConfiguration",
				ConfigName:        item.Name,
				WebhookName:       webhook.Name,
				FailurePolicy:     webhook.FailurePolicy,
				Service:           webhook.ClientConfig.Service,
				NamespaceSelector: webhook.NamespaceSelector,
				Rules:             webhook.Rules,
				TimeoutSeconds:    webhook.TimeoutSeconds,
			})...)
		}
	}

	return findings
}

// webhookSubject is the part of a validating or mutating webhook the checker reasons about.
type webhookSubject struct {
	Kind              string
	ConfigName        string
	WebhookName       string
	FailurePolicy     *admissionv1.FailurePolicyType
	Service           *admissionv1.ServiceReference
	NamespaceSelector *metav1.LabelSelector
	Rules             []admissionv1.RuleWithOperations
	TimeoutSeconds    *int32
}

// collectWebhookBackends indexes services and their ready endpoints cluster-wide:
// a webhook backend commonly lives outside the assessed namespaces, and treating
// it as absent would be a false Critical.
func collectWebhookBackends(snap *snapshot.ClusterSnapshot) webhookBackends {
	backends := webhookBackends{
		services:       make(map[string]struct{}, len(snap.Services)),
		readyServices:  map[string]struct{}{},
		readinessKnown: !snap.Denied(snapshot.ResourceEndpointSlices),
	}

	for _, service := range snap.Services {
		backends.services[serviceKey(service.Namespace, service.Name)] = struct{}{}
	}

	for _, slice := range snap.EndpointSlices {
		serviceName := slice.Labels[discoveryv1.LabelServiceName]
		if serviceName == "" {
			continue
		}
		if !hasReadyEndpoint(slice) {
			continue
		}
		backends.readyServices[serviceKey(slice.Namespace, serviceName)] = struct{}{}
	}

	return backends
}

// hasReadyEndpoint follows the EndpointSlice convention that an unset ready
// condition means ready.
func hasReadyEndpoint(slice discoveryv1.EndpointSlice) bool {
	for _, endpoint := range slice.Endpoints {
		if endpoint.Conditions.Ready == nil || *endpoint.Conditions.Ready {
			return true
		}
	}
	return false
}

func serviceKey(namespace, name string) string { return namespace + "/" + name }

func evaluateWebhook(backends webhookBackends, subject webhookSubject) []upgradev1alpha1.Finding {
	var findings []upgradev1alpha1.Finding

	if subject.FailurePolicy != nil && *subject.FailurePolicy == admissionv1.Fail {
		findings = append(findings, failurePolicyFinding(subject))
	}

	if subject.Service != nil {
		key := serviceKey(subject.Service.Namespace, subject.Service.Name)
		if _, exists := backends.services[key]; !exists {
			findings = append(findings, webhookFinding(subject, upgradev1alpha1.RiskLevelCritical,
				"referenced service is absent",
				"Restore the webhook Service or remove the webhook configuration before upgrade.",
				map[string]string{"service": key}))
		} else if _, ready := backends.readyServices[key]; !ready && backends.readinessKnown {
			findings = append(findings, webhookFinding(subject, upgradev1alpha1.RiskLevelCritical,
				"webhook backend has no ready endpoints",
				"Restore a healthy webhook backend before upgrade: a webhook without ready endpoints rejects or stalls admission as soon as a node is drained.",
				map[string]string{"service": key}))
		}
	}

	if isBroadNamespaceSelector(subject.NamespaceSelector) {
		findings = append(findings, webhookFinding(subject, upgradev1alpha1.RiskLevelInfo,
			"namespaceSelector is absent",
			"Constrain webhook scope with namespaceSelector/objectSelector where possible.",
			nil))
	}

	return findings
}

// failurePolicyFinding grades failurePolicy=Fail by blast radius: a webhook that
// does not intercept pods cannot block a drain, so it is not an upgrade blocker.
func failurePolicyFinding(subject webhookSubject) upgradev1alpha1.Finding {
	severity := upgradev1alpha1.RiskLevelMedium
	recommendation := "Review this webhook before the upgrade window; it does not intercept pod admission, so it cannot block a node drain."
	if interceptsDrain(subject.Rules) {
		severity = upgradev1alpha1.RiskLevelHigh
		recommendation = "Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical: it intercepts pod admission and can block a node drain."
	}

	reason := "failurePolicy=Fail"
	timeout := webhookTimeout(subject.TimeoutSeconds)
	if timeout >= slowWebhookTimeoutSeconds {
		reason = fmt.Sprintf("failurePolicy=Fail with timeoutSeconds=%d", timeout)
	}

	return webhookFinding(subject, severity, reason, recommendation, map[string]string{
		"interceptsPodAdmission": strconv.FormatBool(interceptsDrain(subject.Rules)),
	})
}

// webhookTimeout returns the effective timeout; the API server defaults it to 10s.
func webhookTimeout(timeoutSeconds *int32) int32 {
	if timeoutSeconds == nil {
		return 10
	}
	return *timeoutSeconds
}

// interceptsDrain reports whether the webhook rules cover the core resources a
// node drain touches: pods, pod eviction, or a core-group wildcard.
func interceptsDrain(rules []admissionv1.RuleWithOperations) bool {
	for _, rule := range rules {
		if !matchesCoreGroup(rule.APIGroups) {
			continue
		}
		for _, resource := range rule.Resources {
			switch resource {
			case "pods", "pods/eviction", "*", "*/*":
				return true
			}
		}
	}
	return false
}

func matchesCoreGroup(groups []string) bool {
	for _, group := range groups {
		if group == "" || group == "*" {
			return true
		}
	}
	return false
}

func isBroadNamespaceSelector(selector *metav1.LabelSelector) bool {
	return selector == nil || (len(selector.MatchLabels) == 0 && len(selector.MatchExpressions) == 0)
}

func webhookFinding(subject webhookSubject, severity upgradev1alpha1.RiskLevel, reason, recommendation string, observed map[string]string) upgradev1alpha1.Finding {
	evidence := map[string]string{
		"webhookName":    subject.WebhookName,
		"reason":         reason,
		"timeoutSeconds": strconv.Itoa(int(webhookTimeout(subject.TimeoutSeconds))),
	}
	for key, value := range observed {
		evidence[key] = value
	}

	return upgradev1alpha1.Finding{
		ID:       findingID(upgradev1alpha1.FindingTypeAdmissionWebhookRisk, subject.Kind, subject.ConfigName, subject.WebhookName, reason),
		Type:     upgradev1alpha1.FindingTypeAdmissionWebhookRisk,
		Severity: severity,
		Category: "AdmissionWebhook",
		Resource: resource("admissionregistration.k8s.io/v1", subject.Kind, "", subject.ConfigName),
		Message:  fmt.Sprintf("%s %s webhook %s has risk: %s.", subject.Kind, subject.ConfigName, subject.WebhookName, reason),
		Evidence: []upgradev1alpha1.Evidence{{
			ID:          evidenceID(upgradev1alpha1.FindingTypeAdmissionWebhookRisk, subject.Kind, subject.ConfigName, subject.WebhookName, reason),
			Description: "Admission webhook configuration risk.",
			Observed:    evidence,
		}},
		Recommendation: recommendation,
	}
}
