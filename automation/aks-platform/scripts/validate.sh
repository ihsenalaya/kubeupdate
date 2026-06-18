#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
workspace_root="$(cd "${repo_root}/../.." && pwd)"
operator_chart_dir="${workspace_root}/operator/helm/kubeupgrade-guardian-operator"

terraform -chdir="${repo_root}" fmt -check -recursive
terraform -chdir="${repo_root}" init -backend=false
terraform -chdir="${repo_root}" validate
helm lint "${operator_chart_dir}"
helm lint "${repo_root}/gitops/charts/upgrade-lab"
(cd "${repo_root}/apps/upgrade-lab/services/signals-service" && go test ./...)
terraform -chdir="${repo_root}" plan -out="${repo_root}/tfplan"
