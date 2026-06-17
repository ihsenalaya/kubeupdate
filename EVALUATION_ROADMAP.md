# Evaluation Roadmap

## Phase 1: Controlled Kind Benchmark

- 30-50 scenarios.
- Single-risk and multi-risk cases.
- Ground truth labels.
- Version-pinned Kubernetes targets.
- Output archived for all tools.
- Core evaluated families: deprecated APIs, PDB/eviction, workload availability/readiness, admission/policy risks.
- Secondary-only families until deeper modeling: capacity, observability, add-on/operator compatibility, managed-provider constraints.

## Phase 2: Baseline Comparison

- Pluto.
- kubent.
- Helm API migration context.
- Manual checklist.

## Phase 3: Metrics

- Precision.
- Recall.
- F1.
- False positives.
- False negatives.
- Risk-family coverage.
- Assessment runtime.
- Plan actionability.
- Scoring decision stability.
- Action-order rank correlation.

## Phase 3b: Scoring Sensitivity

- Scoring-A: current weights.
- Scoring-B: equal weights.
- Scoring-C: expert-calibrated weights.
- Scoring-D: blocker-first decision only.
- Report changed decisions and changed action ordering.

## Phase 4: AKS Validation

- Minimal AKS cluster. **Done:** `aks-kug-validation-we`, Kubernetes `1.34.8`, one `Standard_B2s` node, run `20260617T101556Z`.
- Platform-like AKS clusters. **Done for managed add-on execution:** `aks-medium` and `aks-policy`, Kubernetes `1.34.8`, three `Standard_D2s_v3` nodes each, archived under `experiments/aks/r03-aks-medium` and `experiments/aks/r04-aks-policy`.
- Add-ons: ingress-nginx, cert-manager, Prometheus stack, and Azure Policy on `aks-policy`. **Done:** `experiments/aks/multi-cluster-summary.json`.
- Record AKS version, node pools, node image, upgrade settings, surge configuration. **Partial:** node counts, Kubernetes versions, Helm releases, workload status, and completed assessment outputs recorded; actual surge-upgrade execution and node-image upgrade simulation still pending.

## Phase 4b: Resource And Scalability Profiling

- **Done for minimal AKS:** assessment wait duration, local controller peak RSS, scoped object inventory, and cluster-level API-server request-counter delta.
- **Done for multi-cluster AKS:** in-cluster pod measurements for `aks-medium` (13s / 10 MiB RSS) and `aks-policy` (171s / 71 MiB RSS), plus completed patched-controller exports.
- **Done for AKS scale boundary:** one run each at 100 / 1,000 / 5,000 / 10,000 synthetic Deployment/PDB pairs on `aks-medium`; 100 and 1,000 completed, 5,000 and 10,000 failed on Kubernetes request-size limits while writing `UpgradeAssessment` status.
- **Pending:** exact per-controller API attribution, repeated scale runs after result-storage redesign, realistic object mixes, and cache warm-up separation.

## Phase 4c: Adversarial Fixture Error Analysis

- **Done with Codex-authored control labels:** R10 has 20 fixtures, 20 expected-finding files, real-checker observations, and `results-summary.json`.
- **Measured:** 32 TP, 0 FP, 4 FN, precision 1.0000, recall 0.8889, F1 0.9412.
- **Open:** replace Codex-authored labels with independent human labels before claiming independent detection accuracy.

## Phase 5: Expert Review

- 2-3 Kubernetes/platform engineers.
- Rubric for correctness, actionability, prioritization, evidence support, remediation safety.
- Capture disagreements.
- **Prepared:** `UPGRADEPLAN_EXPERT_REVIEW_PACKET.md`.
- **Pending:** actual independent reviewer ratings. Do not claim expert validation until this is completed.

## Phase 5b: Results Section Required For Submission

- RQ1 Results: frequency of risk families in scenarios and validation clusters.
- RQ2 Results: KubeUpgrade Guardian precision/recall/F1 by risk family.
- RQ3 Results: comparison with Pluto, kubent, and manual checklist.
- RQ4 Results: expert-rated UpgradePlan actionability and evidence support.
- Error Analysis: false positives, false negatives, and root causes.

## Expected Comparison Table

| Tool | API | PDB | Probe | Webhook | Policy | Plan |
| --- | --- | --- | --- | --- | --- | --- |
| KubeUpgrade Guardian | measured | measured | measured | measured | measured | measured |
| Pluto | measured | measured | measured | measured | measured | measured |
| kubent | measured | measured | measured | measured | measured | measured |
| Manual checklist | measured | measured | measured | measured | measured | measured |

## Phase 6: Optional Production Or Pre-Production Study

- 3-5 anonymized clusters.
- Bounded-write only.
- No secrets.
- No workload payloads.
- Redaction rules required.
