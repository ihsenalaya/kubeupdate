# R05 Helm Realistic Workloads

This package renders a small corpus of public Helm charts that resemble common
platform add-ons used in managed Kubernetes clusters.

## Scope

The runner renders pinned chart versions for:

- ingress-nginx;
- cert-manager;
- external-dns;
- kube-prometheus-stack;
- kyverno.

It archives the rendered YAML, chart metadata, SHA-256 digests, and Kubernetes
resource inventory by `apiVersion`, `kind`, and namespace.

## Limits

This is a rendered-manifest corpus, not a live-cluster validation. It does not
prove that the add-ons become healthy in Kind or AKS, and it does not provide
independent ground truth labels. It is intended to reduce the gap between purely
hand-written fixtures and realistic public manifests.

## Run

From the repository root:

```bash
python3 experiments/helm/r05-realistic-workloads/render_helm_workloads.py
```

The runner writes a timestamped directory under `results/` with one manifest per
chart plus `metadata.json`, `inventory.json`, and `summary.md`.
