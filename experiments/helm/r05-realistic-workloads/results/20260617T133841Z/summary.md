# R05 Helm Realistic Workloads Summary

- Run ID: `20260617T133841Z`
- Helm version: `v3.20.2+g8fb76d6`
- Charts rendered: `5`
- Total resources: `281`
- Secrets excluded from archived manifests: `2`
- Scope: rendered public Helm charts only; no live-cluster health validation.

## Charts

| Chart | Version | Namespace | Resources | Secrets excluded | Archived bytes | SHA-256 |
| --- | --- | --- | ---: | ---: | ---: | --- |
| ingress-nginx | 4.15.1 | ingress-nginx | 19 | 0 | 22194 | `55a978790a1e9aa2...` |
| cert-manager | v1.20.2 | cert-manager | 52 | 0 | 1020114 | `e488f7dfa28db92b...` |
| external-dns | 1.21.1 | external-dns | 5 | 0 | 4537 | `297f4b922e43920d...` |
| kube-prometheus-stack | 86.2.3 | monitoring | 129 | 2 | 5151877 | `39f298091fc79605...` |
| kyverno | 3.8.1 | kyverno | 76 | 0 | 5717779 | `b68445d14af7ed25...` |

## Top Resource Kinds

| Kind | Count |
| --- | ---: |
| ClusterRole | 38 |
| CustomResourceDefinition | 38 |
| PrometheusRule | 35 |
| ConfigMap | 34 |
| ClusterRoleBinding | 26 |
| Service | 23 |
| ServiceAccount | 19 |
| ServiceMonitor | 13 |
| Deployment | 12 |
| Role | 12 |
| RoleBinding | 12 |
| Job | 8 |
| ValidatingWebhookConfiguration | 3 |
| MutatingWebhookConfiguration | 2 |
| PodDisruptionBudget | 2 |

## API Versions

| apiVersion | Count |
| --- | ---: |
| rbac.authorization.k8s.io/v1 | 88 |
| v1 | 76 |
| monitoring.coreos.com/v1 | 50 |
| apiextensions.k8s.io/v1 | 38 |
| apps/v1 | 13 |
| batch/v1 | 8 |
| admissionregistration.k8s.io/v1 | 5 |
| policy/v1 | 2 |
| networking.k8s.io/v1 | 1 |
