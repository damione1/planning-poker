# Planning Poker - Terraform Infrastructure

AWS Lightsail infrastructure for Planning Poker application.

## Prerequisites

1. **AWS CLI** configured with credentials:
   ```bash
   aws configure
   ```

2. **Terraform** installed (version >= 1.0):
   ```bash
   # macOS
   brew install terraform

   # Linux
   wget https://releases.hashicorp.com/terraform/1.6.0/terraform_1.6.0_linux_amd64.zip
   unzip terraform_1.6.0_linux_amd64.zip
   sudo mv terraform /usr/local/bin/
   ```

3. **SSH key pair** for instance access:
   ```bash
   # Generate if you don't have one
   ssh-keygen -t rsa -b 4096 -C "your-email@example.com"
   ```

## Quick Start

### 1. Configure Variables

Copy the example variables file and customize:

```bash
cp terraform.tfvars.example terraform.tfvars
```

Edit `terraform.tfvars` with your configuration:
- Set your AWS region and availability zone
- Configure SSH key path
- Restrict `allowed_ssh_cidrs` to your IP address for security
- Set `ws_allowed_origins` to your domain (or keep `*` for testing)

### 2. Initialize Terraform

```bash
cd terraform
terraform init
```

### 3. Review the Plan

```bash
terraform plan
```

This shows what resources will be created.

### 4. Apply Infrastructure

```bash
terraform apply
```

Type `yes` when prompted to confirm.

## Infrastructure Details

### Resources Created

- **Lightsail Instance**: Ubuntu 24.04 LTS server
- **Static IP**: Persistent IP address (optional)
- **SSH Key Pair**: For secure instance access
- **Firewall Rules**: Ports 22 (SSH), 80 (HTTP), 443 (HTTPS), 8090 (app)
- **Automatic Snapshots**: Daily backups (optional)

### Instance Configuration

**Default Bundle** (`nano_3_0`):
- **Cost**: $3.50/month
- **RAM**: 512MB
- **CPU**: 1 vCPU
- **Storage**: 20GB SSD
- **Transfer**: 1TB/month

**Alternative Bundles**:
- `micro_3_0`: $5/month - 1GB RAM, 1 vCPU, 40GB SSD
- `small_3_0`: $10/month - 2GB RAM, 1 vCPU, 60GB SSD

To change, edit `bundle_id` in `terraform.tfvars`.

### Security Features

The user-data script automatically configures:
- **Automatic security updates** (unattended-upgrades)
- **SSH brute-force protection** (fail2ban)
- **Firewall** (UFW) with minimal open ports
- **System hardening** (security limits, sysctl tuning)
- **Log rotation** for application logs

## Post-Deployment

### 1. Get Instance Information

```bash
terraform output
```

This shows:
- Instance ID and name
- Public IP and static IP (if enabled)
- SSH command to connect
- Application URL
- Deployment command

### 2. Connect to Instance

```bash
ssh ubuntu@<instance-ip>
```

Or use the output:
```bash
$(terraform output -raw ssh_command)
```

### 3. Verify Instance Setup

```bash
# Check if initialization completed
ssh ubuntu@<instance-ip> "cat /var/lib/planning-poker/.instance-initialized"

# Check user-data script logs
ssh ubuntu@<instance-ip> "sudo tail -f /var/log/user-data.log"
```

### 4. Deploy Application

From your local machine:

```bash
# Build and package application
make release VERSION=1.0.0

# Deploy to Lightsail
REMOTE_HOST=<instance-ip> ./deploy/deploy.sh dist/planning-poker-v1.0.0.tar.gz
```

Or use the Terraform output command:
```bash
# Get deployment command
terraform output -raw deployment_command
# Replace <VERSION> with actual version, e.g., 1.0.0
```

## DNS Configuration (Optional)

### Using AWS Route 53

1. Create a hosted zone for your domain in Route 53
2. Add an A record pointing to your static IP:
   ```hcl
   # In terraform/main.tf, add:
   resource "aws_route53_record" "app" {
     zone_id = "<your-hosted-zone-id>"
     name    = "poker.example.com"
     type    = "A"
     ttl     = 300
     records = [aws_lightsail_static_ip.app[0].ip_address]
   }
   ```

### Manual DNS Configuration

Add an A record in your DNS provider:
- **Host**: `poker` (or `@` for root domain)
- **Type**: `A`
- **Value**: Your static IP (from `terraform output static_ip`)
- **TTL**: 300 seconds

## Monitoring and Maintenance

### Check Application Status

```bash
ssh ubuntu@<instance-ip> "sudo systemctl status planning-poker"
```

### View Application Logs

```bash
ssh ubuntu@<instance-ip> "sudo journalctl -u planning-poker -f"
```

### Database Backup

```bash
# SSH into instance
ssh ubuntu@<instance-ip>

# Backup database
sudo systemctl stop planning-poker
sudo cp /var/lib/planning-poker/pb_data/data.db ~/backup-$(date +%Y%m%d).db
sudo systemctl start planning-poker

# Download backup to local machine
scp ubuntu@<instance-ip>:~/backup-*.db ./backups/
```

### Update Application

```bash
# Build new version locally
make release VERSION=1.0.1

# Deploy update
REMOTE_HOST=<instance-ip> ./deploy/deploy.sh dist/planning-poker-v1.0.1.tar.gz
```

The deployment script automatically:
- Backs up the current binary and database
- Stops the service gracefully
- Installs the new version
- Starts the service
- Performs health checks

## Scaling

### Vertical Scaling (Larger Instance)

1. Update `bundle_id` in `terraform.tfvars`:
   ```hcl
   bundle_id = "small_3_0" # Upgrade to 2GB RAM
   ```

2. Apply changes:
   ```bash
   terraform apply
   ```

   ⚠️ **Note**: This will recreate the instance. Backup data first!

### Horizontal Scaling

For high-traffic scenarios:
1. Use AWS Application Load Balancer
2. Multiple Lightsail instances
3. Shared database (RDS or external)
4. Consider migrating to ECS/Fargate

## Cost Estimation

**Minimal Setup** ($3.50/month):
- Instance: `nano_3_0` - $3.50
- Static IP: Free (while attached)
- Data transfer: 1TB included

**Standard Setup** ($5/month):
- Instance: `micro_3_0` - $5
- Static IP: Free
- Data transfer: 2TB included
- Automatic snapshots: $0.05/GB (~$1/month for daily)

## Troubleshooting

### Cannot Connect via SSH

```bash
# Check security group rules
terraform show | grep -A 10 port_info

# Verify SSH key is correct
ssh-keygen -lf ~/.ssh/id_rsa.pub

# Check instance status in AWS Console
aws lightsail get-instance --instance-name planning-poker-prod
```

### Application Not Responding

```bash
# SSH into instance
ssh ubuntu@<instance-ip>

# Check service status
sudo systemctl status planning-poker

# Check logs
sudo journalctl -u planning-poker -n 100 --no-pager

# Check if port is listening
sudo netstat -tlnp | grep 8090

# Restart service
sudo systemctl restart planning-poker
```

### High Memory Usage

```bash
# Check memory usage
ssh ubuntu@<instance-ip> "free -h"

# Check application memory
ssh ubuntu@<instance-ip> "ps aux | grep planning-poker"

# Consider upgrading to larger bundle
# Edit terraform.tfvars: bundle_id = "micro_3_0"
```

## Cleanup

### Destroy All Resources

```bash
terraform destroy
```

Type `yes` to confirm.

⚠️ **Warning**: This will permanently delete:
- The instance and all data
- Static IP
- Snapshots
- SSH key pair

Backup important data before destroying!

### Partial Cleanup

To keep some resources:

```bash
# Remove only snapshots
terraform destroy -target=aws_lightsail_instance_snapshot.backup

# Remove only static IP
terraform destroy -target=aws_lightsail_static_ip.app
```

## Support

- **Repository**: https://github.com/damione1/planning-poker
- **Issues**: https://github.com/damione1/planning-poker/issues
- **AWS Lightsail Docs**: https://docs.aws.amazon.com/lightsail/
