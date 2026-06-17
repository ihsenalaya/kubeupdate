# R07 Helm Kind Server Dry-Run Summary

- Run ID: `20260617T134937Z`
- R05 input run: `20260617T133841Z`
- Kind image: `kindest/node:v1.31.0`
- Scope: apply CRDs, then server-side dry-run non-CRD resources.

| Chart | CRDs | Resources | Success | Failed commands |
| --- | ---: | ---: | --- | --- |
| cert-manager | 6 | 46 | True |  |
| external-dns | 0 | 5 | True |  |
| ingress-nginx | 0 | 19 | True |  |
| kube-prometheus-stack | 10 | 119 | True |  |
| kyverno | 22 | 54 | True |  |

## Command Diagnostics
