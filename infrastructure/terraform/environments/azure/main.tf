terraform {
  required_version = ">= 1.3.0"
  
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.80"
    }
    azuread = {
      source  = "hashicorp/azuread"
      version = "~> 2.45"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.23"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.11"
    }
  }
  
  # Optional: Configure Azure Storage backend for state
  # backend "azurerm" {
  #   resource_group_name  = "phoenix-terraform-state"
  #   storage_account_name = "phoenixtfstate"
  #   container_name       = "tfstate"
  #   key                  = "azure/terraform.tfstate"
  # }
}

provider "azurerm" {
  features {
    resource_group {
      prevent_deletion_if_contains_resources = false
    }
  }
}

data "azurerm_subscription" "current" {}
data "azuread_client_config" "current" {}

# Resource Group
resource "azurerm_resource_group" "phoenix" {
  name     = "${var.project_name}-rg"
  location = var.azure_location

  tags = local.tags
}

# Virtual Network
resource "azurerm_virtual_network" "phoenix" {
  name                = "${var.project_name}-vnet"
  location            = azurerm_resource_group.phoenix.location
  resource_group_name = azurerm_resource_group.phoenix.name
  address_space       = [var.vnet_address_space]

  tags = local.tags
}

# Subnets
resource "azurerm_subnet" "aks" {
  name                 = "aks-subnet"
  resource_group_name  = azurerm_resource_group.phoenix.name
  virtual_network_name = azurerm_virtual_network.phoenix.name
  address_prefixes     = [var.aks_subnet_cidr]
}

resource "azurerm_subnet" "ingress" {
  name                 = "ingress-subnet"
  resource_group_name  = azurerm_resource_group.phoenix.name
  virtual_network_name = azurerm_virtual_network.phoenix.name
  address_prefixes     = [var.ingress_subnet_cidr]
}

# Network Security Group
resource "azurerm_network_security_group" "phoenix" {
  name                = "${var.project_name}-nsg"
  location            = azurerm_resource_group.phoenix.location
  resource_group_name = azurerm_resource_group.phoenix.name

  security_rule {
    name                       = "AllowHTTPS"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "443"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }

  security_rule {
    name                       = "AllowOTLP"
    priority                   = 110
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_ranges    = ["4317", "4318"]
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }

  tags = local.tags
}

resource "azurerm_subnet_network_security_group_association" "aks" {
  subnet_id                 = azurerm_subnet.aks.id
  network_security_group_id = azurerm_network_security_group.phoenix.id
}

# Log Analytics Workspace
resource "azurerm_log_analytics_workspace" "phoenix" {
  name                = "${var.project_name}-logs"
  location            = azurerm_resource_group.phoenix.location
  resource_group_name = azurerm_resource_group.phoenix.name
  sku                 = "PerGB2018"
  retention_in_days   = 30

  tags = local.tags
}

# AKS Cluster
resource "azurerm_kubernetes_cluster" "phoenix" {
  name                = var.cluster_name
  location            = azurerm_resource_group.phoenix.location
  resource_group_name = azurerm_resource_group.phoenix.name
  dns_prefix          = var.cluster_name
  kubernetes_version  = var.kubernetes_version

  default_node_pool {
    name                = "default"
    node_count          = var.node_count
    vm_size             = var.node_vm_size
    vnet_subnet_id      = azurerm_subnet.aks.id
    type                = "VirtualMachineScaleSets"
    enable_auto_scaling = true
    min_count           = var.node_min_count
    max_count           = var.node_max_count
    os_disk_size_gb     = 128
    os_disk_type        = "Managed"

    node_labels = {
      "workload" = "general"
    }

    tags = local.tags
  }

  # Additional node pool for monitoring
  dynamic "agent_pool_profile" {
    for_each = var.enable_monitoring_pool ? [1] : []
    content {
      name                = "monitoring"
      count               = 2
      vm_size             = "Standard_D4s_v3"
      vnet_subnet_id      = azurerm_subnet.aks.id
      type                = "VirtualMachineScaleSets"
      enable_auto_scaling = true
      min_count           = 2
      max_count           = 4
      os_disk_size_gb     = 256
      os_disk_type        = "Managed"

      node_labels = {
        "workload" = "monitoring"
      }

      node_taints = [
        "monitoring=true:NoSchedule"
      ]
    }
  }

  identity {
    type = "SystemAssigned"
  }

  network_profile {
    network_plugin    = "azure"
    network_policy    = "azure"
    load_balancer_sku = "standard"
    outbound_type     = "loadBalancer"
  }

  oms_agent {
    log_analytics_workspace_id = azurerm_log_analytics_workspace.phoenix.id
  }

  azure_active_directory_role_based_access_control {
    managed                = true
    azure_rbac_enabled     = true
    admin_group_object_ids = var.aks_admin_group_ids
  }

  auto_scaler_profile {
    scale_down_delay_after_add = "10m"
    scale_down_unneeded        = "10m"
  }

  tags = local.tags
}

# Additional node pool for monitoring workloads
resource "azurerm_kubernetes_cluster_node_pool" "monitoring" {
  name                  = "monitoring"
  kubernetes_cluster_id = azurerm_kubernetes_cluster.phoenix.id
  vm_size               = "Standard_D4s_v3"
  node_count            = 2
  vnet_subnet_id        = azurerm_subnet.aks.id
  
  enable_auto_scaling = true
  min_count           = 2
  max_count           = 4
  
  os_disk_size_gb = 256
  os_disk_type    = "Managed"

  node_labels = {
    "workload" = "monitoring"
  }

  node_taints = [
    "monitoring=true:NoSchedule"
  ]

  tags = local.tags
}

# Storage Account for Phoenix data
resource "azurerm_storage_account" "phoenix" {
  name                     = "${lower(replace(var.project_name, "-", ""))}data"
  resource_group_name      = azurerm_resource_group.phoenix.name
  location                 = azurerm_resource_group.phoenix.location
  account_tier             = "Standard"
  account_replication_type = "LRS"
  account_kind             = "StorageV2"

  blob_properties {
    versioning_enabled = true
  }

  tags = local.tags
}

# Storage containers
resource "azurerm_storage_container" "benchmarks" {
  name                  = "benchmarks"
  storage_account_name  = azurerm_storage_account.phoenix.name
  container_access_type = "private"
}

resource "azurerm_storage_container" "metrics" {
  name                  = "metrics"
  storage_account_name  = azurerm_storage_account.phoenix.name
  container_access_type = "private"
}

# Azure Files for persistent storage
resource "azurerm_storage_share" "phoenix" {
  name                 = "phoenix-data"
  storage_account_name = azurerm_storage_account.phoenix.name
  quota                = 100
}

# Role assignments for AKS
resource "azurerm_role_assignment" "aks_network" {
  scope                = azurerm_virtual_network.phoenix.id
  role_definition_name = "Network Contributor"
  principal_id         = azurerm_kubernetes_cluster.phoenix.identity[0].principal_id
}

resource "azurerm_role_assignment" "aks_storage" {
  scope                = azurerm_storage_account.phoenix.id
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = azurerm_kubernetes_cluster.phoenix.identity[0].principal_id
}

# Container Registry
resource "azurerm_container_registry" "phoenix" {
  name                = "${lower(replace(var.project_name, "-", ""))}registry"
  resource_group_name = azurerm_resource_group.phoenix.name
  location            = azurerm_resource_group.phoenix.location
  sku                 = "Standard"
  admin_enabled       = false

  tags = local.tags
}

# Grant AKS access to ACR
resource "azurerm_role_assignment" "aks_acr" {
  scope                = azurerm_container_registry.phoenix.id
  role_definition_name = "AcrPull"
  principal_id         = azurerm_kubernetes_cluster.phoenix.kubelet_identity[0].object_id
}

# Application Insights for monitoring
resource "azurerm_application_insights" "phoenix" {
  name                = "${var.project_name}-insights"
  location            = azurerm_resource_group.phoenix.location
  resource_group_name = azurerm_resource_group.phoenix.name
  workspace_id        = azurerm_log_analytics_workspace.phoenix.id
  application_type    = "web"

  tags = local.tags
}

# Configure kubectl
resource "null_resource" "kubectl_config" {
  depends_on = [azurerm_kubernetes_cluster.phoenix]

  provisioner "local-exec" {
    command = "az aks get-credentials --resource-group ${azurerm_resource_group.phoenix.name} --name ${azurerm_kubernetes_cluster.phoenix.name} --overwrite-existing"
  }
}

# Outputs
output "kube_config" {
  value     = azurerm_kubernetes_cluster.phoenix.kube_config_raw
  sensitive = true
}

output "acr_login_server" {
  value = azurerm_container_registry.phoenix.login_server
}

output "storage_account_name" {
  value = azurerm_storage_account.phoenix.name
}

output "application_insights_key" {
  value     = azurerm_application_insights.phoenix.instrumentation_key
  sensitive = true
}

# Locals
locals {
  tags = merge(var.tags, {
    Project     = var.project_name
    Environment = var.environment
    Cluster     = var.cluster_name
    ManagedBy   = "Terraform"
  })
}