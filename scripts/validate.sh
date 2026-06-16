#!/usr/bin/env bash
set -euo pipefail

terraform fmt -check -recursive
terraform init -backend=false
terraform validate
helm lint gitops/charts/kubeupgrade-guardian-operator
helm lint gitops/charts/upgrade-lab
(cd apps/upgrade-lab/services/signals-service && go test ./...)
terraform plan -out=tfplan
