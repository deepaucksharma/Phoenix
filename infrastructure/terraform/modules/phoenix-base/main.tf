# Phoenix Base Infrastructure Module
# This module contains shared infrastructure components for Phoenix

terraform {
  required_version = ">= 1.5"
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.23"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.11"
    }
  }
}

# Variables
variable "namespace" {
  description = "Kubernetes namespace for Phoenix"
  type        = string
  default     = "phoenix-system"
}

variable "monitoring_namespace" {
  description = "Kubernetes namespace for monitoring components"
  type        = string
  default     = "phoenix-monitoring"
}

variable "phoenix_config" {
  description = "Phoenix configuration parameters"
  type = object({
    collector_replicas = number
    observer_replicas  = number
    resources = object({
      collector_cpu_limit    = string
      collector_memory_limit = string
      collector_cpu_request  = string
      collector_memory_request = string
    })
    image_tag = string
  })
  default = {
    collector_replicas = 2
    observer_replicas  = 1
    resources = {
      collector_cpu_limit      = "1000m"
      collector_memory_limit   = "2Gi"
      collector_cpu_request    = "500m"
      collector_memory_request = "1Gi"
    }
    image_tag = "0.103.1"
  }
}

variable "storage_class" {
  description = "Storage class for persistent volumes"
  type        = string
  default     = "standard"
}

variable "enable_monitoring" {
  description = "Enable Prometheus and Grafana monitoring"
  type        = bool
  default     = true
}

# Namespaces
resource "kubernetes_namespace" "phoenix_system" {
  metadata {
    name = var.namespace
    labels = {
      "app.kubernetes.io/name"    = "phoenix-vnext"
      "app.kubernetes.io/part-of" = "phoenix"
    }
  }
}

resource "kubernetes_namespace" "phoenix_monitoring" {
  count = var.enable_monitoring ? 1 : 0
  metadata {
    name = var.monitoring_namespace
    labels = {
      "app.kubernetes.io/name"    = "phoenix-monitoring"
      "app.kubernetes.io/part-of" = "phoenix"
    }
  }
}

# Service Accounts
resource "kubernetes_service_account" "phoenix_collector" {
  metadata {
    name      = "phoenix-collector"
    namespace = kubernetes_namespace.phoenix_system.metadata[0].name
    labels = {
      "app.kubernetes.io/name"      = "phoenix-collector"
      "app.kubernetes.io/component" = "collector"
    }
  }
}

resource "kubernetes_service_account" "phoenix_control" {
  metadata {
    name      = "phoenix-control"
    namespace = kubernetes_namespace.phoenix_system.metadata[0].name
    labels = {
      "app.kubernetes.io/name"      = "phoenix-control"
      "app.kubernetes.io/component" = "control-plane"
    }
  }
}

# ClusterRole for collector to read cluster metrics
resource "kubernetes_cluster_role" "phoenix_collector" {
  metadata {
    name = "phoenix-collector"
    labels = {
      "app.kubernetes.io/name" = "phoenix-collector"
    }
  }

  rule {
    api_groups = [""]
    resources = [
      "nodes",
      "nodes/metrics",
      "services",
      "endpoints",
      "pods",
      "events",
      "namespaces"
    ]
    verbs = ["get", "list", "watch"]
  }

  rule {
    api_groups = ["apps"]
    resources  = ["deployments", "daemonsets", "replicasets"]
    verbs      = ["get", "list", "watch"]
  }

  rule {
    api_groups = ["extensions"]
    resources  = ["deployments", "daemonsets", "replicasets"]
    verbs      = ["get", "list", "watch"]
  }
}

resource "kubernetes_cluster_role_binding" "phoenix_collector" {
  metadata {
    name = "phoenix-collector"
    labels = {
      "app.kubernetes.io/name" = "phoenix-collector"
    }
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = kubernetes_cluster_role.phoenix_collector.metadata[0].name
  }

  subject {
    kind      = "ServiceAccount"
    name      = kubernetes_service_account.phoenix_collector.metadata[0].name
    namespace = kubernetes_namespace.phoenix_system.metadata[0].name
  }
}

# ConfigMap for control configuration
resource "kubernetes_config_map" "phoenix_control_config" {
  metadata {
    name      = "phoenix-control-config"
    namespace = kubernetes_namespace.phoenix_system.metadata[0].name
    labels = {
      "app.kubernetes.io/name"      = "phoenix-control"
      "app.kubernetes.io/component" = "config"
    }
  }

  data = {
    "optimization_mode.yaml" = yamlencode({
      optimization_profile = "balanced"
      last_updated         = timestamp()
      trigger_reason       = "terraform_deployment"
      current_metrics = {
        full_ts          = 0
        optimized_ts     = 0
        experimental_ts  = 0
        cost_reduction_ratio = 0.0
      }
      config_version = 1
      correlation_id = "terraform-${formatdate("YYYYMMDD-hhmmss", timestamp())}"
      last_profile_change_timestamp = timestamp()
      thresholds = {
        conservative_max_ts = 15000
        aggressive_min_ts   = 25000
      }
      pipelines = {
        full_fidelity_enabled = true
        optimized_enabled     = true
        experimental_enabled  = false
      }
    })
  }
}

# Persistent Volume Claims for data
resource "kubernetes_persistent_volume_claim" "prometheus_data" {
  count = var.enable_monitoring ? 1 : 0
  metadata {
    name      = "prometheus-data"
    namespace = kubernetes_namespace.phoenix_monitoring[0].metadata[0].name
    labels = {
      "app.kubernetes.io/name" = "prometheus"
    }
  }

  spec {
    access_modes       = ["ReadWriteOnce"]
    storage_class_name = var.storage_class

    resources {
      requests = {
        storage = "50Gi"
      }
    }
  }
}

resource "kubernetes_persistent_volume_claim" "grafana_data" {
  count = var.enable_monitoring ? 1 : 0
  metadata {
    name      = "grafana-data"
    namespace = kubernetes_namespace.phoenix_monitoring[0].metadata[0].name
    labels = {
      "app.kubernetes.io/name" = "grafana"
    }
  }

  spec {
    access_modes       = ["ReadWriteOnce"]
    storage_class_name = var.storage_class

    resources {
      requests = {
        storage = "10Gi"
      }
    }
  }
}

# Outputs
output "namespace" {
  description = "Phoenix system namespace"
  value       = kubernetes_namespace.phoenix_system.metadata[0].name
}

output "monitoring_namespace" {
  description = "Phoenix monitoring namespace"
  value       = var.enable_monitoring ? kubernetes_namespace.phoenix_monitoring[0].metadata[0].name : null
}

output "collector_service_account" {
  description = "Phoenix collector service account"
  value       = kubernetes_service_account.phoenix_collector.metadata[0].name
}

output "control_service_account" {
  description = "Phoenix control service account"
  value       = kubernetes_service_account.phoenix_control.metadata[0].name
}

output "control_config_map" {
  description = "Phoenix control configuration ConfigMap"
  value       = kubernetes_config_map.phoenix_control_config.metadata[0].name
}