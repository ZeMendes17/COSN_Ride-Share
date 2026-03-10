output "grafana_cloudwatch_role_arn" {
  description = "ARN of the IAM role for Grafana CloudWatch access"
  value       = aws_iam_role.grafana_cloudwatch.arn
}

output "grafana_service_account_annotation" {
  description = "Annotation to add to Grafana ServiceAccount"
  value       = "eks.amazonaws.com/role-arn: ${aws_iam_role.grafana_cloudwatch.arn}"
}

output "prometheus_endpoint" {
  description = "Prometheus service endpoint"
  value       = "prometheus-kube-prometheus-prometheus.monitoring.svc.cluster.local:9090"
}

output "grafana_service" {
  description = "Grafana service name"
  value       = "prometheus-grafana"
}

output "namespace" {
  description = "Monitoring namespace"
  value       = "carpooling"
}

