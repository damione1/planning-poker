# Deployment Guide

This guide covers deploying Planning Poker to AWS EC2 with automated deployments via GitHub Actions and AWS Systems Manager (SSM).

## Architecture Overview

```
GitHub Actions (on tag push)
    ↓
1. Build & push Docker image to GHCR
    ↓
2. Trigger AWS SSM Run Command
    ↓
3. EC2 pulls image from GHCR
    ↓
4. Docker Compose deploys with Traefik + Let's Encrypt
```

**Key Features**:
- ✅ Zero SSH required for deployments
- ✅ Automated SSL certificate management (Let's Encrypt)
- ✅ Single source of truth (GHCR for images)
- ✅ Health checks and deployment validation
- ✅ Automatic rollback on failure

## Prerequisites

Before deploying, ensure you have:

1. **AWS Account** with appropriate permissions
2. **AWS CLI** installed and configured (`aws configure`)
3. **Terraform** v1.5+ installed
4. **GitHub Repository** with Actions enabled
5. **Domain Name** pointing to AWS
6. **GitHub Personal Access Token** (for GHCR image pulls)

## Infrastructure Setup

### Step 1: Configure Terraform Variables

Create `terraform/terraform.tfvars`:

```hcl
# Required Variables
aws_region      = "us-east-1"
domain_name     = "planningpoker.yourdomain.com"
email           = "admin@yourdomain.com"
github_repo     = "yourusername/planning-poker"

# Optional Variables (defaults shown)
service_name    = "planning-poker"
instance_type   = "t4g.micro"
ebs_volume_size = 20
backup_retention_days = 7
```

### Step 2: Deploy Infrastructure

```bash
cd terraform

# Initialize Terraform
terraform init

# Review deployment plan
terraform plan

# Deploy infrastructure
terraform apply
```

**Expected Resources Created**:
- EC2 instance (t4g.micro) with SSM agent
- Elastic IP for static IP address
- EBS volume (20GB) for persistent data
- Security groups (HTTP/HTTPS only)
- IAM roles for EC2 and GitHub Actions
- AWS Backup vault for daily snapshots

**Important Outputs**:
After `terraform apply`, note the following outputs:
- `instance_id`: EC2 instance ID (needed for GitHub variables)
- `instance_public_ip`: Elastic IP address (for DNS)
- `application_url`: Your application URL
- `github_setup_required`: Configuration instructions

### Step 3: Configure DNS

Point your domain to the Elastic IP from Terraform outputs:

```bash
# Get IP from Terraform output
terraform output instance_public_ip

# Create DNS A record at your DNS provider
planningpoker.yourdomain.com  A  <elastic-ip-address>
```

**DNS Propagation**: Wait 5-15 minutes for DNS to propagate before first deployment.

### Step 4: Configure GitHub Repository

Add repository variables via GitHub CLI or web UI:

**Option 1: GitHub CLI (Recommended)**

```bash
# Set from Terraform outputs
gh variable set EC2_INSTANCE_ID --body "$(terraform output -raw instance_id)" --repo yourusername/planning-poker
gh variable set AWS_REGION --body "$(terraform output -raw aws_region)" --repo yourusername/planning-poker
gh variable set DOMAIN_NAME --body "$(terraform output -raw domain_name)" --repo yourusername/planning-poker

# Verify variables
gh variable list --repo yourusername/planning-poker
```

**Option 2: GitHub Web UI**

Go to: `https://github.com/yourusername/planning-poker/settings/variables/actions`

Add these **Repository Variables**:
- `EC2_INSTANCE_ID` = `i-xxxxxxxxxxxxx` (from Terraform output)
- `AWS_REGION` = `us-east-1` (your AWS region)
- `DOMAIN_NAME` = `planningpoker.yourdomain.com` (your domain)

**GitHub Secrets** (add these in Settings → Secrets):
- `AWS_ACCESS_KEY_ID` = IAM user access key (from Terraform or AWS IAM)
- `AWS_SECRET_ACCESS_KEY` = IAM user secret key

### Step 5: Verify SSM Agent

Check that the EC2 instance is registered with SSM:

```bash
# Check SSM agent status
aws ssm describe-instance-information \
  --filters "Key=InstanceIds,Values=$(terraform output -raw instance_id)" \
  --region us-east-1

# Should show: PingStatus: Online
```

**Troubleshooting**: If not online, wait 2-3 minutes for SSM agent initialization, or check EC2 instance logs via AWS Console.

## Deployment Workflow

### First Deployment

1. **Commit and Push Code**:
   ```bash
   git add .
   git commit -m "Initial deployment"
   git push origin main
   ```

2. **Create Release Tag**:
   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```

3. **Monitor Deployment**:
   ```bash
   # Watch GitHub Actions workflow
   gh run watch

   # Or view in browser
   open https://github.com/yourusername/planning-poker/actions
   ```

4. **Verify Deployment**:
   ```bash
   # Check application URL
   curl https://planningpoker.yourdomain.com

   # Should return HTTP 200 with Planning Poker UI
   ```

### Subsequent Deployments

For every deployment after the first:

```bash
# Create semantic version tag
git tag v0.2.0
git push origin v0.2.0

# GitHub Actions will automatically:
# 1. Build Docker image
# 2. Push to ghcr.io/yourusername/planning-poker:v0.2.0
# 3. Trigger SSM Run Command on EC2
# 4. Pull and deploy new image
# 5. Verify health checks
```

**Deployment Time**: Typically 2-4 minutes from tag push to live.

## Monitoring

### GitHub Actions

View deployment status:
```bash
# List recent workflow runs
gh run list --workflow=deploy.yml

# View specific run
gh run view <run-id> --log

# Watch live deployment
gh run watch
```

### AWS Systems Manager

Monitor SSM commands:
```bash
# List recent commands
aws ssm list-commands --instance-id $(terraform output -raw instance_id) --region us-east-1

# Get command output
aws ssm get-command-invocation \
  --command-id <command-id> \
  --instance-id $(terraform output -raw instance_id) \
  --region us-east-1
```

### Application Logs

SSH or use AWS Session Manager:

```bash
# SSH (if SSH key configured)
ssh ec2-user@$(terraform output -raw instance_public_ip)

# Or use SSM Session Manager (no SSH key needed)
aws ssm start-session --target $(terraform output -raw instance_id)

# View deployment logs
tail -f /var/log/user-data.log

# View application logs
docker compose -f /opt/planning-poker/docker-compose.prod.yml logs -f app

# View Traefik logs
docker compose -f /opt/planning-poker/docker-compose.prod.yml logs -f traefik
```

### Health Checks

```bash
# Application health
curl https://planningpoker.yourdomain.com/monitoring/health

# Docker container status
ssh ec2-user@<ip> docker compose -f /opt/planning-poker/docker-compose.prod.yml ps

# Expected output:
# planning-poker  running  (healthy)
# traefik         running
```

## Troubleshooting

### Deployment Fails

**Symptoms**: GitHub Actions workflow fails at "Trigger deployment via SSM" step.

**Diagnosis**:
```bash
# Check SSM agent status
aws ssm describe-instance-information \
  --filters "Key=InstanceIds,Values=$(terraform output -raw instance_id)"

# Should show PingStatus: Online
```

**Solutions**:
1. **SSM Agent Offline**: Wait 2-3 minutes for initialization, or restart:
   ```bash
   ssh ec2-user@<ip> sudo systemctl restart amazon-ssm-agent
   ```

2. **IAM Permissions**: Verify GitHub Actions IAM user has SSM permissions:
   ```bash
   aws iam get-user-policy --user-name github-actions-planning-poker --policy-name GitHubActionsSSMPolicy
   ```

3. **GitHub Variables**: Verify EC2_INSTANCE_ID, AWS_REGION, DOMAIN_NAME are set correctly:
   ```bash
   gh variable list --repo yourusername/planning-poker
   ```

### Container Fails to Start

**Symptoms**: Deployment succeeds but application is unreachable.

**Diagnosis**:
```bash
# Check container status
ssh ec2-user@<ip> docker compose -f /opt/planning-poker/docker-compose.prod.yml ps

# Check container logs
ssh ec2-user@<ip> docker compose -f /opt/planning-poker/docker-compose.prod.yml logs app
```

**Solutions**:
1. **Image Pull Failure**: Verify GHCR access:
   ```bash
   ssh ec2-user@<ip> docker pull ghcr.io/yourusername/planning-poker:latest
   ```

2. **Environment Variables**: Check environment variables are set:
   ```bash
   ssh ec2-user@<ip> cat /etc/environment
   # Should contain DOMAIN_NAME and LETS_ENCRYPT_EMAIL
   ```

3. **Port Conflicts**: Verify ports 80/443 are available:
   ```bash
   ssh ec2-user@<ip> sudo netstat -tlnp | grep -E ':(80|443)'
   ```

### SSL Certificate Issues

**Symptoms**: HTTPS not working, certificate errors.

**Diagnosis**:
```bash
# Check Traefik logs
ssh ec2-user@<ip> docker compose -f /opt/planning-poker/docker-compose.prod.yml logs traefik | grep acme

# Check certificate files
ssh ec2-user@<ip> ls -la /mnt/data/traefik/acme/
```

**Solutions**:
1. **DNS Not Propagated**: Wait 15 minutes, verify DNS:
   ```bash
   dig planningpoker.yourdomain.com +short
   # Should return your Elastic IP
   ```

2. **Let's Encrypt Rate Limit**: Check rate limits at https://letsencrypt.org/docs/rate-limits/
   - Limit: 50 certificates per domain per week
   - Solution: Use Let's Encrypt staging during testing

3. **Firewall**: Verify ports 80/443 are open in security group:
   ```bash
   aws ec2 describe-security-groups \
     --group-ids $(terraform output -raw security_group_id) \
     --region us-east-1
   ```

### Rollback Deployment

**Quick Rollback** (deploy previous version):

```bash
# Find previous working tag
git tag -l

# Deploy previous version
git tag v0.1.9-rollback
git push origin v0.1.9-rollback

# Manually trigger deployment to specific version
aws ssm send-command \
  --instance-ids $(terraform output -raw instance_id) \
  --document-name "AWS-RunShellScript" \
  --parameters 'commands=["/opt/planning-poker/scripts/deploy.sh v0.1.8 v0.1.8"]' \
  --region us-east-1
```

**Complete Rollback** (restore from backup):

```bash
# List available backups
aws backup list-recovery-points-by-backup-vault \
  --backup-vault-name planning-poker-backup \
  --region us-east-1

# Restore EBS volume from backup (requires AWS Console or additional CLI commands)
# See AWS Backup documentation: https://docs.aws.amazon.com/aws-backup/
```

## Backup and Recovery

### Automated Backups

AWS Backup automatically creates daily snapshots of the EBS volume at 2 AM UTC:

```bash
# View backup plan
terraform output backup_plan

# List backups
aws backup list-recovery-points-by-backup-vault \
  --backup-vault-name $(terraform output -raw backup_vault) \
  --region us-east-1
```

**Retention**: 7 days (configurable via `backup_retention_days` in terraform.tfvars)

### Manual Backup

Create an on-demand snapshot before risky changes:

```bash
# Create EBS snapshot
aws ec2 create-snapshot \
  --volume-id $(terraform output -raw ebs_volume_id) \
  --description "Manual backup before major update" \
  --region us-east-1
```

### Disaster Recovery

Full disaster recovery process:

1. **Restore Infrastructure**: Re-run `terraform apply` (infrastructure is immutable)
2. **Restore Data**: Restore EBS volume from latest AWS Backup snapshot
3. **Redeploy Application**: Push a new tag to trigger deployment
4. **Verify DNS**: Ensure domain points to new Elastic IP

**Recovery Time Objective (RTO)**: 15-30 minutes
**Recovery Point Objective (RPO)**: 24 hours (daily backups)

## Infrastructure Maintenance

### Updating Terraform Configuration

After modifying Terraform files:

```bash
cd terraform

# Review changes
terraform plan

# Apply changes
terraform apply

# Update GitHub variables if instance ID changed
gh variable set EC2_INSTANCE_ID --body "$(terraform output -raw instance_id)" --repo yourusername/planning-poker
```

### Upgrading EC2 Instance Type

1. **Update terraform.tfvars**:
   ```hcl
   instance_type = "t4g.small"  # From t4g.micro
   ```

2. **Apply changes**:
   ```bash
   terraform apply
   # Instance will be stopped, upgraded, and restarted
   ```

3. **Redeploy application**:
   ```bash
   git tag v0.x.x-redeployed
   git push origin v0.x.x-redeployed
   ```

### Destroying Infrastructure

**WARNING**: This will permanently delete all resources and data.

```bash
cd terraform

# Review what will be deleted
terraform plan -destroy

# Destroy all resources
terraform destroy

# Confirm deletion of:
# - EC2 instance
# - EBS volume (all data lost)
# - Elastic IP
# - Security groups
# - IAM roles
# - AWS Backup vault and snapshots
```

**Before destroying**:
1. Create manual EBS snapshot for data recovery
2. Download any critical data from `/mnt/data/pb_data`
3. Document any custom configurations

## Security Considerations

### IAM Permissions

The deployment uses two IAM entities:

1. **EC2 Instance Role**:
   - `AmazonSSMManagedInstanceCore` - SSM agent access
   - No internet egress beyond AWS services

2. **GitHub Actions User**:
   - SSM SendCommand and GetCommandInvocation
   - No EC2 instance access (read-only metadata)
   - No SSH key association

### Network Security

- **Inbound**: Only HTTP (80) and HTTPS (443) allowed
- **Outbound**: Full internet access (for Docker pulls, DNS)
- **No SSH**: SSM Session Manager provides secure shell access without SSH keys
- **Elastic IP**: Static IP prevents IP changes on instance restart

### Secret Management

Secrets stored in GitHub Secrets (encrypted at rest):
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`

**Never commit**:
- AWS credentials
- PocketBase admin passwords
- Let's Encrypt certificates (auto-generated)

### SSL/TLS

- **Let's Encrypt**: Free, automated SSL certificates
- **Auto-renewal**: Traefik automatically renews certificates before expiration
- **HTTPS Redirect**: All HTTP traffic redirected to HTTPS
- **TLS Version**: TLS 1.2+ (Traefik default)

## Cost Estimation

**Monthly AWS Costs** (us-east-1, approximate):

| Service | Configuration | Monthly Cost |
|---------|---------------|--------------|
| EC2 t4g.micro | 1 instance, 24/7 | $6.14 |
| EBS gp3 | 20 GB | $1.60 |
| Elastic IP | 1 IP, attached | $0.00 |
| Data Transfer | 10 GB/month out | $0.90 |
| AWS Backup | 7 days retention | $0.40 |
| **Total** | | **~$9/month** |

**Additional Costs**:
- Domain registration: $10-15/year (external provider)
- Let's Encrypt: Free
- GitHub Actions: Free (public repo) or included in plan
- GHCR storage: Free (public images) or included in plan

**Scaling Costs**:
- Upgrade to t4g.small: ~$12/month (+$6)
- Increase EBS to 50 GB: ~$4/month (+$2.40)
- Add CloudWatch monitoring: ~$3/month

## Support and Resources

### AWS Documentation
- [AWS Systems Manager](https://docs.aws.amazon.com/systems-manager/)
- [Amazon EC2](https://docs.aws.amazon.com/ec2/)
- [AWS Backup](https://docs.aws.amazon.com/aws-backup/)

### Terraform Documentation
- [AWS Provider](https://registry.terraform.io/providers/hashicorp/aws/latest/docs)
- [Terraform CLI](https://developer.hashicorp.com/terraform/cli)

### GitHub Actions
- [GitHub Actions Workflows](https://docs.github.com/en/actions)
- [GitHub Packages (GHCR)](https://docs.github.com/en/packages)

### Application Stack
- [Traefik](https://doc.traefik.io/traefik/)
- [Docker Compose](https://docs.docker.com/compose/)
- [Let's Encrypt](https://letsencrypt.org/docs/)
