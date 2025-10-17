#!/bin/bash
set -e

# Variables from Terraform
DOMAIN="${domain}"
EMAIL="${email}"
DATA_VOLUME_ID="${data_volume_id}"
AWS_REGION="${aws_region}"
SERVICE_NAME="${service_name}"

# Log everything
exec > >(tee -a /var/log/user-data.log)
exec 2>&1

echo "==== Starting Planning Poker setup ===="

# Update system
dnf update -y

# Install Docker
dnf install -y docker

# Start and enable Docker
systemctl start docker
systemctl enable docker

# Add ec2-user to docker group
usermod -aG docker ec2-user

# Install Docker Compose Plugin (v2) system-wide
mkdir -p /usr/local/lib/docker/cli-plugins
curl -SL https://github.com/docker/compose/releases/latest/download/docker-compose-linux-aarch64 -o /usr/local/lib/docker/cli-plugins/docker-compose
chmod +x /usr/local/lib/docker/cli-plugins/docker-compose

# Verify SSM agent is running (pre-installed on Amazon Linux 2023)
systemctl enable amazon-ssm-agent
systemctl start amazon-ssm-agent

# Wait for EBS volume to attach
echo "Waiting for EBS volume $DATA_VOLUME_ID to attach..."
while [ ! -e /dev/xvdf ]; do
    sleep 1
done

# Format and mount EBS volume if not already formatted
if ! blkid /dev/xvdf; then
    echo "Formatting EBS volume..."
    mkfs -t ext4 /dev/xvdf
fi

# Create mount point and mount
mkdir -p /mnt/data
mount /dev/xvdf /mnt/data

# Add to fstab for automatic mounting on reboot
UUID=$(blkid -s UUID -o value /dev/xvdf)
echo "UUID=$UUID /mnt/data ext4 defaults,nofail 0 2" >> /etc/fstab

# Create directory structure
mkdir -p /opt/planning-poker/scripts
mkdir -p /mnt/data/pb_data
mkdir -p /mnt/data/traefik/acme
chown -R ec2-user:ec2-user /opt/planning-poker
chown -R ec2-user:ec2-user /mnt/data

# Create deployment script
cat > /opt/planning-poker/scripts/deploy.sh << 'EOF'
#!/bin/bash
set -e

VERSION="${1:-latest}"
IMAGE_TAG="${2:-latest}"

echo "=== Deploying Planning Poker version: $VERSION ==="

cd /opt/planning-poker

# Load environment variables
if [ -f /etc/environment ]; then
    set -a
    source /etc/environment
    set +a
fi

# Export required variables
export DOMAIN_NAME
export LETS_ENCRYPT_EMAIL

# Verify environment variables
if [ -z "$DOMAIN_NAME" ] || [ -z "$LETS_ENCRYPT_EMAIL" ]; then
    echo "❌ Required environment variables not set!"
    exit 1
fi

# Pull latest image from GHCR
echo "Pulling image: ghcr.io/damione1/planning-poker:$IMAGE_TAG"
docker pull "ghcr.io/damione1/planning-poker:$IMAGE_TAG"

# Stop existing containers
echo "Stopping existing containers..."
docker compose -f docker-compose.prod.yml down || true

# Start services with new image
echo "Starting services..."
docker compose -f docker-compose.prod.yml up -d

# Wait for health check
echo "Waiting for services to start..."
sleep 15

# Verify containers are running
if docker compose -f docker-compose.prod.yml ps | grep -q "Up"; then
    echo "✅ Deployment successful!"
    docker compose -f docker-compose.prod.yml ps
    exit 0
else
    echo "❌ Deployment failed - containers not running"
    docker compose -f docker-compose.prod.yml logs
    exit 1
fi
EOF

chmod +x /opt/planning-poker/scripts/deploy.sh

# Create docker-compose.prod.yml
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
      - "traefik.http.routers.app.rule=Host(\`${DOMAIN}\`)"
      - "traefik.http.routers.app.entrypoints=websecure"
      - "traefik.http.routers.app.tls=true"
      - "traefik.http.routers.app.tls.certresolver=letsencrypt"
      - "traefik.http.services.app.loadbalancer.server.port=8090"
    networks:
      - app

networks:
  app:
    driver: bridge
EOF

# Store environment variables
cat > /etc/environment << ENV
DOMAIN_NAME=${domain}
LETS_ENCRYPT_EMAIL=${email}
ENV

echo "==== Infrastructure setup complete! ===="
echo "Application URL: https://${domain}"
echo "SSM agent status: $$(systemctl is-active amazon-ssm-agent)"
echo "Ready for SSM-triggered deployments"
