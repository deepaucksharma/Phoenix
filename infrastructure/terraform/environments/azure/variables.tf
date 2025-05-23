variable "azure_location" {
  description = "Azure region for deployment"
  type        = string
  default     = "East US"
}

variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
  default     = "dev"
}

variable "project_name" {
  description = "Project name"
  type        = string
  default     = "phoenix-vnext"
}

variable "cluster_name" {
  description = "AKS cluster name"
  type        = string
  default     = "phoenix-aks"
}

variable "kubernetes_version" {
  description = "Kubernetes version"
  type        = string
  default     = "1.28.3"
}

variable "vnet_address_space" {
  description = "Virtual network address space"
  type        = string
  default     = "10.0.0.0/16"
}

variable "aks_subnet_cidr" {
  description = "AKS subnet CIDR"
  type        = string
  default     = "10.0.1.0/24"
}

variable "ingress_subnet_cidr" {
  description = "Ingress subnet CIDR"
  type        = string
  default     = "10.0.2.0/24"
}

variable "node_vm_size" {
  description = "VM size for AKS nodes"
  type        = string
  default     = "Standard_D4s_v3"
}

variable "node_count" {
  description = "Initial node count"
  type        = number
  default     = 3
}

variable "node_min_count" {
  description = "Minimum node count for autoscaling"
  type        = number
  default     = 2
}

variable "node_max_count" {
  description = "Maximum node count for autoscaling"
  type        = number
  default     = 10
}

variable "enable_monitoring_pool" {
  description = "Enable dedicated monitoring node pool"
  type        = bool
  default     = true
}

variable "aks_admin_group_ids" {
  description = "Azure AD group IDs for AKS administrators"
  type        = list(string)
  default     = []
}

variable "tags" {
  description = "Additional tags for resources"
  type        = map(string)
  default     = {}
}