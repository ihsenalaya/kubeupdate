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

resource "azurerm_subnet" "private_endpoints" {
  name                              = "snet-private-endpoints"
  resource_group_name               = azurerm_resource_group.platform.name
  virtual_network_name              = azurerm_virtual_network.platform.name
  address_prefixes                  = var.private_endpoint_subnet_prefixes
  private_endpoint_network_policies = "Disabled"
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

resource "terraform_data" "container_registry" {
  input = {
    name                = "acr${replace(local.name_slug, "-", "")}${local.compact_suffix}"
    resource_group_name = azurerm_resource_group.platform.name
    location            = azurerm_resource_group.platform.location
  }

  provisioner "local-exec" {
    interpreter = ["/bin/bash", "-c"]
    command     = <<-EOT
      set -euo pipefail
      if ! az acr show --resource-group '${self.input.resource_group_name}' --name '${self.input.name}' >/dev/null 2>&1; then
        az acr create \
          --resource-group '${self.input.resource_group_name}' \
          --name '${self.input.name}' \
          --sku Premium \
          --admin-enabled false \
          --location '${self.input.location}' \
          --only-show-errors \
          --output none
      fi

      az acr update \
        --resource-group '${self.input.resource_group_name}' \
        --name '${self.input.name}' \
        --allow-trusted-services true \
        --default-action Allow \
        --only-show-errors \
        --output none
    EOT
  }

  provisioner "local-exec" {
    interpreter = ["/bin/bash", "-c"]
    when        = destroy
    command     = "az acr delete --resource-group '${self.input.resource_group_name}' --name '${self.input.name}' --yes --only-show-errors || true"
  }

  depends_on = [azurerm_resource_group.platform]
}

data "azurerm_container_registry" "platform" {
  name                = terraform_data.container_registry.input.name
  resource_group_name = azurerm_resource_group.platform.name

  depends_on = [terraform_data.container_registry]
}

resource "azurerm_private_dns_zone" "acr" {
  name                = "privatelink.azurecr.io"
  resource_group_name = azurerm_resource_group.platform.name
  tags                = local.common_tags
}

resource "azurerm_private_dns_zone_virtual_network_link" "acr" {
  name                  = "pdnslink-${local.name_slug}-acr-${local.compact_suffix}"
  resource_group_name   = azurerm_resource_group.platform.name
  private_dns_zone_name = azurerm_private_dns_zone.acr.name
  virtual_network_id    = azurerm_virtual_network.platform.id
  registration_enabled  = false
  tags                  = local.common_tags
}

resource "azurerm_private_endpoint" "acr" {
  name                = "pe-${local.name_slug}-acr-${local.compact_suffix}"
  location            = azurerm_resource_group.platform.location
  resource_group_name = azurerm_resource_group.platform.name
  subnet_id           = azurerm_subnet.private_endpoints.id
  tags                = local.common_tags

  private_service_connection {
    name                           = "psc-${local.name_slug}-acr-${local.compact_suffix}"
    private_connection_resource_id = data.azurerm_container_registry.platform.id
    is_manual_connection           = false
    subresource_names              = ["registry"]
  }

  private_dns_zone_group {
    name                 = "default"
    private_dns_zone_ids = [azurerm_private_dns_zone.acr.id]
  }
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

resource "azurerm_role_assignment" "current_aks_rbac_cluster_admin" {
  scope                = azurerm_kubernetes_cluster.platform.id
  role_definition_name = "Azure Kubernetes Service RBAC Cluster Admin"
  principal_id         = data.azurerm_client_config.current.object_id
}

resource "azurerm_role_assignment" "aks_acr_pull" {
  scope                            = data.azurerm_container_registry.platform.id
  role_definition_name             = "AcrPull"
  principal_id                     = azurerm_kubernetes_cluster.platform.kubelet_identity[0].object_id
  skip_service_principal_aad_check = true
}

resource "azurerm_role_assignment" "jump_host_acr_push" {
  scope                            = data.azurerm_container_registry.platform.id
  role_definition_name             = "AcrPush"
  principal_id                     = azurerm_linux_virtual_machine.jump_host.identity[0].principal_id
  skip_service_principal_aad_check = true
}

resource "azurerm_role_assignment" "jump_host_storage_blob_reader" {
  scope                            = azurerm_storage_account.velero.id
  role_definition_name             = "Storage Blob Data Reader"
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

resource "random_password" "lab_postgres_admin" {
  length  = 28
  special = false
}

resource "random_password" "lab_sql_admin" {
  length  = 28
  special = false
}

resource "azurerm_postgresql_flexible_server" "lab" {
  name                          = "psql-${local.name_slug}-${local.lab_database_compact_suffix}"
  resource_group_name           = azurerm_resource_group.platform.name
  location                      = var.lab_database_location
  version                       = "16"
  administrator_login           = "labadmin"
  administrator_password        = random_password.lab_postgres_admin.result
  sku_name                      = "B_Standard_B1ms"
  storage_mb                    = 32768
  public_network_access_enabled = true
  backup_retention_days         = 7
  tags                          = local.common_tags

  lifecycle {
    ignore_changes = [zone]
  }
}

resource "azurerm_postgresql_flexible_server_database" "catalog" {
  name      = "catalog"
  server_id = azurerm_postgresql_flexible_server.lab.id
  charset   = "UTF8"
  collation = "en_US.utf8"
}

resource "azurerm_postgresql_flexible_server_firewall_rule" "allow_aks_nat" {
  name             = "allow-aks-nat"
  server_id        = azurerm_postgresql_flexible_server.lab.id
  start_ip_address = azurerm_public_ip.nat.ip_address
  end_ip_address   = azurerm_public_ip.nat.ip_address
}

resource "azurerm_mssql_server" "lab" {
  name                          = "sql-${local.name_slug}-${local.lab_database_compact_suffix}"
  resource_group_name           = azurerm_resource_group.platform.name
  location                      = var.lab_database_location
  version                       = "12.0"
  administrator_login           = "labadmin"
  administrator_login_password  = random_password.lab_sql_admin.result
  minimum_tls_version           = "1.2"
  public_network_access_enabled = true
  tags                          = local.common_tags
}

resource "azurerm_mssql_database" "orders" {
  name           = "orders"
  server_id      = azurerm_mssql_server.lab.id
  sku_name       = "Basic"
  max_size_gb    = 2
  zone_redundant = false
  tags           = local.common_tags
}

resource "azurerm_mssql_firewall_rule" "allow_aks_nat" {
  name             = "allow-aks-nat"
  server_id        = azurerm_mssql_server.lab.id
  start_ip_address = azurerm_public_ip.nat.ip_address
  end_ip_address   = azurerm_public_ip.nat.ip_address
}

resource "azurerm_redis_cache" "lab" {
  name                          = "redis-${local.name_slug}-${local.compact_suffix}"
  location                      = azurerm_resource_group.platform.location
  resource_group_name           = azurerm_resource_group.platform.name
  capacity                      = 0
  family                        = "C"
  sku_name                      = "Basic"
  minimum_tls_version           = "1.2"
  public_network_access_enabled = true
  redis_version                 = "6"
  tags                          = local.common_tags
}

resource "azurerm_redis_firewall_rule" "allow_aks_nat" {
  name                = "allow_aks_nat"
  redis_cache_name    = azurerm_redis_cache.lab.name
  resource_group_name = azurerm_resource_group.platform.name
  start_ip            = azurerm_public_ip.nat.ip_address
  end_ip              = azurerm_public_ip.nat.ip_address
}

resource "azurerm_cosmosdb_account" "lab" {
  name                          = "cosmos-${local.name_slug}-${local.compact_suffix}"
  location                      = var.lab_database_location
  resource_group_name           = azurerm_resource_group.platform.name
  offer_type                    = "Standard"
  kind                          = "MongoDB"
  mongo_server_version          = "7.0"
  public_network_access_enabled = true
  ip_range_filter               = [azurerm_public_ip.nat.ip_address]
  tags                          = local.common_tags

  capabilities {
    name = "EnableMongo"
  }

  consistency_policy {
    consistency_level = "Session"
  }

  geo_location {
    location          = var.lab_database_location
    failover_priority = 0
    zone_redundant    = false
  }
}

resource "azurerm_cosmosdb_mongo_database" "signals" {
  name                = "signals"
  resource_group_name = azurerm_resource_group.platform.name
  account_name        = azurerm_cosmosdb_account.lab.name
  throughput          = 400
}

resource "azurerm_cosmosdb_mongo_collection" "events" {
  name                = "events"
  resource_group_name = azurerm_resource_group.platform.name
  account_name        = azurerm_cosmosdb_account.lab.name
  database_name       = azurerm_cosmosdb_mongo_database.signals.name
  shard_key           = "service"

  index {
    keys   = ["_id"]
    unique = true
  }
}

resource "azurerm_key_vault_certificate" "lab_client" {
  name         = local.lab_client_certificate_name
  key_vault_id = azurerm_key_vault.platform.id
  tags         = local.common_tags

  certificate_policy {
    issuer_parameters {
      name = "Self"
    }

    key_properties {
      exportable = true
      key_size   = 2048
      key_type   = "RSA"
      reuse_key  = false
    }

    lifetime_action {
      action {
        action_type = "AutoRenew"
      }
      trigger {
        days_before_expiry = 30
      }
    }

    secret_properties {
      content_type = "application/x-pkcs12"
    }

    x509_certificate_properties {
      subject            = "CN=upgrade-lab.internal"
      validity_in_months = 12
      key_usage          = ["digitalSignature", "keyEncipherment"]
      extended_key_usage = ["1.3.6.1.5.5.7.3.2"]
    }
  }

  depends_on = [time_sleep.wait_for_key_vault_rbac]
}

resource "azurerm_key_vault_secret" "lab_postgres_dsn" {
  name         = "lab-postgres-dsn"
  value        = "postgresql://labadmin:${urlencode(random_password.lab_postgres_admin.result)}@${azurerm_postgresql_flexible_server.lab.fqdn}:5432/${azurerm_postgresql_flexible_server_database.catalog.name}?sslmode=require"
  key_vault_id = azurerm_key_vault.platform.id
  tags         = local.common_tags

  depends_on = [time_sleep.wait_for_key_vault_rbac]
}

resource "azurerm_key_vault_secret" "lab_sqlserver_jdbc_url" {
  name         = "lab-sqlserver-jdbc-url"
  value        = "jdbc:sqlserver://${azurerm_mssql_server.lab.fully_qualified_domain_name}:1433;databaseName=${azurerm_mssql_database.orders.name};encrypt=true;trustServerCertificate=false;hostNameInCertificate=*.database.windows.net;loginTimeout=30;"
  key_vault_id = azurerm_key_vault.platform.id
  tags         = local.common_tags

  depends_on = [time_sleep.wait_for_key_vault_rbac]
}

resource "azurerm_key_vault_secret" "lab_sqlserver_username" {
  name         = "lab-sqlserver-username"
  value        = "labadmin"
  key_vault_id = azurerm_key_vault.platform.id
  tags         = local.common_tags

  depends_on = [time_sleep.wait_for_key_vault_rbac]
}

resource "azurerm_key_vault_secret" "lab_sqlserver_password" {
  name         = "lab-sqlserver-password"
  value        = random_password.lab_sql_admin.result
  key_vault_id = azurerm_key_vault.platform.id
  tags         = local.common_tags

  depends_on = [time_sleep.wait_for_key_vault_rbac]
}

resource "azurerm_key_vault_secret" "lab_redis_url" {
  name         = "lab-redis-url"
  value        = "rediss://:${urlencode(azurerm_redis_cache.lab.primary_access_key)}@${azurerm_redis_cache.lab.hostname}:${azurerm_redis_cache.lab.ssl_port}"
  key_vault_id = azurerm_key_vault.platform.id
  tags         = local.common_tags

  depends_on = [time_sleep.wait_for_key_vault_rbac]
}

resource "azurerm_key_vault_secret" "lab_cosmos_mongo_uri" {
  name         = "lab-cosmos-mongo-uri"
  value        = azurerm_cosmosdb_account.lab.primary_mongodb_connection_string
  key_vault_id = azurerm_key_vault.platform.id
  tags         = local.common_tags

  depends_on = [time_sleep.wait_for_key_vault_rbac]
}
