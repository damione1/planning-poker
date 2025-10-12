# Planning Poker Deployment Guide

Simple deployment guide for AWS Lightsail.

## Prerequisites

- AWS account
- GitHub account with repository access
- GitHub CLI (`gh`) installed
- Terraform installed
- SSH key pair for server access

## One-Time Setup

### 1. AWS Infrastructure Setup

Run the setup script to configure AWS resources and GitHub secrets:

```bash
./scripts/setup-aws.sh
```

This script will:
- Create IAM user `github-actions-planning-poker`
- Create S3 bucket for Terraform state (with versioning and encryption)
- Create and attach IAM policy with necessary permissions
- Generate access keys
- Output GitHub CLI commands to set secrets

After the script completes, run the provided `gh secret set` commands to configure GitHub secrets:

```bash
gh secret set AWS_ACCESS_KEY_ID --body "your-access-key"
gh secret set AWS_SECRET_ACCESS_KEY --body "your-secret-key"
gh secret set TF_STATE_BUCKET --body "your-bucket-name"
gh variable set AWS_REGION --body "us-east-1"
```

### 2. Configure Terraform Backend

After running `setup-aws.sh`, you'll have an S3 bucket name. Create a backend config file:

```bash
cd terraform

# For staging
cp backend-staging.hcl.example backend-staging.hcl
# Edit backend-staging.hcl and replace the bucket name with yours

# For production
cp backend-production.hcl.example backend-production.hcl
# Edit backend-production.hcl and replace the bucket name with yours
```

Example `backend-staging.hcl`:
```hcl
bucket = "planning-poker-terraform-state-abc123"  # Your actual bucket name
key    = "staging/terraform.tfstate"
region = "us-east-1"
```

### 3. Create Environment Variables File

Create your tfvars file for the environment you're deploying:

```bash
# For staging (already exists in your local setup)
# Edit terraform/staging.tfvars with your settings

# For production (already exists in your local setup)
# Edit terraform/production.tfvars with your settings
```

### 4. Initialize and Apply Terraform

Initialize Terraform with the backend configuration:

```bash
cd terraform

# For staging
terraform init -backend-config=backend-staging.hcl

# For production
terraform init -backend-config=backend-production.hcl
```

Plan and apply:

```bash
# For staging
terraform plan -var-file=staging.tfvars
terraform apply -var-file=staging.tfvars

# For production
terraform plan -var-file=production.tfvars
terraform apply -var-file=production.tfvars
```

Terraform will create:
- AWS Lightsail instance (Ubuntu)
- Static IP (optional)
- Firewall rules (SSH, HTTP, HTTPS, app port)
- System configuration via user-data script

Note the instance IP address from the output:

```bash
terraform output instance_public_ip
```

### 5. Verify Instance Setup

SSH into the instance to verify setup:

```bash
ssh ubuntu@<instance-ip>
```

Check that the instance was properly initialized:

```bash
# Verify directories exist
ls -la /opt/planning-poker
ls -la /var/lib/planning-poker

# Check user was created
id planning-poker

# Verify firewall
sudo ufw status
```

## Automated Deployment

### Release Process

The CI/CD pipeline automatically builds and creates releases when you push a version tag:

```bash
# Create and push a version tag
git tag v1.0.0
git push origin v1.0.0
```

### What Happens Automatically

1. **test.yml** - Runs on every push to validate code
2. **release.yml** - Triggered by version tags (v*):
   - Sets up Go and Node.js
   - Installs dependencies
   - Generates templ templates
   - Builds frontend assets
   - Runs tests
   - Builds production binary for linux/amd64
   - Creates release package with:
     - Binary
     - install.sh script
     - systemd service file
     - README
   - Generates checksums
   - Creates GitHub release with artifacts

### Deployment to Server

After the GitHub release is created, deploy to your Lightsail instance:

```bash
# Download the release
wget https://github.com/YOUR-ORG/planning-poker/releases/download/v1.0.0/planning-poker-v1.0.0.tar.gz

# Extract
tar -xzf planning-poker-v1.0.0.tar.gz

# Upload to server
scp -r planning-poker/ ubuntu@<instance-ip>:/tmp/

# SSH to server and install
ssh ubuntu@<instance-ip>
cd /tmp/planning-poker
sudo ./install.sh
```

The `install.sh` script will:
- Install binary to `/opt/planning-poker/`
- Install systemd service
- Create application user and directories
- Set proper permissions
- Start and enable the service

## Service Management

### Check Service Status

```bash
sudo systemctl status planning-poker
```

### View Logs

```bash
# Follow logs in real-time
sudo journalctl -u planning-poker -f

# View recent logs
sudo journalctl -u planning-poker -n 100
```

### Restart Service

```bash
sudo systemctl restart planning-poker
```

### Stop/Start Service

```bash
sudo systemctl stop planning-poker
sudo systemctl start planning-poker
```

## Monitoring

### Health Check

```bash
curl http://localhost:8090
```

### Database

SQLite database is stored at:
```
/var/lib/planning-poker/pb_data/data.db
```

### Application Logs

Application logs are managed by systemd and can be viewed with `journalctl`.

## Troubleshooting

### Service Won't Start

Check logs for errors:
```bash
sudo journalctl -u planning-poker -n 50
```

Verify binary permissions:
```bash
ls -la /opt/planning-poker/planning-poker
```

### Port Already in Use

Check what's using port 8090:
```bash
sudo lsof -i :8090
```

### Database Issues

Check database file permissions:
```bash
ls -la /var/lib/planning-poker/pb_data/
```

Verify the planning-poker user owns the database:
```bash
sudo chown -R planning-poker:planning-poker /var/lib/planning-poker
```

### Build Failures

If GitHub Actions build fails:

1. Check workflow logs in GitHub Actions tab
2. Verify all tests pass locally: `go test -v ./...`
3. Ensure templ templates generate: `templ generate`
4. Check for Go version compatibility

## Manual Build (Optional)

If you need to build locally:

```bash
# Install dependencies
go mod download
npm ci

# Generate templates
templ generate

# Build frontend
npm run build

# Build binary
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-s -w -X main.Version=1.0.0" \
  -o planning-poker \
  .
```

## Security Notes

- Change `ws_allowed_origins = "*"` in production to your domain
- Keep AWS credentials secure - never commit them
- Regularly update the instance: `sudo apt update && sudo apt upgrade`
- Monitor fail2ban logs: `sudo fail2ban-client status sshd`
- Consider setting up CloudWatch or monitoring

## Optional: Enable Automatic Snapshots

Lightsail can automatically create daily snapshots of your instance. To enable this, add an `add_on` block to the instance resource in `terraform/main.tf`:

```hcl
resource "aws_lightsail_instance" "app" {
  name              = var.instance_name
  availability_zone = var.availability_zone
  blueprint_id      = var.blueprint_id
  bundle_id         = var.bundle_id
  key_pair_name     = aws_lightsail_key_pair.main.name

  # Enable automatic daily snapshots
  add_on {
    type          = "AutoSnapshot"
    snapshot_time = "06:00"  # UTC time (6am UTC)
    status        = "Enabled"
  }

  user_data = templatefile("${path.module}/user-data.sh", {
    app_port           = var.app_port
    ws_allowed_origins = var.ws_allowed_origins
  })

  tags = var.tags
}
```

Then apply the change:
```bash
cd terraform
terraform apply
```

Lightsail will create daily snapshots at 6:00 AM UTC and retain them for 7 days by default.

## Infrastructure Cleanup

To destroy all AWS resources:

```bash
cd terraform
terraform destroy
```

**Warning**: This will delete your Lightsail instance and all data.

## GitHub Secrets Required

| Secret | Description |
|--------|-------------|
| `AWS_ACCESS_KEY_ID` | IAM user access key |
| `AWS_SECRET_ACCESS_KEY` | IAM user secret key |
| `TF_STATE_BUCKET` | S3 bucket for Terraform state |
| `AWS_REGION` (variable) | AWS region (e.g., us-east-1) |

## Architecture Summary

```
Developer                GitHub Actions              AWS Lightsail
    |                          |                            |
    |-- git push tag v* ------>|                            |
    |                          |-- build binary             |
    |                          |-- run tests                |
    |                          |-- create release           |
    |                          |                            |
    |<-- download release -----|                            |
    |                                                       |
    |-- scp package ---------------------------------------->|
    |-- ssh install.sh --------------------------------------->|
    |                                                       |-- extract
    |                                                       |-- install
    |                                                       |-- start service
    |<-- service running <------------------------------------|
```

## Next Steps

1. Set up custom domain and SSL certificate
2. Configure CloudWatch monitoring
3. Set up automated backups for SQLite database
4. Configure CloudFront CDN for static assets
5. Set up proper CORS for WebSocket in production
