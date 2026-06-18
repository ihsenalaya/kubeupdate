# Upgrade Assessment

- Name: kubeupgrade-guardian-system/aks-assessment
- Source version: 1.34
- Target version: 1.35
- Profile: production
- Phase: Completed
- Decision risk level: Critical
- Score: 576
- Generated plan: aks-assessment-plan
- Artifact: aks-assessment-artifact
## Effective Finding Summary

- Total: 66
- Critical: 0
- High: 52
- Medium: 14
- Low: 0
- Info: 0

## Raw Finding Summary

- Total: 77
- Critical: 0
- High: 58
- Medium: 14
- Low: 0
- Info: 5

## Classification Summary

- Total: 77
- Blocking: 66
- Accepted risk: 0
- Provider managed: 6
- Informational: 5

## Findings

| Classification | Severity | Category | Resource | Message | Recommendation |
| --- | --- | --- | --- | --- | --- |
| ProviderManaged | High | AdmissionWebhook | MutatingWebhookConfiguration aks-node-mutating-webhook | MutatingWebhookConfiguration aks-node-mutating-webhook webhook aks-node-mutating-webhook.azmk8s.io has risk: failurePolicy=Fail. | Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical. |
| ProviderManaged | High | AdmissionWebhook | MutatingWebhookConfiguration aks-node-mutating-webhook | MutatingWebhookConfiguration aks-node-mutating-webhook webhook aks-node-mutating-webhook.azmk8s.io has risk: namespaceSelector is absent. | Constrain webhook scope with namespaceSelector/objectSelector where possible. |
| ProviderManaged | High | AdmissionWebhook | MutatingWebhookConfiguration aks-webhook-admission-controller | MutatingWebhookConfiguration aks-webhook-admission-controller webhook aks-webhook-admission-controller.azmk8s.io has risk: failurePolicy=Fail. | Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical. |
| ProviderManaged | High | AdmissionWebhook | MutatingWebhookConfiguration aks-webhook-admission-controller | MutatingWebhookConfiguration aks-webhook-admission-controller webhook aks-webhook-admission-controller.azmk8s.io has risk: namespaceSelector is absent. | Constrain webhook scope with namespaceSelector/objectSelector where possible. |
| Blocking | High | AdmissionWebhook | MutatingWebhookConfiguration azure-wi-webhook-mutating-webhook-configuration | MutatingWebhookConfiguration azure-wi-webhook-mutating-webhook-configuration webhook mutation.azure-workload-identity.io has risk: failurePolicy=Fail. | Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical. |
| Blocking | High | AdmissionWebhook | MutatingWebhookConfiguration azure-wi-webhook-mutating-webhook-configuration | MutatingWebhookConfiguration azure-wi-webhook-mutating-webhook-configuration webhook mutation.azure-workload-identity.io has risk: namespaceSelector is absent. | Constrain webhook scope with namespaceSelector/objectSelector where possible. |
| Blocking | High | AdmissionWebhook | MutatingWebhookConfiguration cert-manager-webhook | MutatingWebhookConfiguration cert-manager-webhook webhook webhook.cert-manager.io has risk: failurePolicy=Fail. | Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical. |
| Blocking | High | AdmissionWebhook | MutatingWebhookConfiguration istio-sidecar-injector | MutatingWebhookConfiguration istio-sidecar-injector webhook namespace.sidecar-injector.istio.io has risk: failurePolicy=Fail. | Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical. |
| Blocking | High | AdmissionWebhook | MutatingWebhookConfiguration istio-sidecar-injector | MutatingWebhookConfiguration istio-sidecar-injector webhook object.sidecar-injector.istio.io has risk: failurePolicy=Fail. | Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical. |
| Blocking | High | AdmissionWebhook | MutatingWebhookConfiguration istio-sidecar-injector | MutatingWebhookConfiguration istio-sidecar-injector webhook rev.namespace.sidecar-injector.istio.io has risk: failurePolicy=Fail. | Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical. |
| Blocking | High | AdmissionWebhook | MutatingWebhookConfiguration istio-sidecar-injector | MutatingWebhookConfiguration istio-sidecar-injector webhook rev.object.sidecar-injector.istio.io has risk: failurePolicy=Fail. | Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical. |
| Blocking | High | AdmissionWebhook | MutatingWebhookConfiguration kyverno-policy-mutating-webhook-cfg | MutatingWebhookConfiguration kyverno-policy-mutating-webhook-cfg webhook mutate-policy.kyverno.svc has risk: failurePolicy=Fail. | Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical. |
| Blocking | High | AdmissionWebhook | MutatingWebhookConfiguration kyverno-policy-mutating-webhook-cfg | MutatingWebhookConfiguration kyverno-policy-mutating-webhook-cfg webhook mutate-policy.kyverno.svc has risk: namespaceSelector is absent. | Constrain webhook scope with namespaceSelector/objectSelector where possible. |
| Blocking | High | AdmissionWebhook | MutatingWebhookConfiguration kyverno-verify-mutating-webhook-cfg | MutatingWebhookConfiguration kyverno-verify-mutating-webhook-cfg webhook monitor-webhooks.kyverno.svc has risk: namespaceSelector is absent. | Constrain webhook scope with namespaceSelector/objectSelector where possible. |
| ProviderManaged | High | AdmissionWebhook | ValidatingWebhookConfiguration aks-node-validating-webhook | ValidatingWebhookConfiguration aks-node-validating-webhook webhook aks-node-validating-webhook.azmk8s.io has risk: failurePolicy=Fail. | Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical. |
| ProviderManaged | High | AdmissionWebhook | ValidatingWebhookConfiguration aks-node-validating-webhook | ValidatingWebhookConfiguration aks-node-validating-webhook webhook aks-node-validating-webhook.azmk8s.io has risk: namespaceSelector is absent. | Constrain webhook scope with namespaceSelector/objectSelector where possible. |
| Blocking | High | AdmissionWebhook | ValidatingWebhookConfiguration cert-manager-webhook | ValidatingWebhookConfiguration cert-manager-webhook webhook webhook.cert-manager.io has risk: failurePolicy=Fail. | Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical. |
| Blocking | High | AdmissionWebhook | ValidatingWebhookConfiguration externalsecret-validate | ValidatingWebhookConfiguration externalsecret-validate webhook validate.externalsecret.external-secrets.io has risk: failurePolicy=Fail. | Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical. |
| Blocking | High | AdmissionWebhook | ValidatingWebhookConfiguration istiod-default-validator | ValidatingWebhookConfiguration istiod-default-validator webhook validation.istio.io has risk: failurePolicy=Fail. | Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical. |
| Blocking | High | AdmissionWebhook | ValidatingWebhookConfiguration istio-validator-istio-system | ValidatingWebhookConfiguration istio-validator-istio-system webhook rev.validation.istio.io has risk: failurePolicy=Fail. | Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical. |
| Blocking | High | AdmissionWebhook | ValidatingWebhookConfiguration kyverno-cel-exception-validating-webhook-cfg | ValidatingWebhookConfiguration kyverno-cel-exception-validating-webhook-cfg webhook kyverno-svc.kyverno.svc has risk: failurePolicy=Fail. | Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical. |
| Blocking | High | AdmissionWebhook | ValidatingWebhookConfiguration kyverno-cel-exception-validating-webhook-cfg | ValidatingWebhookConfiguration kyverno-cel-exception-validating-webhook-cfg webhook kyverno-svc.kyverno.svc has risk: namespaceSelector is absent. | Constrain webhook scope with namespaceSelector/objectSelector where possible. |
| Blocking | High | AdmissionWebhook | ValidatingWebhookConfiguration kyverno-cleanup-validating-webhook-cfg | ValidatingWebhookConfiguration kyverno-cleanup-validating-webhook-cfg webhook kyverno-cleanup-controller.kyverno.svc has risk: failurePolicy=Fail. | Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical. |
| Blocking | High | AdmissionWebhook | ValidatingWebhookConfiguration kyverno-cleanup-validating-webhook-cfg | ValidatingWebhookConfiguration kyverno-cleanup-validating-webhook-cfg webhook kyverno-cleanup-controller.kyverno.svc has risk: namespaceSelector is absent. | Constrain webhook scope with namespaceSelector/objectSelector where possible. |
| Blocking | High | AdmissionWebhook | ValidatingWebhookConfiguration kyverno-exception-validating-webhook-cfg | ValidatingWebhookConfiguration kyverno-exception-validating-webhook-cfg webhook kyverno-svc.kyverno.svc has risk: failurePolicy=Fail. | Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical. |
| Blocking | High | AdmissionWebhook | ValidatingWebhookConfiguration kyverno-exception-validating-webhook-cfg | ValidatingWebhookConfiguration kyverno-exception-validating-webhook-cfg webhook kyverno-svc.kyverno.svc has risk: namespaceSelector is absent. | Constrain webhook scope with namespaceSelector/objectSelector where possible. |
| Blocking | High | AdmissionWebhook | ValidatingWebhookConfiguration kyverno-global-context-validating-webhook-cfg | ValidatingWebhookConfiguration kyverno-global-context-validating-webhook-cfg webhook kyverno-svc.kyverno.svc has risk: failurePolicy=Fail. | Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical. |
| Blocking | High | AdmissionWebhook | ValidatingWebhookConfiguration kyverno-global-context-validating-webhook-cfg | ValidatingWebhookConfiguration kyverno-global-context-validating-webhook-cfg webhook kyverno-svc.kyverno.svc has risk: namespaceSelector is absent. | Constrain webhook scope with namespaceSelector/objectSelector where possible. |
| Blocking | High | AdmissionWebhook | ValidatingWebhookConfiguration kyverno-policy-validating-webhook-cfg | ValidatingWebhookConfiguration kyverno-policy-validating-webhook-cfg webhook validate-policy.kyverno.svc has risk: failurePolicy=Fail. | Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical. |
| Blocking | High | AdmissionWebhook | ValidatingWebhookConfiguration kyverno-policy-validating-webhook-cfg | ValidatingWebhookConfiguration kyverno-policy-validating-webhook-cfg webhook validate-policy.kyverno.svc has risk: namespaceSelector is absent. | Constrain webhook scope with namespaceSelector/objectSelector where possible. |
| Blocking | High | AdmissionWebhook | ValidatingWebhookConfiguration kyverno-resource-validating-webhook-cfg | ValidatingWebhookConfiguration kyverno-resource-validating-webhook-cfg webhook validate.kyverno.svc-fail has risk: failurePolicy=Fail. | Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical. |
| Blocking | High | AdmissionWebhook | ValidatingWebhookConfiguration kyverno-ttl-validating-webhook-cfg | ValidatingWebhookConfiguration kyverno-ttl-validating-webhook-cfg webhook kyverno-cleanup-controller.kyverno.svc has risk: namespaceSelector is absent. | Constrain webhook scope with namespaceSelector/objectSelector where possible. |
| Blocking | High | AdmissionWebhook | ValidatingWebhookConfiguration secretstore-validate | ValidatingWebhookConfiguration secretstore-validate webhook validate.clustersecretstore.external-secrets.io has risk: failurePolicy=Fail. | Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical. |
| Blocking | High | AdmissionWebhook | ValidatingWebhookConfiguration secretstore-validate | ValidatingWebhookConfiguration secretstore-validate webhook validate.secretstore.external-secrets.io has risk: failurePolicy=Fail. | Set failurePolicy to Ignore during controlled upgrade windows if the webhook is not upgrade-critical. |
| Blocking | High | Capacity | cluster | Cluster may not have enough requested capacity headroom to tolerate one worker node loss. | Add capacity or reduce requests before upgrade. Estimated remaining capacity after one-node loss: 1900m CPU. |
| Blocking | Medium | ReadinessProbes | Deployment cert-manager/cert-manager-cainjector | Container cert-manager-cainjector in Deployment cert-manager/cert-manager-cainjector has no readinessProbe. | Add a readinessProbe that reflects whether the container can safely receive traffic. |
| Blocking | Medium | ReadinessProbes | Deployment cert-manager/cert-manager | Container cert-manager-controller in Deployment cert-manager/cert-manager has no readinessProbe. | Add a readinessProbe that reflects whether the container can safely receive traffic. |
| Blocking | Medium | ReadinessProbes | Deployment external-secrets/external-secrets | Container external-secrets in Deployment external-secrets/external-secrets has no readinessProbe. | Add a readinessProbe that reflects whether the container can safely receive traffic. |
| Blocking | Medium | ReadinessProbes | Deployment istio-ingress/istio-ingress | Container istio-proxy in Deployment istio-ingress/istio-ingress has no readinessProbe. | Add a readinessProbe that reflects whether the container can safely receive traffic. |
| Blocking | Medium | ReadinessProbes | Deployment kyverno/kyverno-background-controller | Container controller in Deployment kyverno/kyverno-background-controller has no readinessProbe. | Add a readinessProbe that reflects whether the container can safely receive traffic. |
| Blocking | Medium | ReadinessProbes | Deployment kyverno/kyverno-reports-controller | Container controller in Deployment kyverno/kyverno-reports-controller has no readinessProbe. | Add a readinessProbe that reflects whether the container can safely receive traffic. |
| Blocking | Medium | ReadinessProbes | Deployment monitoring/kube-prometheus-stack-grafana | Container grafana-sc-dashboard in Deployment monitoring/kube-prometheus-stack-grafana has no readinessProbe. | Add a readinessProbe that reflects whether the container can safely receive traffic. |
| Blocking | Medium | ReadinessProbes | Deployment monitoring/kube-prometheus-stack-grafana | Container grafana-sc-datasources in Deployment monitoring/kube-prometheus-stack-grafana has no readinessProbe. | Add a readinessProbe that reflects whether the container can safely receive traffic. |
| Blocking | Medium | ReadinessProbes | StatefulSet monitoring/alertmanager-kube-prometheus-stack-alertmanager | Container config-reloader in StatefulSet monitoring/alertmanager-kube-prometheus-stack-alertmanager has no readinessProbe. | Add a readinessProbe that reflects whether the container can safely receive traffic. |
| Blocking | Medium | ReadinessProbes | StatefulSet monitoring/prometheus-kube-prometheus-stack-prometheus | Container config-reloader in StatefulSet monitoring/prometheus-kube-prometheus-stack-prometheus has no readinessProbe. | Add a readinessProbe that reflects whether the container can safely receive traffic. |
| Blocking | Medium | ReadinessProbes | DaemonSet neuvector/neuvector-enforcer-pod | Container neuvector-enforcer-pod in DaemonSet neuvector/neuvector-enforcer-pod has no readinessProbe. | Add a readinessProbe that reflects whether the container can safely receive traffic. |
| Blocking | Medium | ReadinessProbes | Deployment neuvector/neuvector-manager-pod | Container neuvector-manager-pod in Deployment neuvector/neuvector-manager-pod has no readinessProbe. | Add a readinessProbe that reflects whether the container can safely receive traffic. |
| Blocking | Medium | ReadinessProbes | Deployment neuvector/neuvector-scanner-pod | Container neuvector-scanner-pod in Deployment neuvector/neuvector-scanner-pod has no readinessProbe. | Add a readinessProbe that reflects whether the container can safely receive traffic. |
| Informational | Info | Observability | CustomResourceDefinition podmonitors.monitoring.coreos.com | Observability CRD detected: podmonitors.monitoring.coreos.com | Use existing observability to validate workload health before and after upgrade. |
| Informational | Info | Observability | CustomResourceDefinition prometheuses.monitoring.coreos.com | Observability CRD detected: prometheuses.monitoring.coreos.com | Use existing observability to validate workload health before and after upgrade. |
| Informational | Info | Observability | CustomResourceDefinition servicemonitors.monitoring.coreos.com | Observability CRD detected: servicemonitors.monitoring.coreos.com | Use existing observability to validate workload health before and after upgrade. |
| Informational | Info | PolicyRisk | CustomResourceDefinition clusterpolicies.kyverno.io | Kyverno policy engine CRD detected. | Review policy reports and admission behavior before upgrade. |
| Informational | Info | PolicyRisk | CustomResourceDefinition policies.kyverno.io | Kyverno policy engine CRD detected. | Review policy reports and admission behavior before upgrade. |
| Blocking | High | PolicyRisk | Deployment upgrade-lab/upgrade-lab-upgrade-lab-catalog | Deployment upgrade-lab/upgrade-lab-upgrade-lab-catalog may violate restricted policy: runAsNonRoot absent. | Adjust the pod security context or namespace policy before upgrade. |
| Blocking | High | PolicyRisk | Deployment upgrade-lab/upgrade-lab-upgrade-lab-edge | Deployment upgrade-lab/upgrade-lab-upgrade-lab-edge may violate restricted policy: runAsNonRoot absent. | Adjust the pod security context or namespace policy before upgrade. |
| Blocking | High | PolicyRisk | Deployment upgrade-lab/upgrade-lab-upgrade-lab-orders | Deployment upgrade-lab/upgrade-lab-upgrade-lab-orders may violate restricted policy: runAsNonRoot absent. | Adjust the pod security context or namespace policy before upgrade. |
| Blocking | High | PolicyRisk | Deployment upgrade-lab/upgrade-lab-upgrade-lab-signals | Deployment upgrade-lab/upgrade-lab-upgrade-lab-signals may violate restricted policy: runAsNonRoot absent. | Adjust the pod security context or namespace policy before upgrade. |
| Blocking | Medium | PolicyRisk | Namespace upgrade-lab | Namespace upgrade-lab enforces Pod Security restricted. | Validate all workloads in this namespace against the restricted Pod Security profile before upgrade. |
| Blocking | High | WorkloadAvailability | Deployment cert-manager/cert-manager | Deployment cert-manager/cert-manager has fewer than 2 replicas. | Increase replicas to at least 2 or document why this workload can tolerate disruption. |
| Blocking | High | WorkloadAvailability | Deployment cert-manager/cert-manager-cainjector | Deployment cert-manager/cert-manager-cainjector has fewer than 2 replicas. | Increase replicas to at least 2 or document why this workload can tolerate disruption. |
| Blocking | High | WorkloadAvailability | Deployment cert-manager/cert-manager-webhook | Deployment cert-manager/cert-manager-webhook has fewer than 2 replicas. | Increase replicas to at least 2 or document why this workload can tolerate disruption. |
| Blocking | High | WorkloadAvailability | Deployment external-secrets/external-secrets | Deployment external-secrets/external-secrets has fewer than 2 replicas. | Increase replicas to at least 2 or document why this workload can tolerate disruption. |
| Blocking | High | WorkloadAvailability | Deployment external-secrets/external-secrets-cert-controller | Deployment external-secrets/external-secrets-cert-controller has fewer than 2 replicas. | Increase replicas to at least 2 or document why this workload can tolerate disruption. |
| Blocking | High | WorkloadAvailability | Deployment external-secrets/external-secrets-webhook | Deployment external-secrets/external-secrets-webhook has fewer than 2 replicas. | Increase replicas to at least 2 or document why this workload can tolerate disruption. |
| Blocking | High | WorkloadAvailability | Deployment istio-ingress/istio-ingress | Deployment istio-ingress/istio-ingress has fewer than 2 replicas. | Increase replicas to at least 2 or document why this workload can tolerate disruption. |
| Blocking | High | WorkloadAvailability | Deployment kyverno/kyverno-admission-controller | Deployment kyverno/kyverno-admission-controller has fewer than 2 replicas. | Increase replicas to at least 2 or document why this workload can tolerate disruption. |
| Blocking | High | WorkloadAvailability | Deployment kyverno/kyverno-background-controller | Deployment kyverno/kyverno-background-controller has fewer than 2 replicas. | Increase replicas to at least 2 or document why this workload can tolerate disruption. |
| Blocking | High | WorkloadAvailability | Deployment kyverno/kyverno-cleanup-controller | Deployment kyverno/kyverno-cleanup-controller has fewer than 2 replicas. | Increase replicas to at least 2 or document why this workload can tolerate disruption. |
| Blocking | High | WorkloadAvailability | Deployment kyverno/kyverno-reports-controller | Deployment kyverno/kyverno-reports-controller has fewer than 2 replicas. | Increase replicas to at least 2 or document why this workload can tolerate disruption. |
| Blocking | High | WorkloadAvailability | Deployment monitoring/kube-prometheus-stack-grafana | Deployment monitoring/kube-prometheus-stack-grafana has fewer than 2 replicas. | Increase replicas to at least 2 or document why this workload can tolerate disruption. |
| Blocking | High | WorkloadAvailability | Deployment monitoring/kube-prometheus-stack-kube-state-metrics | Deployment monitoring/kube-prometheus-stack-kube-state-metrics has fewer than 2 replicas. | Increase replicas to at least 2 or document why this workload can tolerate disruption. |
| Blocking | High | WorkloadAvailability | Deployment monitoring/kube-prometheus-stack-operator | Deployment monitoring/kube-prometheus-stack-operator has fewer than 2 replicas. | Increase replicas to at least 2 or document why this workload can tolerate disruption. |
| Blocking | High | WorkloadAvailability | StatefulSet monitoring/alertmanager-kube-prometheus-stack-alertmanager | StatefulSet monitoring/alertmanager-kube-prometheus-stack-alertmanager has fewer than 2 replicas. | Increase replicas to at least 2 or document why this workload can tolerate disruption. |
| Blocking | High | WorkloadAvailability | StatefulSet monitoring/prometheus-kube-prometheus-stack-prometheus | StatefulSet monitoring/prometheus-kube-prometheus-stack-prometheus has fewer than 2 replicas. | Increase replicas to at least 2 or document why this workload can tolerate disruption. |
| Blocking | High | WorkloadAvailability | Deployment neuvector/neuvector-controller-pod | Deployment neuvector/neuvector-controller-pod has fewer than 2 replicas. | Increase replicas to at least 2 or document why this workload can tolerate disruption. |
| Blocking | High | WorkloadAvailability | Deployment neuvector/neuvector-manager-pod | Deployment neuvector/neuvector-manager-pod has fewer than 2 replicas. | Increase replicas to at least 2 or document why this workload can tolerate disruption. |
| Blocking | High | WorkloadAvailability | Deployment neuvector/neuvector-scanner-pod | Deployment neuvector/neuvector-scanner-pod has fewer than 2 replicas. | Increase replicas to at least 2 or document why this workload can tolerate disruption. |


