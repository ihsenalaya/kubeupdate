# Evaluation Roadmap

## Phase 1: Controlled Kind Benchmark

- 30-50 scenarios.
- Single-risk and multi-risk cases.
- Ground truth labels.
- Version-pinned Kubernetes targets.
- Output archived for all tools.

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

## Phase 4: AKS Validation

- Minimal AKS cluster.
- Platform-like AKS cluster.
- Add-ons: ingress-nginx, cert-manager, Prometheus, Kyverno or Gatekeeper, GitOps optional.
- Record AKS version, node pools, node image, upgrade settings, surge configuration.

## Phase 5: Expert Review

- 2-3 Kubernetes/platform engineers.
- Rubric for correctness, actionability, prioritization, evidence support, remediation safety.
- Capture disagreements.

## Phase 6: Optional Production Or Pre-Production Study

- 3-5 anonymized clusters.
- Bounded-write only.
- No secrets.
- No workload payloads.
- Redaction rules required.
