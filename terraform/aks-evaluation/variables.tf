variable "location" {
  description = "Azure region used for the AKS evaluation clusters."
  type        = string
  default     = "westeurope"
}

variable "kubernetes_version" {
  description = "AKS Kubernetes version used by all evaluation clusters. Leave empty to use the AKS default."
  type        = string
  default     = ""
}

variable "tags" {
  description = "Extra tags merged with the required research/project tags."
  type        = map(string)
  default     = {}
}

variable "aks_small_name" {
  description = "Name of Cluster A: one-node smoke/regression AKS cluster."
  type        = string
  default     = "aks-small"
}

variable "aks_small_node_count" {
  description = "Node count for Cluster A."
  type        = number
  default     = 1
}

variable "aks_small_vm_size" {
  description = "VM size for Cluster A."
  type        = string
  default     = "Standard_B2s"
}

variable "aks_medium_name" {
  description = "Name of Cluster B: three-node readiness AKS cluster."
  type        = string
  default     = "aks-medium"
}

variable "aks_medium_node_count" {
  description = "Node count for Cluster B."
  type        = number
  default     = 3
}

variable "aks_medium_vm_size" {
  description = "VM size for Cluster B."
  type        = string
  default     = "Standard_D2s_v3"
}

variable "aks_policy_name" {
  description = "Name of Cluster C: policy-enabled AKS cluster."
  type        = string
  default     = "aks-policy"
}

variable "aks_policy_node_count" {
  description = "Node count for Cluster C."
  type        = number
  default     = 3
}

variable "aks_policy_vm_size" {
  description = "VM size for Cluster C."
  type        = string
  default     = "Standard_D2s_v3"
}

variable "kubeconfig_path_a" {
  description = "Local kubeconfig output path for Cluster A."
  type        = string
  default     = "~/.kube/config-aks-a"
}

variable "kubeconfig_path_b" {
  description = "Local kubeconfig output path for Cluster B."
  type        = string
  default     = "~/.kube/config-aks-b"
}

variable "kubeconfig_path_c" {
  description = "Local kubeconfig output path for Cluster C."
  type        = string
  default     = "~/.kube/config-aks-c"
}
