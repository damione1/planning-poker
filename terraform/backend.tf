# Terraform Backend Configuration
# Store state in S3 bucket created by scripts/setup-aws.sh
#
# Initialize with:
#   terraform init -backend-config="bucket=your-bucket-name"
#
# Or create a backend-config.hcl file:
#   bucket = "planning-poker-terraform-state-xxxxx"
#   key    = "production/terraform.tfstate"
#   region = "us-east-1"
#
# Then run:
#   terraform init -backend-config=backend-config.hcl

terraform {
  backend "s3" {
    # Bucket name will be provided via -backend-config flag or environment variable
    # bucket = "planning-poker-terraform-state-xxxxx"  # Set via -backend-config

    key            = "terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true

    # DynamoDB table for state locking (optional but recommended)
    # dynamodb_table = "planning-poker-terraform-locks"
  }
}
