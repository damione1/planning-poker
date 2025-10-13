terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region  = var.aws_region
  profile = var.aws_profile
}

# Data source for latest Ubuntu 24.04 LTS ARM64 AMI
data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"] # Canonical

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-arm64-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
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

# EC2 Instance
resource "aws_instance" "app" {
  ami           = data.aws_ami.ubuntu.id
  instance_type = var.instance_type
  key_name      = aws_key_pair.deployer.key_name

  vpc_security_group_ids = [aws_security_group.app.id]
  availability_zone      = var.availability_zone

  user_data = templatefile("${path.module}/user-data.sh", {
    domain         = var.domain_name
    email          = var.lets_encrypt_email
    github_repo    = var.github_repo
    github_ref     = var.github_ref
    data_volume_id = aws_ebs_volume.data.id
    aws_region     = var.aws_region
    DOMAIN         = var.domain_name
    EMAIL          = var.lets_encrypt_email
  })

  tags = merge(var.tags, {
    Name = var.service_name
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
