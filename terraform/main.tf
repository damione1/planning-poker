terraform {
  required_version = ">= 1.5.7, < 2.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "= 6.71.0"
    }
  }
}

provider "aws" {
  region              = var.aws_region
  profile             = var.aws_profile
  allowed_account_ids = ["912928333407"]
}

# Production is imported; the former EC2 stack stays removed.
resource "aws_lightsail_instance" "production" {
  name              = var.lightsail_instance_name
  availability_zone = "us-east-1a"
  blueprint_id      = "amazon_linux_2023"
  bundle_id         = "micro_3_0"
  key_pair_name     = "planning-poker-prod"
  ip_address_type   = "dualstack"
  user_data         = file("${path.module}/lightsail-user-data.sh")
  add_on {
    type          = "AutoSnapshot"
    snapshot_time = "08:00"
    status        = "Enabled"
  }
  tags = {
    Project     = "planning-poker"
    Environment = "production"
    ManagedBy   = "terraform"
    Owner       = "Damien"
  }
  lifecycle {
    prevent_destroy = true
    ignore_changes  = [user_data]
  }
}

resource "aws_lightsail_static_ip" "production" {
  name = "planning-poker-ip"
  lifecycle {
    prevent_destroy = true
  }
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
    for_each = length(var.ssh_allowed_cidrs) == 0 ? [] : [true]
    content {
      protocol  = "tcp"
      from_port = 22
      to_port   = 22
      cidrs     = var.ssh_allowed_cidrs
    }
  }
}

import {
  to = aws_lightsail_instance.production
  id = "planning-poker"
}
import {
  to = aws_lightsail_static_ip.production
  id = "planning-poker-ip"
}
import {
  to = aws_lightsail_static_ip_attachment.production
  id = "planning-poker-ip"
}
