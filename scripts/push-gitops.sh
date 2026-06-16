#!/usr/bin/env bash
set -euo pipefail

repo_url="$(terraform output -raw gitops_repo_url)"
target_revision="$(terraform output -raw gitops_target_revision)"
gitops_file="gitops/argocd/platform.yaml"
gitops_checkout_dir="${GITOPS_CHECKOUT_DIR:-.local/gitops-repo}"

if [[ ! -f "${gitops_file}" ]]; then
  echo "Missing ${gitops_file}; run terraform apply first."
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

mkdir -p "${gitops_checkout_dir}/$(dirname "${gitops_file}")"
cp "${gitops_file}" "${gitops_checkout_dir}/${gitops_file}"

git -C "${gitops_checkout_dir}" add -f "${gitops_file}"

if git -C "${gitops_checkout_dir}" diff --cached --quiet -- "${gitops_file}"; then
  echo "GitOps manifest unchanged."
else
  git -C "${gitops_checkout_dir}" commit -m "Update rendered Argo CD platform apps" -- "${gitops_file}"
fi

git -C "${gitops_checkout_dir}" push -u origin "${target_revision}"
