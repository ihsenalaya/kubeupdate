variable "name" {
  description = "AKS cluster name."
  type        = string
}

variable "location" {
  description = "Azure region."
  type        = string
}

variable "kubernetes_version" {
  description = "AKS Kubernetes version. Empty string uses provider default."
  type        = string
}

variable "node_count" {
  description = "Default node pool node count."
  type        = number
}

variable "auto_scaling_enabled" {
  description = "Enable cluster autoscaler for the default node pool."
  type        = bool
  default     = false
}

variable "min_count" {
  description = "Minimum node count when autoscaling is enabled."
  type        = number
  default     = null
}

variable "max_count" {
  description = "Maximum node count when autoscaling is enabled."
  type        = number
  default     = null
}

variable "vm_size" {
  description = "Default node pool VM size."
  type        = string
}

variable "azure_policy_enabled" {
  description = "Enable AKS Azure Policy add-on."
  type        = bool
  default     = false
}

variable "tags" {
  description = "Tags applied to Azure resources."
  type        = map(string)
}
