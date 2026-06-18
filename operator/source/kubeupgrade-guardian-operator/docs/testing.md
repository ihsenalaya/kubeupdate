# Testing

KubeUpgrade Guardian is assessment-only. The test suite must prove that the controller only reads cluster resources, writes `UpgradeAssessment` status, and creates or updates `UpgradePlan` objects with recommendations.

## Local Required Checks

Run these before every push:

```sh
make generate
make manifests
go test ./...
```

`go test ./...` is intentionally local by default. Scaffolded envtest and e2e suites are guarded by build tags so that the default command does not depend on a live cluster, deleted AKS credentials, or external kubeconfig state.

## Unit Coverage

Current unit tests cover:

- scoring weights, global risk boundaries, and decision ordering;
- `UpgradePlan` generation, action extraction, evidence references, and default recommended order;
- workload availability detection for replicas below two;
- readiness probe detection;
- PDB blocking detection for a single-replica workload with `minAvailable: 1`;
- admission webhook `failurePolicy: Fail` and missing Service detection;
- policy risk detection for restricted namespaces and privileged workloads;
- conservative one-node-loss capacity detection;
- observability gaps;
- RBAC-denied list calls producing `RBAC_ASSESSMENT_GAP`;
- controller idempotence so repeated reconciles keep a single generated plan.

## Optional Envtest

The Kubebuilder envtest suite is kept behind the `envtest` build tag:

```sh
go test -tags=envtest ./internal/controller
```

This requires compatible Kubernetes API server and etcd assets. Install or configure them with the Kubebuilder `setup-envtest` workflow before running the command.

## Optional E2E

The scaffolded e2e suite is behind the `e2e` build tag:

```sh
go test -tags=e2e ./test/e2e
```

Use this only with an explicit disposable test cluster. The operator still must not execute upgrades, node drains, workload patches, or destructive actions.
