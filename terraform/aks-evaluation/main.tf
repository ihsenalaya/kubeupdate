terraform {
  required_version = ">= 1.6.0"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.117"
    }
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5"
    }
  }
}

provider "azurerm" {
  features {}
}

locals {
  tags = merge(
    {
      environment = "research"
      project     = "kubeupgrade-guardian"
    },
    var.tags
  )
}

module "aks_small" {
  source = "./modules/aks-cluster"

  name                 = var.aks_small_name
  location             = var.location
  kubernetes_version   = var.kubernetes_version
  node_count           = var.aks_small_node_count
  vm_size              = var.aks_small_vm_size
  azure_policy_enabled = false
  tags                 = local.tags
}

module "aks_medium" {
  source = "./modules/aks-cluster"

  name                 = var.aks_medium_name
  location             = var.location
  kubernetes_version   = var.kubernetes_version
  node_count           = var.aks_medium_node_count
  vm_size              = var.aks_medium_vm_size
  azure_policy_enabled = false
  tags                 = local.tags
}

module "aks_policy" {
  source = "./modules/aks-cluster"

  name                 = var.aks_policy_name
  location             = var.location
  kubernetes_version   = var.kubernetes_version
  node_count           = var.aks_policy_node_count
  vm_size              = var.aks_policy_vm_size
  azure_policy_enabled = true
  tags                 = local.tags
}

resource "local_sensitive_file" "kubeconfig_a" {
  content         = module.aks_small.kube_config_raw
  filename        = pathexpand(var.kubeconfig_path_a)
  file_permission = "0600"
}

resource "local_sensitive_file" "kubeconfig_b" {
  content         = module.aks_medium.kube_config_raw
  filename        = pathexpand(var.kubeconfig_path_b)
  file_permission = "0600"
}

resource "local_sensitive_file" "kubeconfig_c" {
  content         = module.aks_policy.kube_config_raw
  filename        = pathexpand(var.kubeconfig_path_c)
  file_permission = "0600"
}
