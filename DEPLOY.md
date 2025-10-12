# Planning Poker Deployment Guide

Automated deployment to AWS Lightsail Container Service using GitHub Actions.

## Prerequisites

- AWS account
- GitHub account with repository access
- GitHub CLI (`gh`) installed
- Terraform installed
- AWS CLI installed

## Architecture Overview

```
Developer                GitHub Actions              AWS Lightsail
    |                          |                            |
    |-- git push tag v* ------>|                            |
    |                          |-- run tests                |
    |                          |-- build Docker image       |
    |                          |-- push to Lightsail ------>|
    |                          |-- deploy container ------->|
    |                                                       |-- running service
    |<-- deployment complete <---------------------------------|
```

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

After the script completes, run the provided `gh secret set` commands:

```bash
gh secret set AWS_ACCESS_KEY_ID --body "your-access-key"
gh secret set AWS_SECRET_ACCESS_KEY --body "your-secret-key"
gh secret set TF_STATE_BUCKET --body "your-bucket-name"
gh variable set AWS_REGION --body "us-east-1"
gh variable set LIGHTSAIL_SERVICE_NAME --body "planning-poker-prod"
```

### 2. Configure Terraform Backend

After running `setup-aws.sh`, create backend config files:

```bash
cd terraform

# For staging
cp backend-staging.hcl.example backend-staging.hcl
# Edit backend-staging.hcl and replace the bucket name

# For production
cp backend-production.hcl.example backend-production.hcl
# Edit backend-production.hcl and replace the bucket name
```

Example `backend-production.hcl`:
```hcl
bucket = "planning-poker-terraform-state-abc123"  # Your actual bucket name
key    = "production/terraform.tfstate"
region = "us-east-1"
```

### 3. Update Environment Variables

Edit your environment tfvars file:

```bash
# For production
vim terraform/production.tfvars
```

Update these values:
```hcl
service_name    = "planning-poker-prod"
container_power = "micro"  # or "nano" for lower cost
container_scale = 1
domain_name     = "your-domain.com"  # optional
```

### 4. Deploy Infrastructure with Terraform

Initialize and apply Terraform:

```bash
cd terraform

# For production
terraform init -backend-config=backend-production.hcl
terraform plan -var-file=production.tfvars
terraform apply -var-file=production.tfvars
```

Terraform will create:
- AWS Lightsail Container Service
- Container service configuration
- Optional: SSL certificate for custom domain

Note the service URL from the output:

```bash
terraform output service_url
```

## Automated Deployment

### How It Works

1. **Push a version tag** to trigger automated deployment:

```bash
git tag v1.0.0
git push origin v1.0.0
```

2. **GitHub Actions automatically**:
   - Runs all tests
   - Builds Docker image with version info
   - Pushes image to Lightsail Container Service
   - Deploys the new container
   - Updates the running service

3. **Monitor the deployment**:
   - Go to GitHub Actions tab
   - Watch the "Deploy to AWS Lightsail" workflow
   - Check the deployment logs

### Deployment Workflow

The `.github/workflows/deploy.yml` workflow:

1. **Test Job**:
   - Checks out code
   - Sets up Go environment
   - Generates templ templates
   - Runs all tests

2. **Deploy Job** (runs after tests pass):
   - Builds Docker image with version metadata
   - Pushes image to Lightsail
   - Creates deployment configuration
   - Deploys to container service
   - Verifies deployment success

### Environment Variables

The container runs with these environment variables:
- `PP_ENV=production` - Environment name
- Application listens on port `8090`

## Monitoring

### Service Status

Check service status via AWS CLI:

```bash
# Get service details
aws lightsail get-container-services --service-name planning-poker-prod

# View service state
aws lightsail get-container-services \
  --service-name planning-poker-prod \
  --query 'containerServices[0].state' \
  --output text

# Get service URL
aws lightsail get-container-services \
  --service-name planning-poker-prod \
  --query 'containerServices[0].url' \
  --output text
```

### View Logs

```bash
# View container logs
aws lightsail get-container-log \
  --service-name planning-poker-prod \
  --container-name planning-poker
```

### Health Checks

The container includes a health check:
- Endpoint: `http://localhost:8090/`
- Interval: 30 seconds
- Timeout: 3 seconds
- Unhealthy threshold: 3 failures

## Scaling

### Vertical Scaling (Change Container Power)

Update `terraform/production.tfvars`:

```hcl
container_power = "small"  # Upgrade to 0.5 vCPU, 2 GB RAM
```

Apply changes:

```bash
terraform apply -var-file=production.tfvars
```

### Horizontal Scaling (Add More Containers)

Update `terraform/production.tfvars`:

```hcl
container_scale = 2  # Run 2 container instances
```

Apply changes:

```bash
terraform apply -var-file=production.tfvars
```

## Rollback

### Rollback to Previous Version

```bash
# List recent deployments
aws lightsail get-container-service-deployments \
  --service-name planning-poker-prod

# The workflow automatically tags images with versions
# To rollback, redeploy a previous tag:
git push origin v1.0.0  # Re-push old tag to trigger redeployment
```

## Custom Domain Setup

### 1. Update Terraform Configuration

Edit `terraform/production.tfvars`:

```hcl
domain_name = "poker.example.com"
# domain_alternative_names = ["www.poker.example.com"]
```

Apply:

```bash
terraform apply -var-file=production.tfvars
```

### 2. Update DNS

Point your domain to the Lightsail service:

```bash
# Get the Lightsail service URL
terraform output service_url

# Create a CNAME record:
# poker.example.com -> [lightsail-service-url]
```

### 3. Enable HTTPS

Lightsail automatically provides HTTPS with Let's Encrypt certificates when you configure a custom domain.

## Troubleshooting

### Deployment Fails

1. **Check GitHub Actions logs**:
   - Go to Actions tab in GitHub
   - Click on the failed workflow
   - Review error messages

2. **Check service state**:
```bash
aws lightsail get-container-services --service-name planning-poker-prod
```

3. **View container logs**:
```bash
aws lightsail get-container-log \
  --service-name planning-poker-prod \
  --container-name planning-poker
```

### Container Won't Start

1. **Verify image was pushed**:
```bash
aws lightsail get-container-images --service-name planning-poker-prod
```

2. **Check deployment status**:
```bash
aws lightsail get-container-service-deployments --service-name planning-poker-prod
```

3. **Test image locally**:
```bash
docker build -t planning-poker:test .
docker run -p 8090:8090 planning-poker:test
curl http://localhost:8090
```

### Service Unhealthy

1. **Check health check configuration** in deployment
2. **Verify application starts correctly** via logs
3. **Ensure port 8090 is exposed** and application binds to 0.0.0.0

## Cost Optimization

### Container Service Pricing

- **Nano**: 0.25 vCPU, 512 MB RAM - $7/month
- **Micro**: 0.25 vCPU, 1 GB RAM - $10/month
- **Small**: 0.5 vCPU, 2 GB RAM - $20/month
- **Medium**: 1 vCPU, 4 GB RAM - $40/month

Multiply by `container_scale` for total cost.

### Cost Saving Tips

1. **Start with Nano power** for development
2. **Use scale=1** unless you need high availability
3. **Monitor resource usage** and adjust power accordingly
4. **Destroy staging environments** when not in use:

```bash
terraform destroy -var-file=staging.tfvars
```

## Infrastructure Cleanup

To destroy all AWS resources:

```bash
cd terraform
terraform destroy -var-file=production.tfvars
```

**Warning**: This will delete your container service and all data.

## Security Notes

- Container runs as non-root user (UID 1000)
- Image built with `CGO_ENABLED=0` for security
- Health checks monitor service availability
- HTTPS provided automatically with custom domains
- Keep AWS credentials secure - never commit them
- Rotate AWS access keys periodically

## GitHub Secrets Required

| Secret | Description | Required |
|--------|-------------|----------|
| `AWS_ACCESS_KEY_ID` | IAM user access key | Yes |
| `AWS_SECRET_ACCESS_KEY` | IAM user secret key | Yes |

## GitHub Variables Required

| Variable | Description | Default |
|----------|-------------|---------|
| `AWS_REGION` | AWS region | us-east-1 |
| `LIGHTSAIL_SERVICE_NAME` | Container service name | planning-poker-prod |

## Local Development vs Production

### Local Development (Docker Compose)

```bash
# Start local development
docker compose up
```

### Production Deployment

```bash
# Deploy to production
git tag v1.0.0
git push origin v1.0.0
```

The production deployment is fully automated via GitHub Actions.

## Next Steps

1. ✅ Set up custom domain and SSL certificate
2. ✅ Monitor container metrics via AWS Console
3. ✅ Configure automatic backups for PocketBase data
4. Set up CloudWatch alarms for service health
5. Configure WebSocket allowed origins for production domain
