# R01 Benchmark Summary

- Run ID: `20260617T083259Z`
- Cluster context: `kind-kug-r01`
- Kind image: `kindest/node:v1.24.15`
- Target version: `1.32`
- Expected findings: `18`
- Observed KubeUpgrade Guardian findings: `18`

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
10:36AM INF version 0.7.3 (git sha 57480c07b3f91238f12a35d0ec88d9368aae99aa)
```

## Overall Metrics

| Tool | TP | FP | FN | Precision | Recall | F1 |
| --- | --- | --- | --- | --- | --- | --- |
| KubeUpgrade Guardian | 18 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| Pluto files | 3 | 0 | 15 | 1.0 | 0.1667 | 0.2857 |
| Pluto cluster | 3 | 0 | 15 | 1.0 | 0.1667 | 0.2857 |
| kubent files | 3 | 0 | 15 | 1.0 | 0.1667 | 0.2857 |
| kubent cluster | 3 | 0 | 15 | 1.0 | 0.1667 | 0.2857 |

## Metrics By Family

| Tool | Family | TP | FP | FN | Precision | Recall | F1 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| KubeUpgrade Guardian | AdmissionWebhook | 3 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| KubeUpgrade Guardian | DeprecatedAPI | 3 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| KubeUpgrade Guardian | PDB | 4 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| KubeUpgrade Guardian | PolicyRisk | 5 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| KubeUpgrade Guardian | ReadinessProbes | 1 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| KubeUpgrade Guardian | WorkloadAvailability | 2 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| Pluto files | AdmissionWebhook | 0 | 0 | 3 | 0.0 | 0.0 | 0.0 |
| Pluto files | DeprecatedAPI | 3 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| Pluto files | PDB | 0 | 0 | 4 | 0.0 | 0.0 | 0.0 |
| Pluto files | PolicyRisk | 0 | 0 | 5 | 0.0 | 0.0 | 0.0 |
| Pluto files | ReadinessProbes | 0 | 0 | 1 | 0.0 | 0.0 | 0.0 |
| Pluto files | WorkloadAvailability | 0 | 0 | 2 | 0.0 | 0.0 | 0.0 |
| Pluto cluster | AdmissionWebhook | 0 | 0 | 3 | 0.0 | 0.0 | 0.0 |
| Pluto cluster | DeprecatedAPI | 3 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| Pluto cluster | PDB | 0 | 0 | 4 | 0.0 | 0.0 | 0.0 |
| Pluto cluster | PolicyRisk | 0 | 0 | 5 | 0.0 | 0.0 | 0.0 |
| Pluto cluster | ReadinessProbes | 0 | 0 | 1 | 0.0 | 0.0 | 0.0 |
| Pluto cluster | WorkloadAvailability | 0 | 0 | 2 | 0.0 | 0.0 | 0.0 |
| kubent files | AdmissionWebhook | 0 | 0 | 3 | 0.0 | 0.0 | 0.0 |
| kubent files | DeprecatedAPI | 3 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| kubent files | PDB | 0 | 0 | 4 | 0.0 | 0.0 | 0.0 |
| kubent files | PolicyRisk | 0 | 0 | 5 | 0.0 | 0.0 | 0.0 |
| kubent files | ReadinessProbes | 0 | 0 | 1 | 0.0 | 0.0 | 0.0 |
| kubent files | WorkloadAvailability | 0 | 0 | 2 | 0.0 | 0.0 | 0.0 |
| kubent cluster | AdmissionWebhook | 0 | 0 | 3 | 0.0 | 0.0 | 0.0 |
| kubent cluster | DeprecatedAPI | 3 | 0 | 0 | 1.0 | 1.0 | 1.0 |
| kubent cluster | PDB | 0 | 0 | 4 | 0.0 | 0.0 | 0.0 |
| kubent cluster | PolicyRisk | 0 | 0 | 5 | 0.0 | 0.0 | 0.0 |
| kubent cluster | ReadinessProbes | 0 | 0 | 1 | 0.0 | 0.0 | 0.0 |
| kubent cluster | WorkloadAvailability | 0 | 0 | 2 | 0.0 | 0.0 | 0.0 |

## Notes

- Pluto and kubent are API-deprecation baselines; their non-deprecated family false negatives are expected and make the scope difference explicit.
- This run is a controlled benchmark, not yet a production-cluster evaluation.
