# AWS Phoenix Environment
# This deploys Phoenix-vNext on AWS EKS

terraform {
  required_version = ">= 1.5"
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
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

  # Uncomment and configure for remote state
  # backend "s3" {
  #   bucket  = "phoenix-terraform-state"
  #   key     = "aws/terraform.tfstate"
  #   region  = "us-west-2"
  #   encrypt = true
  # }
}

# Configure providers
provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "Phoenix-vNext"
      Environment = var.environment
      ManagedBy   = "Terraform"
    }
  }
}

provider "kubernetes" {
  host                   = module.aws_phoenix.cluster_endpoint
  cluster_ca_certificate = base64decode(module.aws_phoenix.cluster_certificate_authority_data)

  exec {
    api_version = "client.authentication.k8s.io/v1beta1"
    command     = "aws"
    args        = ["eks", "get-token", "--cluster-name", module.aws_phoenix.cluster_name]
  }
}

provider "helm" {
  kubernetes {
    host                   = module.aws_phoenix.cluster_endpoint
    cluster_ca_certificate = base64decode(module.aws_phoenix.cluster_certificate_authority_data)

    exec {
      api_version = "client.authentication.k8s.io/v1beta1"
      command     = "aws"
      args        = ["eks", "get-token", "--cluster-name", module.aws_phoenix.cluster_name]
    }
  }
}

# Local values
locals {
  cluster_name = "${var.project_name}-${var.environment}-eks"
  
  phoenix_config = {
    ecr_repository_prefix = var.project_name
    image_tag            = var.phoenix_image_tag
    enable_new_relic     = var.enable_new_relic
    storage_class        = "gp3-csi"
  }

  base_phoenix_config = {
    collector_replicas = var.environment == "production" ? 3 : 2
    observer_replicas  = 1
    resources = {
      collector_cpu_limit      = var.environment == "production" ? "2000m" : "1000m"
      collector_memory_limit   = var.environment == "production" ? "4Gi" : "2Gi"
      collector_cpu_request    = var.environment == "production" ? "1000m" : "500m"
      collector_memory_request = var.environment == "production" ? "2Gi" : "1Gi"
    }
    image_tag = var.phoenix_image_tag
  }
}

# Deploy AWS infrastructure
module "aws_phoenix" {
  source = "../../modules/aws-phoenix"

  cluster_name           = local.cluster_name
  region                = var.aws_region
  vpc_cidr              = var.vpc_cidr
  private_subnet_cidrs  = var.private_subnet_cidrs
  public_subnet_cidrs   = var.public_subnet_cidrs
  node_instance_types   = var.node_instance_types
  node_desired_capacity = var.node_desired_capacity
  node_min_capacity     = var.node_min_capacity
  node_max_capacity     = var.node_max_capacity
  phoenix_config        = local.phoenix_config
}

# Deploy Phoenix base infrastructure
module "phoenix_base" {
  source = "../../modules/phoenix-base"

  namespace             = var.phoenix_namespace
  monitoring_namespace  = var.monitoring_namespace
  phoenix_config        = local.base_phoenix_config
  storage_class         = local.phoenix_config.storage_class
  enable_monitoring     = var.enable_monitoring

  depends_on = [module.aws_phoenix]
}

# Install AWS Load Balancer Controller
resource "helm_release" "aws_load_balancer_controller" {
  name       = "aws-load-balancer-controller"
  repository = "https://aws.github.io/eks-charts"
  chart      = "aws-load-balancer-controller"
  namespace  = "kube-system"
  version    = "1.6.2"

  set {
    name  = "clusterName"
    value = module.aws_phoenix.cluster_name
  }

  set {
    name  = "serviceAccount.create"
    value = "false"
  }

  set {
    name  = "serviceAccount.name"
    value = "aws-load-balancer-controller"
  }

  depends_on = [module.aws_phoenix]
}

# Install EBS CSI Driver
resource "helm_release" "ebs_csi_driver" {
  name       = "aws-ebs-csi-driver"
  repository = "https://kubernetes-sigs.github.io/aws-ebs-csi-driver"
  chart      = "aws-ebs-csi-driver"
  namespace  = "kube-system"
  version    = "2.25.0"

  set {
    name  = "storageClasses[0].name"
    value = "gp3-csi"
  }

  set {
    name  = "storageClasses[0].parameters.type"
    value = "gp3"
  }

  set {
    name  = "storageClasses[0].volumeBindingMode"
    value = "WaitForFirstConsumer"
  }

  depends_on = [module.aws_phoenix]
}

# Install Phoenix application
resource "helm_release" "phoenix" {
  name       = "phoenix-vnext"
  chart      = "../../../helm/phoenix"
  namespace  = module.phoenix_base.namespace
  version    = var.phoenix_chart_version

  values = [
    yamlencode({
      global = {
        cloudProvider = "aws"
        region       = var.aws_region
        environment  = var.environment
      }

      collector = {
        replicaCount = local.base_phoenix_config.collector_replicas
        image = {
          tag = local.base_phoenix_config.image_tag
        }
        resources = local.base_phoenix_config.resources
        serviceAccount = {
          name = module.phoenix_base.collector_service_account
        }
      }

      controlActuator = {
        image = {
          repository = "${module.aws_phoenix.ecr_repositories["control-actuator-go"]}"
          tag       = var.phoenix_image_tag
        }
        serviceAccount = {
          name = module.phoenix_base.control_service_account
        }
      }

      monitoring = {
        enabled = var.enable_monitoring
        prometheus = {
          storageClass = local.phoenix_config.storage_class
        }
        grafana = {
          storageClass = local.phoenix_config.storage_class
        }
      }

      newRelic = {
        enabled    = var.enable_new_relic
        licenseKey = var.new_relic_license_key
      }
    })
  ]

  depends_on = [
    module.phoenix_base,
    helm_release.aws_load_balancer_controller,
    helm_release.ebs_csi_driver
  ]
}