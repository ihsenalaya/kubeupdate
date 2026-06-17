# R04 AKS Scale Boundary Results

Date: 2026-06-17

This run used `aks-medium` with Kubernetes `1.34.8` and the patched local controller from `kubeupgrade-guardian-operator`. The in-cluster operator deployment was scaled to zero, so pod RSS metrics were not available for this run; local `ps` samples are archived under `experiments/scale/resource-samples`.

## Result Summary

| Deployment/PDB pairs | Completed | Duration | API request delta | Outcome |
| ---: | :---: | ---: | ---: | --- |
| 100 | yes | 7s | 18 | Assessment completed. |
| 1,000 | yes | 41s | 3 | Assessment completed. |
| 5,000 | no | 271s | 20 | `UpgradeAssessment` status update failed with `Request entity too large: limit is 3145728`. |
| 10,000 | no | 601s | n/a | Setup saw API-server `ServiceUnavailable`; assessment then failed with the same status-size limit. |

Aggregated metrics are in `experiments/scale/r04-scalability-summary.json`.

## Interpretation

This is a measured storage-design limit. The operator stores detailed findings in `UpgradeAssessment.status`. At thousands of synthetic PDB-blocking findings, the status update exceeds the Kubernetes request-size limit. A production-scale design needs bounded status summaries plus paginated or externalized detailed findings.

## Resource Samples

- 5,000-pair failure: `experiments/scale/resource-samples/r04-local-controller-5000-sample.txt`, RSS approximately 306 MiB.
- 10,000-pair failure: `experiments/scale/resource-samples/r04-local-controller-10000-sample.txt`, RSS approximately 764 MiB.

## Reproduction

```bash
KUBECONFIG=/home/ihsen/.kube/config-aks-b \
SIZES="100 1000 5000 10000" \
RUNS=1 \
CONTROLLER_METRICS_URL=http://127.0.0.1:18084/metrics \
experiments/scale/run-scale-study.sh

python3 experiments/scale/aggregate-results.py
```
