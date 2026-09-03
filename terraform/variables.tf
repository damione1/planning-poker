variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "aws_profile" {
  description = "AWS CLI profile to use for authentication"
  type        = string
  default     = null
}

variable "lightsail_instance_name" {
  description = "Lightsail instance name"
  type        = string
  default     = "planning-poker"
}

variable "domain_name" {
  description = "Domain name for the application"
  type        = string
  default     = "pokerplanning.net"
}
