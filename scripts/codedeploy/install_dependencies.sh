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

# Create docker-compose.prod.yml if it doesn't exist (from user-data.sh template)
if [ ! -f docker-compose.prod.yml ]; then
    echo "Creating docker-compose.prod.yml..."
    cat > docker-compose.prod.yml <<'COMPOSE'
services:
  traefik:
    image: traefik:v3.3
    container_name: traefik
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - /mnt/data/traefik/acme:/acme
    command:
      - "--api.dashboard=false"
      - "--providers.docker=true"
      - "--providers.docker.exposedbydefault=false"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
      - "--entrypoints.web.http.redirections.entryPoint.to=websecure"
      - "--entrypoints.web.http.redirections.entryPoint.scheme=https"
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge=true"
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=web"
      - "--certificatesresolvers.letsencrypt.acme.email=${LETS_ENCRYPT_EMAIL}"
      - "--certificatesresolvers.letsencrypt.acme.storage=/acme/acme.json"

  app:
    build: .
    container_name: planning-poker
    restart: unless-stopped
    volumes:
      - /mnt/data/pb_data:/app/pb_data
    environment:
      - PP_ENV=production
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.app.rule=Host(\`${DOMAIN_NAME}\`)"
      - "traefik.http.routers.app.entrypoints=websecure"
      - "traefik.http.routers.app.tls=true"
      - "traefik.http.routers.app.tls.certresolver=letsencrypt"
      - "traefik.http.services.app.loadbalancer.server.port=8090"
COMPOSE
fi

# Ensure data directories exist
mkdir -p /mnt/data/pb_data
mkdir -p /mnt/data/traefik/acme

# Set proper ownership
chown -R ec2-user:ec2-user /opt/planning-poker
chown -R ec2-user:ec2-user /mnt/data

echo "=== Dependency installation complete ==="
