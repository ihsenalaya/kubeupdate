# Independent Labeling Instructions

This package prepares independent ground truth for KubeUpgrade Guardian. The labeler must not inspect checker implementation code before labeling.

## Risk Families

- `WorkloadAvailability`: low replica count, singleton controllers, unmanaged pods, or disruption-intolerant workload topology.
- `PDB`: PodDisruptionBudget selectors that match no pods, budgets that block voluntary disruption, ambiguous percentage semantics, or missing PDBs where disruption safety is expected.
- `AdmissionWebhook`: fail-closed webhooks, unavailable webhook services, overly broad webhook scope, invalid TLS material, or restrictive admission behavior likely to block recreated objects.
- `PolicyRisk`: Pod Security or policy-engine settings that can reject workload recreation, including enforce labels and incompatible pod templates.
- `Capacity`: resource requests or persistent-storage assumptions that can prevent rescheduling, node drain, or surge upgrade.
- `DeprecatedAPI`: Kubernetes API versions removed before the target version.
- `Observability`: missing readiness, telemetry, probe, or monitoring signals that make upgrade validation weak.
- `RBAC`: assessment cannot list/get/create the objects needed to establish readiness.

## Labeling Rules

1. Label every fixture in `fixtures/`.
2. Do not inspect the operator checker source before labeling.
3. Create one file under `expected-findings/` per fixture, named `expected-findings-<fixture-name>.yaml`.
4. Use `expected-findings-template.yaml` as the schema.
5. Use `negative_control: true` only when the fixture should produce zero findings.
6. For every finding, provide:
   - `family`
   - `type`
   - `severity`
   - `confidence`: `certain`, `likely`, or `uncertain`
   - `resource.kind`
   - `resource.name`
   - optional `resource.namespace`
   - optional `message_contains`
7. If the fixture is ambiguous, keep the finding and set `confidence: uncertain`; do not silently drop it.

## Required Human Work

`[HUMAIN]` A platform engineer or reviewer who did not implement the checkers must fill the `expected-findings/*.yaml` files. Codex only prepares fixtures, schema, and comparison tooling.
