#!/bin/bash
# Phoenix Azure Deployment Script
# Deploys Phoenix to Azure AKS with all components

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
AZURE_LOCATION="${AZURE_LOCATION:-eastus}"
RESOURCE_GROUP="${RESOURCE_GROUP:-phoenix-vnext-rg}"
CLUSTER_NAME="${CLUSTER_NAME:-phoenix-aks}"
ENVIRONMENT="${ENVIRONMENT:-dev}"
TERRAFORM_DIR="infrastructure/terraform/environments/azure"
HELM_RELEASE="phoenix"
NAMESPACE="phoenix-system"

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                  Phoenix Azure Deployment                        ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════════╝${NC}"
echo

# Check prerequisites
check_prerequisites() {
    echo -e "${YELLOW}Checking prerequisites...${NC}"
    
    local missing=false
    
    # Check Azure CLI
    if ! command -v az &> /dev/null; then
        echo -e "${RED}✗ Azure CLI not found${NC}"
        missing=true
    else
        echo -e "${GREEN}✓ Azure CLI${NC}"
        # Check Azure login
        if ! az account show &> /dev/null; then
            echo -e "${RED}✗ Not logged in to Azure${NC}"
            missing=true
        else
            echo -e "${GREEN}✓ Azure authenticated${NC}"
        fi
    fi
    
    # Check Terraform
    if ! command -v terraform &> /dev/null; then
        echo -e "${RED}✗ Terraform not found${NC}"
        missing=true
    else
        echo -e "${GREEN}✓ Terraform${NC}"
    fi
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        echo -e "${RED}✗ kubectl not found${NC}"
        missing=true
    else
        echo -e "${GREEN}✓ kubectl${NC}"
    fi
    
    # Check Helm
    if ! command -v helm &> /dev/null; then
        echo -e "${RED}✗ Helm not found${NC}"
        missing=true
    else
        echo -e "${GREEN}✓ Helm${NC}"
    fi
    
    if [ "$missing" = true ]; then
        echo -e "\n${RED}Please install missing prerequisites before continuing.${NC}"
        echo "Installation guides:"
        echo "- Azure CLI: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli"
        echo "- Terraform: https://www.terraform.io/downloads"
        echo "- kubectl: https://kubernetes.io/docs/tasks/tools/"
        echo "- Helm: https://helm.sh/docs/intro/install/"
        exit 1
    fi
}

# Deploy infrastructure with Terraform
deploy_infrastructure() {
    echo -e "\n${YELLOW}Deploying Azure infrastructure...${NC}"
    
    cd "$TERRAFORM_DIR"
    
    # Initialize Terraform
    echo -e "${BLUE}Initializing Terraform...${NC}"
    terraform init
    
    # Get current user's object ID for AKS admin
    USER_OBJECT_ID=$(az ad signed-in-user show --query id -o tsv)
    
    # Create terraform.tfvars
    cat > terraform.tfvars <<EOF
azure_location = "${AZURE_LOCATION}"
environment = "${ENVIRONMENT}"
cluster_name = "${CLUSTER_NAME}"
project_name = "phoenix-vnext"
node_vm_size = "Standard_D4s_v3"
aks_admin_group_ids = ["${USER_OBJECT_ID}"]
EOF
    
    # Plan deployment
    echo -e "${BLUE}Planning infrastructure deployment...${NC}"
    terraform plan -out=tfplan
    
    # Apply deployment
    read -p "Deploy infrastructure? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${BLUE}Deploying infrastructure...${NC}"
        terraform apply tfplan
    else
        echo -e "${RED}Deployment cancelled.${NC}"
        exit 1
    fi
    
    # Get outputs
    ACR_LOGIN_SERVER=$(terraform output -raw acr_login_server)
    STORAGE_ACCOUNT=$(terraform output -raw storage_account_name)
    APP_INSIGHTS_KEY=$(terraform output -raw application_insights_key)
    
    cd - > /dev/null
}

# Configure kubectl
configure_kubectl() {
    echo -e "\n${YELLOW}Configuring kubectl...${NC}"
    
    az aks get-credentials \
        --resource-group "${RESOURCE_GROUP}" \
        --name "${CLUSTER_NAME}" \
        --overwrite-existing
    
    # Verify connection
    if kubectl cluster-info &> /dev/null; then
        echo -e "${GREEN}✓ Successfully connected to AKS cluster${NC}"
    else
        echo -e "${RED}✗ Failed to connect to AKS cluster${NC}"
        exit 1
    fi
}

# Install NGINX Ingress Controller
install_nginx_ingress() {
    echo -e "\n${YELLOW}Installing NGINX Ingress Controller...${NC}"
    
    helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
    helm repo update
    
    helm upgrade --install ingress-nginx ingress-nginx/ingress-nginx \
        --namespace ingress-nginx \
        --create-namespace \
        --set controller.service.type=LoadBalancer \
        --set controller.service.annotations."service\.beta\.kubernetes\.io/azure-load-balancer-health-probe-request-path"=/healthz
}

# Install Azure File CSI Driver (if needed)
install_azure_file_csi() {
    echo -e "\n${YELLOW}Checking Azure File CSI Driver...${NC}"
    
    # AKS includes CSI drivers by default in recent versions
    if kubectl get storageclass azurefile-csi &> /dev/null; then
        echo -e "${GREEN}✓ Azure File CSI Driver already installed${NC}"
    else
        echo -e "${YELLOW}Installing Azure File CSI Driver...${NC}"
        kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/install-driver.sh
    fi
}

# Create storage class for Phoenix
create_storage_class() {
    echo -e "\n${YELLOW}Creating storage classes...${NC}"
    
    cat <<EOF | kubectl apply -f -
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: phoenix-premium
provisioner: disk.csi.azure.com
parameters:
  skuName: Premium_LRS
reclaimPolicy: Retain
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: phoenix-files
provisioner: file.csi.azure.com
parameters:
  skuName: Standard_LRS
  storageAccount: ${STORAGE_ACCOUNT}
reclaimPolicy: Retain
volumeBindingMode: Immediate
allowVolumeExpansion: true
mountOptions:
  - dir_mode=0777
  - file_mode=0777
  - uid=65534
  - gid=65534
  - mfsymlinks
  - cache=strict
  - actimeo=30
EOF
}

# Deploy Phoenix
deploy_phoenix() {
    echo -e "\n${YELLOW}Deploying Phoenix application...${NC}"
    
    # Create namespace
    kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -
    
    # Add Helm repositories
    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
    helm repo add grafana https://grafana.github.io/helm-charts
    helm repo update
    
    # Get ingress IP
    INGRESS_IP=$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "pending")
    
    # Create values file for Azure
    cat > azure-values.yaml <<EOF
global:
  cloudProvider: azure
  domain: "${INGRESS_IP}.nip.io"

collector:
  service:
    type: LoadBalancer
    annotations:
      service.beta.kubernetes.io/azure-load-balancer-internal: "false"
      service.beta.kubernetes.io/azure-load-balancer-health-probe-request-path: "/health"

storage:
  controlSignals:
    storageClass: phoenix-files
  benchmarkData:
    storageClass: phoenix-premium

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: phoenix.${INGRESS_IP}.nip.io
      paths:
        - path: /
          pathType: Prefix
          service: grafana

azure:
  location: ${AZURE_LOCATION}
  workloadIdentityEnabled: true
  storageAccount: ${STORAGE_ACCOUNT}
  resourceGroup: ${RESOURCE_GROUP}

prometheus:
  server:
    persistentVolume:
      storageClass: phoenix-premium

grafana:
  persistence:
    storageClassName: phoenix-premium
  
  env:
    AZURE_DEFAULT_SUBSCRIPTION_ID: $(az account show --query id -o tsv)
    
  grafana.ini:
    azure:
      managed_identity_enabled = true
EOF
    
    # Deploy Phoenix
    echo -e "${BLUE}Installing Phoenix with Helm...${NC}"
    helm upgrade --install "${HELM_RELEASE}" ./infrastructure/helm/phoenix \
        --namespace "${NAMESPACE}" \
        --values azure-values.yaml \
        --wait \
        --timeout 10m
    
    rm -f azure-values.yaml
}

# Get deployment info
get_deployment_info() {
    echo -e "\n${BLUE}═══════════════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}Phoenix deployed successfully on Azure!${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════════${NC}"
    
    # Get Load Balancer IP
    LB_IP=$(kubectl get svc phoenix-collector -n "${NAMESPACE}" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "pending")
    INGRESS_IP=$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "pending")
    
    echo -e "\n${YELLOW}Access Information:${NC}"
    echo -e "  • OTLP Endpoint: ${LB_IP}:4318"
    echo -e "  • Grafana: http://phoenix.${INGRESS_IP}.nip.io"
    echo -e "  • Prometheus: http://phoenix.${INGRESS_IP}.nip.io/prometheus"
    
    echo -e "\n${YELLOW}Azure Resources:${NC}"
    echo -e "  • Resource Group: ${RESOURCE_GROUP}"
    echo -e "  • AKS Cluster: ${CLUSTER_NAME}"
    echo -e "  • Container Registry: ${ACR_LOGIN_SERVER}"
    echo -e "  • Storage Account: ${STORAGE_ACCOUNT}"
    
    echo -e "\n${YELLOW}Useful Commands:${NC}"
    echo -e "  • Check pods: kubectl get pods -n ${NAMESPACE}"
    echo -e "  • View logs: kubectl logs -n ${NAMESPACE} -l app.kubernetes.io/name=phoenix-collector"
    echo -e "  • Port forward Grafana: kubectl port-forward -n ${NAMESPACE} svc/phoenix-grafana 3000:80"
    echo -e "  • View Azure Monitor: az monitor metrics list --resource \$AKS_RESOURCE_ID"
    
    echo -e "\n${YELLOW}Next Steps:${NC}"
    echo -e "  1. Wait for Load Balancer IPs to be assigned (2-3 minutes)"
    echo -e "  2. Configure your OTLP exporters to send to: ${LB_IP}:4318"
    echo -e "  3. Access Grafana dashboard at: http://phoenix.${INGRESS_IP}.nip.io"
    echo -e "  4. Enable Azure Monitor integration in Grafana"
    
    # Save deployment info
    cat > phoenix-azure-deployment.txt <<EOF
Phoenix Azure Deployment Information
====================================
Date: $(date)
Location: ${AZURE_LOCATION}
Resource Group: ${RESOURCE_GROUP}
Cluster: ${CLUSTER_NAME}
Namespace: ${NAMESPACE}

OTLP Endpoint: ${LB_IP}:4318
Grafana: http://phoenix.${INGRESS_IP}.nip.io
Prometheus: http://phoenix.${INGRESS_IP}.nip.io/prometheus

Azure Resources:
- Container Registry: ${ACR_LOGIN_SERVER}
- Storage Account: ${STORAGE_ACCOUNT}
- Application Insights Key: ${APP_INSIGHTS_KEY}

To delete deployment:
- Helm: helm uninstall ${HELM_RELEASE} -n ${NAMESPACE}
- Infrastructure: cd ${TERRAFORM_DIR} && terraform destroy
EOF
    
    echo -e "\n${GREEN}Deployment information saved to: phoenix-azure-deployment.txt${NC}"
}

# Main deployment flow
main() {
    check_prerequisites
    
    # Check if resource group exists
    if az group show --name "${RESOURCE_GROUP}" &> /dev/null; then
        echo -e "${GREEN}Resource group already exists. Checking for AKS cluster...${NC}"
        if az aks show --resource-group "${RESOURCE_GROUP}" --name "${CLUSTER_NAME}" &> /dev/null; then
            echo -e "${GREEN}AKS cluster already exists. Skipping infrastructure deployment.${NC}"
            # Get terraform outputs
            cd "$TERRAFORM_DIR"
            ACR_LOGIN_SERVER=$(terraform output -raw acr_login_server 2>/dev/null || echo "")
            STORAGE_ACCOUNT=$(terraform output -raw storage_account_name 2>/dev/null || echo "")
            APP_INSIGHTS_KEY=$(terraform output -raw application_insights_key 2>/dev/null || echo "")
            cd - > /dev/null
        else
            deploy_infrastructure
        fi
    else
        deploy_infrastructure
    fi
    
    configure_kubectl
    install_nginx_ingress
    install_azure_file_csi
    create_storage_class
    deploy_phoenix
    get_deployment_info
}

# Handle arguments
case "${1:-}" in
    destroy)
        echo -e "${YELLOW}Destroying Phoenix deployment...${NC}"
        helm uninstall "${HELM_RELEASE}" -n "${NAMESPACE}" || true
        kubectl delete namespace "${NAMESPACE}" || true
        cd "${TERRAFORM_DIR}" && terraform destroy
        ;;
    info)
        get_deployment_info
        ;;
    *)
        main
        ;;
esac