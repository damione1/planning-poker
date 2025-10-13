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
  value       = "ssh ubuntu@${aws_eip.app.public_ip}"
}

output "deployment_instructions" {
  description = "Manual deployment instructions"
  value       = <<EOT
To deploy updates:
1. SSH into the instance: ssh ubuntu@${aws_eip.app.public_ip}
2. Run deployment script: sudo bash /opt/planning-poker/scripts/deploy.sh

Or use GitHub Actions with SSH deployment.
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
