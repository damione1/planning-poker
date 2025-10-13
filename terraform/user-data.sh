#!/bin/bash
set -e

# Variables from Terraform
DOMAIN="${domain}"
EMAIL="${email}"
GITHUB_REPO="${github_repo}"
GITHUB_REF="${github_ref}"
DATA_VOLUME_ID="${data_volume_id}"
AWS_REGION="${aws_region}"

# Log everything
exec > >(tee -a /var/log/user-data.log)
exec 2>&1

echo "==== Starting Planning Poker setup ===="

# Update system
apt-get update
apt-get upgrade -y

# Install dependencies
apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release \
    unzip \
    git

# Install AWS CLI v2 for ARM64
curl "https://awscli.amazonaws.com/awscli-exe-linux-aarch64.zip" -o "awscliv2.zip"
unzip -q awscliv2.zip
./aws/install
rm -rf aws awscliv2.zip

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh
rm get-docker.sh

# Install Docker Compose Plugin (v2)
apt-get update
apt-get install -y docker-compose-plugin

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
mkdir -p /opt/planning-poker
mkdir -p /mnt/data/pb_data
mkdir -p /mnt/data/traefik/acme

# Clone repository
cd /opt/planning-poker
git clone https://github.com/$GITHUB_REPO.git .
git checkout $GITHUB_REF

# Create docker-compose.yml
cat > docker-compose.yml <<'EOF'
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
      - "traefik.http.routers.app.rule=Host(\`${DOMAIN}\`)"
      - "traefik.http.routers.app.entrypoints=websecure"
      - "traefik.http.routers.app.tls=true"
      - "traefik.http.routers.app.tls.certresolver=letsencrypt"
      - "traefik.http.services.app.loadbalancer.server.port=8090"
EOF

# Replace domain and email placeholders
sed -i "s/\${DOMAIN}/$DOMAIN/g" docker-compose.yml
sed -i "s/\${EMAIL}/$EMAIL/g" docker-compose.yml

# Build and start containers
docker compose build --no-cache
docker compose up -d

echo "==== Setup complete! ===="
echo "Application URL: https://$DOMAIN"
echo "Logs: docker compose -f /opt/planning-poker/docker-compose.yml logs -f"
