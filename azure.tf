resource "azurerm_resource_group" "platform" {
  name     = var.resource_group_name
  location = var.location
  tags     = local.common_tags
}

resource "azurerm_virtual_network" "platform" {
  name                = "vnet-${local.name_slug}-${var.location}"
  location            = azurerm_resource_group.platform.location
  resource_group_name = azurerm_resource_group.platform.name
  address_space       = var.vnet_address_space
  tags                = local.common_tags
}

resource "azurerm_subnet" "nodes" {
  name                 = "snet-aks-nodes"
  resource_group_name  = azurerm_resource_group.platform.name
  virtual_network_name = azurerm_virtual_network.platform.name
  address_prefixes     = var.node_subnet_prefixes
}

resource "azurerm_subnet" "api_server" {
  name                 = "snet-aks-apiserver"
  resource_group_name  = azurerm_resource_group.platform.name
  virtual_network_name = azurerm_virtual_network.platform.name
  address_prefixes     = var.api_server_subnet_prefixes

  delegation {
    name = "aks-apiserver"

    service_delegation {
      name = "Microsoft.ContainerService/managedClusters"
      actions = [
        "Microsoft.Network/virtualNetworks/subnets/join/action",
      ]
    }
  }
}

resource "azurerm_subnet" "jump_host" {
  name                 = "snet-jump-host"
  resource_group_name  = azurerm_resource_group.platform.name
  virtual_network_name = azurerm_virtual_network.platform.name
  address_prefixes     = var.jump_host_subnet_prefixes
}

resource "azurerm_private_dns_zone" "platform" {
  name                = local.base_domain
  resource_group_name = azurerm_resource_group.platform.name
  tags                = local.common_tags
}

resource "azurerm_private_dns_zone_virtual_network_link" "platform" {
  name                  = "pdnslink-${local.name_slug}-${local.compact_suffix}"
  resource_group_name   = azurerm_resource_group.platform.name
  private_dns_zone_name = azurerm_private_dns_zone.platform.name
  virtual_network_id    = azurerm_virtual_network.platform.id
  registration_enabled  = false
  tags                  = local.common_tags
}

resource "azurerm_public_ip" "nat" {
  name                = "pip-${local.name_slug}-nat-${local.compact_suffix}"
  location            = azurerm_resource_group.platform.location
  resource_group_name = azurerm_resource_group.platform.name
  allocation_method   = "Static"
  sku                 = "Standard"
  tags                = local.common_tags
}

resource "azurerm_nat_gateway" "platform" {
  name                    = "nat-${local.name_slug}-${local.compact_suffix}"
  location                = azurerm_resource_group.platform.location
  resource_group_name     = azurerm_resource_group.platform.name
  sku_name                = "Standard"
  idle_timeout_in_minutes = 4
  tags                    = local.common_tags
}

resource "azurerm_nat_gateway_public_ip_association" "platform" {
  nat_gateway_id       = azurerm_nat_gateway.platform.id
  public_ip_address_id = azurerm_public_ip.nat.id
}

resource "azurerm_subnet_nat_gateway_association" "nodes" {
  subnet_id      = azurerm_subnet.nodes.id
  nat_gateway_id = azurerm_nat_gateway.platform.id
}

resource "azurerm_subnet_nat_gateway_association" "jump_host" {
  subnet_id      = azurerm_subnet.jump_host.id
  nat_gateway_id = azurerm_nat_gateway.platform.id
}

resource "azurerm_public_ip" "jump_host" {
  name                = "pip-${local.name_slug}-jump-${local.compact_suffix}"
  location            = azurerm_resource_group.platform.location
  resource_group_name = azurerm_resource_group.platform.name
  allocation_method   = "Static"
  sku                 = "Standard"
  tags                = local.common_tags
}

resource "azurerm_network_security_group" "jump_host" {
  name                = "nsg-${local.name_slug}-jump-${local.compact_suffix}"
  location            = azurerm_resource_group.platform.location
  resource_group_name = azurerm_resource_group.platform.name
  tags                = local.common_tags

  security_rule {
    name                       = "AllowSsh"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "22"
    source_address_prefixes    = var.jump_host_ssh_allowed_cidrs
    destination_address_prefix = "*"
  }
}

resource "azurerm_subnet_network_security_group_association" "jump_host" {
  subnet_id                 = azurerm_subnet.jump_host.id
  network_security_group_id = azurerm_network_security_group.jump_host.id
}

resource "azurerm_user_assigned_identity" "aks" {
  name                = "id-${local.name_slug}-aks"
  location            = azurerm_resource_group.platform.location
  resource_group_name = azurerm_resource_group.platform.name
  tags                = local.common_tags
}

resource "azurerm_user_assigned_identity" "external_dns" {
  name                = "id-${local.name_slug}-external-dns"
  location            = azurerm_resource_group.platform.location
  resource_group_name = azurerm_resource_group.platform.name
  tags                = local.common_tags
}

resource "azurerm_user_assigned_identity" "cert_manager" {
  name                = "id-${local.name_slug}-cert-manager"
  location            = azurerm_resource_group.platform.location
  resource_group_name = azurerm_resource_group.platform.name
  tags                = local.common_tags
}

resource "azurerm_user_assigned_identity" "external_secrets" {
  name                = "id-${local.name_slug}-external-secrets"
  location            = azurerm_resource_group.platform.location
  resource_group_name = azurerm_resource_group.platform.name
  tags                = local.common_tags
}

resource "azurerm_user_assigned_identity" "velero" {
  name                = "id-${local.name_slug}-velero"
  location            = azurerm_resource_group.platform.location
  resource_group_name = azurerm_resource_group.platform.name
  tags                = local.common_tags
}

resource "azurerm_role_assignment" "aks_network_contributor_vnet" {
  scope                            = azurerm_virtual_network.platform.id
  role_definition_name             = "Network Contributor"
  principal_id                     = azurerm_user_assigned_identity.aks.principal_id
  skip_service_principal_aad_check = true
}

resource "azurerm_role_assignment" "aks_network_contributor_rg" {
  scope                            = azurerm_resource_group.platform.id
  role_definition_name             = "Network Contributor"
  principal_id                     = azurerm_user_assigned_identity.aks.principal_id
  skip_service_principal_aad_check = true
}

resource "azurerm_role_assignment" "external_dns_private_zone_contributor" {
  scope                            = azurerm_private_dns_zone.platform.id
  role_definition_name             = "Private DNS Zone Contributor"
  principal_id                     = azurerm_user_assigned_identity.external_dns.principal_id
  skip_service_principal_aad_check = true
}

resource "azurerm_role_assignment" "external_dns_platform_rg_reader" {
  scope                            = azurerm_resource_group.platform.id
  role_definition_name             = "Reader"
  principal_id                     = azurerm_user_assigned_identity.external_dns.principal_id
  skip_service_principal_aad_check = true
}

resource "azurerm_role_assignment" "cert_manager_zone_contributor" {
  scope                            = data.azurerm_dns_zone.public.id
  role_definition_name             = "DNS Zone Contributor"
  principal_id                     = azurerm_user_assigned_identity.cert_manager.principal_id
  skip_service_principal_aad_check = true
}

resource "azurerm_key_vault" "platform" {
  name                          = "kv${replace(local.name_slug, "-", "")}${local.compact_suffix}"
  location                      = azurerm_resource_group.platform.location
  resource_group_name           = azurerm_resource_group.platform.name
  tenant_id                     = local.tenant_id
  sku_name                      = "standard"
  rbac_authorization_enabled    = true
  purge_protection_enabled      = true
  soft_delete_retention_days    = 7
  public_network_access_enabled = true
  tags                          = local.common_tags
}

resource "azurerm_role_assignment" "current_key_vault_admin" {
  scope                = azurerm_key_vault.platform.id
  role_definition_name = "Key Vault Administrator"
  principal_id         = data.azurerm_client_config.current.object_id
}

resource "azurerm_role_assignment" "external_secrets_key_vault_user" {
  scope                            = azurerm_key_vault.platform.id
  role_definition_name             = "Key Vault Secrets User"
  principal_id                     = azurerm_user_assigned_identity.external_secrets.principal_id
  skip_service_principal_aad_check = true
}

resource "time_sleep" "wait_for_key_vault_rbac" {
  depends_on = [
    azurerm_role_assignment.current_key_vault_admin,
    azurerm_role_assignment.external_secrets_key_vault_user,
  ]

  create_duration = "45s"
}

resource "random_password" "grafana_admin" {
  length           = 24
  special          = true
  override_special = "!#%*-_=+?"
}

resource "random_password" "neuvector_admin" {
  length           = 24
  special          = true
  override_special = "!#%*-_=+?"
}

resource "random_password" "jump_host_admin" {
  length           = 24
  special          = true
  override_special = "!#%*-_=+?"
}

resource "azurerm_key_vault_secret" "grafana_admin_user" {
  name         = "grafana-admin-user"
  value        = "admin"
  key_vault_id = azurerm_key_vault.platform.id
  tags         = local.common_tags

  depends_on = [time_sleep.wait_for_key_vault_rbac]
}

resource "azurerm_key_vault_secret" "grafana_admin_password" {
  name         = "grafana-admin-password"
  value        = random_password.grafana_admin.result
  key_vault_id = azurerm_key_vault.platform.id
  tags         = local.common_tags

  depends_on = [time_sleep.wait_for_key_vault_rbac]
}

resource "azurerm_key_vault_secret" "neuvector_admin_password" {
  name         = "neuvector-admin-password"
  value        = random_password.neuvector_admin.result
  key_vault_id = azurerm_key_vault.platform.id
  tags         = local.common_tags

  depends_on = [time_sleep.wait_for_key_vault_rbac]
}

resource "azurerm_key_vault_secret" "jump_host_admin_password" {
  name         = "jump-host-admin-password"
  value        = random_password.jump_host_admin.result
  key_vault_id = azurerm_key_vault.platform.id
  tags         = local.common_tags

  depends_on = [time_sleep.wait_for_key_vault_rbac]
}

resource "azurerm_network_interface" "jump_host" {
  name                = "nic-${local.name_slug}-jump-${local.compact_suffix}"
  location            = azurerm_resource_group.platform.location
  resource_group_name = azurerm_resource_group.platform.name
  tags                = local.common_tags

  ip_configuration {
    name                          = "private"
    subnet_id                     = azurerm_subnet.jump_host.id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = azurerm_public_ip.jump_host.id
  }
}

resource "azurerm_linux_virtual_machine" "jump_host" {
  name                            = "vm-${local.name_slug}-jump"
  location                        = azurerm_resource_group.platform.location
  resource_group_name             = azurerm_resource_group.platform.name
  size                            = var.jump_host_vm_size
  admin_username                  = var.jump_host_admin_username
  admin_password                  = random_password.jump_host_admin.result
  disable_password_authentication = false
  network_interface_ids           = [azurerm_network_interface.jump_host.id]
  custom_data = base64encode(templatefile("${path.module}/templates/jump-host-cloud-init.yaml.tftpl", {
    cluster_name        = var.cluster_name
    resource_group_name = azurerm_resource_group.platform.name
    gitops_repo_url     = var.gitops_repo_url
    gitops_revision     = var.gitops_target_revision
    admin_username      = var.jump_host_admin_username
  }))
  tags = local.common_tags

  identity {
    type = "SystemAssigned"
  }

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "Premium_LRS"
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "0001-com-ubuntu-server-jammy"
    sku       = "22_04-lts-gen2"
    version   = "latest"
  }

  depends_on = [
    azurerm_subnet_nat_gateway_association.jump_host,
    azurerm_subnet_network_security_group_association.jump_host,
  ]
}

resource "azurerm_role_assignment" "jump_host_aks_admin" {
  scope                            = azurerm_kubernetes_cluster.platform.id
  role_definition_name             = "Azure Kubernetes Service Cluster Admin Role"
  principal_id                     = azurerm_linux_virtual_machine.jump_host.identity[0].principal_id
  skip_service_principal_aad_check = true
}

resource "azurerm_virtual_machine_extension" "jump_host_configure_aks" {
  name                 = "configure-aks-access"
  virtual_machine_id   = azurerm_linux_virtual_machine.jump_host.id
  publisher            = "Microsoft.Azure.Extensions"
  type                 = "CustomScript"
  type_handler_version = "2.1"

  settings = jsonencode({
    commandToExecute = "bash -lc 'while [ ! -f /var/lib/cloud/instance/boot-finished ]; do sleep 10; done; /usr/local/bin/configure-aks-access'"
  })

  depends_on = [
    azurerm_role_assignment.jump_host_aks_admin,
  ]
}

resource "azurerm_storage_account" "velero" {
  name                            = "stvelero${local.compact_suffix}${substr(md5(local.subscription_id), 0, 5)}"
  resource_group_name             = azurerm_resource_group.platform.name
  location                        = azurerm_resource_group.platform.location
  account_tier                    = "Standard"
  account_replication_type        = "LRS"
  account_kind                    = "StorageV2"
  min_tls_version                 = "TLS1_2"
  allow_nested_items_to_be_public = false
  shared_access_key_enabled       = true
  tags                            = local.common_tags
}

resource "azurerm_storage_container" "velero" {
  name                  = var.velero_backup_container_name
  storage_account_id    = azurerm_storage_account.velero.id
  container_access_type = "private"
}

resource "azurerm_role_assignment" "velero_storage_blob_contributor" {
  scope                            = azurerm_storage_account.velero.id
  role_definition_name             = "Storage Blob Data Contributor"
  principal_id                     = azurerm_user_assigned_identity.velero.principal_id
  skip_service_principal_aad_check = true
}

resource "azurerm_role_assignment" "velero_storage_reader" {
  scope                            = azurerm_storage_account.velero.id
  role_definition_name             = "Reader"
  principal_id                     = azurerm_user_assigned_identity.velero.principal_id
  skip_service_principal_aad_check = true
}

resource "azurerm_role_definition" "velero_aks_snapshots" {
  name        = "Velero AKS Snapshot Operator ${local.compact_suffix}"
  scope       = data.azurerm_subscription.current.id
  description = "Minimum compute snapshot permissions for Velero on the AKS node resource group."

  permissions {
    actions = [
      "Microsoft.Compute/disks/read",
      "Microsoft.Compute/disks/write",
      "Microsoft.Compute/disks/endGetAccess/action",
      "Microsoft.Compute/disks/beginGetAccess/action",
      "Microsoft.Compute/snapshots/read",
      "Microsoft.Compute/snapshots/write",
      "Microsoft.Compute/snapshots/delete",
    ]
  }

  assignable_scopes = [
    data.azurerm_subscription.current.id,
  ]
}
