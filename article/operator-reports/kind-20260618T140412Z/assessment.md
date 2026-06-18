# Upgrade Assessment

- Name: kubeupgrade-guardian-system/kind-assessment
- Source version: 1.34
- Target version: 1.35
- Profile: production
- Phase: Completed
- Decision risk level: Critical
- Score: 68
- Generated plan: kind-assessment-plan
- Artifact: kind-assessment-artifact
## Effective Finding Summary

- Total: 8
- Critical: 1
- High: 3
- Medium: 3
- Low: 1
- Info: 0

## Raw Finding Summary

- Total: 8
- Critical: 1
- High: 3
- Medium: 3
- Low: 1
- Info: 0

## Classification Summary

- Total: 8
- Blocking: 8
- Accepted risk: 0
- Provider managed: 0
- Informational: 0

## Findings

| Classification | Severity | Category | Resource | Message | Recommendation |
| --- | --- | --- | --- | --- | --- |
| Blocking | High | Capacity | cluster | Cluster may not have enough requested capacity headroom to tolerate one worker node loss. | Add capacity or reduce requests before upgrade. Estimated remaining capacity after one-node loss: 0m CPU. |
| Blocking | Medium | ReadinessProbes | Deployment upgrade-lab/single-replica-api | Container app in Deployment upgrade-lab/single-replica-api has no readinessProbe. | Add a readinessProbe that reflects whether the container can safely receive traffic. |
| Blocking | Medium | Observability | cluster | No monitoring, prometheus, or observability namespace found. | Install or validate monitoring coverage before upgrade. |
| Blocking | Low | Observability | cluster | Prometheus CRD not detected; upgrade observability validation may be incomplete. | Install or validate monitoring coverage before upgrade. |
| Blocking | Critical | PDB | PodDisruptionBudget upgrade-lab/single-replica-api | PDB upgrade-lab/single-replica-api may block disruption for Deployment upgrade-lab/single-replica-api. | Increase workload replicas or relax the PodDisruptionBudget before upgrade. |
| Blocking | High | PolicyRisk | Deployment upgrade-lab/single-replica-api | Deployment upgrade-lab/single-replica-api may violate restricted policy: runAsNonRoot absent. | Adjust the pod security context or namespace policy before upgrade. |
| Blocking | Medium | PolicyRisk | Namespace upgrade-lab | Namespace upgrade-lab enforces Pod Security restricted. | Validate all workloads in this namespace against the restricted Pod Security profile before upgrade. |
| Blocking | High | WorkloadAvailability | Deployment upgrade-lab/single-replica-api | Deployment upgrade-lab/single-replica-api has fewer than 2 replicas. | Increase replicas to at least 2 or document why this workload can tolerate disruption. |
