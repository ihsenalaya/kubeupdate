#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
workspace_root="$(cd "${repo_root}/../.." && pwd)"

repo_url="$(terraform -chdir="${repo_root}" output -raw gitops_repo_url)"
target_revision="$(terraform -chdir="${repo_root}" output -raw gitops_target_revision)"
gitops_dir="gitops"
gitops_file="${repo_root}/gitops/argocd/platform.yaml"
gitops_checkout_dir="${GITOPS_CHECKOUT_DIR:-${repo_root}/.local/gitops-repo}"
operator_chart_dir="${OPERATOR_CHART_DIR:-${workspace_root}/operator/helm/kubeupgrade-guardian-operator}"

if [[ ! -f "${gitops_file}" ]]; then
  echo "Missing ${gitops_file}; run terraform apply first."
  exit 1
fi

if [[ ! -d "${operator_chart_dir}" ]]; then
  echo "Missing operator chart directory: ${operator_chart_dir}"
  exit 1
fi

if [[ -d "${gitops_checkout_dir}" && ! -d "${gitops_checkout_dir}/.git" ]]; then
  echo "${gitops_checkout_dir} exists but is not a git checkout."
  exit 1
fi

if [[ ! -d "${gitops_checkout_dir}/.git" ]]; then
  mkdir -p "$(dirname "${gitops_checkout_dir}")"
  git clone "${repo_url}" "${gitops_checkout_dir}"
fi

if git -C "${gitops_checkout_dir}" remote get-url origin >/dev/null 2>&1; then
  git -C "${gitops_checkout_dir}" remote set-url origin "${repo_url}"
else
  git -C "${gitops_checkout_dir}" remote add origin "${repo_url}"
fi

if [[ -n "$(git -C "${gitops_checkout_dir}" status --short)" ]]; then
  echo "${gitops_checkout_dir} has local changes; commit, stash, or remove them before pushing rendered GitOps manifests."
  git -C "${gitops_checkout_dir}" status --short
  exit 1
fi

git -C "${gitops_checkout_dir}" fetch origin

if git -C "${gitops_checkout_dir}" show-ref --verify --quiet "refs/heads/${target_revision}"; then
  git -C "${gitops_checkout_dir}" checkout "${target_revision}"
elif git -C "${gitops_checkout_dir}" show-ref --verify --quiet "refs/remotes/origin/${target_revision}"; then
  git -C "${gitops_checkout_dir}" checkout -b "${target_revision}" "origin/${target_revision}"
else
  git -C "${gitops_checkout_dir}" checkout --orphan "${target_revision}"
  git -C "${gitops_checkout_dir}" rm -rf . >/dev/null 2>&1 || true
fi

if git -C "${gitops_checkout_dir}" show-ref --verify --quiet "refs/remotes/origin/${target_revision}"; then
  git -C "${gitops_checkout_dir}" pull --ff-only origin "${target_revision}"
fi

mkdir -p "${gitops_checkout_dir}/${gitops_dir}"
rsync -a --delete "${repo_root}/${gitops_dir}/" "${gitops_checkout_dir}/${gitops_dir}/"
mkdir -p "${gitops_checkout_dir}/${gitops_dir}/charts/kubeupgrade-guardian-operator"
rsync -a --delete "${operator_chart_dir}/" "${gitops_checkout_dir}/${gitops_dir}/charts/kubeupgrade-guardian-operator/"

git -C "${gitops_checkout_dir}" add -f "${gitops_dir}"

if git -C "${gitops_checkout_dir}" diff --cached --quiet -- "${gitops_dir}"; then
  echo "GitOps content unchanged."
else
  git -C "${gitops_checkout_dir}" commit -m "Update rendered Argo CD platform apps" -- "${gitops_dir}"
fi

git -C "${gitops_checkout_dir}" push -u origin "${target_revision}"
