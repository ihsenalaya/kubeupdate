# R07 Helm Kind Server Dry-Run

This package validates the rendered Helm corpus from
`article/evidence/experiments/helm/r05-realistic-workloads` against a real Kubernetes API server.

## Scope

The runner creates a temporary Kind cluster, applies CRDs from the rendered Helm
corpus, waits for those CRDs to become established, creates target namespaces,
and runs `kubectl apply --dry-run=server` for the non-CRD resources in each
chart.

## Limits

This is not a live add-on installation or health test. Non-CRD resources are not
persisted, pods are not scheduled, and admission webhooks from the charts are not
made active. The result is a server-side schema/API acceptance smoke test for
the rendered corpus.

## Run

From the repository root:

```bash
python3 article/evidence/experiments/helm/r07-kind-server-dryrun/run_kind_server_dryrun.py \
  --r05-run-id 20260617T133841Z
```

The runner writes a timestamped directory under `results/` with command
outcomes and a Markdown summary.
