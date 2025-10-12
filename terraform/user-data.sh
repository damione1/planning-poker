#!/bin/bash
set -eu

# User data script for AWS Lightsail instance initialization
# Runs once on first boot to prepare the instance

exec > >(tee /var/log/user-data.log)
exec 2>&1

echo "==================================="
echo "Planning Poker Instance Setup"
echo "==================================="
echo "Started at: $(date)"

# Update system packages
echo "Updating system packages..."
export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get upgrade -y -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold"

# Install required packages
echo "Installing required packages..."
apt-get install -y \
    curl \
    wget \
    ca-certificates \
    tzdata \
    unattended-upgrades \
    fail2ban

# Configure timezone
echo "Configuring timezone..."
timedatectl set-timezone UTC

# Enable automatic security updates
echo "Enabling automatic security updates..."
cat > /etc/apt/apt.conf.d/50unattended-upgrades << 'EOF'
Unattended-Upgrade::Allowed-Origins {
    "$${distro_id}:$${distro_codename}-security";
};
Unattended-Upgrade::AutoFixInterruptedDpkg "true";
Unattended-Upgrade::MinimalSteps "true";
Unattended-Upgrade::Remove-Unused-Kernel-Packages "true";
Unattended-Upgrade::Remove-Unused-Dependencies "true";
Unattended-Upgrade::Automatic-Reboot "false";
EOF

cat > /etc/apt/apt.conf.d/20auto-upgrades << 'EOF'
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Download-Upgradeable-Packages "1";
APT::Periodic::AutocleanInterval "7";
APT::Periodic::Unattended-Upgrade "1";
EOF

# Configure fail2ban for SSH protection
echo "Configuring fail2ban..."
systemctl enable fail2ban
systemctl start fail2ban

# Create application directories
echo "Creating application directories..."
mkdir -p /opt/planning-poker
mkdir -p /var/lib/planning-poker/pb_data

# Create application user
echo "Creating application user..."
if ! id planning-poker &>/dev/null; then
    useradd -r -s /bin/false -d /var/lib/planning-poker -c "Planning Poker Service User" planning-poker
fi

# Set ownership
chown -R planning-poker:planning-poker /var/lib/planning-poker

# Create environment file for systemd service overrides
echo "Creating environment configuration..."
mkdir -p /etc/systemd/system/planning-poker.service.d
cat > /etc/systemd/system/planning-poker.service.d/override.conf << 'EOF'
[Service]
Environment="PP_ENV=production"
Environment="PP_WS_ALLOWED_ORIGINS=${ws_allowed_origins}"
Environment="PP_LOG_LEVEL=info"
EOF

# Configure firewall (UFW)
echo "Configuring firewall..."
apt-get install -y ufw

# Allow SSH
ufw allow 22/tcp comment 'SSH'

# Allow application port
ufw allow ${app_port}/tcp comment 'Planning Poker HTTP'

# Allow HTTP/HTTPS for future reverse proxy
ufw allow 80/tcp comment 'HTTP'
ufw allow 443/tcp comment 'HTTPS'

# Enable firewall (non-interactive)
echo "y" | ufw enable

# Setup log rotation for application
echo "Configuring log rotation..."
cat > /etc/logrotate.d/planning-poker << 'EOF'
/var/log/planning-poker/*.log {
    daily
    rotate 7
    compress
    delaycompress
    notifempty
    create 0640 planning-poker planning-poker
    sharedscripts
    postrotate
        systemctl reload planning-poker > /dev/null 2>&1 || true
    endscript
}
EOF

# Create motd with deployment instructions
echo "Creating MOTD..."
cat > /etc/update-motd.d/99-planning-poker << 'EOF'
#!/bin/sh
cat << 'MOTD'

╔══════════════════════════════════════════════════════════════╗
║                  Planning Poker Server                        ║
╚══════════════════════════════════════════════════════════════╝

📦 Deployment Instructions:
  1. Upload package: scp planning-poker-v*.tar.gz ubuntu@<server-ip>:/tmp/
  2. Extract: tar -xzf /tmp/planning-poker-v*.tar.gz
  3. Install: sudo ./install.sh

🔧 Service Management:
  Status:  sudo systemctl status planning-poker
  Logs:    sudo journalctl -u planning-poker -f
  Restart: sudo systemctl restart planning-poker

📊 Monitoring:
  Application: http://localhost:${app_port}
  Database:    /var/lib/planning-poker/pb_data/data.db

MOTD
EOF
chmod +x /etc/update-motd.d/99-planning-poker

# Create health check script
echo "Creating health check script..."
cat > /usr/local/bin/planning-poker-health << 'EOF'
#!/bin/bash
curl -sf http://localhost:${app_port} > /dev/null
exit $?
EOF
chmod +x /usr/local/bin/planning-poker-health

# Setup cron job for health monitoring (optional)
# echo "*/5 * * * * root /usr/local/bin/planning-poker-health || echo 'Planning Poker health check failed' | logger -t planning-poker" >> /etc/crontab

# Final system configuration
echo "Finalizing system configuration..."

# Disable unnecessary services
systemctl disable snapd.service || true
systemctl stop snapd.service || true

# Optimize system limits for web application
cat >> /etc/security/limits.conf << 'EOF'
planning-poker soft nofile 65536
planning-poker hard nofile 65536
EOF

# Update sysctl for better network performance
cat >> /etc/sysctl.conf << 'EOF'
# Planning Poker optimizations
net.core.somaxconn = 1024
net.ipv4.tcp_max_syn_backlog = 2048
net.ipv4.ip_local_port_range = 1024 65535
EOF
sysctl -p

# Create deployment marker
touch /var/lib/planning-poker/.instance-initialized
date > /var/lib/planning-poker/.instance-initialized

echo "==================================="
echo "Instance setup complete!"
echo "Completed at: $(date)"
echo "==================================="
echo ""
echo "Instance is ready for application deployment"
echo "Deploy with: ./deploy/deploy.sh <package> <server-ip>"
