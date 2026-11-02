# Planning Poker AWS deployment

Production currently runs on Lightsail `planning-poker` in account `912928333407`, region `us-east-1`. HTTP and HTTPS are public. Deployment and administration use the Systems Manager managed node; regular deployment does not require port 22.

## Ownership

- This Terraform configuration imports and manages the production instance, static IP, attachment, firewall and automatic snapshots. It prevents instance/IP destruction.
- Account IAM, OIDC, restricted SSM documents, audit logging, security controls and application backups are managed by `../homelab/terraform/aws-account/` in the sibling homelab repository.
- The new Applications account is `675704851310`. Its baseline is in `../homelab/terraform/aws-applications/`. Production remains in the original account until the planned migration and DNS cutover are completed.

## Terraform

```sh
aws sso login --profile aws-admin
terraform init -backend-config=backend-security.hcl
terraform plan -var aws_profile=aws-admin
terraform apply -var aws_profile=aws-admin
```

Use this local backend configuration (it contains names only):

```hcl
bucket         = "planning-poker-terraform-state-72cf7d8d"
key            = "production/terraform.tfstate"
region         = "us-east-1"
profile        = "aws-terraform"
encrypt        = true
dynamodb_table = "panetone-terraform-locks"
```

`aws-terraform` is a local `credential_process` profile that obtains temporary credentials from `aws-admin`; it supports the existing Terraform 1.5 S3 backend without storing access keys. The AWS provider version is pinned in `main.tf` and `.terraform.lock.hcl`.

The default `ssh_allowed_cidrs = []` closes SSH. A temporary recovery source can be added through Terraform if SSM needs repair; remove it after recovery. Keep ports 80 and 443 available for the application and certificate renewal.

## GitHub Actions

The release workflow builds the application image and assumes `deploy-planning-poker` through GitHub OIDC. Its trust allows this repository's release tags and the `master` branch used by the access verification workflow. Only the deploy job requests `id-token: write`.

Repository variables:

| Variable | Value |
|---|---|
| `AWS_REGION` | `us-east-1` |
| `AWS_DEPLOY_ROLE_ARN` | `arn:aws:iam::912928333407:role/deploy-planning-poker` |
| `SSM_INSTANCE_ID` | `mi-085330c5d4f0b3c52` |
| `DOMAIN_NAME` | `pokerplanning.net` |

No permanent AWS access key or SSH private key is needed by these workflows. The old `scripts/setup-aws.sh` no longer creates credentials.

`Verify AWS deployment access` can be dispatched from GitHub Actions. It assumes the role and runs the read-only `PlanningPoker-Health` document. The deployment role can invoke only the deployment and health documents on the configured managed node; it cannot run arbitrary SSM shell commands.

## Manual deployment

```sh
export AWS_PROFILE=aws-admin AWS_REGION=us-east-1
export SSM_INSTANCE_ID=mi-085330c5d4f0b3c52
./scripts/ci-deploy-lightsail.sh 0.6.0 0.6.0
```

The restricted document runs `/opt/planning-poker/scripts/deploy.sh` as `ec2-user`. It pulls the selected image and recreates only the app container, leaving Traefik running. Versions and image tags are validated both in the client script and in the SSM document.

## Backups and recovery

Automatic Lightsail snapshots run at 08:00 UTC. An SSM association creates an application backup at 06:00 UTC: SQLite online backup with integrity checks plus application files, uploaded into the private, encrypted, versioned bucket `panetone-planning-poker-backups-912928333407`. The node can append backups but cannot read or delete them. Current archives have a 90-day lifecycle.

For application restoration, retrieve a selected archive with an administrator role, verify its checksum and SQLite integrity, stop the app, restore `pb_data`, then start it and validate the public health endpoint. File copies are taken online; restoring a disk snapshot or coordinating a maintenance window provides a stronger point-in-time boundary for concurrent file changes.

`lightsail-user-data.sh` describes host bootstrap for reconstruction. An imported instance ignores user-data changes, so editing it does not replace production. A replacement node must be registered in SSM, its identifier updated in account Terraform and GitHub variables, and its data restored before any DNS cutover.
