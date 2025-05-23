# Phoenix-vNext Infrastructure Guide

This document provides a comprehensive overview of the streamlined infrastructure setup for Phoenix-vNext.

## 📋 Overview

The infrastructure has been streamlined and consolidated to eliminate redundancy and provide a unified deployment experience across all environments.

### Key Improvements

- **Unified Docker Compose**: Single base configuration with environment-specific overrides
- **Modular Terraform**: Reusable modules for AWS and Azure deployments
- **Kustomize-based K8s**: Clean Kubernetes manifests with overlay-based customization
- **Comprehensive Helm Chart**: Production-ready chart with cloud-specific configurations
- **Unified Deployment Scripts**: Single script for all deployment targets

## 🏗️ Architecture

```
phoenix-vnext/
├── docker-compose.yaml          # Main Docker Compose configuration
├── docker-compose.override.yml  # Environment-specific overrides
├── docker-compose.dev.yml       # Development-specific settings
├── scripts/
│   ├── deploy.sh                # Unified deployment script
│   └── cleanup.sh               # Unified cleanup script
└── infrastructure/
    ├── terraform/
    │   ├── modules/
    │   │   ├── phoenix-base/     # Shared Kubernetes resources
    │   │   └── aws-phoenix/      # AWS-specific infrastructure
    │   └── environments/
    │       ├── aws/              # AWS deployment configuration
    │       └── azure/            # Azure deployment configuration
    ├── k8s/
    │   ├── base/                 # Base Kubernetes manifests
    │   └── overlays/
    │       ├── aws/              # AWS-specific Kustomize overlay
    │       └── azure/            # Azure-specific Kustomize overlay
    └── helm/
        └── phoenix/              # Comprehensive Helm chart
```

## 🐳 Docker Deployment

### Local Development

```bash
# Quick start
docker-compose up -d

# Development mode with debug logging
docker-compose -f docker-compose.yaml -f docker-compose.dev.yml up -d

# Using unified script
./scripts/deploy.sh local
```

### Docker Compose Structure

- **`docker-compose.yaml`**: Main configuration with all services
- **`docker-compose.override.yml`**: Production-ready defaults
- **`docker-compose.dev.yml`**: Development overrides with hot reload

### Service Configuration

| Service | Ports | Purpose |
|---------|-------|---------|
| otelcol-main | 4317-4318, 8888-8890, 13133 | Main collector with 3 pipelines |
| otelcol-observer | 9888, 13134 | KPI monitoring collector |
| control-actuator-go | 8081 | PID controller API |
| anomaly-detector | 8082 | Anomaly detection API |
| benchmark-controller | 8083 | Performance validation API |
| prometheus | 9090 | Metrics storage |
| grafana | 3000 | Dashboards (admin/admin) |

## ☁️ Cloud Deployment

### AWS EKS

```bash
# Deploy to AWS
./scripts/deploy.sh aws --environment production

# Or use existing script
./deploy-aws.sh production

# Cleanup
./scripts/cleanup.sh aws --environment production
```

### Azure AKS

```bash
# Deploy to Azure
./scripts/deploy.sh azure --environment production

# Or use existing script
./deploy-azure.sh production

# Cleanup
./scripts/cleanup.sh azure --environment production
```

### Cloud Infrastructure Features

#### AWS Components
- **VPC**: Custom VPC with public/private subnets
- **EKS**: Managed Kubernetes cluster
- **ECR**: Container registries for Phoenix images
- **Load Balancer Controller**: NLB integration
- **EBS CSI Driver**: GP3 storage class
- **IAM**: Service accounts with IRSA

#### Azure Components
- **VNET**: Virtual network with subnets
- **AKS**: Managed Kubernetes cluster
- **ACR**: Container registry
- **NGINX Ingress**: External access
- **Managed Disks**: Premium storage
- **RBAC**: Azure AD integration

## 🚀 Kubernetes Deployment

### Using Kustomize

```bash
# Deploy to AWS overlay
kubectl apply -k infrastructure/k8s/overlays/aws

# Deploy to Azure overlay
kubectl apply -k infrastructure/k8s/overlays/azure

# Deploy base configuration
kubectl apply -k infrastructure/k8s/base
```

### Using Helm

```bash
# Install with default values
helm install phoenix-vnext infrastructure/helm/phoenix

# Install with custom values
helm install phoenix-vnext infrastructure/helm/phoenix \
  --set global.cloudProvider=aws \
  --set global.environment=production \
  --set monitoring.enabled=true

# Using unified script
./scripts/deploy.sh k8s --namespace phoenix-prod
```

### Kubernetes Architecture

#### Namespaces
- **phoenix-system**: Main application services
- **phoenix-monitoring**: Prometheus and Grafana

#### Service Accounts
- **phoenix-collector**: RBAC for metrics collection
- **phoenix-control**: Control plane operations

#### Storage
- **Prometheus**: 50Gi persistent storage
- **Grafana**: 10Gi persistent storage
- **Cloud-specific storage classes** (gp3-csi for AWS, managed-csi for Azure)

## 🔧 Terraform Modules

### Module Structure

#### phoenix-base
- Kubernetes namespaces and RBAC
- Service accounts
- ConfigMaps and Secrets
- Persistent Volume Claims

#### aws-phoenix
- VPC and networking
- EKS cluster and node groups
- ECR repositories
- IAM roles and policies

#### azure-phoenix (planned)
- VNET and networking
- AKS cluster and node pools
- ACR registry
- Azure AD integration

### Usage

```bash
# AWS deployment
cd infrastructure/terraform/environments/aws
terraform init
terraform plan -var="environment=production"
terraform apply -var="environment=production"

# Azure deployment
cd infrastructure/terraform/environments/azure
terraform init
terraform plan -var="environment=production"
terraform apply -var="environment=production"
```

## 📊 Monitoring Configuration

### Prometheus
- **Retention**: 30 days
- **Storage**: 50Gi persistent volume
- **Scrape Configs**: All Phoenix services
- **Recording Rules**: 25+ efficiency and control metrics

### Grafana
- **Storage**: 10Gi persistent volume
- **Dashboards**: Provisioned from configs
- **Data Sources**: Automatic Prometheus connection
- **Plugins**: Clock panel, Simple JSON datasource

### Key Metrics
- `phoenix:signal_preservation_score`
- `phoenix:cardinality_efficiency_ratio`
- `phoenix:control_stability_score`
- `phoenix:control_loop_effectiveness`

## 🛠️ Development Tools

### Unified Scripts

```bash
# Deploy to any environment
./scripts/deploy.sh <target> [options]

# Clean up any environment
./scripts/cleanup.sh <target> [options]

# Show available options
./scripts/deploy.sh --help
./scripts/cleanup.sh --help
```

### Development Features

#### Hot Reload (Development)
- Air configuration for Go services
- Volume mounts for live code editing
- Debug logging enabled

#### Testing
- Benchmark controller with 4 scenarios
- Health check endpoints
- Integration test scripts

#### Monitoring
- pprof endpoints for profiling
- Debug APIs for control state
- Prometheus metrics for all services

## 🔐 Security Configuration

### Container Security
- Non-root containers
- Read-only root filesystems
- Minimal capabilities
- Security contexts enforced

### Network Security
- Network policies (configurable)
- Service mesh ready
- TLS termination at ingress

### Cloud Security
- IAM/RBAC least privilege
- VPC/VNET isolation
- Encryption at rest and in transit

## 📈 Scaling Configuration

### Horizontal Pod Autoscaling
```yaml
autoscaling:
  enabled: true
  hpa:
    minReplicas: 2
    maxReplicas: 10
    targetCPUUtilizationPercentage: 70
```

### Vertical Pod Autoscaling
```yaml
autoscaling:
  vpa:
    enabled: true
    updateMode: "Auto"
```

### Resource Recommendations

| Component | CPU Request | Memory Request | CPU Limit | Memory Limit |
|-----------|-------------|----------------|-----------|--------------|
| Collector | 500m | 1Gi | 1000m | 2Gi |
| Observer | 100m | 256Mi | 200m | 512Mi |
| Control Actuator | 100m | 128Mi | 200m | 256Mi |
| Anomaly Detector | 100m | 128Mi | 200m | 256Mi |
| Prometheus | 500m | 1Gi | 1000m | 2Gi |
| Grafana | 200m | 512Mi | 500m | 1Gi |

## 🚨 Troubleshooting

### Common Issues

#### Docker Compose
```bash
# Check service logs
docker-compose logs -f <service-name>

# Restart specific service
docker-compose restart <service-name>

# View service health
curl http://localhost:13133  # Collector health
```

#### Kubernetes
```bash
# Check pod status
kubectl get pods -n phoenix-system

# View pod logs
kubectl logs -f deployment/otelcol-main -n phoenix-system

# Debug service connectivity
kubectl port-forward service/otelcol-main 8888:8888 -n phoenix-system
```

#### Terraform
```bash
# Debug Terraform issues
terraform plan -var="environment=development"
terraform refresh
terraform show
```

### Health Check Endpoints

| Service | Endpoint | Port |
|---------|----------|------|
| Main Collector | `/` | 13133 |
| Observer | `/` | 13134 |
| Control Actuator | `/health` | 8081 |
| Anomaly Detector | `/health` | 8082 |
| Benchmark | `/health` | 8083 |
| Prometheus | `/-/healthy` | 9090 |
| Grafana | `/api/health` | 3000 |

## 📚 Next Steps

1. **Review Configuration**: Update `.env` files with your settings
2. **Choose Deployment**: Select local, AWS, or Azure deployment
3. **Run Deployment**: Use unified scripts for consistent deployment
4. **Monitor**: Access Grafana dashboards for system monitoring
5. **Scale**: Configure autoscaling based on your requirements

## 🤝 Contributing

When adding new infrastructure components:

1. Update base Terraform modules
2. Add cloud-specific configurations to overlays
3. Update Helm chart with new services
4. Add health checks and monitoring
5. Update this documentation

For questions or issues, please refer to the troubleshooting section or create an issue in the repository.