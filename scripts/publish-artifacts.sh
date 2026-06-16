#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
operator_dir="${OPERATOR_DIR:-$(cd "${repo_root}/.." && pwd)/kubeupgrade-guardian-operator}"

acr_name="$(terraform -chdir="${repo_root}" output -raw acr_name)"
acr_login_server="$(terraform -chdir="${repo_root}" output -raw acr_login_server)"
artifact_tag="$(terraform -chdir="${repo_root}" output -raw artifact_tag)"

if [[ ! -d "${operator_dir}" ]]; then
  echo "Missing operator directory: ${operator_dir}"
  exit 1
fi

az acr login --name "${acr_name}"

build_and_push() {
  local context="$1"
  local image="$2"
  local full_image="${acr_login_server}/${image}:${artifact_tag}"

  echo "Building ${full_image}"
  docker build -t "${full_image}" "${context}"
  docker push "${full_image}"
}

build_and_push "${operator_dir}" "kubeupgrade-guardian-operator"
build_and_push "${repo_root}/apps/upgrade-lab/services/edge-api" "upgrade-lab/edge-api"
build_and_push "${repo_root}/apps/upgrade-lab/services/catalog-service" "upgrade-lab/catalog-service"
build_and_push "${repo_root}/apps/upgrade-lab/services/orders-service" "upgrade-lab/orders-service"
build_and_push "${repo_root}/apps/upgrade-lab/services/signals-service" "upgrade-lab/signals-service"

rm -rf "${repo_root}/build/charts"
mkdir -p "${repo_root}/build/charts"

for chart in kubeupgrade-guardian-operator upgrade-lab; do
  helm lint "${repo_root}/gitops/charts/${chart}"
  helm package "${repo_root}/gitops/charts/${chart}" \
    --version "${artifact_tag}" \
    --app-version "${artifact_tag}" \
    --destination "${repo_root}/build/charts"
  helm push "${repo_root}/build/charts/${chart}-${artifact_tag}.tgz" "oci://${acr_login_server}/helm"
done
