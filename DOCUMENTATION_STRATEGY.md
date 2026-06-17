# Documentation Strategy

Date: 2026-06-17

Scope: user-facing documentation needed before broader publication of KubeUpgrade Guardian.

## Required User Docs

1. Installation
   - CRD installation with `kubectl apply -k config/crd`.
   - Helm chart installation from `charts/kubeupgrade-guardian-operator`.
   - Required Kubernetes version and RBAC permissions.

2. Configuration
   - Minimal `UpgradeAssessment` for one namespace.
   - Multi-namespace assessment with explicit include/exclude lists.
   - Check selection for deprecated APIs, readiness, PDB, admission webhooks, policy risks, capacity, and observability.
   - Recommended bounded-write RBAC profile.

3. Output Interpretation
   - Finding fields: type, severity, category, resource, evidence, recommendation.
   - Score and decision meanings.
   - `UpgradePlan` required action ordering.
   - Difference between scenario findings, provider observations, and RBAC assessment gaps.

4. Examples
   - Safe deployment with no findings.
   - Low-replica workload.
   - Blocking PDB.
   - Missing readiness probe.
   - Fail-closed webhook with missing service.
   - Restricted namespace with incompatible workload.
   - AKS managed validation example.

5. Operations
   - Cleanup commands.
   - How to archive assessment and plan artifacts.
   - How to restrict read access to generated plans.
   - Known limitations and non-goals.

## Existing Material To Reuse

- Operator README: `/mnt/c/Users/IhsenAlaya/Documents/ihsen/kubeupgrade-guardian-operator/README.md`
- Kind benchmark package: `experiments/kind/r01-benchmark`
- AKS validation package: `experiments/aks/r02-managed-validation`
- Expert review packet: `UPGRADEPLAN_EXPERT_REVIEW_PACKET.md`

## Acceptance Criteria

- A new user can install the CRDs and controller.
- A new user can run a namespace-scoped assessment.
- A new user can explain every field in a generated finding.
- A platform engineer can review an `UpgradePlan` without reading operator code.
- Documentation clearly states that the tool is bounded-write and non-executing.
