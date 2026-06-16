resource "local_sensitive_file" "jump_host_credentials" {
  filename             = "${path.module}/secrets/jump-host-credentials.txt"
  file_permission      = "0600"
  directory_permission = "0700"
  content              = <<-EOT
    host=${azurerm_public_ip.jump_host.ip_address}
    username=${var.jump_host_admin_username}
    password=${random_password.jump_host_admin.result}

    ssh ${var.jump_host_admin_username}@${azurerm_public_ip.jump_host.ip_address}
  EOT
}
