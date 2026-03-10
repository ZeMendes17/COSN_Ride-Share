#!/bin/bash

# Terraform Deployment Script
# This script helps you deploy the infrastructure step by step

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR/terraform"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  G03 Carpooling Infrastructure Deployment${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Step 1: Check for .env file
if [ ! -f "../.env" ]; then
    echo -e "${RED}❌ Error: .env file not found!${NC}"
    echo ""
    echo -e "${YELLOW}Please create .env file:${NC}"
    echo "  1. Copy .env.example to .env"
    echo "     cp .env.example .env"
    echo ""
    echo "  2. Edit .env with your AWS credentials:"
    echo "     - AWS_ACCESS_KEY_ID"
    echo "     - AWS_SECRET_ACCESS_KEY"
    echo "     - AWS_ACCOUNT_ID"
    echo "     - TF_VAR_kafka_brokers (from 3rd party)"
    echo ""
    exit 1
fi

# Step 2: Load environment variables
echo -e "${GREEN}✓${NC} Loading environment variables from .env..."
set -a
source ../.env
set +a

# Step 3: Verify required variables
echo -e "${GREEN}✓${NC} Verifying AWS credentials..."
if [ -z "$AWS_ACCESS_KEY_ID" ] || [ -z "$AWS_SECRET_ACCESS_KEY" ]; then
    echo -e "${RED}❌ Error: AWS credentials not set in .env${NC}"
    exit 1
fi

# Step 4: Verify AWS region
echo -e "${GREEN}✓${NC} Using AWS region: ${AWS_DEFAULT_REGION:-us-east-1}"

# Step 5: Initialize Terraform
echo ""
echo -e "${BLUE}Step 1: Terraform Init${NC}"
echo -e "${YELLOW}Running: terraform init${NC}"
terraform init

# Step 6: Select or create workspace
echo ""
echo -e "${BLUE}Step 2: Terraform Workspace${NC}"
WORKSPACE="${TF_VAR_environment:-production}"
if terraform workspace list | grep -q "$WORKSPACE"; then
    echo -e "${GREEN}✓${NC} Selecting existing workspace: $WORKSPACE"
    terraform workspace select "$WORKSPACE"
else
    echo -e "${GREEN}✓${NC} Creating new workspace: $WORKSPACE"
    terraform workspace new "$WORKSPACE"
fi

# Step 7: Run terraform plan
echo ""
echo -e "${BLUE}Step 3: Terraform Plan${NC}"
echo -e "${YELLOW}Running: terraform plan${NC}"
echo ""
terraform plan -out=tfplan

# Step 8: Ask for confirmation
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}Review the plan above.${NC}"
echo ""
read -p "Do you want to apply these changes? (yes/no): " -r
echo ""

if [ "$REPLY" != "yes" ]; then
    echo -e "${YELLOW}Deployment cancelled.${NC}"
    rm -f tfplan
    exit 0
fi

# Step 9: Apply terraform
echo ""
echo -e "${BLUE}Step 4: Terraform Apply${NC}"
echo -e "${YELLOW}Running: terraform apply${NC}"
echo ""
terraform apply tfplan

# Step 10: Save outputs
echo ""
echo -e "${GREEN}✓${NC} Saving Terraform outputs..."
terraform output -json > outputs.json
terraform output > outputs.txt

# Step 11: Display key outputs
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Deployment Complete!${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${YELLOW}Key Information:${NC}"
echo ""
echo "EKS Cluster:"
echo "  Name: $(terraform output -raw eks_cluster_name)"
echo "  Endpoint: $(terraform output -raw eks_cluster_endpoint)"
echo ""
echo "Aurora Database:"
echo "  Host: $(terraform output -raw aurora_cluster_endpoint)"
echo "  Database: $(terraform output -raw aurora_database_name)"
echo "  Secret ARN: $(terraform output -raw aurora_secret_arn)"
echo ""
echo "ECR Repositories:"
echo "  Request Service: $(terraform output -raw ecr_request_service_url)"
echo "  Matching Service: $(terraform output -raw ecr_matching_service_url)"
echo ""
echo -e "${YELLOW}Next Steps:${NC}"
echo ""
echo "1. Configure kubectl:"
echo "   aws eks update-kubeconfig --name $(terraform output -raw eks_cluster_name) --region $AWS_DEFAULT_REGION"
echo ""
echo "2. Set GitLab CI/CD variables (see outputs.txt or GITLAB_SECRETS.md)"
echo ""
echo "3. Deploy services via GitLab CI/CD or manually:"
echo "   cd ../kubernetes"
echo "   kubectl apply -f ."
echo ""
echo "4. Get ALB URL after deployment:"
echo "   kubectl get ingress carpooling-ingress -n carpooling"
echo ""

# Step 12: Offer to configure kubectl
echo ""
read -p "Configure kubectl now? (y/n) " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${GREEN}✓${NC} Configuring kubectl..."
    aws eks update-kubeconfig --name $(terraform output -raw eks_cluster_name) --region $AWS_DEFAULT_REGION
    echo -e "${GREEN}✓${NC} kubectl configured!"
    echo ""
    kubectl get nodes
fi

echo ""
echo -e "${GREEN}All done! 🎉${NC}"

