#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
workspace_root="$(cd "${script_dir}/../../.." && pwd)"

namespace="${OPERATOR_NAMESPACE:-kubeupgrade-guardian-system}"
assessment_name="${ASSESSMENT_NAME:-post-automation-assessment}"
target_version="${TARGET_VERSION:-1.35}"
profile="${ASSESSMENT_PROFILE:-production}"
output_root="${OUTPUT_ROOT:-${workspace_root}/article/operator-reports}"
timestamp="${REPORT_TIMESTAMP:-$(date -u +%Y%m%dT%H%M%SZ)}"
output_dir="${OUTPUT_DIR:-${output_root}/${timestamp}}"
raw_dir="${output_dir}/raw"

source_version="${SOURCE_VERSION:-}"
if [[ -z "${source_version}" ]]; then
  node_version="$(kubectl get nodes -o jsonpath='{.items[0].status.nodeInfo.kubeletVersion}' 2>/dev/null || true)"
  if [[ "${node_version}" =~ ^v?([0-9]+\.[0-9]+) ]]; then
    source_version="${BASH_REMATCH[1]}"
  fi
fi
if [[ -z "${source_version}" ]]; then
  source_version="1.34"
fi

if [[ -n "${ASSESSMENT_NAMESPACES:-}" ]]; then
  read -r -a candidate_namespaces <<<"${ASSESSMENT_NAMESPACES//,/ }"
else
  candidate_namespaces=(
    cert-manager
    external-secrets
    istio-ingress
    kubeupgrade-guardian-system
    kyverno
    monitoring
    neuvector
    upgrade-lab
  )
fi

include_namespaces=()
for candidate_namespace in "${candidate_namespaces[@]}"; do
  [[ -z "${candidate_namespace}" ]] && continue
  if kubectl get namespace "${candidate_namespace}" >/dev/null 2>&1; then
    include_namespaces+=("${candidate_namespace}")
  fi
done
if [[ "${#include_namespaces[@]}" -eq 0 ]]; then
  include_namespaces=("${namespace}")
fi

mkdir -p "${raw_dir}"

capture() {
  local file="$1"
  shift
  {
    printf '# command:'
    printf ' %q' "$@"
    printf '\n'
    "$@"
  } >"${file}" 2>&1
}

cat >"${raw_dir}/assessment.yaml" <<EOF
apiVersion: upgrade.guardian.io/v1alpha1
kind: UpgradeAssessment
metadata:
  name: ${assessment_name}
  namespace: ${namespace}
  labels:
    app.kubernetes.io/managed-by: operator-report-export
spec:
  sourceVersion: "${source_version}"
  targetVersion: "${target_version}"
  profile: ${profile}
  mode: ReadOnly
  scope:
    namespaces:
      exclude:
        - kube-system
      include:
EOF
for include_namespace in "${include_namespaces[@]}"; do
  printf '        - %s\n' "${include_namespace}" >>"${raw_dir}/assessment.yaml"
done
cat >>"${raw_dir}/assessment.yaml" <<EOF
  checks:
    deprecatedApis: true
    workloadAvailability: true
    pdb: true
    readinessProbes: true
    admissionWebhooks: true
    policyRisks: true
    capacity: true
    observability: true
EOF

kubectl get namespace "${namespace}" >/dev/null
kubectl apply -f "${raw_dir}/assessment.yaml" >"${raw_dir}/assessment-apply.txt"
kubectl -n "${namespace}" wait \
  --for=condition=AssessmentCompleted \
  "upgradeassessment/${assessment_name}" \
  --timeout="${ASSESSMENT_TIMEOUT:-10m}" >"${raw_dir}/assessment-wait.txt"

plan_name="$(kubectl -n "${namespace}" get "upgradeassessment/${assessment_name}" -o jsonpath='{.status.generatedPlanRef.name}')"
artifact_name="$(kubectl -n "${namespace}" get "upgradeassessment/${assessment_name}" -o jsonpath='{.status.artifactRef.name}')"

if [[ -z "${plan_name}" ]]; then
  plan_name="${assessment_name}-plan"
fi
if [[ -z "${artifact_name}" ]]; then
  artifact_name="${assessment_name}-artifact"
fi

for _ in $(seq 1 60); do
  if kubectl -n "${namespace}" get "configmap/${artifact_name}" >/dev/null 2>&1; then
    break
  fi
  sleep 5
done
kubectl -n "${namespace}" get "configmap/${artifact_name}" >/dev/null

kubectl -n "${namespace}" get "configmap/${artifact_name}" -o json >"${raw_dir}/artifact-configmap.json"
jq -r '.data["assessment.md"] // ""' "${raw_dir}/artifact-configmap.json" >"${output_dir}/assessment.md"
jq -r '.data["plan.md"] // ""' "${raw_dir}/artifact-configmap.json" >"${output_dir}/plan.md"

capture "${raw_dir}/upgradeassessment.json" kubectl -n "${namespace}" get "upgradeassessment/${assessment_name}" -o json
capture "${raw_dir}/upgradeassessment.yaml" kubectl -n "${namespace}" get "upgradeassessment/${assessment_name}" -o yaml
capture "${raw_dir}/upgradeplan.json" kubectl -n "${namespace}" get "upgradeplan/${plan_name}" -o json
capture "${raw_dir}/upgradeplan.yaml" kubectl -n "${namespace}" get "upgradeplan/${plan_name}" -o yaml
capture "${raw_dir}/artifact-configmap.yaml" kubectl -n "${namespace}" get "configmap/${artifact_name}" -o yaml
capture "${raw_dir}/nodes.txt" kubectl get nodes -o wide
capture "${raw_dir}/pods-not-ready.txt" kubectl get pods -A --field-selector=status.phase!=Running,status.phase!=Succeeded -o wide
capture "${raw_dir}/operator-workloads.txt" kubectl -n "${namespace}" get deploy,pod,svc,pdb -o wide
capture "${raw_dir}/upgradeassessments.txt" kubectl -n "${namespace}" get upgradeassessment -o wide
capture "${raw_dir}/upgradeplans.txt" kubectl -n "${namespace}" get upgradeplan -o wide

while read -r pod; do
  [[ -z "${pod}" ]] && continue
  capture "${raw_dir}/operator-logs-${pod}.txt" kubectl -n "${namespace}" logs "${pod}" --all-containers --tail=500
done < <(kubectl -n "${namespace}" get pods -l app.kubernetes.io/name=kubeupgrade-guardian-operator -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null || true)

cat >"${output_dir}/README.md" <<EOF
# Operator Report Export

- Generated UTC: ${timestamp}
- Namespace: ${namespace}
- Assessment: ${assessment_name}
- Plan: ${plan_name}
- Artifact ConfigMap: ${artifact_name}
- Source version: ${source_version}
- Target version: ${target_version}
- Profile: ${profile}

Human-readable entry points:

- assessment.md
- plan.md

Raw Kubernetes exports are stored under raw/.
EOF

printf '%s\n' "${output_dir}"
