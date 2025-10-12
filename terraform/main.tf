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

# Lightsail Container Service
resource "aws_lightsail_container_service" "app" {
  name  = var.service_name
  power = var.container_power
  scale = var.container_scale

  tags = var.tags

  # Note: Custom domains can be configured via AWS Console after deployment
  # The container service will be accessible via auto-generated Lightsail URL
}
