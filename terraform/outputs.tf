output "service_name" {
  description = "Name of the Lightsail container service"
  value       = aws_lightsail_container_service.app.name
}

output "service_url" {
  description = "Public URL of the container service"
  value       = "https://${aws_lightsail_container_service.app.url}"
}

output "service_state" {
  description = "Current state of the container service"
  value       = aws_lightsail_container_service.app.state
}

output "service_power" {
  description = "Power configuration of the container service"
  value       = aws_lightsail_container_service.app.power
}

output "service_scale" {
  description = "Scale (number of nodes) of the container service"
  value       = aws_lightsail_container_service.app.scale
}

output "deployment_instructions" {
  description = "Deployment is automated via GitHub Actions"
  value       = "Push a git tag (e.g., v1.0.0) to automatically deploy to this container service"
}

output "service_details" {
  description = "Container service configuration details"
  value = {
    name     = aws_lightsail_container_service.app.name
    power    = aws_lightsail_container_service.app.power
    scale    = aws_lightsail_container_service.app.scale
    region   = var.aws_region
    url      = aws_lightsail_container_service.app.url
  }
}
