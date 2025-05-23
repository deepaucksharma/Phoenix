#!/bin/bash
# Phoenix AWS Deployment Script
# Deploys Phoenix to AWS using Terraform

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
TERRAFORM_DIR="$PROJECT_ROOT/infrastructure/terraform/environments/aws"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

echo "=== Phoenix AWS Deployment ==="
echo

# Check prerequisites
check_prerequisites() {
    echo "Checking prerequisites..."
    
    # Check Terraform
    if ! command -v terraform &> /dev/null; then
        echo -e "${RED}✗${NC} Terraform not found. Please install Terraform."
        exit 1
    fi
    
    # Check AWS CLI
    if ! command -v aws &> /dev/null; then
        echo -e "${YELLOW}!${NC} AWS CLI not found. Terraform will use environment credentials."
    else
        # Check AWS credentials
        if ! aws sts get-caller-identity &> /dev/null; then
            echo -e "${RED}✗${NC} AWS credentials not configured."
            exit 1
        fi
        echo -e "${GREEN}✓${NC} AWS credentials configured"
    fi
    
    echo
}

# Deploy infrastructure
deploy_infrastructure() {
    echo "Deploying infrastructure..."
    cd "$TERRAFORM_DIR"
    
    # Initialize Terraform
    echo "Initializing Terraform..."
    terraform init
    
    # Create plan
    echo "Creating deployment plan..."
    terraform plan -out=tfplan
    
    # Apply plan
    echo "Applying deployment plan..."
    terraform apply tfplan
    
    # Clean up plan file
    rm -f tfplan
    
    echo -e "${GREEN}✓${NC} Infrastructure deployed successfully"
    echo
}

# Get outputs
get_outputs() {
    echo "Getting deployment outputs..."
    cd "$TERRAFORM_DIR"
    
    # Get load balancer URL
    LB_URL=$(terraform output -raw load_balancer_url 2>/dev/null || echo "N/A")
    echo "Load Balancer URL: $LB_URL"
    
    # Get monitoring endpoints
    GRAFANA_URL=$(terraform output -raw grafana_url 2>/dev/null || echo "N/A")
    PROMETHEUS_URL=$(terraform output -raw prometheus_url 2>/dev/null || echo "N/A")
    
    echo "Grafana URL: $GRAFANA_URL"
    echo "Prometheus URL: $PROMETHEUS_URL"
    echo
}

# Main deployment flow
main() {
    check_prerequisites
    deploy_infrastructure
    get_outputs
    
    echo -e "${GREEN}=== Deployment Complete ===${NC}"
    echo
    echo "Next steps:"
    echo "1. Access Grafana at the provided URL (admin/admin)"
    echo "2. Configure data sources if needed"
    echo "3. Monitor system health via Prometheus"
}

# Handle arguments
case "${1:-deploy}" in
    deploy)
        main
        ;;
    destroy)
        echo "Destroying AWS infrastructure..."
        cd "$TERRAFORM_DIR"
        terraform destroy -auto-approve
        ;;
    output)
        get_outputs
        ;;
    *)
        echo "Usage: $0 [deploy|destroy|output]"
        exit 1
        ;;
esac