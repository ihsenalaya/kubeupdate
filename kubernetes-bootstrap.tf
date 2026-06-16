locals {
  namespaces = {
    argocd             = {}
    "cert-manager"     = {}
    "external-dns"     = {}
    "external-secrets" = {}
    "istio-system"     = {}
    "istio-ingress"    = {}
    monitoring         = {}
    loki               = {}
    tracing            = {}
    keda               = {}
    kyverno            = {}
    kubecost           = {}
    velero = {
      "pod-security.kubernetes.io/enforce" = "privileged"
      "pod-security.kubernetes.io/audit"   = "privileged"
      "pod-security.kubernetes.io/warn"    = "privileged"
    }
    neuvector = {
      "pod-security.kubernetes.io/enforce" = "privileged"
      "pod-security.kubernetes.io/audit"   = "privileged"
      "pod-security.kubernetes.io/warn"    = "privileged"
    }
  }

  rendered_namespace_documents = [
    for name, labels in local.namespaces : {
      apiVersion = "v1"
      kind       = "Namespace"
      metadata = merge(
        {
          name = name
        },
        length(labels) > 0 ? { labels = labels } : {}
      )
    }
  ]

  rendered_bootstrap_secret_documents = [
    {
      apiVersion = "v1"
      kind       = "Secret"
      metadata = {
        name      = local.external_dns_azure_secret_name
        namespace = "external-dns"
      }
      type = "Opaque"
      stringData = {
        "azure.json" = jsonencode({
          tenantId                     = local.tenant_id
          subscriptionId               = local.subscription_id
          resourceGroup                = azurerm_resource_group.platform.name
          useWorkloadIdentityExtension = true
        })
      }
    },
    {
      apiVersion = "v1"
      kind       = "Secret"
      metadata = {
        name      = local.neuvector_bootstrap_secret_name
        namespace = "neuvector"
      }
      type = "Opaque"
      stringData = {
        bootstrapPassword = random_password.neuvector_admin.result
      }
    }
  ]
}

resource "local_file" "bootstrap_namespaces" {
  filename             = "${path.module}/build/bootstrap/namespaces.yaml"
  content              = join("\n---\n", [for document in local.rendered_namespace_documents : yamlencode(document)])
  file_permission      = "0644"
  directory_permission = "0755"
}

resource "local_sensitive_file" "bootstrap_secrets" {
  filename             = "${path.module}/build/bootstrap/bootstrap-secrets.yaml"
  content              = join("\n---\n", [for document in local.rendered_bootstrap_secret_documents : yamlencode(document)])
  file_permission      = "0600"
  directory_permission = "0700"
}

resource "local_file" "bootstrap_argocd_values" {
  filename             = "${path.module}/build/bootstrap/argocd-values.yaml"
  content              = yamlencode(local.argocd_values)
  file_permission      = "0644"
  directory_permission = "0755"
}

resource "local_file" "bootstrap_argocd_apps_values" {
  filename             = "${path.module}/build/bootstrap/argocd-apps-values.yaml"
  content              = yamlencode(local.argocd_apps_values)
  file_permission      = "0644"
  directory_permission = "0755"
}

resource "local_file" "bootstrap_script" {
  filename             = "${path.module}/build/bootstrap/bootstrap.sh"
  file_permission      = "0755"
  directory_permission = "0755"
  content              = <<-EOT
    #!/usr/bin/env bash
    set -euo pipefail

    cd "$(dirname "$0")"

    kubectl apply -f namespaces.yaml
    kubectl apply -f bootstrap-secrets.yaml

    helm repo add argo https://argoproj.github.io/argo-helm >/dev/null 2>&1 || true
    helm repo update

    helm upgrade --install argocd argo/argo-cd \
      --namespace argocd \
      --version ${local.chart_versions.argocd} \
      --values argocd-values.yaml \
      --wait \
      --timeout 15m

    helm upgrade --install argocd-apps argo/argocd-apps \
      --namespace argocd \
      --version ${local.chart_versions.argocd_apps} \
      --values argocd-apps-values.yaml \
      --wait \
      --timeout 10m

    kubectl -n argocd annotate application kubeupdate-root argocd.argoproj.io/refresh=hard --overwrite >/dev/null 2>&1 || true
  EOT
}
