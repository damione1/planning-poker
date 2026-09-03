output "lightsail_instance_name" {
  description = "Lightsail instance name"
  value       = var.lightsail_instance_name
}

output "application_url" {
  description = "Application URL"
  value       = "https://${var.domain_name}"
}

output "container_registry" {
  description = "Container registry for Docker images"
  value       = "ghcr.io/damione1/planning-poker"
}
