#!/bin/bash

# Service Deployment Script
# Deploys services to EKS cluster after infrastructure is provisioned

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  G03 Carpooling Service Deployment${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Check if terraform outputs exist
if [ ! -f "terraform/outputs.json" ]; then
    echo -e "${RED}❌ Error: terraform/outputs.json not found${NC}"
    echo "Please run: cd terraform && terraform output -json > outputs.json"
    exit 1
fi

# Load environment variables from .env if it exists
if [ -f ".env" ]; then
    echo -e "${GREEN}✓${NC} Loading environment variables from .env..."
    set -a
    source .env
    set +a
fi

# Extract values from terraform outputs
echo -e "${GREEN}✓${NC} Reading Terraform outputs..."
export EKS_CLUSTER_NAME=$(cd terraform && terraform output -raw eks_cluster_name)
export ECR_REGISTRY=$(cd terraform && terraform output -raw ecr_request_service_url | cut -d'/' -f1)
export DB_HOST=$(cd terraform && terraform output -raw aurora_cluster_endpoint)
export DB_NAME=$(cd terraform && terraform output -raw aurora_database_name)
export DB_SECRET_ARN=$(cd terraform && terraform output -raw aurora_secret_arn)
export REQUEST_SERVICE_ROLE_ARN=$(cd terraform && terraform output -raw request_service_role_arn)
export MATCHING_SERVICE_ROLE_ARN=$(cd terraform && terraform output -raw matching_service_role_arn)
export IMAGE_TAG=${IMAGE_TAG:-latest}

echo ""
echo -e "${YELLOW}Configuration:${NC}"
echo "  EKS Cluster: $EKS_CLUSTER_NAME"
echo "  ECR Registry: $ECR_REGISTRY"
echo "  DB Host: $DB_HOST"
echo "  DB Name: $DB_NAME"
echo "  Image Tag: $IMAGE_TAG"
echo ""

# Step 1: Configure kubectl
echo -e "${BLUE}Step 1: Configure kubectl${NC}"
if ! command -v aws &> /dev/null; then
    echo -e "${RED}❌ AWS CLI not found${NC}"
    echo "Please install: brew install awscli"
    exit 1
fi

if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}❌ kubectl not found${NC}"
    echo "Please install: brew install kubectl"
    exit 1
fi

echo -e "${GREEN}✓${NC} Configuring kubectl for EKS cluster..."
aws eks update-kubeconfig --name $EKS_CLUSTER_NAME --region ${AWS_DEFAULT_REGION:-us-east-1}

echo -e "${GREEN}✓${NC} Verifying cluster connection..."
kubectl get nodes

# Step 2: Deploy namespace first (required before secrets)
echo ""
echo -e "${BLUE}Step 2: Deploy Namespace${NC}"
kubectl apply -f kubernetes/namespace.yaml

# Step 3: Deploy external credentials secret
echo ""
echo -e "${BLUE}Step 3: Deploy External Credentials Secret${NC}"
if [ -f "kubernetes/external-credentials-secret.yaml" ]; then
    if [ -s "kubernetes/external-credentials-secret.yaml" ]; then
        echo -e "${GREEN}✓${NC} Applying external credentials secret..."
        kubectl apply -f kubernetes/external-credentials-secret.yaml
    else
        echo -e "${YELLOW}⚠️${NC}  External credentials secret file is empty, skipping..."
    fi
else
    echo -e "${YELLOW}⚠️${NC}  No external credentials secret found, skipping..."
fi

# Step 4: Deploy services
echo ""
echo -e "${BLUE}Step 4: Deploy Services${NC}"

# Deploy Request Service
echo -e "${YELLOW}Deploying request-service...${NC}"
envsubst < kubernetes/request-service/configmap.yaml | kubectl apply -f -
envsubst < kubernetes/request-service/serviceaccount.yaml | kubectl apply -f -
envsubst < kubernetes/request-service/deployment.yaml | kubectl apply -f -
envsubst < kubernetes/request-service/service.yaml | kubectl apply -f -
envsubst < kubernetes/request-service/hpa.yaml | kubectl apply -f -

# Deploy Matching Service
echo -e "${YELLOW}Deploying matching-service...${NC}"
envsubst < kubernetes/matching-service/configmap.yaml | kubectl apply -f -
envsubst < kubernetes/matching-service/serviceaccount.yaml | kubectl apply -f -
envsubst < kubernetes/matching-service/deployment.yaml | kubectl apply -f -
envsubst < kubernetes/matching-service/service.yaml | kubectl apply -f -
envsubst < kubernetes/matching-service/hpa.yaml | kubectl apply -f -

# Deploy Ingress
echo ""
echo -e "${BLUE}Step 5: Deploy Ingress${NC}"
envsubst < kubernetes/ingress.yaml | kubectl apply -f -

# Step 6: Wait for deployments
echo ""
echo -e "${BLUE}Step 6: Wait for Deployments${NC}"
echo -e "${YELLOW}Waiting for request-service...${NC}"
kubectl rollout status deployment/request-service -n carpooling --timeout=5m || true

echo -e "${YELLOW}Waiting for matching-service...${NC}"
kubectl rollout status deployment/matching-service -n carpooling --timeout=5m || true

# Step 7: Get status
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Deployment Complete!${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

echo -e "${YELLOW}Deployment Status:${NC}"
kubectl get pods -n carpooling
echo ""

echo -e "${YELLOW}Services:${NC}"
kubectl get svc -n carpooling
echo ""

# Wait for ALB to be provisioned
echo -e "${YELLOW}Waiting for ALB to be provisioned (this may take 2-3 minutes)...${NC}"
sleep 30

echo -e "${YELLOW}Ingress:${NC}"
kubectl get ingress -n carpooling
echo ""

# Try to get ALB URL
ALB_URL=$(kubectl get ingress carpooling-ingress -n carpooling -o jsonpath='{.status.loadBalancer.ingress[0].hostname}' 2>/dev/null || echo "")

if [ -n "$ALB_URL" ]; then
    echo -e "${GREEN}✅ ALB URL:${NC} http://$ALB_URL"
    echo ""
    echo -e "${YELLOW}API Endpoints:${NC}"
    echo "  • Request Service: http://$ALB_URL/requests"
    echo "  • Matching Service: http://$ALB_URL/matches"
    echo "  • Grafana (Monitoring): http://$ALB_URL/monitor"
    echo ""
    echo -e "${YELLOW}Health Check:${NC}"
    echo "  curl http://$ALB_URL/requests/health"
else
    echo -e "${YELLOW}⚠️${NC}  ALB not ready yet. Check status with:"
    echo "  kubectl get ingress carpooling-ingress -n carpooling"
fi

echo ""
echo -e "${GREEN}All done! 🎉${NC}"
echo ""
echo -e "${YELLOW}Monitoring:${NC}"
echo "  Prometheus/Grafana already deployed via Terraform"
echo ""
echo -e "${YELLOW}Access Grafana (Monitoring UI):${NC}"
if [ -n "$ALB_URL" ]; then
    echo "  http://$ALB_URL/monitor"
else
    echo "  (ALB URL not ready yet, check again soon)"
fi
echo "  Username: admin"
echo "  Password: ChangeMe123!SecurePassword"
echo ""

