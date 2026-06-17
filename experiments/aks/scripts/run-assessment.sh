#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

CLUSTER_NAME="${1:-$(kubectl config current-context)}"
TARGET_VERSION="${2:-1.35}"
ASSESSMENT_FILE="${3:-${ROOT_DIR}/experiments/aks/r03-aks-medium/assessment.yaml}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-300}"

if [[ ! -f "${ASSESSMENT_FILE}" ]]; then
  echo "assessment file not found: ${ASSESSMENT_FILE}" >&2
  exit 2
fi

TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
RESULT_DIR="${ROOT_DIR}/experiments/aks/results/${CLUSTER_NAME}/${TIMESTAMP}"
mkdir -p "${RESULT_DIR}"

RENDERED="${RESULT_DIR}/assessment-rendered.yaml"
sed -E "s/(targetVersion: ).*/\\1\"${TARGET_VERSION}\"/" "${ASSESSMENT_FILE}" > "${RENDERED}"

ASSESSMENT_NAME="$(awk '/^metadata:/{meta=1; next} meta && /^  name:/{print $2; exit}' "${RENDERED}")"
ASSESSMENT_NAMESPACE="$(awk '/^metadata:/{meta=1; next} meta && /^  namespace:/{print $2; exit}' "${RENDERED}")"

if [[ -z "${ASSESSMENT_NAME}" || -z "${ASSESSMENT_NAMESPACE}" ]]; then
  echo "assessment manifest must include metadata.name and metadata.namespace" >&2
  exit 2
fi

kubectl create namespace "${ASSESSMENT_NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

START_EPOCH="$(date +%s)"
kubectl apply -f "${RENDERED}"

deadline=$((START_EPOCH + TIMEOUT_SECONDS))
phase=""
while (( "$(date +%s)" < deadline )); do
  phase="$(kubectl -n "${ASSESSMENT_NAMESPACE}" get upgradeassessment "${ASSESSMENT_NAME}" \
    -o jsonpath='{.status.phase}' 2>/dev/null || true)"
  if [[ "${phase}" == "Completed" ]]; then
    break
  fi
  if [[ "${phase}" == "Failed" ]]; then
    break
  fi
  sleep 5
done
END_EPOCH="$(date +%s)"
DURATION_SECONDS=$((END_EPOCH - START_EPOCH))

kubectl -n "${ASSESSMENT_NAMESPACE}" get upgradeassessment "${ASSESSMENT_NAME}" -o json \
  > "${RESULT_DIR}/findings-${CLUSTER_NAME}-${TIMESTAMP}.json"

kubectl get upgradeplan -A -o json > "${RESULT_DIR}/upgradeplans-${CLUSTER_NAME}-${TIMESTAMP}.json" || true

kubectl top pod -A -l app.kubernetes.io/name=kubeupgrade-guardian-operator --no-headers \
  > "${RESULT_DIR}/controller-top.txt" 2> "${RESULT_DIR}/controller-top.stderr" || true

RSS_MIB="$(
  awk '
    function to_mib(v) {
      if (v ~ /Gi$/) { sub(/Gi$/, "", v); return v * 1024 }
      if (v ~ /Mi$/) { sub(/Mi$/, "", v); return v + 0 }
      if (v ~ /Ki$/) { sub(/Ki$/, "", v); return v / 1024 }
      return 0
    }
    NF >= 4 { sum += to_mib($4) }
    END { if (sum == "") sum = 0; printf "%.3f", sum }
  ' "${RESULT_DIR}/controller-top.txt"
)"

cat > "${RESULT_DIR}/run-metadata.json" <<EOF
{
  "cluster": "${CLUSTER_NAME}",
  "targetVersion": "${TARGET_VERSION}",
  "assessmentName": "${ASSESSMENT_NAME}",
  "assessmentNamespace": "${ASSESSMENT_NAMESPACE}",
  "phase": "${phase}",
  "duration_s": ${DURATION_SECONDS},
  "controller_rss_mib": ${RSS_MIB},
  "timestamp": "${TIMESTAMP}"
}
EOF

if [[ "${phase}" != "Completed" ]]; then
  echo "assessment did not complete: phase=${phase}, result=${RESULT_DIR}" >&2
  exit 1
fi

echo "${RESULT_DIR}"
