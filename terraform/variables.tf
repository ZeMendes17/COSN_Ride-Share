variable "aws_region" {
  description = "AWS region for resources"
  type        = string
  default     = "us-east-1"
}

variable "project_name" {
  description = "Project name used for resource naming"
  type        = string
  default     = "g03-carpooling"
}

variable "environment" {
  description = "Environment name (staging, production)"
  type        = string
}

variable "vpc_cidr" {
  description = "CIDR block for VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "eks_cluster_version" {
  description = "Kubernetes version for EKS cluster"
  type        = string
  default     = "1.34"
}

variable "eks_node_instance_types" {
  description = "Instance types for EKS node groups"
  type        = list(string)
  default     = ["c7i-flex.large"]
}

variable "eks_node_desired_size" {
  description = "Desired number of nodes"
  type        = number
  default     = 2
}

variable "eks_node_min_size" {
  description = "Minimum number of nodes"
  type        = number
  default     = 1
}

variable "eks_node_max_size" {
  description = "Maximum number of nodes"
  type        = number
  default     = 4
}

# Aurora RDS Variables
variable "database_name" {
  description = "Name of the database to create"
  type        = string
  default     = "carpooling"
}

variable "db_master_username" {
  description = "Master username for RDS (cannot be 'admin' - reserved word)"
  type        = string
  default     = "dbadmin"
}

variable "aurora_instance_class" {
  description = "Instance class for Aurora"
  type        = string
  default     = "db.t4g.micro"
}

variable "aurora_instance_count" {
  description = "Number of Aurora instances (1 for free tier)"
  type        = number
  default     = 1
}

variable "aurora_backup_retention" {
  description = "Backup retention period in days"
  type        = number
  default     = 7
}

# External SNS/SQS Configuration (provided by other company)
variable "sns_match_created_arn" {
  description = "SNS topic ARN for match_created_event"
  type        = string
  default     = "arn:aws:sns:us-east-1:670636167354:match_created_event"
}

variable "sns_match_cancelled_arn" {
  description = "SNS topic ARN for match_cancelled_event"
  type        = string
  default     = "arn:aws:sns:us-east-1:670636167354:match_cancelled_event"
}

variable "sqs_trip_available_url" {
  description = "SQS queue URL for offer_tripAvailable_event"
  type        = string
  default     = "https://sqs.us-east-1.amazonaws.com/670636167354/offer_tripAvailable_event-request_match_service"
}

variable "sqs_update_offer_url" {
  description = "SQS queue URL for offer_updateOffer_event"
  type        = string
  default     = "https://sqs.us-east-1.amazonaws.com/670636167354/offer_updateOffer_event-request_match_service"
}

variable "tags" {
  description = "Common tags for all resources"
  type        = map(string)
  default     = {}
}

