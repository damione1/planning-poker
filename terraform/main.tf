terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }
}

provider "aws" {
  region  = var.aws_region
  profile = var.aws_profile
}

# Data source for latest Amazon Linux 2023 ARM64 AMI
data "aws_ami" "amazon_linux" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["al2023-ami-2023.*-arm64"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  filter {
    name   = "architecture"
    values = ["arm64"]
  }
}

# SSH Key Pair (Terraform-managed)
resource "aws_key_pair" "deployer" {
  key_name   = "${var.service_name}-key"
  public_key = file(pathexpand(var.ssh_public_key_path))

  tags = merge(var.tags, {
    Name = "${var.service_name}-key"
  })
}

# Security Group
resource "aws_security_group" "app" {
  name        = "${var.service_name}-sg"
  description = "Security group for ${var.service_name}"

  # HTTP
  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "HTTP"
  }

  # HTTPS
  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "HTTPS"
  }

  # SSH (restricted to your IP for security)
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.ssh_allowed_cidr
    description = "SSH"
  }

  # Outbound traffic
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "All outbound traffic"
  }

  tags = merge(var.tags, {
    Name = "${var.service_name}-sg"
  })
}

# EBS Volume for persistent data
resource "aws_ebs_volume" "data" {
  availability_zone = var.availability_zone
  size              = var.data_volume_size
  type              = "gp3"
  encrypted         = true

  tags = merge(var.tags, {
    Name = "${var.service_name}-data"
  })
}

# IAM Role for EC2 to access CodeDeploy and S3
resource "aws_iam_role" "ec2_codedeploy" {
  name = "${var.service_name}-ec2-codedeploy-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      }
    ]
  })

  tags = var.tags
}

# Attach policy for CodeDeploy agent
resource "aws_iam_role_policy_attachment" "ec2_codedeploy_agent" {
  role       = aws_iam_role.ec2_codedeploy.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2RoleforAWSCodeDeploy"
}

# Create specific policy for CodeDeploy S3 bucket access
resource "aws_iam_role_policy" "ec2_codedeploy_s3" {
  name = "${var.service_name}-ec2-s3-policy"
  role = aws_iam_role.ec2_codedeploy.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:ListBucket"
        ]
        Resource = [
          aws_s3_bucket.codedeploy.arn,
          "${aws_s3_bucket.codedeploy.arn}/*"
        ]
      }
    ]
  })
}

# Instance profile for EC2
resource "aws_iam_instance_profile" "ec2_codedeploy" {
  name = "${var.service_name}-ec2-profile"
  role = aws_iam_role.ec2_codedeploy.name

  tags = var.tags
}

# EC2 Instance
resource "aws_instance" "app" {
  ami                    = data.aws_ami.amazon_linux.id
  instance_type          = var.instance_type
  key_name               = aws_key_pair.deployer.key_name
  iam_instance_profile   = aws_iam_instance_profile.ec2_codedeploy.name

  vpc_security_group_ids = [aws_security_group.app.id]
  availability_zone      = var.availability_zone

  user_data = templatefile("${path.module}/user-data.sh", {
    domain         = var.domain_name
    email          = var.lets_encrypt_email
    github_repo    = var.github_repo
    github_ref     = var.github_ref
    data_volume_id = aws_ebs_volume.data.id
    aws_region     = var.aws_region
    service_name   = var.service_name
  })

  tags = merge(var.tags, {
    Name = var.service_name
    Application = var.service_name
  })

  lifecycle {
    ignore_changes = [user_data]
  }
}

# Attach EBS volume to EC2
resource "aws_volume_attachment" "data" {
  device_name = "/dev/xvdf"
  volume_id   = aws_ebs_volume.data.id
  instance_id = aws_instance.app.id
}

# Elastic IP for stable public address
resource "aws_eip" "app" {
  instance = aws_instance.app.id
  domain   = "vpc"

  tags = merge(var.tags, {
    Name = "${var.service_name}-eip"
  })
}

# AWS Backup Vault
resource "aws_backup_vault" "data" {
  name = "${var.service_name}-backup-vault"

  tags = merge(var.tags, {
    Name = "${var.service_name}-backup-vault"
  })
}

# AWS Backup Plan - Daily snapshots with 7-day retention
resource "aws_backup_plan" "data" {
  name = "${var.service_name}-daily-backup"

  rule {
    rule_name         = "daily_backup"
    target_vault_name = aws_backup_vault.data.name
    schedule          = "cron(0 2 * * ? *)" # Daily at 2 AM UTC

    lifecycle {
      delete_after = var.backup_retention_days
    }
  }

  tags = merge(var.tags, {
    Name = "${var.service_name}-backup-plan"
  })
}

# IAM Role for AWS Backup
resource "aws_iam_role" "backup" {
  name = "${var.service_name}-backup-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "backup.amazonaws.com"
        }
      }
    ]
  })

  tags = var.tags
}

# Attach AWS managed backup policy
resource "aws_iam_role_policy_attachment" "backup" {
  role       = aws_iam_role.backup.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSBackupServiceRolePolicyForBackup"
}

resource "aws_iam_role_policy_attachment" "restore" {
  role       = aws_iam_role.backup.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSBackupServiceRolePolicyForRestores"
}

# Backup Selection - Target EBS volume
resource "aws_backup_selection" "data" {
  name         = "${var.service_name}-ebs-selection"
  plan_id      = aws_backup_plan.data.id
  iam_role_arn = aws_iam_role.backup.arn

  resources = [
    aws_ebs_volume.data.arn
  ]
}

# Random suffix for S3 bucket name
resource "random_id" "codedeploy_bucket" {
  byte_length = 4
}

# S3 Bucket for CodeDeploy deployment packages
resource "aws_s3_bucket" "codedeploy" {
  bucket = "${var.service_name}-codedeploy-${random_id.codedeploy_bucket.hex}"

  tags = merge(var.tags, {
    Name = "${var.service_name}-codedeploy"
  })
}

# Enable versioning for CodeDeploy bucket
resource "aws_s3_bucket_versioning" "codedeploy" {
  bucket = aws_s3_bucket.codedeploy.id

  versioning_configuration {
    status = "Enabled"
  }
}

# Lifecycle rule to clean up old deployment packages
resource "aws_s3_bucket_lifecycle_configuration" "codedeploy" {
  bucket = aws_s3_bucket.codedeploy.id

  rule {
    id     = "cleanup-old-deployments"
    status = "Enabled"

    filter {} # Apply to all objects

    expiration {
      days = 30
    }

    noncurrent_version_expiration {
      noncurrent_days = 7
    }
  }
}

# Block public access to CodeDeploy bucket
resource "aws_s3_bucket_public_access_block" "codedeploy" {
  bucket = aws_s3_bucket.codedeploy.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Data source for existing GitHub Actions IAM user
data "aws_iam_user" "github_actions" {
  user_name = var.github_actions_user
}

# IAM policy for GitHub Actions to upload to CodeDeploy bucket
resource "aws_iam_user_policy" "github_actions_codedeploy" {
  name = "${var.service_name}-github-actions-codedeploy-policy"
  user = data.aws_iam_user.github_actions.user_name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:GetObject",
          "s3:ListBucket"
        ]
        Resource = [
          aws_s3_bucket.codedeploy.arn,
          "${aws_s3_bucket.codedeploy.arn}/*"
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "codedeploy:CreateDeployment",
          "codedeploy:GetDeployment",
          "codedeploy:GetDeploymentConfig",
          "codedeploy:GetApplicationRevision",
          "codedeploy:RegisterApplicationRevision"
        ]
        Resource = [
          aws_codedeploy_app.app.arn,
          "arn:aws:codedeploy:${var.aws_region}:${data.aws_caller_identity.current.account_id}:deploymentgroup:${aws_codedeploy_app.app.name}/*",
          "arn:aws:codedeploy:${var.aws_region}:${data.aws_caller_identity.current.account_id}:deploymentconfig:*"
        ]
      }
    ]
  })
}

# Data source to get current AWS account ID
data "aws_caller_identity" "current" {}

# IAM Role for CodeDeploy Service
resource "aws_iam_role" "codedeploy" {
  name = "${var.service_name}-codedeploy-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "codedeploy.amazonaws.com"
        }
      }
    ]
  })

  tags = var.tags
}

# Attach AWS managed CodeDeploy policy
resource "aws_iam_role_policy_attachment" "codedeploy" {
  role       = aws_iam_role.codedeploy.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSCodeDeployRole"
}

# CodeDeploy Application
resource "aws_codedeploy_app" "app" {
  name             = var.service_name
  compute_platform = "Server"

  tags = var.tags
}

# CodeDeploy Deployment Group
resource "aws_codedeploy_deployment_group" "app" {
  app_name              = aws_codedeploy_app.app.name
  deployment_group_name = "${var.service_name}-deployment-group"
  service_role_arn      = aws_iam_role.codedeploy.arn

  deployment_config_name = "CodeDeployDefault.AllAtOnce"

  ec2_tag_set {
    ec2_tag_filter {
      key   = "Application"
      type  = "KEY_AND_VALUE"
      value = var.service_name
    }
  }

  auto_rollback_configuration {
    enabled = true
    events  = ["DEPLOYMENT_FAILURE"]
  }

  tags = var.tags
}
