# R04 Kind Ablation Smoke

This package runs a controlled ablation smoke study for KubeUpgrade Guardian on
the existing R01 benchmark fixtures.

## Scope

The runner creates a Kind cluster, applies the R01 scenario manifest, and runs
multiple `UpgradeAssessment` variants:

- full checker set;
- full set without deprecated API checks;
- full set without workload availability checks;
- full set without PDB checks;
- full set without readiness-probe checks;
- full set without admission-webhook checks;
- full set without policy-risk checks.

Each variant is scored against `article/evidence/experiments/kind/r01-benchmark/ground-truth.json`
using the same matching semantics as the R01 benchmark runner.

## Limits

This is not a full Q1 ablation study. It reuses author-controlled R01 fixtures,
does not include independent labels, and does not evaluate expert actionability.
It is useful as a reproducible sanity check that the measured coverage depends
on the individual checker families rather than on one monolithic detector.

## Run

From the repository root:

```bash
python3 article/evidence/experiments/kind/r04-ablation-smoke/run_ablation_smoke.py \
  --operator-repo operator/source/kubeupgrade-guardian-operator
```

The runner writes a timestamped directory under `results/` with raw
`UpgradeAssessment`/`UpgradePlan` JSON for every variant, normalized findings,
metrics, and a Markdown summary.
