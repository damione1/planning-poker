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
  region = var.aws_region
}

# SSH Key Pair
resource "aws_lightsail_key_pair" "main" {
  name       = var.ssh_key_name
  public_key = file(pathexpand(var.ssh_public_key_path))
}

# Lightsail Instance
resource "aws_lightsail_instance" "app" {
  name              = var.instance_name
  availability_zone = var.availability_zone
  blueprint_id      = var.blueprint_id
  bundle_id         = var.bundle_id
  key_pair_name     = aws_lightsail_key_pair.main.name

  user_data = templatefile("${path.module}/user-data.sh", {
    app_port           = var.app_port
    ws_allowed_origins = var.ws_allowed_origins
  })

  tags = var.tags
}

# Static IP (Optional)
resource "aws_lightsail_static_ip" "app" {
  count = var.enable_static_ip ? 1 : 0
  name  = "${var.instance_name}-static-ip"
}

resource "aws_lightsail_static_ip_attachment" "app" {
  count          = var.enable_static_ip ? 1 : 0
  static_ip_name = aws_lightsail_static_ip.app[0].name
  instance_name  = aws_lightsail_instance.app.name
}

# Firewall Rules
resource "aws_lightsail_instance_public_ports" "app" {
  instance_name = aws_lightsail_instance.app.name

  # SSH
  port_info {
    protocol  = "tcp"
    from_port = 22
    to_port   = 22
    cidrs     = var.allowed_ssh_cidrs
  }

  # HTTP (Application)
  port_info {
    protocol  = "tcp"
    from_port = var.app_port
    to_port   = var.app_port
    cidrs     = ["0.0.0.0/0"]
  }

  # HTTPS (for future use with reverse proxy)
  port_info {
    protocol  = "tcp"
    from_port = 443
    to_port   = 443
    cidrs     = ["0.0.0.0/0"]
  }

  # HTTP (for reverse proxy)
  port_info {
    protocol  = "tcp"
    from_port = 80
    to_port   = 80
    cidrs     = ["0.0.0.0/0"]
  }
}
