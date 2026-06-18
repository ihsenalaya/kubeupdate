#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKLOAD_DIR="$(cd "${SCRIPT_DIR}/../workloads" && pwd)"

helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx >/dev/null
helm repo add jetstack https://charts.jetstack.io >/dev/null
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts >/dev/null
helm repo update >/dev/null

helm upgrade --install ingress-nginx ingress-nginx/ingress-nginx \
  --version 4.10.0 \
  --namespace ingress-nginx \
  --create-namespace \
  --wait \
  --timeout 10m

helm upgrade --install cert-manager jetstack/cert-manager \
  --version v1.14.5 \
  --namespace cert-manager \
  --create-namespace \
  --set installCRDs=true \
  --wait \
  --timeout 10m

helm upgrade --install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
  --version 58.5.0 \
  --namespace monitoring \
  --create-namespace \
  --wait \
  --timeout 15m

kubectl apply -f "${WORKLOAD_DIR}/pdb-realistic.yaml"
kubectl apply -f "${WORKLOAD_DIR}/statefulset-with-pvc.yaml"

kubectl -n kug-real-workloads wait --for=condition=Available deployment \
  --selector=app.kubernetes.io/part-of=kug-real-workloads \
  --timeout=180s
