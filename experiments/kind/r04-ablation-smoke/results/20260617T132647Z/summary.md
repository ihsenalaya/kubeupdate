# R04 Kind Ablation Smoke Summary

- Run ID: `20260617T132647Z`
- Kind image: `kindest/node:v1.24.15`
- Target version: `1.32`
- Expected findings: `31`
- Negative controls: `5`
- Scope: R01 author-controlled fixtures; one checker family disabled per variant.
- Interpretation: ablation smoke only, not independent detection accuracy.
- Diagnostics: `controller.log` contains 435 `ERROR` lines, including 428 optimistic-concurrency or `StorageError` lines; all 7 assessment variants still reached `Completed`.

## Overall Metrics

| Variant | Disabled | Findings | TP | FP | FN | Recall | F1 | NegCtrls | DurationSec |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| full | none | 31 | 31 | 0 | 0 | 1.0 | 1.0 | 0 | 9.478 |
| without-deprecated-apis | deprecatedApis | 27 | 27 | 0 | 4 | 0.871 | 0.931 | 0 | 5.304 |
| without-workload-availability | workloadAvailability | 28 | 28 | 0 | 3 | 0.9032 | 0.9492 | 0 | 5.14 |
| without-pdb | pdb | 24 | 24 | 0 | 7 | 0.7742 | 0.8727 | 0 | 2.892 |
| without-readiness-probes | readinessProbes | 26 | 26 | 0 | 5 | 0.8387 | 0.9123 | 0 | 2.769 |
| without-admission-webhooks | admissionWebhooks | 27 | 27 | 0 | 4 | 0.871 | 0.931 | 0 | 12.198 |
| without-policy-risks | policyRisks | 23 | 23 | 0 | 8 | 0.7419 | 0.8519 | 0 | 5.22 |

## Metrics By Family

| Variant | Family | TP | FP | FN | Recall | F1 |
| --- | --- | --- | --- | --- | --- | --- |
| full | AdmissionWebhook | 4 | 0 | 0 | 1.0 | 1.0 |
| full | DeprecatedAPI | 4 | 0 | 0 | 1.0 | 1.0 |
| full | PDB | 7 | 0 | 0 | 1.0 | 1.0 |
| full | PolicyRisk | 8 | 0 | 0 | 1.0 | 1.0 |
| full | ReadinessProbes | 5 | 0 | 0 | 1.0 | 1.0 |
| full | WorkloadAvailability | 3 | 0 | 0 | 1.0 | 1.0 |
| without-deprecated-apis | AdmissionWebhook | 4 | 0 | 0 | 1.0 | 1.0 |
| without-deprecated-apis | DeprecatedAPI | 0 | 0 | 4 | 0.0 | 0.0 |
| without-deprecated-apis | PDB | 7 | 0 | 0 | 1.0 | 1.0 |
| without-deprecated-apis | PolicyRisk | 8 | 0 | 0 | 1.0 | 1.0 |
| without-deprecated-apis | ReadinessProbes | 5 | 0 | 0 | 1.0 | 1.0 |
| without-deprecated-apis | WorkloadAvailability | 3 | 0 | 0 | 1.0 | 1.0 |
| without-workload-availability | AdmissionWebhook | 4 | 0 | 0 | 1.0 | 1.0 |
| without-workload-availability | DeprecatedAPI | 4 | 0 | 0 | 1.0 | 1.0 |
| without-workload-availability | PDB | 7 | 0 | 0 | 1.0 | 1.0 |
| without-workload-availability | PolicyRisk | 8 | 0 | 0 | 1.0 | 1.0 |
| without-workload-availability | ReadinessProbes | 5 | 0 | 0 | 1.0 | 1.0 |
| without-workload-availability | WorkloadAvailability | 0 | 0 | 3 | 0.0 | 0.0 |
| without-pdb | AdmissionWebhook | 4 | 0 | 0 | 1.0 | 1.0 |
| without-pdb | DeprecatedAPI | 4 | 0 | 0 | 1.0 | 1.0 |
| without-pdb | PDB | 0 | 0 | 7 | 0.0 | 0.0 |
| without-pdb | PolicyRisk | 8 | 0 | 0 | 1.0 | 1.0 |
| without-pdb | ReadinessProbes | 5 | 0 | 0 | 1.0 | 1.0 |
| without-pdb | WorkloadAvailability | 3 | 0 | 0 | 1.0 | 1.0 |
| without-readiness-probes | AdmissionWebhook | 4 | 0 | 0 | 1.0 | 1.0 |
| without-readiness-probes | DeprecatedAPI | 4 | 0 | 0 | 1.0 | 1.0 |
| without-readiness-probes | PDB | 7 | 0 | 0 | 1.0 | 1.0 |
| without-readiness-probes | PolicyRisk | 8 | 0 | 0 | 1.0 | 1.0 |
| without-readiness-probes | ReadinessProbes | 0 | 0 | 5 | 0.0 | 0.0 |
| without-readiness-probes | WorkloadAvailability | 3 | 0 | 0 | 1.0 | 1.0 |
| without-admission-webhooks | AdmissionWebhook | 0 | 0 | 4 | 0.0 | 0.0 |
| without-admission-webhooks | DeprecatedAPI | 4 | 0 | 0 | 1.0 | 1.0 |
| without-admission-webhooks | PDB | 7 | 0 | 0 | 1.0 | 1.0 |
| without-admission-webhooks | PolicyRisk | 8 | 0 | 0 | 1.0 | 1.0 |
| without-admission-webhooks | ReadinessProbes | 5 | 0 | 0 | 1.0 | 1.0 |
| without-admission-webhooks | WorkloadAvailability | 3 | 0 | 0 | 1.0 | 1.0 |
| without-policy-risks | AdmissionWebhook | 4 | 0 | 0 | 1.0 | 1.0 |
| without-policy-risks | DeprecatedAPI | 4 | 0 | 0 | 1.0 | 1.0 |
| without-policy-risks | PDB | 7 | 0 | 0 | 1.0 | 1.0 |
| without-policy-risks | PolicyRisk | 0 | 0 | 8 | 0.0 | 0.0 |
| without-policy-risks | ReadinessProbes | 5 | 0 | 0 | 1.0 | 1.0 |
| without-policy-risks | WorkloadAvailability | 3 | 0 | 0 | 1.0 | 1.0 |

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
```
