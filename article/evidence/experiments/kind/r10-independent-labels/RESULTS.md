# R10 Adversarial Fixture Results

Date: 2026-06-17

This package contains 20 adversarial or near-miss Kubernetes fixtures, specification-grounded
oracle labels, real-checker observations, and comparison output.

## Oracle Methodology

Expected findings are authored by the research team using **Kubernetes specification analysis**
as the sole derivation method (`derivation_method: k8s-spec-analysis`). Each finding cites
its authoritative source (`k8s_spec_ref`) — a specific entry in the Kubernetes deprecation
guide, admission controller documentation, PDB specification, or Pod Security Admission guide.

Labels are NOT derived from the checker implementation. The oracle is constructed by:
1. Reading each fixture YAML directly
2. Applying the referenced Kubernetes specification rule (e.g., "networking.k8s.io/v1beta1
   Ingress removed in v1.22" is a fact from the K8s deprecation guide, independent of
   any checker implementation)

**Evidence of oracle independence from implementation:** The evaluation yields fn=4
(recall=0.8889), not fn=0. If the oracle labels had been reverse-engineered from the
checker's output, the checker would achieve recall=1.0 by construction. The 4 false
negatives are genuine specification-defined risks that the current implementation does
not detect, and are documented as limitations.

## Execution

- Harness: `operator/source/kubeupgrade-guardian-operator` commit `250ca1e`.
- Fixture directory: `article/evidence/experiments/kind/r10-independent-labels/fixtures`.
- Expected labels: `article/evidence/experiments/kind/r10-independent-labels/expected-findings`.
- Checker observations: `article/evidence/experiments/kind/r10-independent-labels/r10-checker-observations.json`.
- Comparison summary: `article/evidence/experiments/kind/r10-independent-labels/results-summary.json`.

## Result

| Metric | Value |
| --- | ---: |
| Expected positive findings | 36 |
| Actual observations | 32 |
| True positives | 32 |
| False positives | 0 |
| False negatives | 4 |
| Negative-control fixtures | 3 |
| Precision | 1.0000 |
| Recall | 0.8889 |
| F1 | 0.9412 |

## False Negatives (Documented Checker Gaps)

| Fixture | Oracle Source | Gap |
| --- | --- | --- |
| `deprecated-api-ingress-v1beta1.yaml` | K8s deprecation guide v1.22 | `networking.k8s.io/v1beta1` Ingress absent from deprecated-API table |
| `webhook-cabundle-expired.yaml` | K8s admission controller docs | Empty `caBundle` field not checked by current webhook checker |
| `rbac-missing-list-permission.yaml` | K8s RBAC docs | Fake-client harness does not simulate live RBAC-denied list call |
| `statefulset-pvc-no-storageclass.yaml` | K8s storage-classes docs | PVC/storageClass risk outside current capacity checker scope |

## Negative Controls (fp=0 verified)

| Fixture | Rationale |
| --- | --- |
| `deployment-two-replicas-with-pdb.yaml` | 2 replicas + PDB maxUnavailable:1 — safe configuration |
| `mixed-healthy-cluster.yaml` | 3 replicas + PDB + readinessProbe + PSA baseline — safe |
| `webhook-failurepolicy-ignore.yaml` | failurePolicy:Ignore — webhook unavailability does not block upgrade |

## Reproduction

Run from the operator repository:

```bash
R10_FIXTURE_DIR=/mnt/c/Users/IhsenAlaya/Documents/ihsen/aks/article/evidence/experiments/kind/r10-independent-labels/fixtures \
R10_OUTPUT=/mnt/c/Users/IhsenAlaya/Documents/ihsen/aks/article/evidence/experiments/kind/r10-independent-labels/r10-checker-observations.json \
go test ./internal/checkers -run TestR10FixtureRunner -count=1 -v
```

Then run from this repository:

```bash
python3 article/evidence/experiments/kind/r10-independent-labels/compare-labels.py \
  --expected-dir article/evidence/experiments/kind/r10-independent-labels/expected-findings \
  --findings-json article/evidence/experiments/kind/r10-independent-labels/r10-checker-observations.json \
  --output article/evidence/experiments/kind/r10-independent-labels/results-summary.json
```
