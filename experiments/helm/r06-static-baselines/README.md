# R06 Helm Static Baselines

This package runs static analysis baselines on the rendered Helm corpus from
`experiments/helm/r05-realistic-workloads`.

## Scope

The runner executes these containerized tools on every rendered chart manifest:

- kube-score;
- kube-linter;
- Polaris.

It records command lines, image digests, return codes, durations, stdout/stderr
byte counts, stdout SHA-256 digests, and normalized finding counts by tool and
chart.

## Limits

This is not a TP/FP/FN comparison because the Helm corpus has no independent
ground-truth labels. It is a workload realism and baseline feasibility artifact:
it shows what the existing static tools report on realistic public add-on
manifests and provides a reproducible input for later manual or expert labeling.

## Run

From the repository root:

```bash
python3 experiments/helm/r06-static-baselines/run_helm_static_baselines.py
```

The runner uses the newest `r05-realistic-workloads/results/*` directory unless
`--r05-run-id` is provided.
