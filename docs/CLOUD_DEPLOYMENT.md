# Phoenix Cloud Deployment Guide

## Overview

Phoenix can be deployed to major cloud providers (AWS, Azure, GCP) using Kubernetes. This guide covers deployment to AWS EKS and Azure AKS with full production configurations.

## Table of Contents

- [Prerequisites](#prerequisites)
- [AWS Deployment](#aws-deployment)
- [Azure Deployment](#azure-deployment)
- [Configuration Options](#configuration-options)
- [Monitoring & Operations](#monitoring--operations)
- [Cost Optimization](#cost-optimization)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### Required Tools
- **Cloud CLI**: AWS CLI or Azure CLI
- **Terraform**: >= 1.3.0
- **kubectl**: >= 1.25.0
- **Helm**: >= 3.10.0
- **Docker**: For building custom images

### Cloud Permissions
Ensure you have sufficient permissions to create:
- VPCs/VNets and subnets
- Kubernetes clusters
- Load balancers
- Storage accounts/S3 buckets
- IAM roles and policies

## AWS Deployment

### Quick Start
```bash
# Set environment variables
export AWS_REGION=us-east-1
export CLUSTER_NAME=phoenix-eks
export ENVIRONMENT=dev

# Deploy Phoenix to AWS
./deploy-aws.sh
```

### Architecture on AWS

```
┌─────────────────────────────────────────────────────────────┐
│                        AWS Account                          │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────┐   │
│  │                    VPC (10.0.0.0/16)                 │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │   │
│  │  │  Public      │  │  Public     │  │  Public     │ │   │
│  │  │  Subnet 1    │  │  Subnet 2   │  │  Subnet 3   │ │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘ │   │
│  │         │                │                │          │   │
│  │    ┌────┴───────────┬────┴───────────┬────┴──────┐  │   │
│  │    │          NAT Gateway            │           │  │   │
│  │    └────┬───────────┴────┬───────────┴────┬──────┘  │   │
│  │         │                │                │          │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │   │
│  │  │  Private     │  │  Private    │  │  Private    │ │   │
│  │  │  Subnet 1    │  │  Subnet 2   │  │  Subnet 3   │ │   │
│  │  │              │  │             │  │             │ │   │
│  │  │  EKS Nodes   │  │  EKS Nodes  │  │  EKS Nodes  │ │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘ │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────┐  ┌─────────────┐  ┌──────────────┐   │
│  │   S3 Bucket     │  │     EFS     │  │  CloudWatch  │   │
│  │  (Phoenix Data) │  │  (Storage)  │  │  (Logs)      │   │
│  └─────────────────┘  └─────────────┘  └──────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### AWS-Specific Features

1. **Network Load Balancer (NLB)**
   - High-performance OTLP ingestion
   - Cross-zone load balancing
   - Static IP addresses available

2. **EBS CSI Driver**
   - GP3 volumes for better performance
   - Automatic volume expansion
   - Snapshot support

3. **IRSA (IAM Roles for Service Accounts)**
   - Fine-grained permissions
   - No credential management
   - Automatic rotation

4. **CloudWatch Integration**
   - Native metrics export
   - Log aggregation
   - Alarms and notifications

### Advanced AWS Configuration

```hcl
# terraform.tfvars for production
aws_region = "us-east-1"
environment = "prod"
cluster_name = "phoenix-eks-prod"

# High availability across 3 AZs
private_subnet_cidrs = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
public_subnet_cidrs = ["10.0.101.0/24", "10.0.102.0/24", "10.0.103.0/24"]

# Production node configuration
node_instance_types = ["m5.xlarge", "m5.2xlarge"]
node_capacity_type = "ON_DEMAND"
node_group_min_size = 3
node_group_max_size = 20
node_group_desired_size = 6

# Enable all monitoring
enable_monitoring = true
```

## Azure Deployment

### Quick Start
```bash
# Set environment variables
export AZURE_LOCATION=eastus
export RESOURCE_GROUP=phoenix-vnext-rg
export CLUSTER_NAME=phoenix-aks
export ENVIRONMENT=dev

# Deploy Phoenix to Azure
./deploy-azure.sh
```

### Architecture on Azure

```
┌─────────────────────────────────────────────────────────────┐
│                    Azure Subscription                       │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Resource Group (phoenix-rg)             │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │  ┌─────────────────────────────────────────────┐    │   │
│  │  │          Virtual Network (10.0.0.0/16)      │    │   │
│  │  ├─────────────────────────────────────────────┤    │   │
│  │  │  ┌─────────────────┐  ┌─────────────────┐  │    │   │
│  │  │  │   AKS Subnet    │  │ Ingress Subnet  │  │    │   │
│  │  │  │   10.0.1.0/24   │  │  10.0.2.0/24    │  │    │   │
│  │  │  │                 │  │                 │  │    │   │
│  │  │  │   AKS Nodes     │  │  Load Balancer  │  │    │   │
│  │  │  └─────────────────┘  └─────────────────┘  │    │   │
│  │  └─────────────────────────────────────────────┘    │   │
│  │                                                      │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────┐  │   │
│  │  │ Storage Acct │  │  Container   │  │   Log    │  │   │
│  │  │  (Metrics)   │  │   Registry   │  │Analytics │  │   │
│  │  └──────────────┘  └──────────────┘  └──────────┘  │   │
│  │                                                      │   │
│  │  ┌─────────────────────────────────────────────┐    │   │
│  │  │          AKS Cluster (phoenix-aks)          │    │   │
│  │  │  ┌────────────┐  ┌────────────┐            │    │   │
│  │  │  │  General   │  │ Monitoring │            │    │   │
│  │  │  │ Node Pool  │  │ Node Pool  │            │    │   │
│  │  │  └────────────┘  └────────────┘            │    │   │
│  │  └─────────────────────────────────────────────┘    │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### Azure-Specific Features

1. **Azure Load Balancer**
   - Standard SKU with zone redundancy
   - Health probes for reliability
   - Azure Private Link support

2. **Azure Files CSI Driver**
   - Shared storage for control signals
   - Premium and Standard tiers
   - SMB and NFS support

3. **Workload Identity**
   - Managed identity for pods
   - Azure RBAC integration
   - Key Vault access

4. **Azure Monitor Integration**
   - Container insights
   - Log Analytics workspace
   - Application Insights

### Advanced Azure Configuration

```hcl
# terraform.tfvars for production
azure_location = "eastus"
environment = "prod"
cluster_name = "phoenix-aks-prod"

# Production node configuration
node_vm_size = "Standard_D8s_v3"
node_count = 6
node_min_count = 3
node_max_count = 20

# Enable monitoring pool
enable_monitoring_pool = true

# Azure AD integration
aks_admin_group_ids = ["your-aad-group-id"]
```

## Configuration Options

### Helm Values Override

Create a custom values file for your deployment:

```yaml
# custom-values.yaml
global:
  cloudProvider: aws  # or azure
  domain: your-domain.com

collector:
  replicas: 5
  resources:
    requests:
      memory: "1Gi"
      cpu: "1000m"
    limits:
      memory: "4Gi"
      cpu: "4000m"
  
  autoscaling:
    enabled: true
    minReplicas: 5
    maxReplicas: 20
    targetCPUUtilizationPercentage: 70

storage:
  controlSignals:
    size: 10Gi
  benchmarkData:
    size: 100Gi

prometheus:
  server:
    retention: "90d"
    persistentVolume:
      size: 500Gi

grafana:
  adminPassword: "your-secure-password"
  ingress:
    enabled: true
    hosts:
      - grafana.your-domain.com
    tls:
      - secretName: grafana-tls
        hosts:
          - grafana.your-domain.com
```

Deploy with custom values:
```bash
helm upgrade --install phoenix ./infrastructure/helm/phoenix \
  --namespace phoenix-system \
  --values custom-values.yaml
```

### Environment Variables

Key environment variables for cloud deployments:

```bash
# AWS
export AWS_REGION=us-east-1
export AWS_PROFILE=production
export CLUSTER_NAME=phoenix-eks-prod

# Azure
export AZURE_SUBSCRIPTION_ID=your-subscription-id
export AZURE_LOCATION=eastus
export RESOURCE_GROUP=phoenix-prod-rg

# Common
export ENVIRONMENT=prod
export ENABLE_MONITORING=true
export ENABLE_BACKUPS=true
```

## Monitoring & Operations

### Accessing Dashboards

#### AWS
```bash
# Port forward to access locally
kubectl port-forward -n phoenix-system svc/phoenix-grafana 3000:80

# Get Load Balancer URL
kubectl get svc phoenix-collector -n phoenix-system
```

#### Azure
```bash
# Get Ingress IP
kubectl get svc -n ingress-nginx ingress-nginx-controller

# Access via domain
open http://phoenix.<INGRESS_IP>.nip.io
```

### Operational Tasks

#### Scaling
```bash
# Scale collector replicas
kubectl scale deployment phoenix-collector -n phoenix-system --replicas=10

# Enable autoscaling
kubectl autoscale deployment phoenix-collector -n phoenix-system \
  --min=3 --max=20 --cpu-percent=80
```

#### Backup Control Signals
```bash
# AWS - Backup to S3
kubectl exec -n phoenix-system deployment/phoenix-actuator -- \
  aws s3 cp /etc/phoenix/control/optimization_mode.yaml \
  s3://phoenix-backups/control/$(date +%Y%m%d-%H%M%S).yaml

# Azure - Backup to Blob Storage
kubectl exec -n phoenix-system deployment/phoenix-actuator -- \
  az storage blob upload \
    --account-name $STORAGE_ACCOUNT \
    --container-name backups \
    --name control/$(date +%Y%m%d-%H%M%S).yaml \
    --file /etc/phoenix/control/optimization_mode.yaml
```

## Cost Optimization

### AWS Cost Savings
1. **Use Spot Instances** for non-critical workloads
2. **Reserved Instances** for production nodes
3. **S3 Lifecycle Policies** for old metrics
4. **GP3 volumes** instead of GP2
5. **Single NAT Gateway** for dev environments

### Azure Cost Savings
1. **Reserved Instances** for AKS nodes
2. **Spot Node Pools** for batch workloads
3. **Standard tier** storage for archives
4. **Autoscaling** to match demand
5. **Azure Hybrid Benefit** if applicable

### Resource Recommendations

| Environment | Nodes | Instance Type | Storage | Cost/Month |
|-------------|-------|---------------|---------|------------|
| Dev | 3 | t3.large / D2s_v3 | 100GB | ~$300 |
| Staging | 6 | t3.xlarge / D4s_v3 | 500GB | ~$800 |
| Production | 12 | m5.2xlarge / D8s_v3 | 2TB | ~$3000 |

## Troubleshooting

### Common Issues

#### Pods Not Starting
```bash
# Check pod status
kubectl get pods -n phoenix-system
kubectl describe pod <pod-name> -n phoenix-system

# Check events
kubectl get events -n phoenix-system --sort-by='.lastTimestamp'
```

#### Load Balancer Not Getting IP
```bash
# AWS - Check service
kubectl describe svc phoenix-collector -n phoenix-system

# Azure - Check ingress controller
kubectl get svc -n ingress-nginx
kubectl logs -n ingress-nginx deployment/ingress-nginx-controller
```

#### Storage Issues
```bash
# Check PVC status
kubectl get pvc -n phoenix-system
kubectl describe pvc <pvc-name> -n phoenix-system

# Check storage class
kubectl get storageclass
```

### Debug Commands

```bash
# Get all Phoenix resources
kubectl get all -n phoenix-system

# Check collector logs
kubectl logs -n phoenix-system -l app.kubernetes.io/name=phoenix-collector

# Check control loop
kubectl logs -n phoenix-system deployment/phoenix-actuator

# Exec into pod
kubectl exec -it -n phoenix-system deployment/phoenix-collector -- sh
```

## Security Best Practices

1. **Network Policies**: Restrict pod-to-pod communication
2. **RBAC**: Use least-privilege service accounts
3. **Secrets Management**: Use cloud KMS for sensitive data
4. **Image Scanning**: Scan containers for vulnerabilities
5. **Audit Logging**: Enable cluster audit logs
6. **Encryption**: Enable encryption at rest and in transit

## Disaster Recovery

### Backup Strategy
- Control signals: Every 15 minutes
- Prometheus data: Daily snapshots
- Grafana dashboards: Version controlled
- Configuration: Stored in Git

### Recovery Procedures
1. Restore infrastructure with Terraform
2. Deploy Phoenix with Helm
3. Restore control signals from backup
4. Import Prometheus snapshots
5. Verify system functionality

## Support

For issues or questions:
- GitHub Issues: https://github.com/deepaucksharma/Phoenix/issues
- Documentation: https://github.com/deepaucksharma/Phoenix/docs
- Community Slack: #phoenix-support
