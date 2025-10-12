output "instance_id" {
  description = "ID of the Lightsail instance"
  value       = aws_lightsail_instance.app.id
}

output "instance_name" {
  description = "Name of the Lightsail instance"
  value       = aws_lightsail_instance.app.name
}

output "public_ip" {
  description = "Public IP address of the instance"
  value       = aws_lightsail_instance.app.public_ip_address
}

output "static_ip" {
  description = "Static IP address (if enabled)"
  value       = var.enable_static_ip ? aws_lightsail_static_ip.app[0].ip_address : null
}

output "ssh_command" {
  description = "SSH command to connect to the instance"
  value       = "ssh ubuntu@${var.enable_static_ip ? aws_lightsail_static_ip.app[0].ip_address : aws_lightsail_instance.app.public_ip_address}"
}

output "app_url" {
  description = "Application URL"
  value       = "http://${var.enable_static_ip ? aws_lightsail_static_ip.app[0].ip_address : aws_lightsail_instance.app.public_ip_address}:${var.app_port}"
}

output "deployment_command" {
  description = "Command to deploy application package"
  value       = "REMOTE_HOST=${var.enable_static_ip ? aws_lightsail_static_ip.app[0].ip_address : aws_lightsail_instance.app.public_ip_address} ./deploy/deploy.sh dist/planning-poker-v<VERSION>.tar.gz"
}

output "instance_details" {
  description = "Instance configuration details"
  value = {
    blueprint  = var.blueprint_id
    bundle     = var.bundle_id
    az         = var.availability_zone
    region     = var.aws_region
    created_at = aws_lightsail_instance.app.created_at
  }
}
