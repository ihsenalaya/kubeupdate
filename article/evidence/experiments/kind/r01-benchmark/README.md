# R01 Kind Benchmark

This package creates a controlled Kind cluster, deploys reproducible
upgrade-readiness scenarios, runs KubeUpgrade Guardian, runs API-deprecation
baselines, and computes measured precision/recall/F1 against ground truth.

## Scope

The benchmark covers:

- Deprecated API objects served by Kubernetes 1.24 and removed by the target
  version 1.32.
- Workload availability risks: low replicas and standalone pods.
- Readiness probe gaps on Deployments.
- PDB risks: blocking budgets, stale selectors, and missing budgets.
- Admission webhook risks: fail-closed, missing service, and broad scope.
- Policy risks: restricted namespaces with incompatible pod templates.
- Explicit negative controls for safe workloads, scoped fail-open webhooks,
  warn/audit-only Pod Security labels, and modern API source manifests.

The benchmark intentionally disables capacity and observability checkers because
they are not part of the first R01 empirical slice.

The validated run `20260617T091703Z` contains 31 positive labels and 5 negative
controls. Pluto and kubent are evaluated as API-deprecation baselines; their
results are reported separately for the four API labels and the 27 non-API
readiness labels.

## Prerequisites

- Docker
- Kind
- kubectl
- Go
- Optional baselines on `PATH`: `pluto` and `kubent`

The validated run used:

- Kind node image: `kindest/node:v1.24.15`
- Target Kubernetes version: `1.32`
- Pluto: `5.24.0`
- kubent: `0.7.3`

## Run

From the repository root:

```bash
PATH=/tmp/kug-r01-tools/bin:$PATH \
python3 article/evidence/experiments/kind/r01-benchmark/run_kind_benchmark.py \
  --operator-repo operator/source/kubeupgrade-guardian-operator \
  --restore-context aks-ihsen-mvp-we-admin
```

The runner writes one timestamped directory under `results/` with:

- raw `UpgradeAssessment` and `UpgradePlan` JSON;
- raw Pluto and kubent outputs when the binaries are available;
- a normalized metrics file;
- negative-control observations;
- a Markdown summary suitable for manuscript extraction.

The benchmark must only be cited after the runner succeeds, outputs are
archived, and the manuscript results section is updated from those outputs.
