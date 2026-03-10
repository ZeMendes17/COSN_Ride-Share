variable "project_name" {
  description = "Project name"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "eks_cluster_oidc_issuer" {
  description = "OIDC issuer URL for EKS cluster"
  type        = string
}

variable "eks_cluster_oidc_arn" {
  description = "OIDC provider ARN for EKS cluster"
  type        = string
}

variable "rds_secret_arn" {
  description = "ARN of the RDS credentials secret"
  type        = string
}

