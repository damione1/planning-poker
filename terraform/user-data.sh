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

# Install Docker Compose Plugin (v2) system-wide
mkdir -p /usr/local/lib/docker/cli-plugins
curl -SL https://github.com/docker/compose/releases/latest/download/docker-compose-linux-aarch64 -o /usr/local/lib/docker/cli-plugins/docker-compose
chmod +x /usr/local/lib/docker/cli-plugins/docker-compose

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

# Note: Application code will be deployed by CodeDeploy
# user-data.sh only sets up infrastructure (Docker, CodeDeploy agent, directories)

# Store environment variables for CodeDeploy scripts
cat > /etc/environment << ENV
DOMAIN_NAME=$DOMAIN
LETS_ENCRYPT_EMAIL=$EMAIL
ENV

echo "==== Infrastructure setup complete! ===="
echo "Application URL: https://$DOMAIN (will be available after first CodeDeploy deployment)"
echo "CodeDeploy agent status: $(systemctl is-active codedeploy-agent)"
echo "Waiting for CodeDeploy to deploy application..."
