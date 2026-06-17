# R02 AKS Managed Validation Summary

- Run ID: `20260617T101556Z`
- Context: `aks-kug-validation-we-admin`
- Expected findings: `21`
- Scenario findings: `21`
- Managed/provider observations: `8`
- Total KubeUpgrade Guardian findings: `29`
- Negative controls: `5`
- Assessment wait duration: `3.972` seconds
- Controller CPU delta: `0.0` CPU-seconds
- Controller peak RSS: `47.371` MiB

## Overall Metrics

| TP | FP | FN | Precision | Recall | F1 |
| --- | --- | --- | --- | --- | --- |
| 21 | 0 | 0 | 1.0 | 1.0 | 1.0 |

## Metrics By Family

| Family | TP | FP | FN | Precision | Recall | F1 |
| --- | --- | --- | --- | --- | --- | --- |
| AdmissionWebhook | 3 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| PDB | 5 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| PolicyRisk | 6 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| ReadinessProbes | 5 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| WorkloadAvailability | 2 | 0 | 0 | 1.0 | 1.0 | 1.0 |

## Negative Controls

| Control | Resource | KUG |
| --- | --- | --- |
| neg-safe-deployment | Deployment aksv-safe/profile-api | 0 |
| neg-safe-webhook | ValidatingWebhookConfiguration r02-safe-webhook | 0 |
| neg-policy-warn-audit-only | Namespace aksv-policy-warn-audit | 0 |
| neg-modern-hpa | HorizontalPodAutoscaler aksv-modern-api/modern-hpa | 0 |
| neg-modern-pdb | PodDisruptionBudget aksv-modern-api/modern-hpa-target | 0 |

## Scoped Object Inventory

| Kind | Count |
| --- | --- |
| daemonset | 1 |
| deploy | 12 |
| hpa | 1 |
| pdb | 12 |
| pod | 27 |
| statefulset | 2 |

## API-Server Request Counter

- Available: `True`
- Total delta: `11.0`
- Note: Cluster-level API-server request counter delta during the assessment window; includes controller requests and harness polling.

## Notes

- This is a managed AKS validation for Kubernetes-modern manifests only.
- Removed/deprecated API fixtures remain in the Kind benchmark because a modern managed API server rejects removed API versions at admission time.
- AKS-managed webhook observations are reported separately from scenario-label precision and recall.
- API-server request deltas are cluster-level counters and are not exact per-controller attribution.
