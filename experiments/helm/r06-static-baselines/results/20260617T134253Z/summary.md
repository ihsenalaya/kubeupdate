# R06 Helm Static Baselines Summary

- Run ID: `20260617T134253Z`
- R05 input run: `20260617T133841Z`
- Scope: static baseline summaries over rendered Helm workload corpus.
- Interpretation: no TP/FP/FN because no independent labels exist for this corpus.

| Chart | Tool | Return code | Duration (s) | JSON parsed | Primary count | stdout bytes |
| --- | --- | ---: | ---: | --- | ---: | ---: |
| cert-manager | kube-score | 1 | 1.691 | True | 29 | 83374 |
| cert-manager | kube-linter | 1 | 1.276 | True | 9 | 17143 |
| cert-manager | polaris | 0 | 1.5 | True | 20 | 58633 |
| external-dns | kube-score | 1 | 0.784 | True | 8 | 21895 |
| external-dns | kube-linter | 1 | 0.897 | True | 2 | 13485 |
| external-dns | polaris | 0 | 0.843 | True | 9 | 10752 |
| ingress-nginx | kube-score | 1 | 0.806 | True | 19 | 56644 |
| ingress-nginx | kube-linter | 1 | 0.866 | True | 10 | 17677 |
| ingress-nginx | polaris | 0 | 0.947 | True | 25 | 37990 |
| kube-prometheus-stack | kube-score | 1 | 1.081 | True | 47 | 143737 |
| kube-prometheus-stack | kube-linter | 1 | 1.706 | True | 31 | 28111 |
| kube-prometheus-stack | polaris | 0 | 2.086 | True | 86 | 119899 |
| kyverno | kube-score | 1 | 1.166 | True | 39 | 131839 |
| kyverno | kube-linter | 1 | 1.935 | True | 4 | 14297 |
| kyverno | polaris | 0 | 2.1 | True | 63 | 109889 |

## Top Findings

### cert-manager / kube-score

```json
{
  "commentCount": 49,
  "failingCheckInstances": 29,
  "objectCount": 7,
  "parseableJson": true,
  "topFailingChecks": [
    {
      "count": 4,
      "name": "container-ephemeral-storage-request-and-limit"
    },
    {
      "count": 4,
      "name": "container-image-pull-policy"
    },
    {
      "count": 4,
      "name": "container-resources"
    },
    {
      "count": 4,
      "name": "container-security-context-user-group-id"
    },
    {
      "count": 4,
      "name": "pod-networkpolicy"
    },
    {
      "count": 4,
      "name": "pod-topology-spread-constraints"
    },
    {
      "count": 3,
      "name": "deployment-replicas"
    },
    {
      "count": 2,
      "name": "pod-probes"
    }
  ],
  "uniqueFailingChecks": 8
}
```

### cert-manager / kube-linter

```json
{
  "checksStatus": "Failed",
  "parseableJson": true,
  "reportCount": 9,
  "topChecks": [
    {
      "count": 4,
      "name": "unset-cpu-requirements"
    },
    {
      "count": 4,
      "name": "unset-memory-requirements"
    },
    {
      "count": 1,
      "name": "job-ttl-seconds-after-finished"
    }
  ],
  "uniqueChecks": 3
}
```

### cert-manager / polaris

```json
{
  "failedCheckInstances": 20,
  "objectCount": 52,
  "parseableJson": true,
  "score": 91,
  "severityCounts": {
    "warning": 20
  },
  "topFailedChecks": [
    {
      "count": 4,
      "name": "priorityClassNotSet"
    },
    {
      "count": 4,
      "name": "pullPolicyNotAlways"
    },
    {
      "count": 3,
      "name": "deploymentMissingReplicas"
    },
    {
      "count": 3,
      "name": "metadataAndInstanceMismatched"
    },
    {
      "count": 3,
      "name": "missingPodDisruptionBudget"
    },
    {
      "count": 3,
      "name": "topologySpreadConstraint"
    }
  ],
  "uniqueFailedChecks": 6
}
```

### external-dns / kube-score

```json
{
  "commentCount": 12,
  "failingCheckInstances": 8,
  "objectCount": 2,
  "parseableJson": true,
  "topFailingChecks": [
    {
      "count": 1,
      "name": "container-ephemeral-storage-request-and-limit"
    },
    {
      "count": 1,
      "name": "container-image-pull-policy"
    },
    {
      "count": 1,
      "name": "container-resources"
    },
    {
      "count": 1,
      "name": "deployment-replicas"
    },
    {
      "count": 1,
      "name": "deployment-strategy"
    },
    {
      "count": 1,
      "name": "pod-networkpolicy"
    },
    {
      "count": 1,
      "name": "pod-probes-identical"
    },
    {
      "count": 1,
      "name": "pod-topology-spread-constraints"
    }
  ],
  "uniqueFailingChecks": 8
}
```

### external-dns / kube-linter

```json
{
  "checksStatus": "Failed",
  "parseableJson": true,
  "reportCount": 2,
  "topChecks": [
    {
      "count": 1,
      "name": "unset-cpu-requirements"
    },
    {
      "count": 1,
      "name": "unset-memory-requirements"
    }
  ],
  "uniqueChecks": 2
}
```

### external-dns / polaris

```json
{
  "failedCheckInstances": 9,
  "objectCount": 5,
  "parseableJson": true,
  "score": 80,
  "severityCounts": {
    "warning": 9
  },
  "topFailedChecks": [
    {
      "count": 1,
      "name": "cpuLimitsMissing"
    },
    {
      "count": 1,
      "name": "cpuRequestsMissing"
    },
    {
      "count": 1,
      "name": "deploymentMissingReplicas"
    },
    {
      "count": 1,
      "name": "memoryLimitsMissing"
    },
    {
      "count": 1,
      "name": "memoryRequestsMissing"
    },
    {
      "count": 1,
      "name": "missingPodDisruptionBudget"
    },
    {
      "count": 1,
      "name": "priorityClassNotSet"
    },
    {
      "count": 1,
      "name": "pullPolicyNotAlways"
    },
    {
      "count": 1,
      "name": "topologySpreadConstraint"
    }
  ],
  "uniqueFailedChecks": 9
}
```

### ingress-nginx / kube-score

```json
{
  "commentCount": 30,
  "failingCheckInstances": 19,
  "objectCount": 6,
  "parseableJson": true,
  "topFailingChecks": [
    {
      "count": 3,
      "name": "container-ephemeral-storage-request-and-limit"
    },
    {
      "count": 3,
      "name": "container-image-pull-policy"
    },
    {
      "count": 3,
      "name": "container-resources"
    },
    {
      "count": 3,
      "name": "pod-networkpolicy"
    },
    {
      "count": 3,
      "name": "pod-topology-spread-constraints"
    },
    {
      "count": 1,
      "name": "container-security-context-readonlyrootfilesystem"
    },
    {
      "count": 1,
      "name": "container-security-context-user-group-id"
    },
    {
      "count": 1,
      "name": "deployment-has-host-podantiaffinity"
    },
    {
      "count": 1,
      "name": "pod-probes-identical"
    }
  ],
  "uniqueFailingChecks": 9
}
```

### ingress-nginx / kube-linter

```json
{
  "checksStatus": "Failed",
  "parseableJson": true,
  "reportCount": 10,
  "topChecks": [
    {
      "count": 3,
      "name": "unset-memory-requirements"
    },
    {
      "count": 2,
      "name": "unset-cpu-requirements"
    },
    {
      "count": 1,
      "name": "liveness-port"
    },
    {
      "count": 1,
      "name": "no-anti-affinity"
    },
    {
      "count": 1,
      "name": "no-read-only-root-fs"
    },
    {
      "count": 1,
      "name": "pdb-unhealthy-pod-eviction-policy"
    },
    {
      "count": 1,
      "name": "readiness-port"
    }
  ],
  "uniqueChecks": 7
}
```

### ingress-nginx / polaris

```json
{
  "failedCheckInstances": 25,
  "objectCount": 19,
  "parseableJson": true,
  "score": 85,
  "severityCounts": {
    "warning": 25
  },
  "topFailedChecks": [
    {
      "count": 3,
      "name": "cpuLimitsMissing"
    },
    {
      "count": 3,
      "name": "memoryLimitsMissing"
    },
    {
      "count": 3,
      "name": "metadataAndInstanceMismatched"
    },
    {
      "count": 3,
      "name": "priorityClassNotSet"
    },
    {
      "count": 3,
      "name": "pullPolicyNotAlways"
    },
    {
      "count": 2,
      "name": "automountServiceAccountToken"
    },
    {
      "count": 2,
      "name": "cpuRequestsMissing"
    },
    {
      "count": 2,
      "name": "memoryRequestsMissing"
    },
    {
      "count": 2,
      "name": "missingNetworkPolicy"
    },
    {
      "count": 1,
      "name": "notReadOnlyRootFilesystem"
    }
  ],
  "uniqueFailedChecks": 11
}
```

### kube-prometheus-stack / kube-score

```json
{
  "commentCount": 94,
  "failingCheckInstances": 47,
  "objectCount": 17,
  "parseableJson": true,
  "topFailingChecks": [
    {
      "count": 7,
      "name": "service-targets-pod"
    },
    {
      "count": 6,
      "name": "container-ephemeral-storage-request-and-limit"
    },
    {
      "count": 6,
      "name": "container-image-pull-policy"
    },
    {
      "count": 6,
      "name": "container-resources"
    },
    {
      "count": 6,
      "name": "pod-networkpolicy"
    },
    {
      "count": 6,
      "name": "pod-topology-spread-constraints"
    },
    {
      "count": 3,
      "name": "container-security-context-user-group-id"
    },
    {
      "count": 3,
      "name": "deployment-replicas"
    },
    {
      "count": 3,
      "name": "pod-probes-identical"
    },
    {
      "count": 1,
      "name": "container-security-context-readonlyrootfilesystem"
    }
  ],
  "uniqueFailingChecks": 10
}
```

### kube-prometheus-stack / kube-linter

```json
{
  "checksStatus": "Failed",
  "parseableJson": true,
  "reportCount": 31,
  "topChecks": [
    {
      "count": 8,
      "name": "unset-cpu-requirements"
    },
    {
      "count": 8,
      "name": "unset-memory-requirements"
    },
    {
      "count": 7,
      "name": "dangling-service"
    },
    {
      "count": 3,
      "name": "no-read-only-root-fs"
    },
    {
      "count": 3,
      "name": "sensitive-host-mounts"
    },
    {
      "count": 1,
      "name": "host-network"
    },
    {
      "count": 1,
      "name": "host-pid"
    }
  ],
  "uniqueChecks": 7
}
```

### kube-prometheus-stack / polaris

```json
{
  "failedCheckInstances": 86,
  "objectCount": 129,
  "parseableJson": true,
  "score": 79,
  "severityCounts": {
    "danger": 3,
    "warning": 83
  },
  "topFailedChecks": [
    {
      "count": 8,
      "name": "cpuLimitsMissing"
    },
    {
      "count": 8,
      "name": "cpuRequestsMissing"
    },
    {
      "count": 8,
      "name": "memoryLimitsMissing"
    },
    {
      "count": 8,
      "name": "memoryRequestsMissing"
    },
    {
      "count": 8,
      "name": "pullPolicyNotAlways"
    },
    {
      "count": 6,
      "name": "metadataAndInstanceMismatched"
    },
    {
      "count": 6,
      "name": "missingNetworkPolicy"
    },
    {
      "count": 6,
      "name": "priorityClassNotSet"
    },
    {
      "count": 5,
      "name": "automountServiceAccountToken"
    },
    {
      "count": 4,
      "name": "topologySpreadConstraint"
    }
  ],
  "uniqueFailedChecks": 21
}
```

### kyverno / kube-score

```json
{
  "commentCount": 49,
  "failingCheckInstances": 39,
  "objectCount": 14,
  "parseableJson": true,
  "topFailingChecks": [
    {
      "count": 7,
      "name": "container-ephemeral-storage-request-and-limit"
    },
    {
      "count": 7,
      "name": "container-image-pull-policy"
    },
    {
      "count": 7,
      "name": "pod-networkpolicy"
    },
    {
      "count": 7,
      "name": "pod-topology-spread-constraints"
    },
    {
      "count": 4,
      "name": "container-resources"
    },
    {
      "count": 3,
      "name": "deployment-replicas"
    },
    {
      "count": 2,
      "name": "deployment-has-poddisruptionbudget"
    },
    {
      "count": 2,
      "name": "pod-probes"
    }
  ],
  "uniqueFailingChecks": 8
}
```

### kyverno / kube-linter

```json
{
  "checksStatus": "Failed",
  "parseableJson": true,
  "reportCount": 4,
  "topChecks": [
    {
      "count": 3,
      "name": "job-ttl-seconds-after-finished"
    },
    {
      "count": 1,
      "name": "pdb-unhealthy-pod-eviction-policy"
    }
  ],
  "uniqueChecks": 2
}
```

### kyverno / polaris

```json
{
  "failedCheckInstances": 63,
  "objectCount": 76,
  "parseableJson": true,
  "score": 84,
  "severityCounts": {
    "danger": 12,
    "warning": 51
  },
  "topFailedChecks": [
    {
      "count": 8,
      "name": "pullPolicyNotAlways"
    },
    {
      "count": 7,
      "name": "automountServiceAccountToken"
    },
    {
      "count": 7,
      "name": "metadataAndInstanceMismatched"
    },
    {
      "count": 7,
      "name": "missingNetworkPolicy"
    },
    {
      "count": 7,
      "name": "priorityClassNotSet"
    },
    {
      "count": 4,
      "name": "clusterrolePodExecAttach"
    },
    {
      "count": 4,
      "name": "clusterrolebindingClusterAdmin"
    },
    {
      "count": 4,
      "name": "clusterrolebindingPodExecAttach"
    },
    {
      "count": 4,
      "name": "cpuLimitsMissing"
    },
    {
      "count": 4,
      "name": "topologySpreadConstraint"
    }
  ],
  "uniqueFailedChecks": 13
}
```
