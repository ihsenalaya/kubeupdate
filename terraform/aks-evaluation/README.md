# AKS Evaluation Terraform

This stack defines three AKS clusters for KubeUpgrade Guardian evaluation:

- Cluster A, `aks-small`: one `Standard_B2s` node for smoke/regression checks.
- Cluster B, `aks-medium`: three `Standard_D2s_v3` nodes for multi-node readiness checks.
- Cluster C, `aks-policy`: three `Standard_D2s_v3` nodes with Azure Policy enabled. The AKS Azure Policy add-on deploys the managed policy enforcement path used for Gatekeeper-backed policy validation.

All clusters use tags `environment=research` and `project=kubeupgrade-guardian`.

## Commands

```bash
terraform -chdir=terraform/aks-evaluation init
terraform -chdir=terraform/aks-evaluation validate
terraform -chdir=terraform/aks-evaluation plan \
  -var='kubernetes_version=1.34.8'
terraform -chdir=terraform/aks-evaluation apply \
  -var='kubernetes_version=1.34.8'
```

Kubeconfigs are written by Terraform to:

- `~/.kube/config-aks-a`
- `~/.kube/config-aks-b`
- `~/.kube/config-aks-c`

Use one with:

```bash
KUBECONFIG=~/.kube/config-aks-b kubectl get nodes
```

## Destroy

```bash
terraform -chdir=terraform/aks-evaluation destroy \
  -var='kubernetes_version=1.34.8'
```

## Cost Note

This configuration creates seven total AKS nodes: one `Standard_B2s` node and six `Standard_D2s_v3` nodes, plus managed disks, load balancers, public IPs, and control-plane-related managed resources. Actual cost depends on Azure region, reserved capacity, disk type, network egress, and runtime duration. Always run `terraform plan` and check Azure cost management before applying.
