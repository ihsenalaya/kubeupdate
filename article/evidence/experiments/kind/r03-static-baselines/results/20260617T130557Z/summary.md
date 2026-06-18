# R03 Static Baseline Smoke Summary

- Run ID: `20260617T130557Z`
- Input manifest: `experiments/kind/r01-benchmark/manifests/00-scenarios.yaml`
- Scope: static baseline feasibility on the existing R01 scenario manifest.
- Interpretation: raw smoke data only; no TP/FP/FN or superiority claim.

| Tool | Return code | Duration (s) | JSON parsed | Primary count | Version |
| --- | ---: | ---: | --- | ---: | --- |
| kube-score-files | 1 | 0.817 | True | 185 | `kube-score version: v1.20.0-docker-linux/amd64, commit: 81371e9f53b633bec69423fc298295bd71bd869a, built: 2025-04-28T18:55:13+02:00` |
| kube-linter-files | 1 | 0.843 | True | 113 | `0.8.3` |
| polaris-files | 0 | 1.22 | True | 339 | `Polaris version:10.1.8` |

## Normalized Counts

```json
{
  "kube-linter-files": {
    "checksStatus": "Failed",
    "parseableJson": true,
    "reportCount": 113,
    "topChecks": [
      {
        "count": 23,
        "name": "no-read-only-root-fs"
      },
      {
        "count": 23,
        "name": "unset-cpu-requirements"
      },
      {
        "count": 23,
        "name": "unset-memory-requirements"
      },
      {
        "count": 17,
        "name": "no-anti-affinity"
      },
      {
        "count": 15,
        "name": "pdb-unhealthy-pod-eviction-policy"
      },
      {
        "count": 4,
        "name": "run-as-non-root"
      },
      {
        "count": 2,
        "name": "pdb-max-unavailable"
      },
      {
        "count": 2,
        "name": "pdb-min-available"
      },
      {
        "count": 2,
        "name": "privilege-escalation-container"
      },
      {
        "count": 1,
        "name": "dangling-service"
      }
    ],
    "uniqueChecks": 11
  },
  "kube-score-files": {
    "commentCount": 283,
    "failingCheckInstances": 185,
    "objectCount": 41,
    "parseableJson": true,
    "topFailingChecks": [
      {
        "count": 22,
        "name": "container-ephemeral-storage-request-and-limit"
      },
      {
        "count": 22,
        "name": "container-image-pull-policy"
      },
      {
        "count": 22,
        "name": "container-resources"
      },
      {
        "count": 22,
        "name": "container-security-context-readonlyrootfilesystem"
      },
      {
        "count": 22,
        "name": "container-security-context-user-group-id"
      },
      {
        "count": 22,
        "name": "pod-networkpolicy"
      },
      {
        "count": 22,
        "name": "pod-topology-spread-constraints"
      },
      {
        "count": 14,
        "name": "deployment-has-host-podantiaffinity"
      },
      {
        "count": 4,
        "name": "statefulset-has-servicename"
      },
      {
        "count": 3,
        "name": "statefulset-has-host-podantiaffinity"
      }
    ],
    "uniqueFailingChecks": 18
  },
  "polaris-files": {
    "failedCheckInstances": 339,
    "objectCount": 68,
    "parseableJson": true,
    "score": 67,
    "severityCounts": {
      "danger": 10,
      "warning": 329
    },
    "topFailedChecks": [
      {
        "count": 23,
        "name": "cpuLimitsMissing"
      },
      {
        "count": 23,
        "name": "cpuRequestsMissing"
      },
      {
        "count": 23,
        "name": "insecureCapabilities"
      },
      {
        "count": 23,
        "name": "linuxHardening"
      },
      {
        "count": 23,
        "name": "memoryLimitsMissing"
      },
      {
        "count": 23,
        "name": "memoryRequestsMissing"
      },
      {
        "count": 23,
        "name": "notReadOnlyRootFilesystem"
      },
      {
        "count": 23,
        "name": "pullPolicyNotAlways"
      },
      {
        "count": 22,
        "name": "automountServiceAccountToken"
      },
      {
        "count": 22,
        "name": "livenessProbeMissing"
      }
    ],
    "uniqueFailedChecks": 23
  }
}
```

Raw stdout/stderr files are archived next to this summary.
