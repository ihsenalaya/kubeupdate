#!/usr/bin/env bash
set -euo pipefail

cluster_name="$(terraform output -raw cluster_name 2>/dev/null || true)"
resource_group_name="$(terraform output -raw resource_group_name 2>/dev/null || true)"
az_extension_dir="$(mktemp -d)"
trap 'rm -rf "${az_extension_dir}"' EXIT

if [[ -n "${cluster_name}" && -n "${resource_group_name}" ]] && AZURE_EXTENSION_DIR="${az_extension_dir}" az aks show --resource-group "${resource_group_name}" --name "${cluster_name}" >/dev/null 2>&1; then
  AZURE_EXTENSION_DIR="${az_extension_dir}" az aks command invoke \
    --resource-group "${resource_group_name}" \
    --name "${cluster_name}" \
    --command 'set -euo pipefail
if kubectl get namespace argocd >/dev/null 2>&1; then
  kubectl -n argocd delete applications.argoproj.io --all --ignore-not-found --wait=false

  for _ in $(seq 1 240); do
    remaining_apps="$(kubectl -n argocd get applications.argoproj.io --no-headers 2>/dev/null || true)"
    if [[ -z "${remaining_apps}" ]]; then
      break
    fi

    kubectl -n monitoring patch externalsecret grafana-admin --type=merge -p "{\"metadata\":{\"finalizers\":[]}}" >/dev/null 2>&1 || true
    sleep 5
  done

  remaining_apps="$(kubectl -n argocd get applications.argoproj.io --no-headers 2>/dev/null || true)"
  if [[ -n "${remaining_apps}" ]]; then
    echo "${remaining_apps}"
    exit 1
  fi
fi'
fi

terraform destroy -auto-approve
