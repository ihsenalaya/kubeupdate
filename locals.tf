locals {
  subscription_id = data.azurerm_client_config.current.subscription_id
  tenant_id       = data.azurerm_client_config.current.tenant_id

  name_slug      = lower(replace(var.name_prefix, "/[^a-zA-Z0-9-]/", "-"))
  compact_suffix = substr(md5("${var.name_prefix}-${local.subscription_id}-${var.location}"), 0, 6)

  base_domain = "${var.platform_subdomain}.${var.dns_zone_name}"

  platform_hosts = {
    argocd    = "argocd.${local.base_domain}"
    grafana   = "grafana.${local.base_domain}"
    jaeger    = "jaeger.${local.base_domain}"
    kubecost  = "kubecost.${local.base_domain}"
    neuvector = "neuvector.${local.base_domain}"
    lab       = "lab.${local.base_domain}"
  }

  platform_host_list = [
    local.platform_hosts.argocd,
    local.platform_hosts.grafana,
    local.platform_hosts.jaeger,
    local.platform_hosts.kubecost,
    local.platform_hosts.neuvector,
    local.platform_hosts.lab,
  ]

  external_dns_azure_secret_name  = "external-dns-azure"
  neuvector_bootstrap_secret_name = "neuvector-bootstrap-secret"
  lab_secret_name                 = "upgrade-lab-secrets"
  lab_client_certificate_name     = "lab-client-certificate"

  chart_versions = {
    argocd                  = "9.5.21"
    argocd_apps             = "2.0.5"
    cert_manager            = "v1.20.2"
    external_dns            = "1.21.1"
    external_secrets        = "2.6.0"
    istio                   = "1.30.1"
    kube_prometheus_stack   = "86.2.3"
    loki                    = "7.0.0"
    promtail                = "6.17.1"
    jaeger                  = "4.11.1"
    opentelemetry_collector = "0.158.1"
    keda                    = "2.20.1"
    kyverno                 = "3.8.1"
    kyverno_policies        = "3.8.0"
    kubecost                = "2.8.6"
    velero                  = "12.0.3"
    neuvector               = "2.10.2"
    raw                     = "2.0.2"
  }

  lab_services = {
    edge = {
      image = "upgrade-lab/edge-api"
      port  = 3000
    }
    catalog = {
      image = "upgrade-lab/catalog-service"
      port  = 8000
    }
    orders = {
      image = "upgrade-lab/orders-service"
      port  = 8080
    }
    signals = {
      image = "upgrade-lab/signals-service"
      port  = 8090
    }
  }

  common_tags = merge(
    {
      environment = var.environment
      managed_by  = "terraform"
      platform    = "aks-open-source-mvp"
    },
    var.tags
  )
}
