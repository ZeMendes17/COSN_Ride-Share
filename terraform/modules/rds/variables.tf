variable "project_name" {
  description = "Project name"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID"
  type        = string
}

variable "private_subnet_ids" {
  description = "Private subnet IDs for DB subnet group"
  type        = list(string)
}

variable "database_name" {
  description = "Name of the database to create"
  type        = string
}

variable "master_username" {
  description = "Master username for RDS (cannot be 'admin' - reserved word)"
  type        = string
}

variable "instance_class" {
  description = "Instance class for Aurora"
  type        = string
  default     = "db.t4g.micro"
}

variable "instance_count" {
  description = "Number of Aurora instances (1 for free tier)"
  type        = number
  default     = 1
}

variable "backup_retention" {
  description = "Backup retention period in days"
  type        = number
  default     = 1
}

variable "preferred_backup_window" {
  description = "Preferred backup window"
  type        = string
  default     = "03:00-04:00"
}

variable "preferred_maintenance_window" {
  description = "Preferred maintenance window"
  type        = string
  default     = "sun:04:00-sun:05:00"
}

variable "skip_final_snapshot" {
  description = "Skip final snapshot when destroying"
  type        = bool
  default     = false
}

