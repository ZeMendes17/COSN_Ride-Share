# Extract OIDC provider ID from URL
locals {
  oidc_provider_id = replace(var.eks_cluster_oidc_issuer, "https://", "")
}

# IAM Role for Request Service
resource "aws_iam_role" "request_service" {
  name = "${var.project_name}-${var.environment}-request-service"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Federated = var.eks_cluster_oidc_arn
        }
        Action = "sts:AssumeRoleWithWebIdentity"
        Condition = {
          StringEquals = {
            "${local.oidc_provider_id}:sub" = "system:serviceaccount:carpooling:request-service"
            "${local.oidc_provider_id}:aud" = "sts.amazonaws.com"
          }
        }
      }
    ]
  })
}

# IAM Policy for Request Service - Secrets Manager Access
resource "aws_iam_role_policy" "request_service_secrets" {
  name = "secrets-manager-access"
  role = aws_iam_role.request_service.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue",
          "secretsmanager:DescribeSecret"
        ]
        Resource = var.rds_secret_arn
      }
    ]
  })
}

# IAM Policy for Request Service - SNS Publish Access
resource "aws_iam_role_policy" "request_service_sns" {
  name = "sns-publish-access"
  role = aws_iam_role.request_service.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "sns:Publish"
        ]
        Resource = [
          "arn:aws:sns:us-east-1:670636167354:match_created_event",
          "arn:aws:sns:us-east-1:670636167354:match_cancelled_event"
        ]
      }
    ]
  })
}

# IAM Role for Matching Service
resource "aws_iam_role" "matching_service" {
  name = "${var.project_name}-${var.environment}-matching-service"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Federated = var.eks_cluster_oidc_arn
        }
        Action = "sts:AssumeRoleWithWebIdentity"
        Condition = {
          StringEquals = {
            "${local.oidc_provider_id}:sub" = "system:serviceaccount:carpooling:matching-service"
            "${local.oidc_provider_id}:aud" = "sts.amazonaws.com"
          }
        }
      }
    ]
  })
}

# IAM Policy for Matching Service - Secrets Manager Access
resource "aws_iam_role_policy" "matching_service_secrets" {
  name = "secrets-manager-access"
  role = aws_iam_role.matching_service.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue",
          "secretsmanager:DescribeSecret"
        ]
        Resource = var.rds_secret_arn
      }
    ]
  })
}

# Note: SQS consume from external account uses external credentials
# No IAM policy needed here

# IAM Role for GitLab CI/CD
resource "aws_iam_role" "gitlab_ci" {
  name = "${var.project_name}-${var.environment}-gitlab-ci"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Service = "eks.amazonaws.com"
        }
        Action = "sts:AssumeRole"
      }
    ]
  })
}

# IAM Policy for GitLab CI - ECR and EKS Access
resource "aws_iam_role_policy" "gitlab_ci" {
  name = "gitlab-ci-access"
  role = aws_iam_role.gitlab_ci.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ecr:GetAuthorizationToken",
          "ecr:BatchCheckLayerAvailability",
          "ecr:GetDownloadUrlForLayer",
          "ecr:BatchGetImage",
          "ecr:PutImage",
          "ecr:InitiateLayerUpload",
          "ecr:UploadLayerPart",
          "ecr:CompleteLayerUpload"
        ]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "eks:DescribeCluster",
          "eks:ListClusters"
        ]
        Resource = "*"
      }
    ]
  })
}

