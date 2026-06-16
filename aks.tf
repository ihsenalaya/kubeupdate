locals {
  node_resource_group_name = "mc-${var.resource_group_name}-${var.cluster_name}-${var.location}"
}

resource "azurerm_kubernetes_cluster" "platform" {
  name                = var.cluster_name
  location            = azurerm_resource_group.platform.location
  resource_group_name = azurerm_resource_group.platform.name
  dns_prefix          = var.cluster_name
  node_resource_group = local.node_resource_group_name

  kubernetes_version                  = var.kubernetes_version != "" ? var.kubernetes_version : null
  sku_tier                            = "Standard"
  support_plan                        = "KubernetesOfficial"
  role_based_access_control_enabled   = true
  oidc_issuer_enabled                 = true
  workload_identity_enabled           = true
  local_account_disabled              = false
  run_command_enabled                 = true
  azure_policy_enabled                = false
  cost_analysis_enabled               = false
  automatic_upgrade_channel           = "stable"
  node_os_upgrade_channel             = "NodeImage"
  private_cluster_enabled             = true
  private_cluster_public_fqdn_enabled = false
  private_dns_zone_id                 = "System"

  identity {
    type         = "UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.aks.id]
  }

  azure_active_directory_role_based_access_control {
    tenant_id          = local.tenant_id
    azure_rbac_enabled = true
  }

  default_node_pool {
    name                         = "system"
    type                         = "VirtualMachineScaleSets"
    vm_size                      = var.system_node_vm_size
    node_count                   = var.system_node_count
    os_sku                       = "AzureLinux3"
    os_disk_size_gb              = 128
    os_disk_type                 = "Managed"
    vnet_subnet_id               = azurerm_subnet.nodes.id
    max_pods                     = 110
    temporary_name_for_rotation  = "sysrot"
    only_critical_addons_enabled = false
    node_labels = {
      "nodepool"                    = "system"
      "platform.company.io/purpose" = "system"
    }

    upgrade_settings {
      max_surge = "33%"
    }
  }

  api_server_access_profile {
    virtual_network_integration_enabled = true
    subnet_id                           = azurerm_subnet.api_server.id
  }

  network_profile {
    network_plugin      = "azure"
    network_plugin_mode = "overlay"
    network_data_plane  = "cilium"
    network_policy      = "cilium"
    load_balancer_sku   = "standard"
    outbound_type       = "userAssignedNATGateway"
    pod_cidr            = var.pod_cidr
    service_cidr        = var.service_cidr
    dns_service_ip      = var.dns_service_ip
  }

  node_provisioning_profile {
    mode               = "Auto"
    default_node_pools = "Auto"
  }

  storage_profile {
    blob_driver_enabled         = true
    disk_driver_enabled         = true
    file_driver_enabled         = true
    snapshot_controller_enabled = true
  }

  key_vault_secrets_provider {
    secret_rotation_enabled  = true
    secret_rotation_interval = "2m"
  }

  tags = local.common_tags

  depends_on = [
    azurerm_subnet_nat_gateway_association.nodes,
    azurerm_role_assignment.aks_network_contributor_vnet,
    azurerm_role_assignment.aks_network_contributor_rg,
  ]
}

data "azurerm_resource_group" "aks_nodes" {
  name = azurerm_kubernetes_cluster.platform.node_resource_group
}

resource "azurerm_role_assignment" "velero_node_rg_snapshot_operator" {
  scope                            = data.azurerm_resource_group.aks_nodes.id
  role_definition_id               = azurerm_role_definition.velero_aks_snapshots.role_definition_resource_id
  principal_id                     = azurerm_user_assigned_identity.velero.principal_id
  skip_service_principal_aad_check = true
}

resource "azurerm_federated_identity_credential" "external_dns" {
  name                      = "fic-external-dns"
  user_assigned_identity_id = azurerm_user_assigned_identity.external_dns.id
  issuer                    = azurerm_kubernetes_cluster.platform.oidc_issuer_url
  subject                   = "system:serviceaccount:external-dns:external-dns"
  audience                  = ["api://AzureADTokenExchange"]
}

resource "azurerm_federated_identity_credential" "cert_manager" {
  name                      = "fic-cert-manager"
  user_assigned_identity_id = azurerm_user_assigned_identity.cert_manager.id
  issuer                    = azurerm_kubernetes_cluster.platform.oidc_issuer_url
  subject                   = "system:serviceaccount:cert-manager:cert-manager"
  audience                  = ["api://AzureADTokenExchange"]
}

resource "azurerm_federated_identity_credential" "external_secrets" {
  name                      = "fic-external-secrets"
  user_assigned_identity_id = azurerm_user_assigned_identity.external_secrets.id
  issuer                    = azurerm_kubernetes_cluster.platform.oidc_issuer_url
  subject                   = "system:serviceaccount:external-secrets:external-secrets"
  audience                  = ["api://AzureADTokenExchange"]
}

resource "azurerm_federated_identity_credential" "velero" {
  name                      = "fic-velero"
  user_assigned_identity_id = azurerm_user_assigned_identity.velero.id
  issuer                    = azurerm_kubernetes_cluster.platform.oidc_issuer_url
  subject                   = "system:serviceaccount:velero:velero"
  audience                  = ["api://AzureADTokenExchange"]
}
