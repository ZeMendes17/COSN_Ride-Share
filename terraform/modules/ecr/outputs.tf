output "request_service_repository_url" {
  description = "ECR repository URL for request-service"
  value       = aws_ecr_repository.request_service.repository_url
}

output "request_service_repository_arn" {
  description = "ECR repository ARN for request-service"
  value       = aws_ecr_repository.request_service.arn
}

output "matching_service_repository_url" {
  description = "ECR repository URL for matching-service"
  value       = aws_ecr_repository.matching_service.repository_url
}

output "matching_service_repository_arn" {
  description = "ECR repository ARN for matching-service"
  value       = aws_ecr_repository.matching_service.arn
}

