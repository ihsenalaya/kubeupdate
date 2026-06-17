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
- Platform-like AKS cluster.
- Add-ons: ingress-nginx, cert-manager, Prometheus, Kyverno or Gatekeeper, GitOps optional.
- Record AKS version, node pools, node image, upgrade settings, surge configuration. **Partial:** minimal cluster metadata and provider webhooks recorded; platform-like cluster still pending.

## Phase 4b: Resource And Scalability Profiling

- **Done for minimal AKS:** assessment wait duration, local controller peak RSS, scoped object inventory, and cluster-level API-server request-counter delta.
- **Pending:** exact per-controller API attribution, multi-size runs at 100 / 1,000 / 10,000 objects, and cache warm-up separation.

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
