data "azurerm_client_config" "current" {}

data "azurerm_subscription" "current" {}

data "azurerm_dns_zone" "public" {
  name                = var.dns_zone_name
  resource_group_name = var.dns_zone_resource_group_name
}
