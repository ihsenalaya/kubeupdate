# Upgrade Plan

- Name: kubeupgrade-guardian-system/kind-assessment-plan
- Assessment: kubeupgrade-guardian-system/kind-assessment
- Decision: DoNotUpgrade
- Risk level: Critical
- Score: 68
- Source version: 1.34
- Target version: 1.35
## Classification Summary

- Total: 8
- Blocking: 8
- Accepted risk: 0
- Provider managed: 0
- Informational: 0

## Upgrade Chronology

### 1.34 -> 1.35

1. prechecks: Verify blocking findings, accepted risks, provider-managed risks, capacity, observability, and backups before this version hop.
2. control-plane-upgrade: Upgrade the Kubernetes control plane first.
3. post-control-plane-validation: Validate API server health, admission behavior, Argo CD sync state, and core platform controllers.
4. system-nodepool-upgrade: Upgrade system node pools with controlled drain and workload validation.
5. user-nodepool-upgrade: Upgrade application node pools by wave, starting with the lowest-risk workloads.
6. application-validation: Validate application readiness, SLO dashboards, alerts, and rollback criteria before continuing.

## Required Actions

| Severity | Category | Resource | Action |
| --- | --- | --- | --- |
| High | Capacity | cluster | Add capacity or reduce requests before upgrade. Estimated remaining capacity after one-node loss: 0m CPU. |
| Medium | ReadinessProbes | Deployment upgrade-lab/single-replica-api | Add a readinessProbe that reflects whether the container can safely receive traffic. |
| Medium | Observability | cluster | Install or validate monitoring coverage before upgrade. |
| Low | Observability | cluster | Install or validate monitoring coverage before upgrade. |
| Critical | PDB | PodDisruptionBudget upgrade-lab/single-replica-api | Increase workload replicas or relax the PodDisruptionBudget before upgrade. |
| High | PolicyRisk | Deployment upgrade-lab/single-replica-api | Adjust the pod security context or namespace policy before upgrade. |
| Medium | PolicyRisk | Namespace upgrade-lab | Validate all workloads in this namespace against the restricted Pod Security profile before upgrade. |
| High | WorkloadAvailability | Deployment upgrade-lab/single-replica-api | Increase replicas to at least 2 or document why this workload can tolerate disruption. |
## Blocking Findings

| Classification | Severity | Category | Resource | Reason | Message | Recommendation |
| --- | --- | --- | --- | --- | --- | --- |
| Blocking | High | Capacity | cluster | Finding can affect upgrade safety and has not been accepted or classified as provider-managed. | Cluster may not have enough requested capacity headroom to tolerate one worker node loss. | Add capacity or reduce requests before upgrade. Estimated remaining capacity after one-node loss: 0m CPU. |
| Blocking | Medium | ReadinessProbes | Deployment upgrade-lab/single-replica-api | Finding can affect upgrade safety and has not been accepted or classified as provider-managed. | Container app in Deployment upgrade-lab/single-replica-api has no readinessProbe. | Add a readinessProbe that reflects whether the container can safely receive traffic. |
| Blocking | Medium | Observability | cluster | Finding can affect upgrade safety and has not been accepted or classified as provider-managed. | No monitoring, prometheus, or observability namespace found. | Install or validate monitoring coverage before upgrade. |
| Blocking | Low | Observability | cluster | Finding can affect upgrade safety and has not been accepted or classified as provider-managed. | Prometheus CRD not detected; upgrade observability validation may be incomplete. | Install or validate monitoring coverage before upgrade. |
| Blocking | Critical | PDB | PodDisruptionBudget upgrade-lab/single-replica-api | Finding can affect upgrade safety and has not been accepted or classified as provider-managed. | PDB upgrade-lab/single-replica-api may block disruption for Deployment upgrade-lab/single-replica-api. | Increase workload replicas or relax the PodDisruptionBudget before upgrade. |
| Blocking | High | PolicyRisk | Deployment upgrade-lab/single-replica-api | Finding can affect upgrade safety and has not been accepted or classified as provider-managed. | Deployment upgrade-lab/single-replica-api may violate restricted policy: runAsNonRoot absent. | Adjust the pod security context or namespace policy before upgrade. |
| Blocking | Medium | PolicyRisk | Namespace upgrade-lab | Finding can affect upgrade safety and has not been accepted or classified as provider-managed. | Namespace upgrade-lab enforces Pod Security restricted. | Validate all workloads in this namespace against the restricted Pod Security profile before upgrade. |
| Blocking | High | WorkloadAvailability | Deployment upgrade-lab/single-replica-api | Finding can affect upgrade safety and has not been accepted or classified as provider-managed. | Deployment upgrade-lab/single-replica-api has fewer than 2 replicas. | Increase replicas to at least 2 or document why this workload can tolerate disruption. |

## Accepted Risks

No entries.

## Provider Managed Risks

No entries.

## Informational Findings

No entries.

