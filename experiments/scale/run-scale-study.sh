#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

SIZES=(${SIZES:-100 1000 5000 10000})
RUNS="${RUNS:-10}"
TARGET_VERSION="${TARGET_VERSION:-1.35}"
OPERATOR_SELECTOR="${OPERATOR_SELECTOR:-app.kubernetes.io/name=kubeupgrade-guardian-operator}"
CONTROLLER_METRICS_URL="${CONTROLLER_METRICS_URL:-}"

rss_mib() {
  { kubectl top pod -A -l "${OPERATOR_SELECTOR}" --no-headers 2>/dev/null || true; } | awk '
    function to_mib(v) {
      if (v ~ /Gi$/) { sub(/Gi$/, "", v); return v * 1024 }
      if (v ~ /Mi$/) { sub(/Mi$/, "", v); return v + 0 }
      if (v ~ /Ki$/) { sub(/Ki$/, "", v); return v / 1024 }
      return 0
    }
    NF >= 4 { sum += to_mib($4) }
    END { if (sum == "") sum = 0; printf "%.3f", sum }
  '
}

controller_api_requests() {
  if [[ -z "${CONTROLLER_METRICS_URL}" ]]; then
    echo ""
    return
  fi
  curl -fsS "${CONTROLLER_METRICS_URL}" 2>/dev/null | awk '
    /^rest_client_requests_total/ && $NF ~ /^[0-9.]+$/ { sum += $NF }
    END { if (sum == "") print ""; else printf "%.0f", sum }
  '
}

wait_completed() {
  local namespace="$1"
  local name="$2"
  local timeout="$3"
  local start
  start="$(date +%s)"
  while (( "$(date +%s)" - start < timeout )); do
    phase="$(kubectl -n "${namespace}" get upgradeassessment "${name}" -o jsonpath='{.status.phase}' 2>/dev/null || true)"
    if [[ "${phase}" == "Completed" ]]; then
      return 0
    fi
    if [[ "${phase}" == "Failed" ]]; then
      return 1
    fi
    sleep 5
  done
  return 1
}

for size in "${SIZES[@]}"; do
  for run in $(seq 1 "${RUNS}"); do
    namespace="scale-test-${size}"
    run_dir="${SCRIPT_DIR}/r04-scale-${size}/run-${run}"
    mkdir -p "${run_dir}"

    errors=()
    "${SCRIPT_DIR}/generate-load.sh" "${size}" "${namespace}" > "${run_dir}/generate-load.log" 2>&1 || errors+=("generate-load failed")

    assessment="${run_dir}/assessment-scale.yaml"
    sed -e "s/__NAMESPACE__/${namespace}/g" -e "s/targetVersion: .*/targetVersion: \"${TARGET_VERSION}\"/" \
      "${SCRIPT_DIR}/assessment-scale.yaml" > "${assessment}"

    api_before="$(controller_api_requests)"
    timestamp_start="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    start_epoch="$(date +%s)"

    kubectl apply -f "${assessment}" > "${run_dir}/assessment-apply.log" 2>&1 || errors+=("assessment apply failed")
    completed=false
    if wait_completed "${namespace}" "scale-assessment" 600; then
      completed=true
    else
      errors+=("assessment did not complete")
    fi

    end_epoch="$(date +%s)"
    timestamp_end="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    duration_s=$((end_epoch - start_epoch))
    rss="$(rss_mib)"
    api_after="$(controller_api_requests)"
    api_delta="null"
    if [[ -n "${api_before}" && -n "${api_after}" ]]; then
      api_delta=$((api_after - api_before))
    fi

    kubectl -n "${namespace}" get upgradeassessment scale-assessment -o json > "${run_dir}/upgradeassessment.json" 2>/dev/null || true
    kubectl -n "${namespace}" delete upgradeassessment --all --ignore-not-found > "${run_dir}/assessment-delete.log" 2>&1 || true

    errors_json="$(printf '%s\n' "${errors[@]}" | python3 -c 'import json,sys; print(json.dumps([x.strip() for x in sys.stdin if x.strip()]))')"
    cat > "${run_dir}/run-${run}.json" <<EOF
{
  "size": ${size},
  "run": ${run},
  "timestamp_start": "${timestamp_start}",
  "timestamp_end": "${timestamp_end}",
  "duration_s": ${duration_s},
  "rss_mib": ${rss},
  "api_requests_delta": ${api_delta},
  "completed": ${completed},
  "errors": ${errors_json}
}
EOF
  done
done
