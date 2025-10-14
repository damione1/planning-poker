#!/bin/bash
set -e

echo "=== Stopping application ==="

cd /opt/planning-poker

# Load and export environment variables
if [ -f /etc/environment ]; then
    set -a  # automatically export all variables
    source /etc/environment
    set +a
fi

# Explicitly export for docker compose
export DOMAIN_NAME
export LETS_ENCRYPT_EMAIL

# Check if docker compose is running
if docker compose -f docker-compose.prod.yml ps | grep -q "Up"; then
    echo "Stopping containers..."
    docker compose -f docker-compose.prod.yml down
    echo "✅ Containers stopped"
else
    echo "ℹ️ No running containers found"
fi

echo "=== Application stop complete ==="
