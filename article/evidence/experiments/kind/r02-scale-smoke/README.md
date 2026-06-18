# R02 Kind Scale Smoke

This package measures a focused scalability smoke case for KubeUpgrade Guardian.
It is not a full Q1 scalability study and must not be cited as repeated
production evidence.

## Scope

The runner creates a Kind cluster and sweeps synthetic assessed object counts.
Each size generates paired Deployments and PodDisruptionBudgets in one namespace
to stress the workload/PDB matching path without creating Pods. The assessment
enables only the PDB checker so the run isolates this part of the controller.

The produced data answers a narrow question:

> How does a single controller process behave on increasing numbers of synthetic
> Deployment/PDB objects in a local Kind cluster?

It does not replace:

- independent ground truth;
- expert review;
- multi-provider managed-cluster validation;
- repeated scalability measurements with confidence intervals;
- production upgrade incident calibration.

## Run

From the repository root:

```bash
python3 experiments/kind/r02-scale-smoke/run_scale_smoke.py \
  --operator-repo ../kubeupgrade-guardian-operator \
  --sizes 100,500,1000 \
  --repetitions 1
```

The runner writes a timestamped directory under `results/` with raw assessment
JSON, process samples, API request deltas when available, `metrics.json`, and a
Markdown summary.
