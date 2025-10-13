# Planning Poker Deployment Guide

Simple deployment to AWS EC2 with Docker, Traefik, and EBS persistence.

## Architecture Overview

```
Internet
   |
   ▼
Elastic IP (static)
   |
   ▼
EC2 Instance (t4g.micro - Amazon Linux 2023 ARM64)
├── CodeDeploy Agent (automated deployment)
├── Traefik (Port 80/443)
│   ├── Let's Encrypt SSL
│   └── Auto HTTPS redirect
└── Planning Poker Container
    └── /app/pb_data → /mnt/data (EBS Volume)
```

**Key Features:**
- ✅ **Automated Deployment**: AWS CodeDeploy with GitHub Actions
- ✅ **Persistent Storage**: EBS volume for database (survives container restarts)
- ✅ **Automatic SSL**: Traefik handles Let's Encrypt certificates
- ✅ **Simple**: Single EC2 instance with Docker Compose
- ✅ **Cost-Effective**: t4g.micro eligible for AWS free tier

## Prerequisites

- AWS account with SSO configured
- Domain name (for SSL)
- Terraform installed
- AWS CLI v2 installed
- SSH key pair in AWS EC2

## One-Time Setup

### 1. Configure AWS SSO

**Configure AWS SSO profile** in `~/.aws/config`:

```ini
[profile YOUR-PROFILE-NAME]
sso_start_url = https://your-org.awsapps.com/start
sso_region = us-east-1
sso_account_id = YOUR-ACCOUNT-ID
sso_role_name = YOUR-ROLE-NAME
region = us-east-1
output = json
```

**Pro tip**: Use legacy format (inline config) for better Terraform compatibility. If using the new `sso_session` format, create a separate legacy profile for Terraform.

**Environment variables** (optional, add to `~/.zshrc` or `~/.bashrc`):
```bash
export AWS_PROFILE=YOUR-PROFILE-NAME
export AWS_REGION=us-east-1
```

### 2. Run AWS Setup Script

This creates S3 buckets (Terraform state + CodeDeploy), IAM user, and policies:

```bash
./scripts/setup-aws.sh
```

The script will:
- Create IAM user for GitHub Actions
- Create S3 bucket for Terraform state (with versioning and encryption)
- Create S3 bucket for CodeDeploy deployment packages
- Configure IAM policies (Terraform, CodeDeploy, S3)
- Generate access keys
- Output configuration for GitHub secrets

**Important**: Save the output - it contains bucket names and credentials.

### 3. Configure Terraform Backend

```bash
cd terraform

# Create backend configuration
cp backend-config.hcl.example backend-config.hcl
nano backend-config.hcl  # Edit bucket name and profile from setup script output
```

Example `backend-config.hcl`:
```hcl
bucket  = "planning-poker-terraform-state-xxxxx"  # From setup script
key     = "production/terraform.tfstate"
region  = "us-east-1"
encrypt = true
profile = "YOUR-PROFILE-NAME"  # Your AWS SSO profile
```

### 4. Configure Terraform Variables

Create your configuration file:

```bash
cp terraform.tfvars.example terraform.tfvars
nano terraform.tfvars
```

Update these required values:

```hcl
# AWS Configuration
aws_region  = "us-east-1"
aws_profile = "YOUR-PROFILE-NAME"  # Your AWS SSO profile

# SSH Access
ssh_public_key_path = "~/.ssh/id_ed25519.pub"  # Path to YOUR SSH public key (adjust as needed)

# Domain and SSL
domain_name         = "pokerplanning.net"
lets_encrypt_email  = "your-email@example.com"

# Security - IMPORTANT: Restrict SSH to your IP
ssh_allowed_cidr = ["YOUR.IP.ADDRESS/32"]
```

### 5. Deploy Infrastructure

**Login to AWS SSO first** (required for every session):
```bash
aws sso login --profile=YOUR-PROFILE-NAME
```

Then deploy:

```bash
# Initialize Terraform with backend
terraform init -backend-config=backend-config.hcl

# Review plan
terraform plan

# Deploy (takes ~5 minutes)
terraform apply
```

Terraform will create:
- EC2 t4g.micro instance (Amazon Linux 2023 ARM64, free tier eligible)
- EBS 10GB volume (encrypted, persistent storage)
- Elastic IP (static public IP)
- Security Groups (HTTP, HTTPS, SSH)
- IAM roles for EC2 and CodeDeploy
- CodeDeploy application and deployment group
- Auto-install Docker, CodeDeploy agent, Traefik, and application

### 6. Configure DNS

After `terraform apply` completes, note the Elastic IP:

```bash
terraform output instance_public_ip
# Example: 54.123.456.789
```

**Configure your DNS:**
1. Go to your DNS provider (Cloudflare, Route53, etc.)
2. Create an A record:
   ```
   pokerplanning.net  A  54.123.456.789
   ```
3. Wait for DNS propagation (~5-10 minutes)

### 7. Verify Deployment

Wait ~10 minutes for:
1. EC2 instance to boot
2. Docker to install
3. Application to build and start
4. Let's Encrypt to issue SSL certificate

Then visit:
```
https://pokerplanning.net
```

### 8. Configure GitHub Secrets and Variables

Set up GitHub repository secrets and variables for automated deployment:

```bash
# Set secrets (from setup-aws.sh output)
gh secret set AWS_ACCESS_KEY_ID --body "YOUR-ACCESS-KEY-ID"
gh secret set AWS_SECRET_ACCESS_KEY --body "YOUR-SECRET-ACCESS-KEY"

# Set variables
gh variable set AWS_REGION --body "us-east-1"
gh variable set CODEDEPLOY_S3_BUCKET --body "planning-poker-codedeploy-xxxxx"
gh variable set CODEDEPLOY_APP_NAME --body "planning-poker-prod"
gh variable set CODEDEPLOY_DEPLOYMENT_GROUP --body "planning-poker-prod-deployment-group"
gh variable set DOMAIN_NAME --body "pokerplanning.net"
```

Or configure via GitHub web interface:
- Settings → Secrets and variables → Actions → New repository secret/variable

**Check logs** (if needed):
```bash
# SSH into instance (use your private key that corresponds to the public key in tfvars)
ssh ec2-user@$(terraform output -raw instance_public_ip)

# View application logs
cd /opt/planning-poker
sudo docker compose -f docker-compose.prod.yml logs -f app

# View Traefik logs
sudo docker compose -f docker-compose.prod.yml logs -f traefik

# View CodeDeploy agent logs
sudo tail -f /var/log/aws/codedeploy-agent/codedeploy-agent.log
```

## Deployment

### Automated Deployment (Recommended)

Deployments are triggered automatically when you push a git tag:

```bash
# Create and push a tag
git tag v0.1.11
git push origin v0.1.11
```

This will:
1. Run tests in GitHub Actions
2. Create a GitHub release with changelog
3. Package application and upload to S3
4. Create CodeDeploy deployment
5. CodeDeploy will:
   - Stop running containers
   - Deploy new code
   - Build and start new containers
   - Validate service health

Monitor deployment:
- **GitHub Actions**: https://github.com/damione1/planning-poker/actions
- **AWS CodeDeploy Console**: Services → CodeDeploy → Applications
- **CloudWatch Logs**: For CodeDeploy agent logs

### Manual Deployment (For Testing)

```bash
# SSH into instance
ssh ec2-user@$(terraform output -raw instance_public_ip)

# Navigate to app directory
cd /opt/planning-poker

# Pull latest changes
sudo git pull origin main  # or git checkout v1.0.0

# Rebuild and restart manually
sudo docker compose -f docker-compose.prod.yml build --no-cache
sudo docker compose -f docker-compose.prod.yml up -d

# View logs
sudo docker compose -f docker-compose.prod.yml logs -f
```

## Persistence & Backups

**Database Location:**
- Host: `/mnt/data/pb_data`
- Container: `/app/pb_data`
- EBS Volume: Automatically mounted at `/mnt/data`

**What persists:**
- ✅ PocketBase database (rooms, participants, votes)
- ✅ Let's Encrypt SSL certificates
- ✅ Application data

**Automated Backups:**

AWS Backup automatically creates daily EBS snapshots:
- **Schedule**: Daily at 2 AM UTC
- **Retention**: 7 days (configurable)
- **Location**: AWS Backup Vault

**View Backups:**
```bash
# List snapshots via AWS CLI
aws backup list-recovery-points-by-backup-vault \
  --backup-vault-name $(terraform output -raw backup_vault)

# Or view in AWS Console:
# Services → AWS Backup → Backup vaults → planning-poker-prod-backup-vault
```

**Restore from Backup:**

1. **Via AWS Console:**
   - Go to AWS Backup → Backup vaults
   - Select your vault
   - Choose a recovery point
   - Click "Restore"
   - Select "Create new volume"
   - Attach to EC2 instance

2. **Via Terraform (replace volume):**
   ```hcl
   # In terraform/terraform.tfvars, add:
   restore_from_snapshot_id = "snap-xxxxx"
   ```

**Manual Backup (if needed):**
```bash
# Create on-demand snapshot
aws ec2 create-snapshot \
  --volume-id $(terraform output -raw ebs_volume_id) \
  --description "Manual backup $(date +%Y%m%d)"
```

## Monitoring

### Service Status

```bash
# SSH into instance
ssh ec2-user@<instance-ip>

# Check CodeDeploy agent status
sudo systemctl status codedeploy-agent

# Check container status
sudo docker compose -f /opt/planning-poker/docker-compose.prod.yml ps

# View application logs
sudo docker compose -f /opt/planning-poker/docker-compose.prod.yml logs -f

# Check disk usage
df -h /mnt/data

# View recent CodeDeploy deployments
sudo tail -f /opt/codedeploy-agent/deployment-root/deployment-logs/codedeploy-agent-deployments.log
```

### Health Checks

- **Application**: https://pokerplanning.net/
- **SSL Certificate**: https://www.ssllabs.com/ssltest/analyze.html?d=pokerplanning.net

## Scaling

### Vertical Scaling (Bigger Instance)

Update `terraform/terraform.tfvars`:

```hcl
instance_type = "t4g.small"  # 2 vCPU, 2 GB RAM
```

Apply changes:

```bash
terraform apply
```

Note: This will recreate the instance. EBS volume persists.

### Data Volume Size

Update `terraform/terraform.tfvars`:

```hcl
data_volume_size = 20  # Increase to 20 GB
```

Apply and resize:

```bash
terraform apply

# SSH into instance and resize filesystem
ssh ubuntu@<instance-ip>
sudo resize2fs /dev/xvdf
```

## Security

### SSH Access

**Restrict SSH to your IP** (IMPORTANT):

```hcl
# terraform/terraform.tfvars
ssh_allowed_cidr = ["YOUR.IP.ADDRESS/32"]
```

Then apply:

```bash
terraform apply
```

### SSL Certificate

Traefik automatically:
- Requests Let's Encrypt certificate
- Renews certificates before expiration
- Redirects HTTP → HTTPS

### Firewall Rules

Security Group allows:
- ✅ Port 80 (HTTP) - Auto redirects to HTTPS
- ✅ Port 443 (HTTPS) - Application traffic
- ✅ Port 22 (SSH) - Management (restrict to your IP!)

## Troubleshooting

### AWS SSO and Terraform Issues

**Error: `SSOProviderInvalidToken` or `no valid credential sources`**

Solution: Re-login to AWS SSO
```bash
aws sso login --profile=YOUR-PROFILE-NAME
```

Common causes:
1. SSO session expired (re-login required)
2. Not logged into SSO before running Terraform
3. Wrong profile in `backend-config.hcl` or `terraform.tfvars`

**Error: `bucket does not exist`**

Solution: Run the setup script to create the S3 bucket:
```bash
./scripts/setup-aws.sh
```

**Error: Backend initialization failed**

Solution: Initialize with backend config:
```bash
cd terraform
terraform init -backend-config=backend-config.hcl
```

### Application Not Loading

1. **Check DNS propagation:**
   ```bash
   dig pokerplanning.net
   nslookup pokerplanning.net
   ```

2. **Check CodeDeploy deployment status:**
   - Visit AWS CodeDeploy console
   - Check GitHub Actions logs for deployment errors
   - Review CodeDeploy agent logs: `sudo tail -f /var/log/aws/codedeploy-agent/codedeploy-agent.log`

3. **Check containers:**
   ```bash
   ssh ec2-user@<instance-ip>
   sudo docker ps
   sudo docker compose -f /opt/planning-poker/docker-compose.prod.yml logs
   ```

4. **Check Traefik:**
   ```bash
   sudo docker logs traefik
   ```

### SSL Certificate Issues

```bash
# View Traefik logs
sudo docker logs traefik

# Check certificate storage
sudo ls -la /mnt/data/traefik/acme/

# Force certificate renewal (if needed)
sudo docker compose -f /opt/planning-poker/docker-compose.prod.yml restart traefik
```

### CodeDeploy Deployment Failures

```bash
# Check CodeDeploy agent status
ssh ec2-user@<instance-ip>
sudo systemctl status codedeploy-agent

# View CodeDeploy agent logs
sudo tail -100 /var/log/aws/codedeploy-agent/codedeploy-agent.log

# View specific deployment logs
sudo ls -la /opt/codedeploy-agent/deployment-root/
sudo cat /opt/codedeploy-agent/deployment-root/<deployment-id>/logs/scripts.log

# Restart CodeDeploy agent
sudo systemctl restart codedeploy-agent

# Common issues:
# 1. S3 bucket permissions - check IAM policies
# 2. Deployment scripts not executable - check chmod +x in scripts/codedeploy/
# 3. Docker not running - check sudo systemctl status docker
```

### Database Issues

```bash
# Check database location
ssh ec2-user@<instance-ip>
sudo ls -la /mnt/data/pb_data/

# Check EBS mount
df -h /mnt/data

# Access PocketBase admin
https://pokerplanning.net/_/
```

### Out of Disk Space

```bash
# Check disk usage
ssh ec2-user@<instance-ip>
df -h

# Clean Docker images and build cache
sudo docker system prune -a

# View Docker disk usage
sudo docker system df

# Increase EBS volume size (see Scaling section)
```

## Cost Estimate

**Monthly Cost (AWS us-east-1):**

| Resource | Specs | Cost |
|----------|-------|------|
| EC2 t4g.micro | 2 vCPU, 1 GB | $6.14/month (free tier: $0) |
| EBS gp3 10GB | Encrypted | $0.80/month |
| Elastic IP | In use | $0/month |
| AWS Backup | 7 days retention | ~$0.50/month |
| **Total** | | **~$7.50/month** |

**Free Tier:**
- First 12 months: EC2 t4g.micro 750 hours/month FREE
- First 12 months: 30 GB EBS FREE
- AWS Backup: First 10 GB free, then $0.05/GB-month

**Backup Cost Details:**
- Snapshot storage: ~10 GB × 7 snapshots × $0.05/GB = ~$0.50/month
- Daily backups included in AWS Backup free tier (10 GB)

## Infrastructure Cleanup

**To destroy all resources:**

```bash
cd terraform
terraform destroy
```

⚠️ **Warning:** This will delete:
- EC2 instance
- EBS volume (and all data)
- Elastic IP
- Security groups

Backup your data first!

## Environment Variables

The application runs with:
- `PP_ENV=production`
- `DOMAIN_NAME=pokerplanning.net` (from tfvars)
- `LETS_ENCRYPT_EMAIL=your-email@example.com` (from tfvars)

## Local Development vs Production

### Local Development

```bash
# Start with live reload
make dev

# Or use Docker Compose
docker compose up
```

### Production Deployment

```bash
# SSH and deploy
ssh ubuntu@<instance-ip>
cd /opt/planning-poker
sudo bash scripts/deploy.sh
```

## Next Steps

1. ✅ **Automated backups configured** - Daily EBS snapshots via AWS Backup
2. ✅ **Automated deployment** - CodeDeploy with GitHub Actions
3. Configure monitoring alerts (CloudWatch Alarms)
4. Set up log aggregation (CloudWatch Logs)
5. Add health check monitoring (StatusCake, Pingdom)
6. Configure WebSocket allowed origins for production domain
7. Review and test backup restoration procedure
8. Test deployment rollback procedure

## Support

For issues or questions:
- GitHub Issues: https://github.com/damione1/planning-poker/issues
- Check logs: `docker compose logs -f`
- Review Terraform state: `terraform show`
