#!/bin/bash
# Deploy Phoenix Platform to Kubernetes

set -euo pipefail

# Configuration
ENVIRONMENT=${1:-staging}
NAMESPACE="phoenix-${ENVIRONMENT}"
HELM_RELEASE="phoenix"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_requirements() {
    log_info "Checking requirements..."
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed"
        exit 1
    fi
    
    # Check helm
    if ! command -v helm &> /dev/null; then
        log_error "helm is not installed"
        exit 1
    fi
    
    # Check cluster connection
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    log_info "All requirements satisfied"
}

create_namespace() {
    log_info "Creating namespace ${NAMESPACE}..."
    
    if kubectl get namespace ${NAMESPACE} &> /dev/null; then
        log_warn "Namespace ${NAMESPACE} already exists"
    else
        kubectl create namespace ${NAMESPACE}
        kubectl label namespace ${NAMESPACE} \
            name=${NAMESPACE} \
            environment=${ENVIRONMENT} \
            app.kubernetes.io/managed-by=helm
    fi
}

install_crds() {
    log_info "Installing CRDs..."
    kubectl apply -f k8s/crds/
}

create_secrets() {
    log_info "Creating secrets..."
    
    # Check if secrets already exist
    if kubectl get secret newrelic-secret -n ${NAMESPACE} &> /dev/null; then
        log_warn "newrelic-secret already exists, skipping..."
    else
        if [ -z "${NEW_RELIC_API_KEY:-}" ]; then
            log_error "NEW_RELIC_API_KEY environment variable is not set"
            exit 1
        fi
        
        kubectl create secret generic newrelic-secret \
            --namespace=${NAMESPACE} \
            --from-literal=api-key="${NEW_RELIC_API_KEY}"
    fi
    
    if kubectl get secret git-credentials -n ${NAMESPACE} &> /dev/null; then
        log_warn "git-credentials already exists, skipping..."
    else
        if [ -z "${GIT_TOKEN:-}" ]; then
            log_warn "GIT_TOKEN not set, skipping git credentials creation"
        else
            kubectl create secret generic git-credentials \
                --namespace=${NAMESPACE} \
                --from-literal=token="${GIT_TOKEN}"
        fi
    fi
    
    if kubectl get secret postgresql-secret -n ${NAMESPACE} &> /dev/null; then
        log_warn "postgresql-secret already exists, skipping..."
    else
        # Generate random password
        POSTGRES_PASSWORD=$(openssl rand -base64 32)
        kubectl create secret generic postgresql-secret \
            --namespace=${NAMESPACE} \
            --from-literal=postgres-password="${POSTGRES_PASSWORD}" \
            --from-literal=password="${POSTGRES_PASSWORD}"
    fi
}

add_helm_repos() {
    log_info "Adding Helm repositories..."
    
    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
    helm repo add grafana https://grafana.github.io/helm-charts
    helm repo add bitnami https://charts.bitnami.com/bitnami
    helm repo update
}

deploy_dependencies() {
    log_info "Deploying dependencies..."
    
    # Update Helm dependencies
    cd helm/phoenix
    helm dependency update
    cd ../..
}

deploy_phoenix() {
    log_info "Deploying Phoenix platform..."
    
    # Prepare values file
    VALUES_FILE="helm/phoenix/values-${ENVIRONMENT}.yaml"
    if [ ! -f "${VALUES_FILE}" ]; then
        log_warn "Environment-specific values file not found, using default values"
        VALUES_FILE="helm/phoenix/values.yaml"
    fi
    
    # Install or upgrade Phoenix
    helm upgrade --install ${HELM_RELEASE} ./helm/phoenix \
        --namespace ${NAMESPACE} \
        --values ${VALUES_FILE} \
        --set global.domain="${PHOENIX_DOMAIN:-phoenix.example.com}" \
        --set dashboard.image.tag="${VERSION:-latest}" \
        --set experimentController.image.tag="${VERSION:-latest}" \
        --set generator.image.tag="${VERSION:-latest}" \
        --set pipelineOperator.image.tag="${VERSION:-latest}" \
        --set loadsimOperator.image.tag="${VERSION:-latest}" \
        --wait \
        --timeout 10m
}

wait_for_deployment() {
    log_info "Waiting for deployments to be ready..."
    
    # Wait for key deployments
    kubectl wait --for=condition=available --timeout=300s \
        deployment/phoenix-dashboard \
        deployment/phoenix-experiment-controller \
        deployment/phoenix-api-gateway \
        -n ${NAMESPACE} || true
    
    # Check pod status
    kubectl get pods -n ${NAMESPACE}
}

post_deployment() {
    log_info "Running post-deployment tasks..."
    
    # Get ingress information
    if kubectl get ingress -n ${NAMESPACE} phoenix-dashboard &> /dev/null; then
        DASHBOARD_URL=$(kubectl get ingress -n ${NAMESPACE} phoenix-dashboard -o jsonpath='{.spec.rules[0].host}')
        log_info "Dashboard URL: https://${DASHBOARD_URL}"
    fi
    
    # Get load balancer IP (if using LoadBalancer service)
    if kubectl get svc -n ${NAMESPACE} phoenix-api-gateway &> /dev/null; then
        API_IP=$(kubectl get svc -n ${NAMESPACE} phoenix-api-gateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "pending")
        log_info "API Gateway IP: ${API_IP}"
    fi
    
    # Display access information
    cat <<EOF

========================================
Phoenix Platform Deployed Successfully!
========================================

Environment: ${ENVIRONMENT}
Namespace: ${NAMESPACE}

To access the dashboard:
  kubectl port-forward -n ${NAMESPACE} svc/phoenix-dashboard 8080:80
  Open http://localhost:8080

To access Grafana:
  kubectl port-forward -n ${NAMESPACE} svc/phoenix-grafana 3000:80
  Open http://localhost:3000
  Default credentials: admin/changeme

To check deployment status:
  kubectl get all -n ${NAMESPACE}

To view logs:
  kubectl logs -n ${NAMESPACE} -l app.kubernetes.io/name=phoenix --tail=100 -f

EOF
}

# Main execution
main() {
    log_info "Deploying Phoenix Platform to ${ENVIRONMENT} environment"
    
    check_requirements
    create_namespace
    install_crds
    create_secrets
    add_helm_repos
    deploy_dependencies
    deploy_phoenix
    wait_for_deployment
    post_deployment
    
    log_info "Deployment completed successfully!"
}

# Run main function
main