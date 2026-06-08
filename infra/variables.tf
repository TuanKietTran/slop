variable "kubeconfig_path" {
  description = "Path to the kubeconfig file"
  type        = string
  default     = "~/.kube/config"
}

variable "registry_password" {
  description = "Harbor registry admin password"
  type        = string
  sensitive   = true
}
