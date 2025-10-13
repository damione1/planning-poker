variable "aws_region" {
  description = "AWS region"
}

variable "aws_profile" {
  description = "AWS CLI profile to use for authentication"
  type        = string
  default     = null
}

variable "availability_zone" {
  description = "AWS availability zone for EC2 and EBS"
  type        = string
  default     = "us-east-1a"
}

variable "service_name" {
  description = "Name of the service"
  type        = string
  default     = "planning-poker-prod"
}

variable "instance_type" {
  description = "EC2 instance type (ARM64)"
  type        = string
  default     = "t4g.micro" # Free tier eligible
}

variable "ssh_public_key_path" {
  description = "Path to SSH public key file (e.g., ~/.ssh/id_rsa.pub)"
  type        = string
  default     = "~/.ssh/id_rsa.pub"
}

variable "data_volume_size" {
  description = "Size of EBS volume for persistent data in GB"
  type        = number
  default     = 10
}

variable "domain_name" {
  description = "Domain name for the application"
  type        = string
}

variable "lets_encrypt_email" {
  description = "Email address for Let's Encrypt certificates"
  type        = string
}

variable "github_repo" {
  description = "GitHub repository in format owner/repo"
  type        = string
  default     = "damione1/planning-poker"
}

variable "github_ref" {
  description = "Git reference to deploy (branch or tag)"
  type        = string
  default     = "main"
}

variable "ssh_allowed_cidr" {
  description = "CIDR blocks allowed to SSH"
  type        = list(string)
  default     = ["0.0.0.0/0"] # WARNING: Restrict this in production
}

variable "backup_retention_days" {
  description = "Number of days to retain EBS snapshots"
  type        = number
  default     = 7
}

variable "tags" {
  description = "Common tags for all resources"
  type        = map(string)
  default = {
    Environment = "production"
    Project     = "planning-poker"
    ManagedBy   = "terraform"
  }
}
