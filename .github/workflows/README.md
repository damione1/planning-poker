# GitHub Actions Workflows

This directory contains GitHub Actions workflows for automated CI/CD.

## Workflows

### deploy.yml - Production Deployment

**Trigger**: Git tag push matching `v*` (e.g., `v0.1.0`, `v1.2.3`)

**Purpose**: Automated deployment to AWS EC2 via AWS Systems Manager (SSM)

**Architecture**:
```
Tag Push (v*)
    ↓
GitHub Actions
    ↓
1. Build Docker image
    ↓
2. Push to GHCR (ghcr.io/damione1/planning-poker)
    ↓
3. Trigger SSM Run Command on EC2
    ↓
4. EC2 pulls image from GHCR
    ↓
5. Docker Compose deploys containers
    ↓
6. Health check validates deployment
```

**Required GitHub Secrets**:
- `AWS_ACCESS_KEY_ID` - IAM user access key for SSM commands
- `AWS_SECRET_ACCESS_KEY` - IAM user secret access key

**Required GitHub Variables**:
- `EC2_INSTANCE_ID` - EC2 instance ID (from Terraform output)
- `AWS_REGION` - AWS region (e.g., `us-east-1`)
- `DOMAIN_NAME` - Production domain name

**Setup GitHub Variables**:

```bash
# Using GitHub CLI (recommended)
gh variable set EC2_INSTANCE_ID --body "i-xxxxxxxxxxxxx" --repo damione1/planning-poker
gh variable set AWS_REGION --body "us-east-1" --repo damione1/planning-poker
gh variable set DOMAIN_NAME --body "planningpoker.yourdomain.com" --repo damione1/planning-poker

# Verify variables
gh variable list --repo damione1/planning-poker
```

Or via web UI: https://github.com/damione1/planning-poker/settings/variables/actions

**Workflow Steps**:

1. **Extract Version**
   - Extracts semantic version from git tag
   - Sets `version` output for subsequent steps

2. **Set up Docker Buildx**
   - Configures multi-architecture build support

3. **Log in to GitHub Container Registry**
   - Authenticates to GHCR using GitHub token
   - Automatically available via `GITHUB_TOKEN`

4. **Build and Push Docker Image**
   - Builds multi-architecture image (linux/amd64, linux/arm64)
   - Tags with version and `latest`
   - Pushes to `ghcr.io/damione1/planning-poker`

5. **Trigger Deployment via SSM**
   - Sends SSM Run Command to EC2 instance
   - Executes `/opt/planning-poker/scripts/deploy.sh`
   - Returns command ID for status tracking

6. **Wait for Deployment**
   - Polls SSM command status (max 5 minutes)
   - Checks every 10 seconds for completion
   - Fails workflow if deployment fails or times out

7. **Deployment Summary**
   - Posts deployment status to workflow summary
   - Includes version, image URL, instance ID

**Usage**:

```bash
# Create and push semantic version tag
git tag v0.1.0
git push origin v0.1.0

# Monitor workflow
gh run watch

# View workflow logs
gh run list --workflow=deploy.yml
gh run view <run-id> --log
```

**Deployment Time**: Typically 2-4 minutes from tag push to live

**Rollback**:

Deploy previous version by pushing a new tag pointing to old commit:

```bash
# Deploy previous version
git tag v0.1.9-rollback
git push origin v0.1.9-rollback
```

Or manually trigger SSM command:

```bash
aws ssm send-command \
  --instance-ids i-xxxxxxxxxxxxx \
  --document-name "AWS-RunShellScript" \
  --parameters 'commands=["/opt/planning-poker/scripts/deploy.sh v0.1.8 v0.1.8"]' \
  --region us-east-1
```

**Troubleshooting**:

| Issue | Diagnosis | Solution |
|-------|-----------|----------|
| Workflow fails at "Trigger deployment" | SSM agent offline or IAM permissions missing | Verify SSM agent status: `aws ssm describe-instance-information` |
| Image push fails | GHCR authentication issue | Check `GITHUB_TOKEN` permissions include `packages:write` |
| Deployment times out | SSM command not completing | Check EC2 logs: `tail -f /var/log/user-data.log` |
| Health check fails | Container not starting | Check container logs: `docker compose logs app` |

**Monitoring**:

View recent deployments:
```bash
# List workflow runs
gh run list --workflow=deploy.yml --limit 10

# View specific run
gh run view <run-id>

# Watch live deployment
gh run watch
```

View SSM command history:
```bash
# List recent SSM commands
aws ssm list-commands --instance-id i-xxxxxxxxxxxxx --region us-east-1

# Get command details
aws ssm get-command-invocation \
  --command-id <command-id> \
  --instance-id i-xxxxxxxxxxxxx \
  --region us-east-1
```

**Security**:

- **No SSH Required**: Deployment uses SSM Run Command, no SSH keys exposed
- **IAM Least Privilege**: GitHub Actions user has minimal SSM permissions only
- **Encrypted Secrets**: AWS credentials stored in GitHub Secrets (encrypted at rest)
- **Audit Trail**: All deployments logged in GitHub Actions and AWS CloudTrail

**Architecture Benefits**:

1. **Zero SSH Deployment**: No SSH keys, no exposed ports beyond HTTP/HTTPS
2. **Single Source of Truth**: GHCR as the only artifact repository (no S3)
3. **Atomic Deployments**: Docker Compose ensures all-or-nothing updates
4. **Health Validation**: Automated health checks before marking deployment successful
5. **Automatic Rollback**: Failed deployments don't affect running services

**Cost**:

- GitHub Actions: Free for public repositories
- GHCR Storage: Free for public images
- AWS SSM: Free (no additional charge beyond EC2)

## Setup

### 1. Deploy Infrastructure

Deploy AWS infrastructure using Terraform:

```bash
cd terraform

# Initialize Terraform
terraform init

# Review and apply
terraform plan
terraform apply
```

**Terraform Outputs**: Note the instance ID, region, and domain from outputs.

### 2. Configure DNS

Point your domain to the Elastic IP from Terraform outputs:

```bash
# Get IP from Terraform
terraform output instance_public_ip

# Create A record at your DNS provider
planningpoker.yourdomain.com  A  <elastic-ip-address>
```

### 3. Configure GitHub Repository

Using Terraform outputs, configure GitHub variables:

```bash
# Option 1: Using GitHub CLI (recommended)
gh variable set EC2_INSTANCE_ID --body "$(terraform output -raw instance_id)" --repo damione1/planning-poker
gh variable set AWS_REGION --body "$(terraform output -raw aws_region)" --repo damione1/planning-poker
gh variable set DOMAIN_NAME --body "$(terraform output -raw domain_name)" --repo damione1/planning-poker

# Verify variables
gh variable list --repo damione1/planning-poker
```

**GitHub Secrets** (add via Settings → Secrets):
- `AWS_ACCESS_KEY_ID` - From Terraform or AWS IAM
- `AWS_SECRET_ACCESS_KEY` - From Terraform or AWS IAM

### 4. Verify SSM Agent

Ensure EC2 instance is registered with SSM:

```bash
aws ssm describe-instance-information \
  --filters "Key=InstanceIds,Values=$(terraform output -raw instance_id)" \
  --region us-east-1

# Should show: PingStatus: Online
```

### 5. First Deployment

Create and push a version tag:

```bash
git tag v0.1.0
git push origin v0.1.0

# Monitor deployment
gh run watch
```

## Workflow Development

**Testing Workflow Changes**:

1. Create feature branch: `git checkout -b workflow/test-deployment`
2. Modify `.github/workflows/deploy.yml`
3. Push branch: `git push origin workflow/test-deployment`
4. Create test tag: `git tag v0.0.1-test && git push origin v0.0.1-test`
5. Monitor: `gh run watch`
6. Verify deployment on test domain

**Workflow Best Practices**:

- Use semantic versioning for tags (v1.2.3)
- Include rollback instructions in deployment summary
- Set appropriate timeouts for long-running operations
- Use `--fail-fast` for error detection
- Add deployment notifications (Slack, email) for production

**Extending Workflows**:

Add pre-deployment checks:
```yaml
- name: Run Tests
  run: |
    go test ./...

- name: Lint Code
  run: |
    golangci-lint run
```

Add post-deployment validation:
```yaml
- name: Smoke Tests
  run: |
    curl -f https://${{ vars.DOMAIN_NAME }}/monitoring/health
```

Add deployment notifications:
```yaml
- name: Notify Slack
  if: always()
  uses: slackapi/slack-github-action@v1
  with:
    webhook-url: ${{ secrets.SLACK_WEBHOOK }}
    payload: |
      {
        "text": "Deployment ${{ job.status }}: ${{ github.ref_name }}"
      }
```

## References

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [AWS Systems Manager](https://docs.aws.amazon.com/systems-manager/)
- [Docker Build & Push Action](https://github.com/docker/build-push-action)
- [GitHub Container Registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [Complete Deployment Guide](../../terraform/DEPLOY.md)
