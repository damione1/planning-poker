#!/bin/bash
set -e

echo "=== Starting application ==="

cd /opt/planning-poker

# Load and export environment variables
if [ -f /etc/environment ]; then
    set -a  # automatically export all variables
    source /etc/environment
    set +a
fi

# Verify environment variables are set
echo "Environment check:"
echo "  DOMAIN_NAME: ${DOMAIN_NAME:-NOT SET}"
echo "  LETS_ENCRYPT_EMAIL: ${LETS_ENCRYPT_EMAIL:-NOT SET}"

# Pull latest image from GHCR
echo "Pulling latest image from GitHub Container Registry..."
docker compose -f docker-compose.prod.yml pull app

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
