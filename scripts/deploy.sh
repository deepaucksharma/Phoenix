#!/usr/bin/env bash
# Unified Phoenix-vNext Deployment Script
# Supports local Docker, AWS EKS, and Azure AKS deployments

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m'

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $*"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

# Script configuration
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Default values
DEPLOYMENT_TARGET="local"
ENVIRONMENT="development"
NAMESPACE="phoenix-system"
DRY_RUN=false
VERBOSE=false
SKIP_BUILD=false
FORCE=false

# Function to show usage
show_usage() {
    cat << EOF
Phoenix-vNext Unified Deployment Script

Usage: $0 [OPTIONS] TARGET

Targets:
  local     Deploy to local Docker environment
  aws       Deploy to AWS EKS
  azure     Deploy to Azure AKS
  k8s       Deploy to existing Kubernetes cluster

Options:
  -e, --environment ENV     Deployment environment (default: development)
  -n, --namespace NS        Kubernetes namespace (default: phoenix-system)
  -d, --dry-run            Show what would be deployed without executing
  -v, --verbose            Enable verbose output
  -s, --skip-build         Skip building Docker images
  -f, --force              Force deployment without confirmation
  -h, --help               Show this help message

Examples:
  $0 local                          # Deploy locally with Docker Compose
  $0 aws -e production             # Deploy to AWS EKS in production
  $0 k8s -n phoenix-test          # Deploy to existing cluster in custom namespace
  $0 azure --dry-run               # Show what would be deployed to Azure

Environment Variables:
  AWS_REGION                       # AWS region for EKS deployment
  AZURE_REGION                     # Azure region for AKS deployment
  PHOENIX_IMAGE_TAG                # Image tag to deploy (default: latest)
  NEW_RELIC_LICENSE_KEY           # New Relic license key (optional)

EOF
}

# Function to parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -e|--environment)
                ENVIRONMENT="$2"
                shift 2
                ;;
            -n|--namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            -d|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -s|--skip-build)
                SKIP_BUILD=true
                shift
                ;;
            -f|--force)
                FORCE=true
                shift
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            local|aws|azure|k8s)
                DEPLOYMENT_TARGET="$1"
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
}

# Function to check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites for $DEPLOYMENT_TARGET deployment..."
    
    case $DEPLOYMENT_TARGET in
        local)
            command -v docker >/dev/null 2>&1 || { log_error "Docker is required but not installed"; exit 1; }
            command -v docker-compose >/dev/null 2>&1 || { log_error "Docker Compose is required but not installed"; exit 1; }
            ;;
        aws)
            command -v aws >/dev/null 2>&1 || { log_error "AWS CLI is required but not installed"; exit 1; }
            command -v kubectl >/dev/null 2>&1 || { log_error "kubectl is required but not installed"; exit 1; }
            command -v helm >/dev/null 2>&1 || { log_error "Helm is required but not installed"; exit 1; }
            command -v terraform >/dev/null 2>&1 || { log_error "Terraform is required but not installed"; exit 1; }
            ;;
        azure)
            command -v az >/dev/null 2>&1 || { log_error "Azure CLI is required but not installed"; exit 1; }
            command -v kubectl >/dev/null 2>&1 || { log_error "kubectl is required but not installed"; exit 1; }
            command -v helm >/dev/null 2>&1 || { log_error "Helm is required but not installed"; exit 1; }
            command -v terraform >/dev/null 2>&1 || { log_error "Terraform is required but not installed"; exit 1; }
            ;;
        k8s)
            command -v kubectl >/dev/null 2>&1 || { log_error "kubectl is required but not installed"; exit 1; }
            command -v helm >/dev/null 2>&1 || { log_error "Helm is required but not installed"; exit 1; }
            ;;
    esac
    
    log_success "Prerequisites check passed"
}

# Function to build images
build_images() {
    if [ "$SKIP_BUILD" = true ]; then
        log_info "Skipping image build as requested"
        return
    fi
    
    log_info "Building Phoenix images..."
    
    cd "$PROJECT_ROOT"
    
    if [ "$DEPLOYMENT_TARGET" = "local" ]; then
        docker-compose build
    else
        # Build for cloud deployment
        local tag="${PHOENIX_IMAGE_TAG:-latest}"
        
        # Build each service
        for service in control-actuator-go anomaly-detector benchmark synthetic-generator; do
            if [ -d "apps/$service" ] || [ -d "services/$service" ]; then
                local context_dir="apps/$service"
                [ -d "services/$service" ] && context_dir="services/$service"
                
                log_info "Building $service:$tag"
                docker build -t "phoenix/$service:$tag" "$context_dir"
            fi
        done
    fi
    
    log_success "Image build completed"
}

# Function to deploy locally
deploy_local() {
    log_info "Deploying Phoenix to local Docker environment..."
    
    cd "$PROJECT_ROOT"
    
    # Initialize environment if needed
    if [ ! -f ".env" ]; then
        log_info "Creating .env file from template"
        cp .env.template .env
    fi
    
    # Start services
    if [ "$DRY_RUN" = true ]; then
        log_info "Dry run - would execute: docker-compose up -d"
        docker-compose config
    else
        docker-compose up -d
        
        # Wait for services to be ready
        log_info "Waiting for services to start..."
        sleep 10
        
        # Check health
        local health_checks=(
            "http://localhost:13133"  # Main collector
            "http://localhost:13134"  # Observer
            "http://localhost:9090/-/healthy"  # Prometheus
        )
        
        for endpoint in "${health_checks[@]}"; do
            if curl -f -s "$endpoint" >/dev/null; then
                log_success "$(basename "$endpoint") is healthy"
            else
                log_warning "$(basename "$endpoint") health check failed"
            fi
        done
    fi
    
    log_success "Local deployment completed"
    log_info "Access points:"
    log_info "  Grafana: http://localhost:3000 (admin/admin)"
    log_info "  Prometheus: http://localhost:9090"
    log_info "  Control API: http://localhost:8081"
}

# Function to deploy to AWS
deploy_aws() {
    log_info "Deploying Phoenix to AWS EKS..."
    
    local region="${AWS_REGION:-us-west-2}"
    local cluster_name="phoenix-${ENVIRONMENT}-eks"
    
    cd "$PROJECT_ROOT/infrastructure/terraform/environments/aws"
    
    if [ "$DRY_RUN" = true ]; then
        log_info "Dry run - would execute Terraform plan"
        terraform plan -var="environment=$ENVIRONMENT" -var="aws_region=$region"
        return
    fi
    
    # Initialize Terraform
    terraform init
    
    # Plan and apply
    log_info "Creating AWS infrastructure..."
    terraform plan -var="environment=$ENVIRONMENT" -var="aws_region=$region"
    
    if [ "$FORCE" = false ]; then
        read -p "Continue with deployment? [y/N] " -n 1 -r
        echo
        [[ ! $REPLY =~ ^[Yy]$ ]] && { log_info "Deployment cancelled"; exit 0; }
    fi
    
    terraform apply -auto-approve -var="environment=$ENVIRONMENT" -var="aws_region=$region"
    
    # Update kubeconfig
    aws eks update-kubeconfig --region "$region" --name "$cluster_name"
    
    log_success "AWS deployment completed"
    log_info "Cluster: $cluster_name"
    log_info "Region: $region"
}

# Function to deploy to Azure
deploy_azure() {
    log_info "Deploying Phoenix to Azure AKS..."
    
    local region="${AZURE_REGION:-eastus}"
    local cluster_name="phoenix-${ENVIRONMENT}-aks"
    local resource_group="phoenix-${ENVIRONMENT}-rg"
    
    cd "$PROJECT_ROOT/infrastructure/terraform/environments/azure"
    
    if [ "$DRY_RUN" = true ]; then
        log_info "Dry run - would execute Terraform plan"
        terraform plan -var="environment=$ENVIRONMENT" -var="azure_region=$region"
        return
    fi
    
    # Initialize Terraform
    terraform init
    
    # Plan and apply
    log_info "Creating Azure infrastructure..."
    terraform plan -var="environment=$ENVIRONMENT" -var="azure_region=$region"
    
    if [ "$FORCE" = false ]; then
        read -p "Continue with deployment? [y/N] " -n 1 -r
        echo
        [[ ! $REPLY =~ ^[Yy]$ ]] && { log_info "Deployment cancelled"; exit 0; }
    fi
    
    terraform apply -auto-approve -var="environment=$ENVIRONMENT" -var="azure_region=$region"
    
    # Update kubeconfig
    az aks get-credentials --resource-group "$resource_group" --name "$cluster_name"
    
    log_success "Azure deployment completed"
    log_info "Cluster: $cluster_name"
    log_info "Resource Group: $resource_group"
}

# Function to deploy to Kubernetes
deploy_k8s() {
    log_info "Deploying Phoenix to Kubernetes cluster..."
    
    # Check cluster connectivity
    if ! kubectl cluster-info >/dev/null 2>&1; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    local current_context
    current_context=$(kubectl config current-context)
    log_info "Deploying to cluster: $current_context"
    
    if [ "$DRY_RUN" = true ]; then
        log_info "Dry run - would install Helm chart"
        helm template phoenix-vnext "$PROJECT_ROOT/infrastructure/helm/phoenix" \
            --namespace "$NAMESPACE" \
            --set global.environment="$ENVIRONMENT"
        return
    fi
    
    # Create namespace if it doesn't exist
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    
    # Install or upgrade Phoenix
    helm upgrade --install phoenix-vnext "$PROJECT_ROOT/infrastructure/helm/phoenix" \
        --namespace "$NAMESPACE" \
        --set global.environment="$ENVIRONMENT" \
        --set global.cloudProvider="kubernetes" \
        --wait --timeout=10m
    
    # Check deployment status
    kubectl get pods -n "$NAMESPACE"
    
    log_success "Kubernetes deployment completed"
    log_info "Namespace: $NAMESPACE"
    log_info "Check status: kubectl get pods -n $NAMESPACE"
}

# Function to show deployment status
show_status() {
    log_info "Deployment Status for $DEPLOYMENT_TARGET"
    
    case $DEPLOYMENT_TARGET in
        local)
            docker-compose ps
            ;;
        aws|azure|k8s)
            kubectl get pods -n "$NAMESPACE"
            kubectl get services -n "$NAMESPACE"
            ;;
    esac
}

# Main execution
main() {
    log_info "Phoenix-vNext Unified Deployment"
    log_info "================================="
    
    parse_args "$@"
    
    log_info "Configuration:"
    log_info "  Target: $DEPLOYMENT_TARGET"
    log_info "  Environment: $ENVIRONMENT"
    log_info "  Namespace: $NAMESPACE"
    log_info "  Dry Run: $DRY_RUN"
    
    check_prerequisites
    
    if [ "$SKIP_BUILD" = false ]; then
        build_images
    fi
    
    case $DEPLOYMENT_TARGET in
        local)
            deploy_local
            ;;
        aws)
            deploy_aws
            ;;
        azure)
            deploy_azure
            ;;
        k8s)
            deploy_k8s
            ;;
        *)
            log_error "Unknown deployment target: $DEPLOYMENT_TARGET"
            show_usage
            exit 1
            ;;
    esac
    
    if [ "$DRY_RUN" = false ]; then
        show_status
    fi
    
    log_success "Deployment completed successfully!"
}

# Execute main function
main "$@"