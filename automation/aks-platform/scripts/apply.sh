#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"

terraform -chdir="${repo_root}" init
terraform -chdir="${repo_root}" apply -auto-approve
"${script_dir}/publish-artifacts.sh"
"${script_dir}/push-gitops.sh"
"${script_dir}/bootstrap-argocd.sh"
"${script_dir}/post-apply-check.sh"
