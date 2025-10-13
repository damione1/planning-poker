#!/bin/bash
set -e

# Variables from Terraform
DOMAIN="${domain}"
EMAIL="${email}"
GITHUB_REPO="${github_repo}"
GITHUB_REF="${github_ref}"
DATA_VOLUME_ID="${data_volume_id}"
AWS_REGION="${aws_region}"
SERVICE_NAME="${service_name}"

# Log everything
exec > >(tee -a /var/log/user-data.log)
exec 2>&1

echo "==== Starting Planning Poker setup ===="

# Update system
dnf update -y

# Install dependencies
dnf install -y \
    git \
    docker \
    ruby \
    wget

# Start and enable Docker
systemctl start docker
systemctl enable docker

# Add ec2-user to docker group
usermod -aG docker ec2-user

# Install Docker Compose Plugin (v2)
DOCKER_CONFIG=$${DOCKER_CONFIG:-$HOME/.docker/cli-plugins}
mkdir -p $DOCKER_CONFIG
curl -SL https://github.com/docker/compose/releases/latest/download/docker-compose-linux-aarch64 -o $DOCKER_CONFIG/docker-compose
chmod +x $DOCKER_CONFIG/docker-compose

# Install CodeDeploy Agent
cd /home/ec2-user
wget https://aws-codedeploy-$${AWS_REGION}.s3.$${AWS_REGION}.amazonaws.com/latest/install
chmod +x ./install
./install auto
systemctl start codedeploy-agent
systemctl enable codedeploy-agent

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
chown -R ec2-user:ec2-user /opt/planning-poker
chown -R ec2-user:ec2-user /mnt/data

# Clone repository (initial setup only - CodeDeploy will handle updates)
cd /opt/planning-poker
sudo -u ec2-user git clone https://github.com/$GITHUB_REPO.git .
sudo -u ec2-user git checkout $GITHUB_REF

# Create docker-compose.prod.yml
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
      - "--certificatesresolvers.letsencrypt.acme.email=$${EMAIL}"
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
      - "traefik.http.routers.app.rule=Host(\`$${DOMAIN}\`)"
      - "traefik.http.routers.app.entrypoints=websecure"
      - "traefik.http.routers.app.tls=true"
      - "traefik.http.routers.app.tls.certresolver=letsencrypt"
      - "traefik.http.services.app.loadbalancer.server.port=8090"
COMPOSE

# Replace domain and email placeholders
sed -i "s/\$${DOMAIN}/$DOMAIN/g" docker-compose.prod.yml
sed -i "s/\$${EMAIL}/$EMAIL/g" docker-compose.prod.yml

# Build and start containers (initial setup)
sudo -u ec2-user docker compose -f docker-compose.prod.yml build --no-cache
sudo -u ec2-user docker compose -f docker-compose.prod.yml up -d

echo "==== Setup complete! ===="
echo "Application URL: https://$DOMAIN"
echo "CodeDeploy agent status: $(systemctl is-active codedeploy-agent)"
