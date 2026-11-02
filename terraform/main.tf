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

# The management-account Lightsail instance and its IP were retired after
# migration. This state intentionally contains no application resources.
# Production is managed by ./applications and DNS by ./dns.
