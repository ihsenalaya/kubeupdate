variable "name_prefix" {
  description = "Short prefix used for Azure and Kubernetes resource names."
  type        = string
  default     = "ihsen-aks"
}

variable "environment" {
  description = "Environment tag used on Azure resources."
  type        = string
  default     = "mvp"
}

variable "location" {
  description = "Azure region."
  type        = string
  default     = "westeurope"
}

variable "lab_database_location" {
  description = "Azure region for lab PostgreSQL, MySQL, and Cosmos DB when database offers are restricted in the AKS region."
  type        = string
  default     = "francecentral"
}

variable "resource_group_name" {
  description = "Resource group created for the AKS platform."
  type        = string
  default     = "rg-ihsen-aks-mvp-we"
}

variable "cluster_name" {
  description = "AKS cluster name."
  type        = string
  default     = "aks-ihsen-mvp-we"
}

variable "kubernetes_version" {
  description = "AKS Kubernetes version. Empty lets AKS use the regional default."
  type        = string
  default     = ""
}

variable "system_node_vm_size" {
  description = "Initial single-node system pool VM size."
  type        = string
  default     = "Standard_D4ds_v5"
}

variable "system_node_count" {
  description = "Initial system node count."
  type        = number
  default     = 1
}

variable "dns_zone_name" {
  description = "Azure public DNS zone managed by ExternalDNS and cert-manager."
  type        = string
  default     = "ihsenalaya.xyz"
}

variable "dns_zone_resource_group_name" {
  description = "Resource group containing the Azure DNS zone."
  type        = string
  default     = "ihsen"
}

variable "platform_subdomain" {
  description = "Subdomain used for private platform dashboards."
  type        = string
  default     = "aks"
}

variable "letsencrypt_email" {
  description = "Email used for Let's Encrypt ACME registration."
  type        = string
  default     = "Ihsen.alaya@outlook.com"
}

variable "letsencrypt_server" {
  description = "ACME directory URL. Use staging for dry runs, production for trusted certificates."
  type        = string
  default     = "https://acme-v02.api.letsencrypt.org/directory"
}

variable "gitops_repo_url" {
  description = "Git repository consumed by the Argo CD root application."
  type        = string
  default     = "https://github.com/ihsenalaya/kubeupdate.git"
}

variable "gitops_target_revision" {
  description = "Git revision consumed by the Argo CD root application."
  type        = string
  default     = "main"
}

variable "gitops_path" {
  description = "Path inside the GitOps repository containing Argo CD manifests."
  type        = string
  default     = "gitops/argocd"
}

variable "vnet_address_space" {
  description = "AKS virtual network address space."
  type        = list(string)
  default     = ["10.42.0.0/16"]
}

variable "node_subnet_prefixes" {
  description = "AKS node subnet prefixes."
  type        = list(string)
  default     = ["10.42.0.0/22"]
}

variable "api_server_subnet_prefixes" {
  description = "AKS API server delegated subnet prefixes for node auto-provisioning with custom VNet."
  type        = list(string)
  default     = ["10.42.4.0/28"]
}

variable "jump_host_subnet_prefixes" {
  description = "Private subnet prefixes for the Ubuntu jump host."
  type        = list(string)
  default     = ["10.42.5.0/24"]
}

variable "private_endpoint_subnet_prefixes" {
  description = "Private endpoint subnet prefixes."
  type        = list(string)
  default     = ["10.42.6.0/24"]
}

variable "ingress_private_ip" {
  description = "Static private IP assigned to the internal Istio ingress load balancer."
  type        = string
  default     = "10.42.0.100"
}

variable "jump_host_vm_size" {
  description = "Ubuntu jump host VM size."
  type        = string
  default     = "Standard_B2s"
}

variable "jump_host_admin_username" {
  description = "Local username used to connect to the Ubuntu jump host through SSH."
  type        = string
  default     = "ihsenadmin"
}

variable "jump_host_ssh_allowed_cidrs" {
  description = "CIDR ranges allowed to connect to the Ubuntu jump host over SSH. Replace the default with your public IP /32 for production."
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "pod_cidr" {
  description = "Azure CNI overlay pod CIDR."
  type        = string
  default     = "192.168.0.0/16"
}

variable "service_cidr" {
  description = "Kubernetes service CIDR."
  type        = string
  default     = "10.43.0.0/16"
}

variable "dns_service_ip" {
  description = "Kubernetes DNS service IP."
  type        = string
  default     = "10.43.0.10"
}

variable "velero_backup_container_name" {
  description = "Blob container used by Velero backups."
  type        = string
  default     = "velero"
}

variable "artifact_tag" {
  description = "Immutable tag used for locally built lab and operator images."
  type        = string
  default     = "0.1.1"
}

variable "lab_namespace" {
  description = "Namespace used for the upgrade lab microservices."
  type        = string
  default     = "upgrade-lab"
}

variable "operator_namespace" {
  description = "Namespace used for KubeUpgrade Guardian Operator."
  type        = string
  default     = "kubeupgrade-guardian-system"
}

variable "tags" {
  description = "Additional Azure tags."
  type        = map(string)
  default     = {}
}
