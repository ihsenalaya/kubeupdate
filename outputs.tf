output "resource_group_name" {
  description = "Azure resource group containing the platform resources."
  value       = azurerm_resource_group.platform.name
}

output "cluster_name" {
  description = "AKS cluster name."
  value       = azurerm_kubernetes_cluster.platform.name
}

output "lab_database_location" {
  description = "Azure region used by the lab database PaaS services."
  value       = var.lab_database_location
}

output "node_resource_group_name" {
  description = "AKS managed node resource group."
  value       = local.node_resource_group_name
}

output "kubeconfig_command" {
  description = "Command to configure kubectl against this private AKS cluster from the jump host."
  value       = "az aks get-credentials --resource-group ${azurerm_resource_group.platform.name} --name ${azurerm_kubernetes_cluster.platform.name} --admin --overwrite-existing"
}

output "platform_hosts" {
  description = "Private dashboard hostnames exposed through the internal Istio Gateway."
  value       = local.platform_hosts
}

output "istio_private_ip" {
  description = "Static private IP used by the internal Istio ingress gateway."
  value       = var.ingress_private_ip
}

output "jump_host_public_ip" {
  description = "Public IP used for direct SSH to the Ubuntu jump host."
  value       = azurerm_public_ip.jump_host.ip_address
}

output "jump_host_private_ip" {
  description = "Private IP of the Ubuntu jump host."
  value       = azurerm_network_interface.jump_host.private_ip_address
}

output "jump_host_ssh_command" {
  description = "SSH command for the Ubuntu jump host."
  value       = "ssh ${var.jump_host_admin_username}@${azurerm_public_ip.jump_host.ip_address}"
}

output "jump_host_credentials_file" {
  description = "Local file containing the generated jump host username and password. This path is ignored by Git."
  value       = local_sensitive_file.jump_host_credentials.filename
}

output "key_vault_name" {
  description = "Key Vault used by External Secrets."
  value       = azurerm_key_vault.platform.name
}

output "velero_storage_account_name" {
  description = "Storage account used by Velero backup storage location."
  value       = azurerm_storage_account.velero.name
}

output "acr_name" {
  description = "Azure Container Registry name used for lab and operator images."
  value       = azapi_resource.container_registry.name
}

output "acr_login_server" {
  description = "Azure Container Registry login server."
  value       = azapi_resource.container_registry.output.properties.loginServer
}

output "artifact_tag" {
  description = "Image and chart version tag expected by GitOps."
  value       = var.artifact_tag
}

output "lab_host" {
  description = "Private hostname for the upgrade lab edge API."
  value       = local.platform_hosts.lab
}

output "lab_namespace" {
  description = "Namespace used by the upgrade lab application."
  value       = var.lab_namespace
}

output "operator_namespace" {
  description = "Namespace used by KubeUpgrade Guardian Operator."
  value       = var.operator_namespace
}

output "grafana_admin_user_secret" {
  description = "Key Vault secret name for Grafana admin username."
  value       = azurerm_key_vault_secret.grafana_admin_user.name
}

output "grafana_admin_password_command" {
  description = "Command to read the generated Grafana admin password from Key Vault."
  value       = "az keyvault secret show --vault-name ${azurerm_key_vault.platform.name} --name ${azurerm_key_vault_secret.grafana_admin_password.name} --query value -o tsv"
}

output "neuvector_admin_password_command" {
  description = "Command to read the generated NeuVector admin password from Key Vault."
  value       = "az keyvault secret show --vault-name ${azurerm_key_vault.platform.name} --name ${azurerm_key_vault_secret.neuvector_admin_password.name} --query value -o tsv"
}

output "argocd_initial_admin_password_command" {
  description = "Command to read the Argo CD initial admin password from the jump host."
  value       = "ssh ${var.jump_host_admin_username}@${azurerm_public_ip.jump_host.ip_address} \"kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d\""
}

output "gitops_repo_url" {
  description = "Git repository consumed by the Argo CD root application."
  value       = var.gitops_repo_url
}

output "gitops_target_revision" {
  description = "Git revision consumed by the Argo CD root application."
  value       = var.gitops_target_revision
}

output "gitops_root_application" {
  description = "Argo CD root application that syncs this repository."
  value       = "kubeupdate-root"
}
