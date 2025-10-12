# GitHub Actions Workflows

Simple CI/CD workflows for Planning Poker deployment to AWS Lightsail.

## Workflows

### test.yml - Continuous Testing
**Trigger**: Push to any branch, Pull Requests

Quick validation on every push:
- Lints Go code with golangci-lint
- Runs Go tests with race detection and coverage
- Validates templ template compilation

**Status Badge**:
```markdown
![Tests](https://github.com/YOUR-ORG/planning-poker/actions/workflows/test.yml/badge.svg)
```

### release.yml - Build and Release
**Trigger**: Version tags (`v*.*.*`)

Complete build and release pipeline:
- Sets up Go and Node.js
- Generates templ templates
- Builds frontend assets
- Runs full test suite
- Builds production binary (linux/amd64)
- Creates deployment package with:
  - Binary
  - install.sh script
  - systemd service file
- Generates checksums
- Creates GitHub Release

**Usage**:
```bash
git tag v1.0.0
git push origin v1.0.0
```

**Release Artifacts**:
- `planning-poker-v{version}.tar.gz` - Deployment package
- `planning-poker-v{version}.tar.gz.sha256` - Package checksum
- `planning-poker.sha256` - Binary checksum
- `BUILD_INFO.txt` - Build metadata

## Setup

### 1. Configure GitHub Secrets

Run the AWS setup script to get your secrets:

```bash
./scripts/setup-aws.sh
```

Then configure GitHub secrets using the provided commands:

```bash
gh secret set AWS_ACCESS_KEY_ID --body "your-access-key"
gh secret set AWS_SECRET_ACCESS_KEY --body "your-secret-key"
gh secret set TF_STATE_BUCKET --body "your-bucket-name"
gh variable set AWS_REGION --body "us-east-1"
```

### 2. Provision Infrastructure

```bash
cd terraform
terraform init
terraform apply
```

### 3. Deploy

Push a version tag to trigger automated build and release:

```bash
git tag v1.0.0
git push origin v1.0.0
```

Download the release and deploy to your Lightsail instance:

```bash
# Download release
wget https://github.com/YOUR-ORG/planning-poker/releases/download/v1.0.0/planning-poker-v1.0.0.tar.gz

# Extract and upload
tar -xzf planning-poker-v1.0.0.tar.gz
scp -r planning-poker/ ubuntu@<instance-ip>:/tmp/

# Install on server
ssh ubuntu@<instance-ip>
cd /tmp/planning-poker
sudo ./install.sh
```

## Workflow Flow

```
Developer Push → test.yml (validates code)
Developer Tag  → release.yml (builds + creates release)
Manual Deploy  → Download release → SSH install
```

## Troubleshooting

**Test Failures**:
```bash
go test -v -race ./...
golangci-lint run
```

**Build Failures**:
```bash
templ generate
npm run build
go build -o planning-poker .
```

**Deployment Issues**:
See [DEPLOY.md](../../DEPLOY.md) for complete troubleshooting guide.

## Required Secrets

| Secret | Description | Source |
|--------|-------------|--------|
| `AWS_ACCESS_KEY_ID` | IAM user access key | setup-aws.sh |
| `AWS_SECRET_ACCESS_KEY` | IAM user secret key | setup-aws.sh |
| `TF_STATE_BUCKET` | Terraform state bucket | setup-aws.sh |
| `AWS_REGION` (variable) | AWS region | setup-aws.sh |
