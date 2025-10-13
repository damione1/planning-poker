#!/bin/bash
set -e

echo "=== Installing dependencies ==="

cd /opt/planning-poker

# Ensure Docker is running
if ! systemctl is-active --quiet docker; then
    echo "Starting Docker..."
    systemctl start docker
fi

# Verify docker compose plugin is available
if ! docker compose version &>/dev/null; then
    echo "❌ Docker Compose plugin not found"
    exit 1
fi

echo "✅ Docker: $(docker --version)"
echo "✅ Docker Compose: $(docker compose version)"

# Load environment variables
if [ -f /etc/environment ]; then
    source /etc/environment
fi

# Verify docker-compose.prod.yml exists (should be deployed by CodeDeploy)
if [ ! -f docker-compose.prod.yml ]; then
    echo "❌ docker-compose.prod.yml not found - should be in deployment package"
    exit 1
fi

echo "✅ docker-compose.prod.yml found"

# Ensure data directories exist
mkdir -p /mnt/data/pb_data
mkdir -p /mnt/data/traefik/acme

# Set proper ownership
chown -R ec2-user:ec2-user /opt/planning-poker
chown -R ec2-user:ec2-user /mnt/data

echo "=== Dependency installation complete ==="
