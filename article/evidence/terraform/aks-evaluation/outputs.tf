output "kubeconfig_path_a" {
  description = "Path written for Cluster A kubeconfig."
  value       = local_sensitive_file.kubeconfig_a.filename
}

output "kubeconfig_path_b" {
  description = "Path written for Cluster B kubeconfig."
  value       = local_sensitive_file.kubeconfig_b.filename
}

output "kubeconfig_path_c" {
  description = "Path written for Cluster C kubeconfig."
  value       = local_sensitive_file.kubeconfig_c.filename
}

output "cluster_names" {
  description = "AKS cluster names created by this evaluation stack."
  value = {
    a = module.aks_small.name
    b = module.aks_medium.name
    c = module.aks_policy.name
  }
}
