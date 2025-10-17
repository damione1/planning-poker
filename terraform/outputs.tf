output "instance_id" {
  description = "EC2 instance ID"
  value       = aws_instance.app.id
}

output "instance_public_ip" {
  description = "Elastic IP address"
  value       = aws_eip.app.public_ip
}

output "instance_public_dns" {
  description = "Public DNS name"
  value       = aws_eip.app.public_dns
}

output "ebs_volume_id" {
  description = "EBS volume ID for persistent data"
  value       = aws_ebs_volume.data.id
}

output "security_group_id" {
  description = "Security group ID"
  value       = aws_security_group.app.id
}

output "application_url" {
  description = "Application URL"
  value       = "https://${var.domain_name}"
}

output "ssh_command" {
  description = "SSH command to connect to the instance"
  value       = "ssh ec2-user@${aws_eip.app.public_ip}"
}

output "github_setup_required" {
  description = "⚠️  REQUIRED: GitHub repository configuration before first deployment"
  value       = <<EOT
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⚠️  IMPORTANT: Configure GitHub before deploying
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Option 1: Using GitHub CLI (recommended)
────────────────────────────────────────

gh variable set EC2_INSTANCE_ID --body "${aws_instance.app.id}" --repo ${var.github_repo}
gh variable set AWS_REGION --body "${var.aws_region}" --repo ${var.github_repo}
gh variable set DOMAIN_NAME --body "${var.domain_name}" --repo ${var.github_repo}

Verify secrets exist:
gh variable list --repo ${var.github_repo}

Option 2: Using Web UI
──────────────────────

Go to: https://github.com/${var.github_repo}/settings/variables/actions

Add these Repository Variables:
  EC2_INSTANCE_ID = ${aws_instance.app.id}
  AWS_REGION      = ${var.aws_region}
  DOMAIN_NAME     = ${var.domain_name}

Then verify AWS secrets exist (Settings → Secrets):
  ✓ AWS_ACCESS_KEY_ID
  ✓ AWS_SECRET_ACCESS_KEY

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
EOT
}

output "deployment_instructions" {
  description = "Deployment workflow"
  value       = <<EOT
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Deployment Workflow
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Deployments are handled automatically via AWS Systems Manager (SSM).

To deploy:
  1. Commit your changes
  2. Create a git tag: git tag v0.1.0
  3. Push the tag: git push origin v0.1.0
  4. GitHub Actions will:
     → Build Docker image
     → Push to ghcr.io/${var.github_repo}
     → Trigger SSM Run Command on EC2
     → Deploy containers

Monitor deployment:
  • GitHub Actions: https://github.com/${var.github_repo}/actions
  • AWS Systems Manager: https://console.aws.amazon.com/systems-manager/run-command

Access instance:
  • SSH: ssh ec2-user@${aws_eip.app.public_ip}
  • Session Manager: aws ssm start-session --target ${aws_instance.app.id}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
EOT
}

output "dns_setup" {
  description = "DNS configuration instructions"
  value       = <<EOT
Configure your DNS (${var.domain_name}) to point to:
IP: ${aws_eip.app.public_ip}

Create an A record:
${var.domain_name}  A  ${aws_eip.app.public_ip}
EOT
}

output "backup_vault" {
  description = "AWS Backup vault for EBS snapshots"
  value       = aws_backup_vault.data.name
}

output "backup_plan" {
  description = "AWS Backup plan details"
  value = {
    name              = aws_backup_plan.data.name
    schedule          = "Daily at 2 AM UTC"
    retention_days    = var.backup_retention_days
  }
}

output "container_registry" {
  description = "Container registry for Docker images"
  value       = "ghcr.io/${var.github_repo}"
}

output "ssm_agent_status" {
  description = "Command to check SSM agent status"
  value       = "aws ssm describe-instance-information --filters \"Key=InstanceIds,Values=${aws_instance.app.id}\""
}
