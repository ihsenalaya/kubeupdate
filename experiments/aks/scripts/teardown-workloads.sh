#!/usr/bin/env bash
set -euo pipefail

helm uninstall kube-prometheus-stack --namespace monitoring --ignore-not-found || true
helm uninstall cert-manager --namespace cert-manager --ignore-not-found || true
helm uninstall ingress-nginx --namespace ingress-nginx --ignore-not-found || true

kubectl delete namespace kug-real-workloads --ignore-not-found --wait=true
kubectl delete namespace monitoring --ignore-not-found --wait=true
kubectl delete namespace cert-manager --ignore-not-found --wait=true
kubectl delete namespace ingress-nginx --ignore-not-found --wait=true
