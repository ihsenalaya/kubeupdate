# R01 Benchmark Summary

- Run ID: `20260617T091703Z`
- Cluster context: `kind-kug-r01`
- Kind image: `kindest/node:v1.24.15`
- Target version: `1.32`
- Expected findings: `31`
- Negative controls: `5`
- Observed KubeUpgrade Guardian findings: `31`

## Tool Versions

```text
kind v0.31.0 go1.25.5 linux/amd64
{
  "clientVersion": {
    "major": "1",
    "minor": "34",
    "gitVersion": "v1.34.1",
    "gitCommit": "93248f9ae092f571eb870b7664c534bfc7d00f03",
    "gitTreeState": "clean",
    "buildDate": "2025-09-09T19:44:50Z",
    "goVersion": "go1.24.6",
    "compiler": "gc",
    "platform": "linux/amd64"
  },
  "kustomizeVersion": "v5.7.1",
  "serverVersion": {
    "major": "1",
    "minor": "24",
    "gitVersion": "v1.24.15",
    "gitCommit": "2c67202dc0bb96a7a837cbfb8d72e1f34dfc2808",
    "gitTreeState": "clean",
    "buildDate": "2023-06-15T01:09:03Z",
    "goVersion": "go1.19.10",
    "compiler": "gc",
    "platform": "linux/amd64"
  }
}
Warning: version difference between client (1.34) and server (1.24) exceeds the supported minor version skew of +/-1
go version go1.25.8 linux/amd64
Version:5.24.0 Commit:dd5ec8cccce5e42dfe8054b8250baa35546056a0
11:21AM INF version 0.7.3 (git sha 57480c07b3f91238f12a35d0ec88d9368aae99aa)
```

## API-Deprecation Subset

| Tool | TP | FP | FN | Precision | Recall | F1 |
| --- | --- | --- | --- | --- | --- | --- |
| KubeUpgrade Guardian | 4 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| Pluto files | 4 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| Pluto cluster | 4 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| kubent files | 4 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| kubent cluster | 4 | 0 | 0 | 1.0 | 1.0 | 1.0 |

## Non-API Readiness Coverage

| Tool | Covered | Unexpected | Uncovered | Coverage | F1 |
| --- | --- | --- | --- | --- | --- |
| KubeUpgrade Guardian | 27 | 0 | 0 | 1.0 | 1.0 |
| Pluto files | 0 | 0 | 27 | 0.0 | 0.0 |
| Pluto cluster | 0 | 0 | 27 | 0.0 | 0.0 |
| kubent files | 0 | 0 | 27 | 0.0 | 0.0 |
| kubent cluster | 0 | 0 | 27 | 0.0 | 0.0 |

## Non-API Metrics By Family

| Tool | Family | TP | FP | FN | Precision | Recall | F1 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| KubeUpgrade Guardian | AdmissionWebhook | 4 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| KubeUpgrade Guardian | PDB | 7 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| KubeUpgrade Guardian | PolicyRisk | 8 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| KubeUpgrade Guardian | ReadinessProbes | 5 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| KubeUpgrade Guardian | WorkloadAvailability | 3 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| Pluto files | AdmissionWebhook | 0 | 0 | 4 | 0.0 | 0.0 | 0.0 |
| Pluto files | PDB | 0 | 0 | 7 | 0.0 | 0.0 | 0.0 |
| Pluto files | PolicyRisk | 0 | 0 | 8 | 0.0 | 0.0 | 0.0 |
| Pluto files | ReadinessProbes | 0 | 0 | 5 | 0.0 | 0.0 | 0.0 |
| Pluto files | WorkloadAvailability | 0 | 0 | 3 | 0.0 | 0.0 | 0.0 |
| Pluto cluster | AdmissionWebhook | 0 | 0 | 4 | 0.0 | 0.0 | 0.0 |
| Pluto cluster | PDB | 0 | 0 | 7 | 0.0 | 0.0 | 0.0 |
| Pluto cluster | PolicyRisk | 0 | 0 | 8 | 0.0 | 0.0 | 0.0 |
| Pluto cluster | ReadinessProbes | 0 | 0 | 5 | 0.0 | 0.0 | 0.0 |
| Pluto cluster | WorkloadAvailability | 0 | 0 | 3 | 0.0 | 0.0 | 0.0 |
| kubent files | AdmissionWebhook | 0 | 0 | 4 | 0.0 | 0.0 | 0.0 |
| kubent files | PDB | 0 | 0 | 7 | 0.0 | 0.0 | 0.0 |
| kubent files | PolicyRisk | 0 | 0 | 8 | 0.0 | 0.0 | 0.0 |
| kubent files | ReadinessProbes | 0 | 0 | 5 | 0.0 | 0.0 | 0.0 |
| kubent files | WorkloadAvailability | 0 | 0 | 3 | 0.0 | 0.0 | 0.0 |
| kubent cluster | AdmissionWebhook | 0 | 0 | 4 | 0.0 | 0.0 | 0.0 |
| kubent cluster | PDB | 0 | 0 | 7 | 0.0 | 0.0 | 0.0 |
| kubent cluster | PolicyRisk | 0 | 0 | 8 | 0.0 | 0.0 | 0.0 |
| kubent cluster | ReadinessProbes | 0 | 0 | 5 | 0.0 | 0.0 | 0.0 |
| kubent cluster | WorkloadAvailability | 0 | 0 | 3 | 0.0 | 0.0 | 0.0 |

## Negative Controls

| Control | Resource | KUG | Pluto files | Pluto cluster | kubent files | kubent cluster |
| --- | --- | --- | --- | --- | --- | --- |
| neg-safe-deployment | Deployment r01-safe/profile-api | 0 | 0 | 0 | 0 | 0 |
| neg-safe-webhook | ValidatingWebhookConfiguration r01-safe-webhook | 0 | 0 | 0 | 0 | 0 |
| neg-policy-warn-audit-only | Namespace r01-policy-warn-audit | 0 | 0 | 0 | 0 | 0 |
| neg-modern-hpa | HorizontalPodAutoscaler r01-modern-api-negative/modern-hpa | 0 | 0 | 0 | 0 | 0 |
| neg-modern-pdb-served-through-conversion | PodDisruptionBudget r01-modern-api-negative/modern-hpa-target | 0 | 0 | 0 | 0 | 0 |

## Notes

- Pluto and kubent are evaluated as specialized API-deprecation baselines, not as general readiness tools.
- The non-API table reports coverage outside the declared scope of Pluto and kubent; it must not be read as a global superiority claim.
- This run is a controlled benchmark with positive and negative fixtures, not a production-cluster evaluation.
