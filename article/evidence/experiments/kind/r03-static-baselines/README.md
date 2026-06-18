# R03 Static Baseline Smoke

This package runs additional static Kubernetes analysis baselines on the R01
scenario manifest. It is a smoke run for baseline feasibility, not a fair
precision/recall comparison and not an independent benchmark.

## Scope

The runner audits:

- `experiments/kind/r01-benchmark/manifests/00-scenarios.yaml`

It deliberately excludes `10-assessment.yaml` because that file contains the
KubeUpgrade Guardian custom resource used to trigger the assessment, not a
workload scenario for generic static linters.

The current baselines run through Docker images:

- kube-score
- kube-linter
- Polaris

The runner records image digests, versions, commands, exit codes, durations,
stdout, stderr, and small normalized count summaries. Non-zero exit codes are
kept as data because these tools commonly return non-zero when findings are
present.

## Limits

This run does not replace the Q1 baseline study requested in the review plan.
It does not compute TP/FP/FN against independently labeled data, does not run
Popeye or Datree, and does not normalize semantic overlap with KubeUpgrade
Guardian findings. It only establishes that three additional baselines can be
executed on the existing R01 manifest and archived reproducibly.

## Run

From the repository root:

```bash
python3 experiments/kind/r03-static-baselines/run_static_baselines.py
```

The runner writes a timestamped directory under `results/` with raw outputs,
`metadata.json`, `metrics.json`, and `summary.md`.
