#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
operator_dir="${OPERATOR_DIR:-$(cd "${repo_root}/.." && pwd)/kubeupgrade-guardian-operator}"

acr_name="$(terraform -chdir="${repo_root}" output -raw acr_name)"
acr_login_server="$(terraform -chdir="${repo_root}" output -raw acr_login_server)"
artifact_tag="$(terraform -chdir="${repo_root}" output -raw artifact_tag)"
resource_group_name="$(terraform -chdir="${repo_root}" output -raw resource_group_name)"
jump_host_name="$(terraform -chdir="${repo_root}" output -raw jump_host_name)"
storage_account_name="$(terraform -chdir="${repo_root}" output -raw velero_storage_account_name)"

if [[ ! -d "${operator_dir}" ]]; then
  echo "Missing operator directory: ${operator_dir}"
  exit 1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

bundle_dir="${tmpdir}/bundle"
bundle_file="${tmpdir}/artifact-sources.tgz"
blob_name="artifact-builds/source-${artifact_tag}-$(date +%s).tgz"
remote_script="${tmpdir}/publish-on-jump-host.sh"

mkdir -p "${bundle_dir}/kubeupdate" "${bundle_dir}/kubeupgrade-guardian-operator"
rsync -a \
  --exclude '.git' \
  --exclude '.terraform' \
  --exclude '.local' \
  --exclude 'build' \
  --exclude 'secrets' \
  --exclude 'tfplan' \
  "${repo_root}/" "${bundle_dir}/kubeupdate/"
rsync -a \
  --exclude '.git' \
  --exclude 'bin' \
  --exclude 'dist' \
  "${operator_dir}/" "${bundle_dir}/kubeupgrade-guardian-operator/"
tar -C "${bundle_dir}" -czf "${bundle_file}" kubeupdate kubeupgrade-guardian-operator

storage_account_key="$(az storage account keys list \
  --resource-group "${resource_group_name}" \
  --account-name "${storage_account_name}" \
  --query "[0].value" \
  -o tsv)"

az storage blob upload \
  --account-name "${storage_account_name}" \
  --account-key "${storage_account_key}" \
  --container-name velero \
  --name "${blob_name}" \
  --file "${bundle_file}" \
  --overwrite \
  --only-show-errors \
  --output none

cat >"${remote_script}" <<SCRIPT
#!/usr/bin/env bash
set -euo pipefail

export DEBIAN_FRONTEND=noninteractive
workdir="/tmp/upgrade-lab-publish"
archive="/tmp/upgrade-lab-sources.tgz"

if ! command -v docker >/dev/null 2>&1; then
  apt-get update
  apt-get install -y docker.io
fi
systemctl enable --now docker >/dev/null 2>&1 || true

if ! command -v helm >/dev/null 2>&1; then
  curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
fi

az login --identity --allow-no-subscriptions >/dev/null

login_to_acr() {
  for attempt in \$(seq 1 30); do
    if az acr login --name '${acr_name}' >/dev/null; then
      acr_token="\$(az acr login --name '${acr_name}' --expose-token --query accessToken -o tsv)"
      printf '%s' "\${acr_token}" | helm registry login '${acr_login_server}' \
        --username 00000000-0000-0000-0000-000000000000 \
        --password-stdin >/dev/null
      unset acr_token
      return 0
    fi
    sleep 10
  done
  echo "Unable to authenticate to ACR '${acr_name}' from jump host after retries." >&2
  return 1
}

for attempt in \$(seq 1 30); do
  if az storage blob download \
    --account-name '${storage_account_name}' \
    --container-name velero \
    --name '${blob_name}' \
    --file "\${archive}" \
    --auth-mode login \
    --overwrite \
    --only-show-errors \
    --output none; then
    break
  fi
  if [[ "\${attempt}" == "30" ]]; then
    exit 1
  fi
  sleep 10
done

rm -rf "\${workdir}"
mkdir -p "\${workdir}"
tar -C "\${workdir}" -xzf "\${archive}"

login_to_acr

build_and_push() {
  local context="\$1"
  local image="\$2"
  local full_image="${acr_login_server}/\${image}:${artifact_tag}"

  echo "Building \${full_image}"
  docker build -t "\${full_image}" "\${context}"
  docker push "\${full_image}"
}

build_and_push "\${workdir}/kubeupgrade-guardian-operator" "kubeupgrade-guardian-operator"
build_and_push "\${workdir}/kubeupdate/apps/upgrade-lab/services/edge-api" "upgrade-lab/edge-api"
build_and_push "\${workdir}/kubeupdate/apps/upgrade-lab/services/catalog-service" "upgrade-lab/catalog-service"
build_and_push "\${workdir}/kubeupdate/apps/upgrade-lab/services/orders-service" "upgrade-lab/orders-service"
build_and_push "\${workdir}/kubeupdate/apps/upgrade-lab/services/signals-service" "upgrade-lab/signals-service"

rm -rf "\${workdir}/chart-packages"
mkdir -p "\${workdir}/chart-packages"

for chart in kubeupgrade-guardian-operator upgrade-lab; do
  helm lint "\${workdir}/kubeupdate/gitops/charts/\${chart}"
  helm package "\${workdir}/kubeupdate/gitops/charts/\${chart}" \
    --version '${artifact_tag}' \
    --app-version '${artifact_tag}' \
    --destination "\${workdir}/chart-packages"
  helm push "\${workdir}/chart-packages/\${chart}-${artifact_tag}.tgz" "oci://${acr_login_server}/helm"
done
SCRIPT

az vm run-command invoke \
  --resource-group "${resource_group_name}" \
  --name "${jump_host_name}" \
  --command-id RunShellScript \
  --scripts @"${remote_script}" \
  --query "value[].message" \
  -o tsv
