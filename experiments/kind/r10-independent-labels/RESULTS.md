# R10 Adversarial Fixture Results

Date: 2026-06-17

This package contains 20 adversarial or near-miss Kubernetes fixtures, Codex-authored control labels, real-checker observations, and comparison output.

## Execution

- Harness: `kubeupgrade-guardian-operator` commit `250ca1e`.
- Fixture directory: `experiments/kind/r10-independent-labels/fixtures`.
- Expected labels: `experiments/kind/r10-independent-labels/expected-findings`.
- Checker observations: `experiments/kind/r10-independent-labels/r10-checker-observations.json`.
- Comparison summary: `experiments/kind/r10-independent-labels/results-summary.json`.

The harness executes the operator's real checker implementations through a local Kubernetes fake client. These labels are not independent human ground truth.

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

## False Negatives

| Fixture | Gap |
| --- | --- |
| `deprecated-api-ingress-v1beta1.yaml` | `networking.k8s.io/v1beta1` Ingress is absent from the current deprecated-API table. |
| `webhook-cabundle-expired.yaml` | Empty webhook `caBundle` is not checked. |
| `rbac-missing-list-permission.yaml` | The fake-client harness does not simulate a live RBAC-denied list call. |
| `statefulset-pvc-no-storageclass.yaml` | PVC/storageClass risk is outside the current capacity checker. |

## Reproduction

Run from the operator repository:

```bash
R10_FIXTURE_DIR=/mnt/c/Users/IhsenAlaya/Documents/ihsen/aks/experiments/kind/r10-independent-labels/fixtures \
R10_OUTPUT=/mnt/c/Users/IhsenAlaya/Documents/ihsen/aks/experiments/kind/r10-independent-labels/r10-checker-observations.json \
go test ./internal/checkers -run TestR10FixtureRunner -count=1 -v
```

Then run from this repository:

```bash
python3 experiments/kind/r10-independent-labels/compare-labels.py \
  --expected-dir experiments/kind/r10-independent-labels/expected-findings \
  --findings-json experiments/kind/r10-independent-labels/r10-checker-observations.json \
  --output experiments/kind/r10-independent-labels/results-summary.json
```
