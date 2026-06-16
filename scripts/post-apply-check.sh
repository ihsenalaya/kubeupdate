#!/usr/bin/env bash
set -euo pipefail

cluster_name="$(terraform output -raw cluster_name)"
resource_group_name="$(terraform output -raw resource_group_name)"
ingress_private_ip="$(terraform output -raw istio_private_ip)"
az_extension_dir="$(mktemp -d)"
trap 'rm -rf "${az_extension_dir}"' EXIT

remote_command="$(cat <<EOF
set -euo pipefail

expected_ingress_private_ip="${ingress_private_ip}"

apps=(
  kubeupdate-root
  cert-manager
  external-dns
  external-secrets
  istio-base
  istiod
  istio-ingress
  jaeger
  keda
  kubeupgrade-guardian-operator
  kubecost
  kyverno
  loki
  monitoring
  neuvector
  opentelemetry-collector
  platform-config
  promtail
  upgrade-lab
  velero
)

kubectl wait --for=condition=Ready nodes --all --timeout=10m
kubectl -n argocd wait --for=condition=Available deployment/argocd-server --timeout=10m

kubectl -n argocd annotate applications.argoproj.io --all argocd.argoproj.io/refresh=hard --overwrite >/dev/null || true

for app in "\${apps[@]}"; do
  for i in {1..120}; do
    if kubectl -n argocd get "application/\${app}" >/dev/null 2>&1; then
      break
    fi
    sleep 10
  done

  kubectl -n argocd get "application/\${app}" >/dev/null
  kubectl -n argocd wait --for=jsonpath='{.status.sync.status}'=Synced "application/\${app}" --timeout=25m
  kubectl -n argocd wait --for=jsonpath='{.status.health.status}'=Healthy "application/\${app}" --timeout=25m
done

for ns in cert-manager external-dns external-secrets istio-system istio-ingress monitoring loki tracing keda kyverno kubecost velero neuvector kubeupgrade-guardian-system upgrade-lab; do
  kubectl get namespace "\${ns}" >/dev/null
done

kubectl -n istio-system wait --for=condition=Available deployment/istiod --timeout=10m

if kubectl -n istio-ingress get pods -l app=istio-ingress -o jsonpath='{range .items[*]}{range .spec.containers[*]}{.image}{"\n"}{end}{end}' | grep -qx 'auto'; then
  kubectl -n istio-ingress delete pod -l app=istio-ingress --wait=false
fi

kubectl -n istio-ingress rollout status deployment/istio-ingress --timeout=10m
kubectl -n neuvector rollout status deployment/neuvector-controller-pod --timeout=10m
kubectl -n neuvector rollout status deployment/neuvector-manager-pod --timeout=10m
kubectl -n kubeupgrade-guardian-system rollout status deployment/kubeupgrade-guardian-kubeupgrade-guardian-operator --timeout=10m
kubectl -n upgrade-lab rollout status deployment/upgrade-lab-upgrade-lab-edge --timeout=10m
kubectl -n upgrade-lab rollout status deployment/upgrade-lab-upgrade-lab-catalog --timeout=10m
kubectl -n upgrade-lab rollout status deployment/upgrade-lab-upgrade-lab-orders --timeout=10m
kubectl -n upgrade-lab rollout status deployment/upgrade-lab-upgrade-lab-signals --timeout=10m

actual_ingress_ip="\$(kubectl -n istio-ingress get service istio-ingress -o jsonpath='{.status.loadBalancer.ingress[0].ip}')"
if [[ "\${actual_ingress_ip}" != "\${expected_ingress_private_ip}" ]]; then
  echo "Expected internal Istio IP \${expected_ingress_private_ip}, got \${actual_ingress_ip}"
  exit 1
fi

not_ready_pods="\$(kubectl get pods -A --field-selector=status.phase!=Running,status.phase!=Succeeded --no-headers 2>/dev/null || true)"
if [[ -n "\${not_ready_pods}" ]]; then
  echo "\${not_ready_pods}"
  exit 1
fi

kubectl -n argocd get applications.argoproj.io
kubectl -n istio-ingress get service istio-ingress
kubectl get clusterissuer letsencrypt-dns01
kubectl get clustersecretstore azure-keyvault
kubectl get clusterpolicy
kubectl get nodepool.karpenter.sh
kubectl -n monitoring get externalsecret grafana-admin
kubectl -n upgrade-lab get externalsecret upgrade-lab-secrets
kubectl -n upgrade-lab get virtualservice upgrade-lab
kubectl -n velero get backupstoragelocation
EOF
)"

AZURE_EXTENSION_DIR="${az_extension_dir}" az aks command invoke \
  --resource-group "${resource_group_name}" \
  --name "${cluster_name}" \
  --command "${remote_command}"
