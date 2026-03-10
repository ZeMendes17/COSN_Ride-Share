environment             = "production"
aws_region             = "us-east-1"
project_name           = "g03-carpooling"
vpc_cidr               = "10.0.0.0/16"

# EKS Configuration
eks_cluster_version    = "1.34"
eks_node_instance_types = ["c7i-flex.large"]
eks_node_desired_size  = 2
eks_node_min_size      = 1
eks_node_max_size      = 5

# RDS PostgreSQL Configuration (Free Tier)
database_name           = "carpooling"
db_master_username      = "dbadmin"
aurora_instance_class   = "db.t4g.micro"
aurora_instance_count   = 1
aurora_backup_retention = 1

# SNS/SQS endpoints are configured in Kubernetes ConfigMaps
# No Kafka configuration needed here

tags = {
  Project     = "g03-carpooling"
  Environment = "production"
  Team        = "platform"
  CostCenter  = "engineering"
}

