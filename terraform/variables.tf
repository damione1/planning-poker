variable "aws_region" {
  description = "AWS region for Lightsail container service"
  type        = string
  default     = "us-east-1"
}

variable "service_name" {
  description = "Name for the Lightsail container service"
  type        = string
  default     = "planning-poker"
}

variable "container_power" {
  description = "Power of the container service"
  type        = string
  default     = "nano"
  # Options:
  # - nano: 0.25 vCPU, 512 MB RAM - $7/month
  # - micro: 0.25 vCPU, 1 GB RAM - $10/month
  # - small: 0.5 vCPU, 2 GB RAM - $20/month
  # - medium: 1 vCPU, 4 GB RAM - $40/month
}

variable "container_scale" {
  description = "Number of container instances (1-20)"
  type        = number
  default     = 1
}

variable "tags" {
  description = "Tags to apply to resources"
  type        = map(string)
  default = {
    Project     = "planning-poker"
    Environment = "production"
    ManagedBy   = "terraform"
  }
}

variable "domain_name" {
  description = "Optional custom domain name for the application (e.g., poker.example.com)"
  type        = string
  default     = ""
}

variable "domain_alternative_names" {
  description = "Alternative domain names (e.g., [\"www.poker.example.com\"])"
  type        = list(string)
  default     = []
}
