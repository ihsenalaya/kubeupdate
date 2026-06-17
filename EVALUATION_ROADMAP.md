# Evaluation Roadmap

## Current Status

The current manuscript reports implementation fidelity and local artifact validation only. The next phase is to produce empirical evidence that can support a Q1 journal submission without overstating results.

## Benchmark Design

- Create 30 to 50 Kind scenarios.
- Pin Kubernetes versions, tool versions, manifests, and expected outputs.
- Store each scenario with manifests, target version, injected risk mechanism, expected findings, and ground-truth labels.
- Include single-risk and multi-risk scenarios.
- Archive raw outputs from KubeUpgrade Guardian, baselines, and helper scripts.

## Scenario Taxonomy

- API evolution and removed Kubernetes APIs.
- Low-replica workload availability.
- Missing readiness probes.
- Blocking or misleading PodDisruptionBudgets.
- Admission webhook and ValidatingAdmissionPolicy risks.
- Pod Security and policy-engine indicators.
- Capacity drain headroom.
- Observability gaps.
- RBAC-denied evidence gaps.
- Add-on or operator compatibility placeholders.
- Managed-provider readiness placeholders for AKS.

## Baselines

- Pluto for deprecated API detection.
- kubent for deprecated API detection.
- Helm deprecated API guidance and mapkubeapis for Helm-release migration context.
- Manual platform-engineering checklist for broad operational review.

## Metrics

- Precision by risk family.
- Recall by risk family.
- F1 by risk family.
- False positives and false negatives by risk family.
- Runtime per assessment.
- Coverage gap versus API-deprecation baselines.
- Expert-rated correctness, actionability, prioritization, evidence support, and remediation safety.
- For any future kagent layer: unsupported recommendation rate and evidence-grounding rate.

## AKS Validation Plan

- Validate on one minimal AKS cluster.
- Prefer a second platform-like AKS cluster with ingress, cert-manager, monitoring, policy engine, GitOps, and the operator installed.
- Record Kubernetes version, node image state, node pools, upgrade strategy, surge settings, add-ons, and relevant provider settings.
- Treat AKS validation as transferability evidence, not universal generalization.

## Expert Review Rubric

- Is the finding correct?
- Is the severity appropriate?
- Is the evidence sufficient and traceable?
- Is the required action actionable?
- Is the order of actions useful?
- Does the plan avoid unsafe automatic remediation?
- Which findings are false positives?
- Which expected risks were missed?

## Privacy And Anonymization Plan

- Do not collect secrets, workload payloads, tenant-identifying labels, registry credentials, or organization-specific names.
- Replace cluster, namespace, workload, image, and domain names with stable pseudonyms.
- Keep only fields needed for evidence, scoring, and reviewer replication.
- Document retained fields and excluded sensitive data classes.

## Required Scripts

- Scenario creation and cleanup scripts.
- Kind cluster setup scripts.
- KubeUpgrade Guardian install and run scripts.
- Pluto and kubent runner scripts.
- Output normalization scripts.
- Ground-truth comparison scripts.
- Metric calculation scripts.
- AKS evidence collection scripts.
- Expert-review export templates.

## Output Tables

- Benchmark scenario catalog.
- Per-risk-family confusion matrix.
- Precision, recall, F1, false-positive, and false-negative table.
- Baseline coverage comparison table.
- AKS validation environment table.
- Expert-review score table.
- Error taxonomy table.
- kagent ablation table, if the future layer is implemented.
