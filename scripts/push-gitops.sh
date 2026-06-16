#!/usr/bin/env bash
set -euo pipefail

repo_url="$(terraform output -raw gitops_repo_url)"
target_revision="$(terraform output -raw gitops_target_revision)"
gitops_file="gitops/argocd/platform.yaml"

if [[ ! -f "${gitops_file}" ]]; then
  echo "Missing ${gitops_file}; run terraform apply first."
  exit 1
fi

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "This directory must be a git repository before GitOps manifests can be pushed."
  exit 1
fi

remote_url="$(git remote get-url origin 2>/dev/null || true)"
if [[ -z "${remote_url}" ]]; then
  git remote add origin "${repo_url}"
elif [[ "${remote_url}" != "${repo_url}" ]]; then
  git remote set-url origin "${repo_url}"
fi

current_branch="$(git branch --show-current)"
if [[ -z "${current_branch}" ]]; then
  echo "Cannot push GitOps manifests from a detached HEAD."
  exit 1
fi

if [[ "${current_branch}" != "${target_revision}" ]]; then
  git branch -M "${target_revision}"
fi

git add "${gitops_file}"

if git diff --cached --quiet -- "${gitops_file}"; then
  echo "GitOps manifest unchanged."
else
  git commit -m "Update rendered Argo CD platform apps" -- "${gitops_file}"
fi

git push -u origin "${target_revision}"
