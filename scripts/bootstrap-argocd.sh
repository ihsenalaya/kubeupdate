#!/usr/bin/env bash
set -euo pipefail

cluster_name="$(terraform output -raw cluster_name)"
resource_group_name="$(terraform output -raw resource_group_name)"

if [[ ! -d build/bootstrap ]]; then
  echo "Missing build/bootstrap; run terraform apply first."
  exit 1
fi

az aks command invoke \
  --resource-group "${resource_group_name}" \
  --name "${cluster_name}" \
  --file build/bootstrap \
  --command 'set -euo pipefail
cd /mnt/azscripts
if [[ -f bootstrap.sh ]]; then
  bash bootstrap.sh
elif [[ -f bootstrap/bootstrap.sh ]]; then
  bash bootstrap/bootstrap.sh
elif [[ -f build/bootstrap/bootstrap.sh ]]; then
  bash build/bootstrap/bootstrap.sh
else
  find /mnt/azscripts -maxdepth 4 -type f | sort
  exit 1
fi'
