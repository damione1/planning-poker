#!/bin/bash
set -euo pipefail

# Installation script for Planning Poker
# Installs binary, creates service, and starts application

if [ "$EUID" -ne 0 ]; then
    echo "ERROR: This script must be run as root (use sudo)"
    exit 1
fi

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="/opt/planning-poker"
DATA_DIR="/var/lib/planning-poker"
SERVICE_NAME="planning-poker"
USER="planning-poker"

log_info "Planning Poker Installation"
log_info "==========================="
echo ""

# Check if binary exists
if [ ! -f "$SCRIPT_DIR/planning-poker" ]; then
    log_error "Binary 'planning-poker' not found in $SCRIPT_DIR"
    exit 1
fi

# Check if service is running
if systemctl is-active --quiet $SERVICE_NAME; then
    log_warn "Service is currently running"
    read -p "Stop service and continue installation? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "Stopping service..."
        systemctl stop $SERVICE_NAME
    else
        log_error "Installation cancelled"
        exit 1
    fi
fi

# Backup existing installation if it exists
if [ -d "$INSTALL_DIR" ]; then
    BACKUP_DIR="${INSTALL_DIR}.backup.$(date +%Y%m%d_%H%M%S)"
    log_warn "Existing installation found"
    log_info "Creating backup at $BACKUP_DIR"
    cp -r "$INSTALL_DIR" "$BACKUP_DIR"
fi

# Backup database if it exists
if [ -f "$DATA_DIR/pb_data/data.db" ]; then
    BACKUP_FILE="$DATA_DIR/pb_data/data.db.backup.$(date +%Y%m%d_%H%M%S)"
    log_info "Backing up database to $BACKUP_FILE"
    cp "$DATA_DIR/pb_data/data.db" "$BACKUP_FILE"
fi

# Create user if doesn't exist
if ! id "$USER" &>/dev/null; then
    log_info "Creating user: $USER"
    useradd -r -s /bin/false -d "$DATA_DIR" -c "Planning Poker Service User" "$USER"
else
    log_info "User $USER already exists"
fi

# Create directories
log_info "Creating directories..."
mkdir -p "$INSTALL_DIR"
mkdir -p "$DATA_DIR/pb_data"

# Copy binary
log_info "Installing binary..."
cp "$SCRIPT_DIR/planning-poker" "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/planning-poker"

# Copy web assets if they exist
if [ -d "$SCRIPT_DIR/web" ]; then
    log_info "Installing web assets..."
    cp -r "$SCRIPT_DIR/web" "$INSTALL_DIR/"
fi

# Set ownership
log_info "Setting permissions..."
chown -R "$USER":"$USER" "$INSTALL_DIR"
chown -R "$USER":"$USER" "$DATA_DIR"

# Install systemd service
log_info "Installing systemd service..."
if [ -f "$SCRIPT_DIR/planning-poker.service" ]; then
    cp "$SCRIPT_DIR/planning-poker.service" "/etc/systemd/system/"
    systemctl daemon-reload
else
    log_warn "Service file not found in package, service not installed"
fi

# Enable and start service
log_info "Enabling service..."
systemctl enable $SERVICE_NAME

log_info "Starting service..."
systemctl start $SERVICE_NAME

# Wait for service to start
sleep 2

# Check service status
if systemctl is-active --quiet $SERVICE_NAME; then
    log_info "Service started successfully!"
else
    log_error "Service failed to start"
    log_info "Check logs with: journalctl -u $SERVICE_NAME -n 50"
    exit 1
fi

# Display status
echo ""
log_info "Installation complete!"
echo ""
echo "Service Status:"
systemctl status $SERVICE_NAME --no-pager -l
echo ""
log_info "Useful commands:"
echo "  View logs:        journalctl -u $SERVICE_NAME -f"
echo "  Restart service:  systemctl restart $SERVICE_NAME"
echo "  Stop service:     systemctl stop $SERVICE_NAME"
echo "  Check status:     systemctl status $SERVICE_NAME"
echo ""
log_info "Application is running at: http://localhost:8090"
log_info "Database location: $DATA_DIR/pb_data/data.db"
