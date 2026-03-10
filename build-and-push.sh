#!/bin/bash

# Build and Push Docker Images to ECR
# Run this before deploying if you haven't set up GitLab CI/CD yet

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  Build & Push Docker Images to ECR${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Check if docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Error: Docker is not running${NC}"
    echo "Please start Docker Desktop and try again"
    exit 1
fi

# Load environment variables from .env if it exists
if [ -f ".env" ]; then
    set -a
    source .env
    set +a
fi

# Get ECR registry from terraform outputs
if [ -f "terraform/outputs.json" ]; then
    export ECR_REGISTRY=$(cd terraform && terraform output -raw ecr_request_service_url | cut -d'/' -f1)
else
    echo -e "${RED}❌ Error: terraform/outputs.json not found${NC}"
    echo "Please run: cd terraform && terraform output -json > outputs.json"
    exit 1
fi

echo -e "${YELLOW}ECR Registry:${NC} $ECR_REGISTRY"
echo ""

# Login to ECR
echo -e "${BLUE}Step 1: Login to ECR${NC}"
aws ecr get-login-password --region ${AWS_DEFAULT_REGION:-us-east-1} | \
    docker login --username AWS --password-stdin $ECR_REGISTRY

if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Failed to login to ECR${NC}"
    echo "Please check your AWS credentials"
    exit 1
fi
echo -e "${GREEN}✓${NC} Logged in to ECR"
echo ""

# Build and push request-service
echo -e "${BLUE}Step 2: Build Request Service${NC}"
cd request-service
docker build -t $ECR_REGISTRY/g03-carpooling/production/request-service:latest .
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Failed to build request-service${NC}"
    exit 1
fi
echo -e "${GREEN}✓${NC} Built request-service"

echo -e "${YELLOW}Pushing request-service to ECR...${NC}"
docker push $ECR_REGISTRY/g03-carpooling/production/request-service:latest
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Failed to push request-service${NC}"
    exit 1
fi
echo -e "${GREEN}✓${NC} Pushed request-service"
cd ..
echo ""

# Build and push matching-service
echo -e "${BLUE}Step 3: Build Matching Service${NC}"
cd matching-service
docker build -t $ECR_REGISTRY/g03-carpooling/production/matching-service:latest .
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Failed to build matching-service${NC}"
    exit 1
fi
echo -e "${GREEN}✓${NC} Built matching-service"

echo -e "${YELLOW}Pushing matching-service to ECR...${NC}"
docker push $ECR_REGISTRY/g03-carpooling/production/matching-service:latest
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Failed to push matching-service${NC}"
    exit 1
fi
echo -e "${GREEN}✓${NC} Pushed matching-service"
cd ..
echo ""

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ All images built and pushed successfully!${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

echo -e "${YELLOW}Images in ECR:${NC}"
echo "  • $ECR_REGISTRY/g03-carpooling/production/request-service:latest"
echo "  • $ECR_REGISTRY/g03-carpooling/production/matching-service:latest"
echo ""

echo -e "${GREEN}Next step:${NC} Deploy services with: ${BLUE}./deploy-services.sh${NC}"

