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

# No managed resources. The EC2 stack was destroyed 2026-09-03.
# Production is the Lightsail instance named var.lightsail_instance_name,
# bootstrapped with lightsail-user-data.sh. `terraform apply` is a no-op
# so this config cannot recreate the old t4g.micro.
