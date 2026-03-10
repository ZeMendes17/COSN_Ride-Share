output "vpc_id" {
  description = "VPC ID"
  value       = module.vpc.vpc_id
}

output "private_subnet_ids" {
  description = "Private subnet IDs"
  value       = module.vpc.private_subnet_ids
}

output "eks_cluster_name" {
  description = "EKS cluster name"
  value       = module.eks.cluster_name
}

output "eks_cluster_endpoint" {
  description = "EKS cluster endpoint"
  value       = module.eks.cluster_endpoint
}

output "eks_cluster_oidc_issuer_url" {
  description = "OIDC issuer URL for the EKS cluster"
  value       = module.eks.cluster_oidc_issuer_url
}

output "aurora_cluster_endpoint" {
  description = "Aurora cluster writer endpoint"
  value       = module.rds.cluster_endpoint
}

output "aurora_reader_endpoint" {
  description = "Aurora cluster reader endpoint"
  value       = module.rds.reader_endpoint
}

output "aurora_database_name" {
  description = "Aurora database name"
  value       = module.rds.database_name
}

output "aurora_secret_arn" {
  description = "ARN of the secret containing database credentials"
  value       = module.rds.secret_arn
}

output "aurora_security_group_id" {
  description = "Security group ID for Aurora cluster"
  value       = module.rds.security_group_id
}

output "ecr_request_service_url" {
  description = "ECR repository URL for request-service"
  value       = module.ecr.request_service_repository_url
}

output "ecr_matching_service_url" {
  description = "ECR repository URL for matching-service"
  value       = module.ecr.matching_service_repository_url
}

output "request_service_role_arn" {
  description = "IAM role ARN for request-service"
  value       = module.iam.request_service_role_arn
}

output "matching_service_role_arn" {
  description = "IAM role ARN for matching-service"
  value       = module.iam.matching_service_role_arn
}

output "alb_controller_role_arn" {
  description = "IAM role ARN for AWS Load Balancer Controller"
  value       = module.iam.alb_controller_role_arn
}

output "grafana_cloudwatch_role_arn" {
  description = "IAM role ARN for Grafana CloudWatch access"
  value       = module.monitoring.grafana_cloudwatch_role_arn
}
