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

output "codedeploy_app_name" {
  description = "CodeDeploy application name"
  value       = aws_codedeploy_app.app.name
}

output "codedeploy_deployment_group" {
  description = "CodeDeploy deployment group name"
  value       = aws_codedeploy_deployment_group.app.deployment_group_name
}

output "deployment_instructions" {
  description = "Deployment instructions"
  value       = <<EOT
Deployments are handled automatically via AWS CodeDeploy when you push a tag to GitHub.

To deploy manually:
1. Create a git tag: git tag v0.1.x
2. Push the tag: git push origin v0.1.x
3. GitHub Actions will create a release and trigger CodeDeploy

Monitor deployment status:
- GitHub Actions: https://github.com/${var.github_repo}/actions
- AWS CodeDeploy Console: https://console.aws.amazon.com/codesuite/codedeploy/applications/${aws_codedeploy_app.app.name}

SSH access for debugging:
ssh ec2-user@${aws_eip.app.public_ip}
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

output "codedeploy_s3_bucket" {
  description = "S3 bucket for CodeDeploy deployment packages"
  value       = aws_s3_bucket.codedeploy.bucket
}
