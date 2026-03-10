output "request_service_role_arn" {
  description = "IAM role ARN for request-service"
  value       = aws_iam_role.request_service.arn
}

output "request_service_role_name" {
  description = "IAM role name for request-service"
  value       = aws_iam_role.request_service.name
}

output "matching_service_role_arn" {
  description = "IAM role ARN for matching-service"
  value       = aws_iam_role.matching_service.arn
}

output "matching_service_role_name" {
  description = "IAM role name for matching-service"
  value       = aws_iam_role.matching_service.name
}

output "gitlab_ci_role_arn" {
  description = "IAM role ARN for GitLab CI"
  value       = aws_iam_role.gitlab_ci.arn
}

output "alb_controller_role_arn" {
  description = "IAM role ARN for AWS Load Balancer Controller"
  value       = aws_iam_role.alb_controller.arn
}
