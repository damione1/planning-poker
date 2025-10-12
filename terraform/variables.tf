variable "aws_region" {
  description = "AWS region for Lightsail instance"
  type        = string
  default     = "us-east-1"
}

variable "instance_name" {
  description = "Name for the Lightsail instance"
  type        = string
  default     = "planning-poker"
}

variable "blueprint_id" {
  description = "Lightsail blueprint ID (OS image)"
  type        = string
  default     = "ubuntu_24_04"
}

variable "bundle_id" {
  description = "Lightsail bundle ID (instance size)"
  type        = string
  default     = "nano_3_0" # $3.50/month - 512MB RAM, 1 vCPU, 20GB SSD
  # Options:
  # - nano_3_0: $3.50/month - 512MB RAM, 1 vCPU, 20GB SSD
  # - micro_3_0: $5/month - 1GB RAM, 1 vCPU, 40GB SSD
  # - small_3_0: $10/month - 2GB RAM, 1 vCPU, 60GB SSD
}

variable "availability_zone" {
  description = "Lightsail availability zone"
  type        = string
  default     = "us-east-1a"
}

variable "app_port" {
  description = "Application HTTP port"
  type        = number
  default     = 8090
}

variable "ssh_key_name" {
  description = "Name of SSH key pair for instance access"
  type        = string
  default     = "planning-poker-key"
}

variable "ssh_public_key_path" {
  description = "Path to SSH public key file"
  type        = string
  default     = "~/.ssh/id_rsa.pub"
}

variable "allowed_ssh_cidrs" {
  description = "CIDR blocks allowed to SSH into instance"
  type        = list(string)
  default     = ["0.0.0.0/0"] # Restrict to your IP in production!
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

variable "enable_static_ip" {
  description = "Whether to attach a static IP to the instance"
  type        = bool
  default     = true
}

variable "domain_name" {
  description = "Optional domain name for the application (e.g., poker.example.com)"
  type        = string
  default     = ""
}

variable "ws_allowed_origins" {
  description = "WebSocket allowed origins (comma-separated domains)"
  type        = string
  default     = "*" # Configure with actual domain in production
}
