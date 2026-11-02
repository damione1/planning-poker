terraform {
  required_version = ">= 1.5.7, < 2.0"
  required_providers {
    cloudflare = { source = "cloudflare/cloudflare", version = "= 5.22.0" }
  }
  backend "s3" {
    bucket         = "panetone-terraform-state-912928333407"
    key            = "planning-poker-dns/terraform.tfstate"
    region         = "us-east-1"
    profile        = "aws-terraform"
    encrypt        = true
    dynamodb_table = "panetone-terraform-locks"
  }
}
provider "cloudflare" {}
variable "production_ipv4" {
  type    = string
  default = null
}
data "terraform_remote_state" "application" {
  backend = "s3"
  config = {
    bucket  = "panetone-terraform-state-912928333407"
    key     = "planning-poker-applications/terraform.tfstate"
    region  = "us-east-1"
    profile = "aws-terraform"
    encrypt = true
  }
}
resource "cloudflare_dns_record" "production" {
  zone_id = "b593b6666432b350da6d94ec0968cc85"
  name    = "pokerplanning.net"
  type    = "A"
  content = var.production_ipv4 != null ? var.production_ipv4 : data.terraform_remote_state.application.outputs.public_ip
  proxied = false
  ttl     = 60
  lifecycle { prevent_destroy = true }
}
import {
  to = cloudflare_dns_record.production
  id = "b593b6666432b350da6d94ec0968cc85/24b780113293f91298e43a5b4c160c8a"
}
