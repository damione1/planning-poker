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

  # Public endpoint configuration
  public_domain_names {
    certificate {
      certificate_name = var.domain_name != "" ? aws_lightsail_certificate.app[0].name : null
    }
  }

  # This will be updated by GitHub Actions, but we need an initial deployment
  # The deployment is managed through GitHub Actions after initial creation
}

# Optional: Custom domain certificate
resource "aws_lightsail_certificate" "app" {
  count = var.domain_name != "" ? 1 : 0
  name  = "${var.service_name}-cert"
  domain_name = var.domain_name
  subject_alternative_names = var.domain_alternative_names

  tags = var.tags
}
