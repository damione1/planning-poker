#!/bin/bash
set -e

echo "=== Validating service ==="

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

# Check if containers are running
if ! docker compose -f docker-compose.prod.yml ps | grep -q "Up"; then
    echo "❌ Containers are not running"
    docker compose -f docker-compose.prod.yml logs --tail=50
    exit 1
fi

echo "✅ Containers are running"

# Check if app container is healthy
APP_CONTAINER=$(docker compose -f docker-compose.prod.yml ps -q app)
if [ -z "$APP_CONTAINER" ]; then
    echo "❌ App container not found"
    exit 1
fi

# Try to reach the health endpoint (with retries)
MAX_RETRIES=30
RETRY_COUNT=0
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if docker exec $APP_CONTAINER curl -f http://localhost:8090/api/health &>/dev/null; then
        echo "✅ Application health check passed"
        echo "=== Service validation complete ==="
        exit 0
    fi

    RETRY_COUNT=$((RETRY_COUNT + 1))
    echo "Waiting for application to be ready... (attempt $RETRY_COUNT/$MAX_RETRIES)"
    sleep 2
done

echo "❌ Application health check failed after $MAX_RETRIES attempts"
docker compose -f docker-compose.prod.yml logs --tail=50 app
exit 1
