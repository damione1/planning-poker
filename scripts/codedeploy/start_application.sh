#!/bin/bash
set -e

echo "=== Starting application ==="

cd /opt/planning-poker

# Load environment variables from Terraform (if available)
if [ -f /etc/environment ]; then
    source /etc/environment
fi

# Build the application image
echo "Building Docker image..."
docker compose -f docker-compose.prod.yml build --no-cache app

# Start all services
echo "Starting services..."
docker compose -f docker-compose.prod.yml up -d

# Wait for containers to be healthy
echo "Waiting for containers to start..."
sleep 10

# Show container status
echo "=== Container status ==="
docker compose -f docker-compose.prod.yml ps

echo "=== Application start complete ==="
