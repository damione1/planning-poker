terraform {
  required_version = ">= 1.5.7, < 2.0"
  required_providers {
    aws = { source = "hashicorp/aws", version = "= 6.71.0" }
  }
  backend "s3" {
    bucket         = "panetone-terraform-state-912928333407"
    key            = "planning-poker-applications/terraform.tfstate"
    region         = "us-east-1"
    profile        = "aws-terraform"
    encrypt        = true
    dynamodb_table = "panetone-terraform-locks"
  }
}

provider "aws" {
  region              = "us-east-1"
  profile             = "aws-applications-admin"
  allowed_account_ids = ["675704851310"]
  default_tags { tags = { Project = "planning-poker", Environment = "production", Owner = "Damien", ManagedBy = "terraform" } }
}

variable "bootstrap_ssh_cidr" {
  type    = string
  default = ""
}
variable "planning_poker_managed_instance_id" {
  type    = string
  default = ""
}
variable "migration_in_progress" {
  type    = bool
  default = false
}
locals {
  account_id        = "675704851310"
  backup_bucket_arn = "arn:aws:s3:::panetone-planning-poker-backups-912928333407"
}

resource "aws_lightsail_instance" "production" {
  name              = "planning-poker"
  availability_zone = "us-east-1a"
  blueprint_id      = "amazon_linux_2023"
  bundle_id         = "micro_3_0"
  ip_address_type   = "dualstack"
  user_data         = join("\n", ["#!/bin/bash", "export PLANNING_POKER_BOOTSTRAP_MODE=restore", file("${path.module}/../lightsail-user-data.sh")])
  add_on {
    type          = "AutoSnapshot"
    status        = "Enabled"
    snapshot_time = "08:00"
  }
  lifecycle {
    prevent_destroy = true
    ignore_changes  = [user_data]
  }
}
resource "aws_lightsail_static_ip" "production" {
  name = "planning-poker-ip"
  lifecycle { prevent_destroy = true }
}
resource "aws_lightsail_static_ip_attachment" "production" {
  static_ip_name = aws_lightsail_static_ip.production.id
  instance_name  = aws_lightsail_instance.production.id
}
resource "aws_lightsail_instance_public_ports" "production" {
  instance_name = aws_lightsail_instance.production.name
  port_info {
    protocol   = "tcp"
    from_port  = 80
    to_port    = 80
    cidrs      = ["0.0.0.0/0"]
    ipv6_cidrs = ["::/0"]
  }
  port_info {
    protocol   = "tcp"
    from_port  = 443
    to_port    = 443
    cidrs      = ["0.0.0.0/0"]
    ipv6_cidrs = ["::/0"]
  }
  dynamic "port_info" {
    for_each = var.bootstrap_ssh_cidr == "" ? [] : [var.bootstrap_ssh_cidr]
    content {
      protocol  = "tcp"
      from_port = 22
      to_port   = 22
      cidrs     = [port_info.value]
    }
  }
}
resource "aws_iam_openid_connect_provider" "github" {
  url            = "https://token.actions.githubusercontent.com"
  client_id_list = ["sts.amazonaws.com"]
}
output "public_ip" { value = aws_lightsail_static_ip.production.ip_address }
output "github_deploy_role_arn" { value = aws_iam_role.github_deploy.arn }
