#!/usr/bin/env bash
set -euo pipefail

cluster_name="$(terraform output -raw cluster_name 2>/dev/null || true)"
resource_group_name="$(terraform output -raw resource_group_name 2>/dev/null || true)"
jump_host_name="$(terraform output -raw jump_host_name 2>/dev/null || true)"
jump_host_admin_username="$(terraform output -raw jump_host_ssh_command 2>/dev/null | sed -E 's#^ssh ([^@]+)@.*#\1#' || true)"
remote_script="$(mktemp)"
trap 'rm -f "${remote_script}"' EXIT

if [[ -n "${cluster_name}" && -n "${resource_group_name}" && -n "${jump_host_name}" && -n "${jump_host_admin_username}" ]] &&
  az aks show --resource-group "${resource_group_name}" --name "${cluster_name}" >/dev/null 2>&1 &&
  az vm show --resource-group "${resource_group_name}" --name "${jump_host_name}" >/dev/null 2>&1; then
  cat >"${remote_script}" <<EOF
#!/usr/bin/env bash
set -euo pipefail

jump_host_admin_username="${jump_host_admin_username}"

/usr/local/bin/configure-aks-access >/dev/null
export KUBECONFIG="/home/\${jump_host_admin_username}/.kube/config"

if kubectl get namespace argocd >/dev/null 2>&1; then
  kubectl -n argocd delete applications.argoproj.io --all --ignore-not-found --wait=false

  for _ in \$(seq 1 240); do
    remaining_apps="\$(kubectl -n argocd get applications.argoproj.io --no-headers 2>/dev/null || true)"
    if [[ -z "\${remaining_apps}" ]]; then
      break
    fi

    kubectl -n monitoring patch externalsecret grafana-admin --type=merge -p '{"metadata":{"finalizers":[]}}' >/dev/null 2>&1 || true
    kubectl -n upgrade-lab patch externalsecret upgrade-lab-secrets --type=merge -p '{"metadata":{"finalizers":[]}}' >/dev/null 2>&1 || true
    sleep 5
  done

  remaining_apps="\$(kubectl -n argocd get applications.argoproj.io --no-headers 2>/dev/null || true)"
  if [[ -n "\${remaining_apps}" ]]; then
    echo "\${remaining_apps}"
    exit 1
  fi
fi
EOF

  az vm run-command invoke \
    --resource-group "${resource_group_name}" \
    --name "${jump_host_name}" \
    --command-id RunShellScript \
    --scripts @"${remote_script}" \
    --query "value[].message" \
    -o tsv
fi

terraform destroy -auto-approve
