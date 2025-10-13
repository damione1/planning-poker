#!/bin/bash
set -e

# Configuration
APP_DIR="/opt/planning-poker"
COMPOSE_FILE="docker-compose.prod.yml"
GITHUB_REPO="${GITHUB_REPO:-damione1/planning-poker}"
GITHUB_REF="${GITHUB_REF:-main}"

echo "==== Deploying Planning Poker ===="
echo "Repository: $GITHUB_REPO"
echo "Reference: $GITHUB_REF"

# Navigate to app directory
cd $APP_DIR

# Pull latest changes
echo "📥 Pulling latest changes..."
git fetch origin
git checkout $GITHUB_REF
git pull origin $GITHUB_REF

# Rebuild and restart containers
echo "🔨 Building new image..."
docker compose -f $COMPOSE_FILE build --no-cache app

echo "🔄 Restarting services..."
docker compose -f $COMPOSE_FILE up -d

# Show status
echo "📊 Service status:"
docker compose -f $COMPOSE_FILE ps

# Show logs
echo "📝 Recent logs:"
docker compose -f $COMPOSE_FILE logs --tail=50 app

echo "✅ Deployment complete!"
