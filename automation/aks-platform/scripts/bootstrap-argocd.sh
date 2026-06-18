#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"

cluster_name="$(terraform -chdir="${repo_root}" output -raw cluster_name)"
resource_group_name="$(terraform -chdir="${repo_root}" output -raw resource_group_name)"

bootstrap_dir="${repo_root}/build/bootstrap"
az_extension_dir="$(mktemp -d)"
trap 'rm -rf "${az_extension_dir}"' EXIT

if [[ ! -d "${bootstrap_dir}" ]]; then
  echo "Missing build/bootstrap; run terraform apply first."
  exit 1
fi

(
  cd "${bootstrap_dir}"

  AZURE_EXTENSION_DIR="${az_extension_dir}" az aks command invoke \
    --resource-group "${resource_group_name}" \
    --name "${cluster_name}" \
    --file . \
    --command 'bash -lc '\''
set -euo pipefail
cd /command-files
if [[ -f bootstrap.sh ]]; then
  bash bootstrap.sh
else
  find /command-files -maxdepth 4 -type f | sort
  exit 1
fi
'\'''
)
