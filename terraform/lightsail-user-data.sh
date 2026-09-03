#!/bin/bash
set -e

DOMAIN="pokerplanning.net"
EMAIL="goehrig.damien@protonmail.com"

exec > >(tee -a /var/log/user-data.log)
exec 2>&1

echo "==== Starting Planning Poker setup (Lightsail) ===="

dnf update -y
dnf install -y docker

systemctl start docker
systemctl enable docker
usermod -aG docker ec2-user

# Docker Compose plugin matching the host architecture
ARCH=$(uname -m)
mkdir -p /usr/local/lib/docker/cli-plugins
curl -SL "https://github.com/docker/compose/releases/latest/download/docker-compose-linux-$ARCH" \
  -o /usr/local/lib/docker/cli-plugins/docker-compose
chmod +x /usr/local/lib/docker/cli-plugins/docker-compose

# Swap keeps image pulls from pressuring the 1 GB bundle
if [ ! -f /swapfile ]; then
    dd if=/dev/zero of=/swapfile bs=1M count=1024
    chmod 600 /swapfile
    mkswap /swapfile
    swapon /swapfile
    echo "/swapfile none swap sw 0 0" >> /etc/fstab
fi

# Data lives on the bundle SSD, path kept identical to the EC2 layout
mkdir -p /opt/planning-poker/scripts
mkdir -p /mnt/data/pb_data
mkdir -p /mnt/data/traefik/acme
chown -R ec2-user:ec2-user /opt/planning-poker
chown -R ec2-user:ec2-user /mnt/data

cat > /opt/planning-poker/scripts/deploy.sh << 'EOF'
#!/bin/bash
# Runs on the Lightsail instance. Pulls a GHCR tag and recreates only the app
# container so Traefik/certs stay up.
set -euo pipefail

VERSION="${1:-latest}"
IMAGE_TAG="${2:-$VERSION}"
IMAGE="ghcr.io/damione1/planning-poker"
COMPOSE="/opt/planning-poker/docker-compose.prod.yml"

echo "=== Deploying Planning Poker $VERSION (image tag: $IMAGE_TAG) ==="

if [ -f /etc/environment ]; then
    set -a
    # shellcheck disable=SC1091
    source /etc/environment
    set +a
fi

if [ -z "${DOMAIN_NAME:-}" ] || [ -z "${LETS_ENCRYPT_EMAIL:-}" ]; then
    echo "Required environment variables not set (DOMAIN_NAME / LETS_ENCRYPT_EMAIL)"
    exit 1
fi

cd /opt/planning-poker

echo "Pulling ${IMAGE}:${IMAGE_TAG}"
docker pull "${IMAGE}:${IMAGE_TAG}"

if [ "$IMAGE_TAG" != "latest" ]; then
    docker tag "${IMAGE}:${IMAGE_TAG}" "${IMAGE}:latest"
fi

echo "Recreating app container (Traefik left running)"
docker compose -f "$COMPOSE" up -d --no-deps --force-recreate app

echo "Waiting for container to become healthy..."
ok=0
for i in $(seq 1 36); do
    status=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' planning-poker 2>/dev/null || echo "missing")
    echo "  health=${status} (${i})"
    if [ "$status" = "healthy" ]; then
        ok=1
        break
    fi
    if [ "$status" = "missing" ] || [ "$status" = "exited" ] || [ "$status" = "dead" ]; then
        docker compose -f "$COMPOSE" logs --tail 80 app
        exit 1
    fi
    sleep 5
done

if [ "$ok" != 1 ]; then
    echo "Container did not become healthy"
    docker compose -f "$COMPOSE" logs --tail 80 app
    exit 1
fi

echo "Deployment successful"
docker compose -f "$COMPOSE" ps
EOF

chmod +x /opt/planning-poker/scripts/deploy.sh
chown ec2-user:ec2-user /opt/planning-poker/scripts/deploy.sh

cat > /opt/planning-poker/docker-compose.prod.yml << EOF
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
      - "--certificatesresolvers.letsencrypt.acme.email=${EMAIL}"
      - "--certificatesresolvers.letsencrypt.acme.storage=/acme/acme.json"
    networks:
      - app

  app:
    image: ghcr.io/damione1/planning-poker:latest
    container_name: planning-poker
    restart: unless-stopped
    pull_policy: always
    volumes:
      - /mnt/data/pb_data:/app/pb_data
    environment:
      - PP_ENV=production
    labels:
      - "traefik.enable=true"
      - "traefik.http.services.app.loadbalancer.server.port=8090"
      - "traefik.http.routers.app.rule=Host(\`${DOMAIN}\`)"
      - "traefik.http.routers.app.entrypoints=websecure"
      - "traefik.http.routers.app.tls=true"
      - "traefik.http.routers.app.tls.certresolver=letsencrypt"
      - "traefik.http.routers.app.service=app"
      - "traefik.http.routers.app.priority=100"
      # Pre-cutover validation over the raw IP, served with Traefik's default cert
      - "traefik.http.routers.app-staging.rule=PathPrefix(\`/\`)"
      - "traefik.http.routers.app-staging.entrypoints=websecure"
      - "traefik.http.routers.app-staging.tls=true"
      - "traefik.http.routers.app-staging.service=app"
      - "traefik.http.routers.app-staging.priority=1"
    networks:
      - app

networks:
  app:
    driver: bridge
EOF

chown ec2-user:ec2-user /opt/planning-poker/docker-compose.prod.yml

cat > /etc/environment << ENV
DOMAIN_NAME=${DOMAIN}
LETS_ENCRYPT_EMAIL=${EMAIL}
ENV

cd /opt/planning-poker
docker compose -f docker-compose.prod.yml up -d

echo "==== Infrastructure setup complete! ===="
