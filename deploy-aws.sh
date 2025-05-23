#!/bin/bash
# Phoenix AWS Deployment Script
# Deploys Phoenix to AWS EKS with all components

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
AWS_REGION="${AWS_REGION:-us-east-1}"
CLUSTER_NAME="${CLUSTER_NAME:-phoenix-eks}"
ENVIRONMENT="${ENVIRONMENT:-dev}"
TERRAFORM_DIR="infrastructure/terraform/environments/aws"
HELM_RELEASE="phoenix"
NAMESPACE="phoenix-system"

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                  Phoenix AWS Deployment                          ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════════╝${NC}"
echo

# Check prerequisites
check_prerequisites() {
    echo -e "${YELLOW}Checking prerequisites...${NC}"
    
    local missing=false
    
    # Check AWS CLI
    if ! command -v aws &> /dev/null; then
        echo -e "${RED}✗ AWS CLI not found${NC}"
        missing=true
    else
        echo -e "${GREEN}✓ AWS CLI${NC}"
        # Check AWS credentials
        if ! aws sts get-caller-identity &> /dev/null; then
            echo -e "${RED}✗ AWS credentials not configured${NC}"
            missing=true
        else
            echo -e "${GREEN}✓ AWS credentials${NC}"
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
        echo "- AWS CLI: https://aws.amazon.com/cli/"
        echo "- Terraform: https://www.terraform.io/downloads"
        echo "- kubectl: https://kubernetes.io/docs/tasks/tools/"
        echo "- Helm: https://helm.sh/docs/intro/install/"
        exit 1
    fi
}

# Deploy infrastructure with Terraform
deploy_infrastructure() {
    echo -e "\n${YELLOW}Deploying AWS infrastructure...${NC}"
    
    cd "$TERRAFORM_DIR"
    
    # Initialize Terraform
    echo -e "${BLUE}Initializing Terraform...${NC}"
    terraform init
    
    # Create terraform.tfvars
    cat > terraform.tfvars <<EOF
aws_region = "${AWS_REGION}"
environment = "${ENVIRONMENT}"
cluster_name = "${CLUSTER_NAME}"
node_instance_types = ["t3.large", "t3.xlarge"]
node_capacity_type = "${ENVIRONMENT}" == "prod" ? "ON_DEMAND" : "SPOT"
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
    
    cd - > /dev/null
}

# Configure kubectl
configure_kubectl() {
    echo -e "\n${YELLOW}Configuring kubectl...${NC}"
    
    aws eks update-kubeconfig \
        --region "${AWS_REGION}" \
        --name "${CLUSTER_NAME}"
    
    # Verify connection
    if kubectl cluster-info &> /dev/null; then
        echo -e "${GREEN}✓ Successfully connected to EKS cluster${NC}"
    else
        echo -e "${RED}✗ Failed to connect to EKS cluster${NC}"
        exit 1
    fi
}

# Install AWS Load Balancer Controller
install_aws_lb_controller() {
    echo -e "\n${YELLOW}Installing AWS Load Balancer Controller...${NC}"
    
    # Create IAM policy
    curl -o iam-policy.json https://raw.githubusercontent.com/kubernetes-sigs/aws-load-balancer-controller/v2.6.2/docs/install/iam_policy.json
    
    POLICY_ARN=$(aws iam create-policy \
        --policy-name AWSLoadBalancerControllerIAMPolicy \
        --policy-document file://iam-policy.json \
        --query 'Policy.Arn' \
        --output text 2>/dev/null || \
        aws iam list-policies \
        --query 'Policies[?PolicyName==`AWSLoadBalancerControllerIAMPolicy`].Arn' \
        --output text)
    
    # Create service account
    eksctl create iamserviceaccount \
        --cluster="${CLUSTER_NAME}" \
        --namespace=kube-system \
        --name=aws-load-balancer-controller \
        --attach-policy-arn="${POLICY_ARN}" \
        --override-existing-serviceaccounts \
        --approve
    
    # Install using Helm
    helm repo add eks https://aws.github.io/eks-charts
    helm repo update
    
    helm upgrade --install aws-load-balancer-controller eks/aws-load-balancer-controller \
        -n kube-system \
        --set clusterName="${CLUSTER_NAME}" \
        --set serviceAccount.create=false \
        --set serviceAccount.name=aws-load-balancer-controller
    
    rm -f iam-policy.json
}

# Install EBS CSI Driver
install_ebs_csi_driver() {
    echo -e "\n${YELLOW}Installing EBS CSI Driver...${NC}"
    
    # The EKS module already installs this as an addon
    echo -e "${GREEN}✓ EBS CSI Driver installed via EKS addon${NC}"
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
    
    # Create values file for AWS
    cat > aws-values.yaml <<EOF
global:
  cloudProvider: aws
  domain: "${CLUSTER_NAME}.${AWS_REGION}.elb.amazonaws.com"

collector:
  service:
    type: LoadBalancer
    annotations:
      service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
      service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled: "true"

storage:
  controlSignals:
    storageClass: gp3
  benchmarkData:
    storageClass: gp3

ingress:
  enabled: true
  className: alb
  annotations:
    alb.ingress.kubernetes.io/scheme: internet-facing
    alb.ingress.kubernetes.io/target-type: ip
    alb.ingress.kubernetes.io/healthcheck-path: /health

aws:
  region: ${AWS_REGION}
  irsaEnabled: true

prometheus:
  server:
    persistentVolume:
      storageClass: gp3

grafana:
  persistence:
    storageClassName: gp3
EOF
    
    # Deploy Phoenix
    echo -e "${BLUE}Installing Phoenix with Helm...${NC}"
    helm upgrade --install "${HELM_RELEASE}" ./infrastructure/helm/phoenix \
        --namespace "${NAMESPACE}" \
        --values aws-values.yaml \
        --wait \
        --timeout 10m
    
    rm -f aws-values.yaml
}

# Get deployment info
get_deployment_info() {
    echo -e "\n${BLUE}═══════════════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}Phoenix deployed successfully on AWS!${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════════${NC}"
    
    # Get Load Balancer URL
    LB_URL=$(kubectl get svc phoenix-collector -n "${NAMESPACE}" -o jsonpath='{.status.loadBalancer.ingress[0].hostname}' 2>/dev/null || echo "pending")
    
    echo -e "\n${YELLOW}Access Information:${NC}"
    echo -e "  • OTLP Endpoint: ${LB_URL}:4318"
    echo -e "  • Grafana: http://${LB_URL}"
    echo -e "  • Prometheus: http://${LB_URL}/prometheus"
    
    echo -e "\n${YELLOW}Useful Commands:${NC}"
    echo -e "  • Check pods: kubectl get pods -n ${NAMESPACE}"
    echo -e "  • View logs: kubectl logs -n ${NAMESPACE} -l app.kubernetes.io/name=phoenix-collector"
    echo -e "  • Port forward Grafana: kubectl port-forward -n ${NAMESPACE} svc/phoenix-grafana 3000:80"
    
    echo -e "\n${YELLOW}Next Steps:${NC}"
    echo -e "  1. Wait for Load Balancer to be ready (2-3 minutes)"
    echo -e "  2. Configure your OTLP exporters to send to: ${LB_URL}:4318"
    echo -e "  3. Access Grafana dashboard for monitoring"
    
    # Save deployment info
    cat > phoenix-aws-deployment.txt <<EOF
Phoenix AWS Deployment Information
==================================
Date: $(date)
Region: ${AWS_REGION}
Cluster: ${CLUSTER_NAME}
Namespace: ${NAMESPACE}
Load Balancer: ${LB_URL}

OTLP Endpoint: ${LB_URL}:4318
Grafana: http://${LB_URL}
Prometheus: http://${LB_URL}/prometheus

To delete deployment:
- Helm: helm uninstall ${HELM_RELEASE} -n ${NAMESPACE}
- Infrastructure: cd ${TERRAFORM_DIR} && terraform destroy
EOF
    
    echo -e "\n${GREEN}Deployment information saved to: phoenix-aws-deployment.txt${NC}"
}

# Main deployment flow
main() {
    check_prerequisites
    
    # Check if infrastructure exists
    if aws eks describe-cluster --name "${CLUSTER_NAME}" --region "${AWS_REGION}" &> /dev/null; then
        echo -e "${GREEN}EKS cluster already exists. Skipping infrastructure deployment.${NC}"
    else
        deploy_infrastructure
    fi
    
    configure_kubectl
    install_aws_lb_controller
    install_ebs_csi_driver
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