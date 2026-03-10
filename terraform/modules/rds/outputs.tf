output "instance_identifier" {
  description = "RDS instance identifier"
  value       = aws_db_instance.main.id
}

output "cluster_endpoint" {
  description = "RDS instance endpoint (compatibility name)"
  value       = aws_db_instance.main.address
}

output "reader_endpoint" {
  description = "RDS instance endpoint (same as writer for single instance)"
  value       = aws_db_instance.main.address
}

output "cluster_port" {
  description = "RDS instance port"
  value       = aws_db_instance.main.port
}

output "database_name" {
  description = "Database name"
  value       = aws_db_instance.main.db_name
}

output "master_username" {
  description = "Master username"
  value       = aws_db_instance.main.username
  sensitive   = true
}

output "secret_arn" {
  description = "ARN of the secret containing database credentials"
  value       = aws_secretsmanager_secret.db_credentials.arn
}

output "security_group_id" {
  description = "Security group ID for RDS instance"
  value       = aws_security_group.rds.id
}

