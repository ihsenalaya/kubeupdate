# R04 AKS Scale Results

Date: 2026-06-17

This run used `aks-medium` with Kubernetes `1.34.8`, AKS cluster autoscaler enabled on the default node pool (`min=3`, `max=8`), and the patched local controller from `kubeupgrade-guardian-operator`. The controller bounds `UpgradeAssessment.status.findings` and generated `UpgradePlan` actions to 200 entries while preserving full aggregate counts in `status.summary`.

## Result Summary

| Deployment/PDB pairs | Runs | Completed | Median duration | p95 duration | Median RSS | Median API request delta | Outcome |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| 100 | 10 | 10 | 8.0s | 13.0s | 49.041 MiB | 4.5 | Completed with clean namespace before/after each run. |
| 1,000 | 10 | 10 | 7.5s | 13.0s | 99.227 MiB | 4 | Completed; run-1 created load on a clean cluster and runs 2-10 reused the same load. |
| 5,000 | 10 | 10 | 13.5s | 17.3s | 327.535 MiB | 4 | Completed with bounded status output. |
| 10,000 | 10 | 10 | 19.5s | 37.0s | 726.073 MiB | 3 | Completed with bounded status output. |

Aggregated metrics are in `experiments/scale/r04-scalability-summary.json`. Raw per-run files are under `experiments/scale/r04-scale-*/run-*`.

## Truncation Evidence

The 5,000-pair run produced 10,000 total findings. The archived `run-10/upgradeassessment.json` reports `status.summary.totalFindings=10000`, publishes 200 findings in `status.findings`, and sets `AssessmentOutputTruncated=True`.

The 10,000-pair run produced 20,000 total findings. The archived `run-10/upgradeassessment.json` reports `status.summary.totalFindings=20000`, publishes 200 findings in `status.findings`, and sets `AssessmentOutputTruncated=True`.

## Interpretation

The earlier failure mode was `Request entity too large: limit is 3145728` when the controller wrote all detailed findings into one `UpgradeAssessment.status`. The patched controller turns this into a bounded-write behavior: aggregate counts remain complete, while detailed status and plan action lists are capped. This removes the Kubernetes object-size failure for the tested 5,000 and 10,000 Deployment/PDB-pair cases.

Cluster autoscaling helped absorb workload-creation pressure. In the 1,000-pair rerun, `aks-medium` scaled from 3 to 4 nodes. In the archived 5,000/10,000-pair runs, it scaled up to the configured 8-node maximum; many synthetic pods remained Pending because the benchmark intentionally creates more pods than the configured max-node pool can schedule. The assessment itself uses Kubernetes API objects and completed despite scheduling saturation.

The 100-pair rerun used a clean namespace before and after each repetition. The 1,000-pair rerun used a clean cluster for run-1 and reused the generated load for runs 2-10 to avoid repeated namespace-deletion overhead. RSS values are local process RSS samples from `LOCAL_CONTROLLER_PID`. They are useful resource-cost proxies for this run; the validated median RSS trend is monotone across sizes (49.041, 99.227, 327.535, and 726.073 MiB), but local process RSS should not be interpreted as isolated pod steady-state memory.

## Reproduction

```bash
KUBECONFIG=/home/ihsen/.kube/config-aks-b \
SIZES="100 1000 5000 10000" \
RUNS=10 \
LOCAL_CONTROLLER_PID=<controller-pid> \
CONTROLLER_METRICS_URL=http://127.0.0.1:18084/metrics \
experiments/scale/run-scale-study.sh

# Clean per-run mode used for the 100-pair rerun:
KUBECONFIG=/home/ihsen/.kube/config-aks-b \
SIZES="100" \
RUNS=10 \
SKIP_EXISTING_LOAD=false \
CLEAN_BEFORE_RUN=true \
CLEAN_AFTER_RUN=true \
LOCAL_CONTROLLER_PID=<controller-pid> \
CONTROLLER_METRICS_URL=http://127.0.0.1:18084/metrics \
experiments/scale/run-scale-study.sh

python3 experiments/scale/aggregate-results.py
```
