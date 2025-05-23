#!/bin/bash
# Phoenix Azure Deployment Script
# Deploys Phoenix to Azure using Terraform

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
TERRAFORM_DIR="$PROJECT_ROOT/infrastructure/terraform/environments/azure"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

echo "=== Phoenix Azure Deployment ==="
echo

# Check prerequisites
check_prerequisites() {
    echo "Checking prerequisites..."
    
    # Check Terraform
    if ! command -v terraform &> /dev/null; then
        echo -e "${RED}✗${NC} Terraform not found. Please install Terraform."
        exit 1
    fi
    
    # Check Azure CLI
    if ! command -v az &> /dev/null; then
        echo -e "${YELLOW}!${NC} Azure CLI not found. Terraform will use environment credentials."
    else
        # Check Azure login
        if ! az account show &> /dev/null; then
            echo -e "${RED}✗${NC} Not logged in to Azure. Run 'az login' first."
            exit 1
        fi
        echo -e "${GREEN}✓${NC} Azure credentials configured"
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
    
    # Get AKS cluster details
    CLUSTER_NAME=$(terraform output -raw aks_cluster_name 2>/dev/null || echo "N/A")
    RESOURCE_GROUP=$(terraform output -raw resource_group_name 2>/dev/null || echo "N/A")
    
    echo "AKS Cluster: $CLUSTER_NAME"
    echo "Resource Group: $RESOURCE_GROUP"
    
    # Get ingress URL if available
    INGRESS_IP=$(terraform output -raw ingress_ip 2>/dev/null || echo "N/A")
    echo "Ingress IP: $INGRESS_IP"
    echo
}

# Configure kubectl
configure_kubectl() {
    echo "Configuring kubectl..."
    cd "$TERRAFORM_DIR"
    
    CLUSTER_NAME=$(terraform output -raw aks_cluster_name 2>/dev/null)
    RESOURCE_GROUP=$(terraform output -raw resource_group_name 2>/dev/null)
    
    if [ "$CLUSTER_NAME" != "N/A" ] && [ "$RESOURCE_GROUP" != "N/A" ]; then
        az aks get-credentials --resource-group "$RESOURCE_GROUP" --name "$CLUSTER_NAME" --overwrite-existing
        echo -e "${GREEN}✓${NC} kubectl configured for AKS cluster"
    else
        echo -e "${YELLOW}!${NC} Could not configure kubectl - cluster information not available"
    fi
    echo
}

# Main deployment flow
main() {
    check_prerequisites
    deploy_infrastructure
    configure_kubectl
    get_outputs
    
    echo -e "${GREEN}=== Deployment Complete ===${NC}"
    echo
    echo "Next steps:"
    echo "1. Use 'kubectl get pods -n phoenix' to check pod status"
    echo "2. Access services via the ingress IP when available"
    echo "3. Monitor cluster health in Azure Portal"
}

# Handle arguments
case "${1:-deploy}" in
    deploy)
        main
        ;;
    destroy)
        echo "Destroying Azure infrastructure..."
        cd "$TERRAFORM_DIR"
        terraform destroy -auto-approve
        ;;
    output)
        get_outputs
        ;;
    kubeconfig)
        configure_kubectl
        ;;
    *)
        echo "Usage: $0 [deploy|destroy|output|kubeconfig]"
        exit 1
        ;;
esac