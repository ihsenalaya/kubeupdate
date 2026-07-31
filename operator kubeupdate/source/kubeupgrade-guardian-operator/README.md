# KubeUpgrade Guardian Operator

## Description

KubeUpgrade Guardian Operator analyzes Kubernetes upgrade readiness in read-only mode. It never upgrades a cluster, drains nodes, patches workloads, or performs destructive actions. It watches `UpgradeAssessment` resources, runs observable checks against the cluster, writes findings to assessment status, and creates an idempotent `UpgradePlan` with prioritized recommendations.

The checks cover:

- workload availability risks;
- missing readiness probes;
- PodDisruptionBudget blockers;
- admission webhook risks: failure policy graded by drain blast radius, absent or
  unready backends, webhook timeouts;
- Pod Security Admission (restricted profile, effective pod + container security
  context) and policy engine signals;
- objects still written through a removed API version, from an embedded removal
  table;
- conservative one-node-loss capacity headroom;
- observability gaps and monitoring CRD detection;
- RBAC gaps when a check cannot verify required data.

The cluster is read exactly once per assessment, through `internal/snapshot`: one
paginated List per resource type against the API server, no informer, no cache.
Every checker is a pure function over that snapshot, so all findings of one
assessment describe the same cluster state.

## Getting Started

### Prerequisites
- go version v1.21.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### Run an Assessment

Install the CRDs and deploy the controller:

```sh
make install
make deploy IMG=<some-registry>/kubeupgrade-guardian-operator:tag
```

Apply an `UpgradeAssessment`:

```sh
kubectl apply -f config/samples/upgrade_v1alpha1_upgradeassessment.yaml
kubectl get upgradeassessment
kubectl get upgradeplan
```

Both resources print their outcome directly:

```
NAME                      TARGET   PHASE       RISK       SCORE   AGE
prod-upgrade-assessment   1.32     Completed   Critical   74      3m

NAME                           DECISION       RISK       SCORE   AGE
prod-upgrade-assessment-plan   DoNotUpgrade   Critical   74      3m
```

The generated `UpgradePlan` contains recommendations only. Every recommendation stays non-executing and should be reviewed by an operator before any real cluster upgrade work.

### Assessment spec

| Field | Meaning |
| --- | --- |
| `sourceVersion` | Current minor version, e.g. `1.31`. Defaults to `current` in the report. |
| `targetVersion` | Minor version being assessed, e.g. `1.32`. Required. |
| `mode` | `ReadOnly`, the only supported mode. |
| `profile` | `lab`, `staging` or `production`. Tunes which severities block. |
| `refreshInterval` | Optional. Re-runs the audit on that period once completed, so the assessment tracks cluster drift. Values under `1m` are raised to `1m`. Omit it to assess on demand only. |
| `scope.namespaces.include` / `.exclude` | Namespaces the workload checks look at. Cluster-level checks (capacity, webhooks, observability) always reason about the whole cluster. |
| `checks.*` | Enable individual checkers. When none is set, all of them run. |
| `acceptedRisks` | Documented exceptions, matched by finding id, type or resource. |

### Assessment status

| Field | Meaning |
| --- | --- |
| `phase` | `Pending`, `Running`, `Completed` or `Failed`. |
| `riskLevel`, `score` | Aggregate risk of the blocking findings. |
| `summary`, `rawSummary`, `classificationSummary` | Counts of effective, raw and classified findings. |
| `findings` | Published findings, bounded to keep the object under the API server size limit. |
| `generatedPlanRef`, `artifactRef` | The generated `UpgradePlan` and artifact ConfigMap. |
| `lastAssessedTime` | When the cluster snapshot behind this status was taken. |
| `lastRerunToken` | The re-run annotation value already acted on. |

Conditions: `AssessmentRunning`, `AssessmentCompleted`, `AssessmentFailed`,
`AssessmentDegraded` (the audit completed but a checker failed or a resource type
could not be read, so the findings are partial) and `AssessmentOutputTruncated`.

### Re-running an assessment

An assessment runs once per spec change. To force a fresh audit without touching
the spec, change the value of the re-run annotation:

```sh
kubectl annotate upgradeassessment prod-upgrade-assessment \
  upgrade.guardian.io/rerun="$(date +%s)" --overwrite
```

The controller records the value it acted on in `status.lastRerunToken`, so the
same token never triggers two audits.

### Artifacts

Each assessment publishes a ConfigMap named `<assessment>-artifact` with three keys:

| Key | Content |
| --- | --- |
| `assessment.md` | Human-readable assessment summary. |
| `plan.md` | Operator-grade upgrade plan: decision, blockers, remediation, go/no-go gates, chronology. |
| `assessment.json` | Machine-readable export for CI and dashboards. |

`assessment.json` carries `"exportVersion": "v1"`, the snapshot time as `takenAt`,
the assessment and plan references, source/target versions and profile, the
decision, risk level and score, the three summaries, the published findings with
their full classification and evidence, and the plan's `requiredActions`,
`upgradePath` and `recommendedOrder`. It is bounded exactly like the Markdown
artifacts so the ConfigMap stays under the 1 MiB object limit.

```sh
kubectl get configmap prod-upgrade-assessment-artifact \
  -o jsonpath='{.data.assessment\.json}' | jq '.decision, .score'
```

### Metrics

The manager exposes these on its existing `/metrics` endpoint:

| Metric | Type | Labels |
| --- | --- | --- |
| `guardian_assessment_score` | gauge | `namespace`, `name` |
| `guardian_assessment_findings` | gauge | `namespace`, `name`, `severity` |
| `guardian_checker_duration_seconds` | histogram | `checker` |
| `guardian_assessment_info` | gauge (always 1) | `namespace`, `name`, `decision`, `risk_level` |

Series are removed when the assessment is deleted.

The controller also emits Kubernetes events on the assessment: `Normal
AssessmentCompleted` with the decision and score, `Warning AssessmentDegraded`
and `Warning AssessmentFailed`.

### Testing

Required local validation:

```sh
make generate
make manifests
go test ./...
```

More detail is in [docs/testing.md](docs/testing.md), including optional `envtest` and `e2e` commands.

### Released artifacts

The current release is **0.1.5**. Image and chart are published to GHCR and share
the same version:

| Artifact | Reference |
| --- | --- |
| Image | `ghcr.io/ihsenalaya/kubeupgrade-guardian-operator:0.1.5` (also tagged `latest`) |
| Chart | `oci://ghcr.io/ihsenalaya/charts/kubeupgrade-guardian-operator` version `0.1.5` |

Install straight from the registry:

```sh
helm install kubeupgrade-guardian \
  oci://ghcr.io/ihsenalaya/charts/kubeupgrade-guardian-operator \
  --version 0.1.5 \
  --namespace kubeupgrade-guardian-system --create-namespace
```

The chart ships the CRDs under `crds/`, so Helm installs them on first release.
Helm does not upgrade CRDs on `helm upgrade`: when moving from an earlier version,
apply them first, otherwise the fields added in 0.1.5 (`spec.refreshInterval`,
`status.lastAssessedTime`, `status.lastRerunToken`) are silently dropped.

```sh
kubectl apply -f charts/kubeupgrade-guardian-operator/crds/
```

The chart packaged for release lives in `../../helm/kubeupgrade-guardian-operator`
(the copy carrying `service.yaml`, `servicemonitor.yaml` and `values.schema.json`).
The in-source copy under `charts/` is what the `release` workflow packages for
per-commit `0.1.0-<sha>` builds; both declare the same version and image tag, but
their templates have not been merged yet.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/kubeupgrade-guardian-operator:tag
```

**NOTE:** This image ought to be published in the personal registry you specified. 
And it is required to have access to pull the image from the working environment. 
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/kubeupgrade-guardian-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin 
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/kubeupgrade-guardian-operator:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/kubeupgrade-guardian-operator/<tag or branch>/dist/install.yaml
```

## Contributing
Keep changes assessment-only. A new checker reads the cluster through
`internal/snapshot` and never through a Kubernetes client - a test enforces this
by parsing the package imports. If a checker needs a resource type the snapshot
does not collect yet, add it to `snapshot.Collect` (one paginated List, RBAC
refusal recorded as a gap) and to the RBAC markers plus the Helm chart role.
Findings must carry observable evidence, RBAC denial stays
`RBAC_ASSESSMENT_GAP` rather than an error, and each new finding needs focused
tests on the finding and on the plan output.

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
