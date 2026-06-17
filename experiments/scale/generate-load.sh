#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "Usage: $0 <N> <namespace>" >&2
  exit 2
fi

N="$1"
NAMESPACE="$2"

if ! [[ "${N}" =~ ^[0-9]+$ ]] || [[ "${N}" -lt 1 ]]; then
  echo "N must be a positive integer" >&2
  exit 2
fi

kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

tmpfile="$(mktemp)"
trap 'rm -f "${tmpfile}"' EXIT

for i in $(seq 1 "${N}"); do
  name="$(printf 'load-deploy-%04d' "${i}")"
  cat >> "${tmpfile}" <<EOF
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${name}
  namespace: ${NAMESPACE}
  labels:
    app: ${name}
    scale-study: "true"
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ${name}
  template:
    metadata:
      labels:
        app: ${name}
        scale-study: "true"
    spec:
      containers:
        - name: app
          image: busybox:1.36.1
          command: ["/bin/sh", "-c", "sleep 365d"]
          readinessProbe:
            exec:
              command: ["/bin/sh", "-c", "true"]
          resources:
            requests:
              cpu: 5m
              memory: 16Mi
---
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: ${name}
  namespace: ${NAMESPACE}
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app: ${name}
EOF
done

kubectl apply -f "${tmpfile}"
