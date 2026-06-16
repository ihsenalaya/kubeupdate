#!/usr/bin/env bash
set -euo pipefail

terraform init
terraform apply -auto-approve
"$(dirname "$0")/push-gitops.sh"
"$(dirname "$0")/bootstrap-argocd.sh"
"$(dirname "$0")/post-apply-check.sh"
