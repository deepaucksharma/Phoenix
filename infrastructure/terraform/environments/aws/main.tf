terraform {
  required_version = ">= 1.3.0"
  
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
  
  # Optional: Configure S3 backend for state
  # backend "s3" {
  #   bucket = "phoenix-terraform-state"
  #   key    = "aws/terraform.tfstate"
  #   region = "us-east-1"
  # }
}

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

# Data sources for existing resources
data "aws_availability_zones" "available" {
  state = "available"
}

data "aws_caller_identity" "current" {}

# VPC Module
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"

  name = "${var.project_name}-vpc"
  cidr = var.vpc_cidr

  azs             = slice(data.aws_availability_zones.available.names, 0, 3)
  private_subnets = var.private_subnet_cidrs
  public_subnets  = var.public_subnet_cidrs

  enable_nat_gateway   = true
  single_nat_gateway   = var.environment == "dev" ? true : false
  enable_dns_hostnames = true
  enable_dns_support   = true

  # Enable VPC flow logs
  enable_flow_log                      = true
  create_flow_log_cloudwatch_iam_role  = true
  create_flow_log_cloudwatch_log_group = true

  tags = {
    "kubernetes.io/cluster/${var.cluster_name}" = "shared"
  }

  public_subnet_tags = {
    "kubernetes.io/cluster/${var.cluster_name}" = "shared"
    "kubernetes.io/role/elb"                    = "1"
  }

  private_subnet_tags = {
    "kubernetes.io/cluster/${var.cluster_name}" = "shared"
    "kubernetes.io/role/internal-elb"           = "1"
  }
}

# EKS Module
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 19.0"

  cluster_name    = var.cluster_name
  cluster_version = var.kubernetes_version

  vpc_id                         = module.vpc.vpc_id
  subnet_ids                     = module.vpc.private_subnets
  cluster_endpoint_public_access = true

  # Enable IRSA
  enable_irsa = true

  # Enable cluster encryption
  cluster_encryption_config = {
    provider_key_arn = aws_kms_key.eks.arn
    resources        = ["secrets"]
  }

  # Cluster addons
  cluster_addons = {
    coredns = {
      most_recent = true
    }
    kube-proxy = {
      most_recent = true
    }
    vpc-cni = {
      most_recent = true
      configuration_values = jsonencode({
        env = {
          ENABLE_PREFIX_DELEGATION = "true"
          WARM_PREFIX_TARGET       = "1"
        }
      })
    }
    aws-ebs-csi-driver = {
      most_recent = true
    }
  }

  # Node groups
  eks_managed_node_groups = {
    phoenix_general = {
      name            = "${var.project_name}-general"
      use_name_prefix = false

      min_size     = var.node_group_min_size
      max_size     = var.node_group_max_size
      desired_size = var.node_group_desired_size

      instance_types = var.node_instance_types
      capacity_type  = var.node_capacity_type

      disk_size = 100
      disk_type = "gp3"

      labels = {
        Environment = var.environment
        Workload    = "general"
      }

      taints = []

      update_config = {
        max_unavailable = 1
      }
    }

    phoenix_monitoring = {
      name            = "${var.project_name}-monitoring"
      use_name_prefix = false

      min_size     = 2
      max_size     = 4
      desired_size = 2

      instance_types = ["t3.large"]
      capacity_type  = "ON_DEMAND"

      disk_size = 200
      disk_type = "gp3"

      labels = {
        Environment = var.environment
        Workload    = "monitoring"
      }

      taints = [
        {
          key    = "monitoring"
          value  = "true"
          effect = "NO_SCHEDULE"
        }
      ]
    }
  }

  # Security group rules
  node_security_group_additional_rules = {
    ingress_self_all = {
      description = "Node to node all ports/protocols"
      protocol    = "-1"
      from_port   = 0
      to_port     = 0
      type        = "ingress"
      self        = true
    }
    egress_all = {
      description      = "Node all egress"
      protocol         = "-1"
      from_port        = 0
      to_port          = 0
      type             = "egress"
      cidr_blocks      = ["0.0.0.0/0"]
      ipv6_cidr_blocks = ["::/0"]
    }
  }

  tags = local.tags
}

# KMS key for EKS encryption
resource "aws_kms_key" "eks" {
  description             = "EKS Secret Encryption Key"
  deletion_window_in_days = 7
  enable_key_rotation     = true

  tags = merge(local.tags, {
    Name = "${var.cluster_name}-eks-key"
  })
}

resource "aws_kms_alias" "eks" {
  name          = "alias/${var.cluster_name}-eks"
  target_key_id = aws_kms_key.eks.key_id
}

# S3 bucket for Phoenix data
resource "aws_s3_bucket" "phoenix_data" {
  bucket = "${var.project_name}-data-${data.aws_caller_identity.current.account_id}"

  tags = merge(local.tags, {
    Name = "${var.project_name}-data"
  })
}

resource "aws_s3_bucket_versioning" "phoenix_data" {
  bucket = aws_s3_bucket.phoenix_data.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "phoenix_data" {
  bucket = aws_s3_bucket.phoenix_data.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# EFS for persistent storage
resource "aws_efs_file_system" "phoenix" {
  creation_token = "${var.project_name}-efs"
  encrypted      = true

  performance_mode = "generalPurpose"
  throughput_mode  = "bursting"

  tags = merge(local.tags, {
    Name = "${var.project_name}-efs"
  })
}

resource "aws_efs_mount_target" "phoenix" {
  count = length(module.vpc.private_subnets)

  file_system_id  = aws_efs_file_system.phoenix.id
  subnet_id       = module.vpc.private_subnets[count.index]
  security_groups = [aws_security_group.efs.id]
}

resource "aws_security_group" "efs" {
  name_prefix = "${var.project_name}-efs-"
  vpc_id      = module.vpc.vpc_id

  ingress {
    from_port   = 2049
    to_port     = 2049
    protocol    = "tcp"
    cidr_blocks = [module.vpc.vpc_cidr_block]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(local.tags, {
    Name = "${var.project_name}-efs-sg"
  })
}

# IAM roles for Phoenix services
module "phoenix_irsa" {
  source = "../../modules/aws-eks/irsa"

  cluster_name = module.eks.cluster_name
  
  service_accounts = {
    phoenix-collector = {
      namespace = "phoenix-system"
      policies = [
        "arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy",
        aws_iam_policy.phoenix_collector.arn
      ]
    }
    phoenix-validator = {
      namespace = "phoenix-system"
      policies = [
        aws_iam_policy.phoenix_validator.arn
      ]
    }
  }
}

# IAM policy for collector
resource "aws_iam_policy" "phoenix_collector" {
  name_prefix = "${var.project_name}-collector-"
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:GetObject",
          "s3:ListBucket"
        ]
        Resource = [
          aws_s3_bucket.phoenix_data.arn,
          "${aws_s3_bucket.phoenix_data.arn}/*"
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "cloudwatch:PutMetricData",
          "cloudwatch:GetMetricStatistics",
          "cloudwatch:ListMetrics"
        ]
        Resource = "*"
      }
    ]
  })
}

# IAM policy for validator
resource "aws_iam_policy" "phoenix_validator" {
  name_prefix = "${var.project_name}-validator-"
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "cloudwatch:GetMetricData",
          "cloudwatch:GetMetricStatistics"
        ]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:GetObject"
        ]
        Resource = [
          "${aws_s3_bucket.phoenix_data.arn}/benchmarks/*"
        ]
      }
    ]
  })
}

# Configure kubectl
resource "null_resource" "kubectl_config" {
  depends_on = [module.eks]

  provisioner "local-exec" {
    command = "aws eks update-kubeconfig --region ${var.aws_region} --name ${module.eks.cluster_name}"
  }
}

# Locals
locals {
  tags = {
    Project     = var.project_name
    Environment = var.environment
    Cluster     = var.cluster_name
  }
}