# Generate random password for database
resource "random_password" "master" {
  length           = 32
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

# Store credentials in AWS Secrets Manager
resource "aws_secretsmanager_secret" "db_credentials" {
  name                    = "${var.project_name}-${var.environment}-db-credentials"
  description             = "RDS PostgreSQL credentials for ${var.project_name}"
  recovery_window_in_days = 7
}

resource "aws_secretsmanager_secret_version" "db_credentials" {
  secret_id = aws_secretsmanager_secret.db_credentials.id
  secret_string = jsonencode({
    username = var.master_username
    password = random_password.master.result
    engine   = "postgres"
    host     = aws_db_instance.main.address
    port     = aws_db_instance.main.port
    dbname   = var.database_name
  })
}

# DB Subnet Group
resource "aws_db_subnet_group" "main" {
  name       = "${var.project_name}-${var.environment}-db-subnet"
  subnet_ids = var.private_subnet_ids

  tags = {
    Name = "${var.project_name}-${var.environment}-db-subnet"
  }
}

# Security Group for RDS
resource "aws_security_group" "rds" {
  name        = "${var.project_name}-${var.environment}-rds-sg"
  description = "Security group for RDS PostgreSQL instance"
  vpc_id      = var.vpc_id

  ingress {
    description = "PostgreSQL from VPC"
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.selected.cidr_block]
  }

  egress {
    description = "All outbound traffic"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.project_name}-${var.environment}-rds-sg"
  }
}

data "aws_vpc" "selected" {
  id = var.vpc_id
}

# RDS PostgreSQL Instance (Free Tier: db.t4g.micro)
resource "aws_db_instance" "main" {
  identifier             = "${var.project_name}-${var.environment}"
  engine                 = "postgres"
  engine_version         = "17.6"
  instance_class         = var.instance_class
  allocated_storage      = 20
  max_allocated_storage  = 20
  storage_type           = "gp3"
  storage_encrypted      = true

  db_name  = var.database_name
  username = var.master_username
  password = random_password.master.result

  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [aws_security_group.rds.id]
  publicly_accessible    = false

  backup_retention_period      = var.backup_retention
  backup_window                = var.preferred_backup_window
  maintenance_window           = var.preferred_maintenance_window

  # DISABLED FOR FREE TIER - CloudWatch logs cost money!
  # enabled_cloudwatch_logs_exports = ["postgresql", "upgrade"]
  enabled_cloudwatch_logs_exports = []  # Disabled to save Free Tier quota

  skip_final_snapshot       = var.skip_final_snapshot
  final_snapshot_identifier = var.skip_final_snapshot ? null : "${var.project_name}-${var.environment}-final-${formatdate("YYYY-MM-DD-hhmm", timestamp())}"

  # Free tier compatible settings
  multi_az                   = false
  performance_insights_enabled = false
  deletion_protection        = false

  tags = {
    Name = "${var.project_name}-${var.environment}-rds"
  }
}

# CloudWatch Log Group for RDS - DISABLED FOR FREE TIER
# Uncomment if you need RDS logs and are willing to pay
# resource "aws_cloudwatch_log_group" "rds" {
#   name              = "/aws/rds/instance/${var.project_name}-${var.environment}/postgresql"
#   retention_in_days = 7
# }

