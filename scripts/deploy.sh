#!/bin/bash
#
# NimsForest Deployment Script for Hetzner
#
# This script automates the deployment of NimsForest on a Hetzner server
# It handles service management, backups, and rollback capabilities

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
APP_NAME="nimsforest"
BINARY_NAME="forest"
SERVICE_NAME="nimsforest"
INSTALL_DIR="/usr/local/bin"
DATA_DIR="/var/lib/nimsforest"
LOG_DIR="/var/log/nimsforest"
BACKUP_DIR="/opt/nimsforest/backups"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
SERVICE_USER="forest"

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
    if [ "$EUID" -ne 0 ]; then 
        log_error "This script must be run as root"
        exit 1
    fi
}

# Create system user if it doesn't exist
create_user() {
    if ! id "$SERVICE_USER" &>/dev/null; then
        log_info "Creating system user: $SERVICE_USER"
        useradd -r -s /bin/false -d "$DATA_DIR" "$SERVICE_USER"
    else
        log_info "User $SERVICE_USER already exists"
    fi
}

# Create necessary directories
create_directories() {
    log_info "Creating directories..."
    
    mkdir -p "$DATA_DIR"
    mkdir -p "$LOG_DIR"
    mkdir -p "$BACKUP_DIR"
    
    chown -R "$SERVICE_USER:$SERVICE_USER" "$DATA_DIR"
    chown -R "$SERVICE_USER:$SERVICE_USER" "$LOG_DIR"
    chmod 755 "$DATA_DIR"
    chmod 755 "$LOG_DIR"
}

# Backup current binary if it exists
backup_current() {
    if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
        log_info "Backing up current binary..."
        cp "$INSTALL_DIR/$BINARY_NAME" "$BACKUP_DIR/${BINARY_NAME}.backup.$(date +%Y%m%d_%H%M%S)"
        cp "$INSTALL_DIR/$BINARY_NAME" "$BACKUP_DIR/${BINARY_NAME}.backup"
        log_info "Backup created at $BACKUP_DIR"
    else
        log_warn "No existing binary found to backup"
    fi
}

# Stop the service if running
stop_service() {
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log_info "Stopping $SERVICE_NAME service..."
        systemctl stop "$SERVICE_NAME"
        sleep 2
    else
        log_info "Service $SERVICE_NAME is not running"
    fi
}

# Install the new binary
install_binary() {
    log_info "Installing new binary..."
    
    if [ ! -f "./$BINARY_NAME" ]; then
        log_error "Binary $BINARY_NAME not found in current directory"
        exit 1
    fi
    
    cp "./$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
    chown root:root "$INSTALL_DIR/$BINARY_NAME"
    
    log_info "Binary installed to $INSTALL_DIR/$BINARY_NAME"
}

# Install systemd service
install_service() {
    log_info "Installing systemd service..."
    
    if [ -f "./nimsforest.service" ]; then
        cp "./nimsforest.service" "$SERVICE_FILE"
    else
        # Create default service file if not provided
        log_warn "Service file not found, creating default..."
        cat > "$SERVICE_FILE" << 'EOF'
[Unit]
Description=NimsForest Event Orchestration System
After=network.target nats.service
Wants=nats.service

[Service]
Type=simple
User=forest
Group=forest
WorkingDirectory=/var/lib/nimsforest
Environment="NATS_URL=nats://localhost:4222"
ExecStart=/usr/local/bin/forest
Restart=on-failure
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=nimsforest

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/nimsforest /var/log/nimsforest

[Install]
WantedBy=multi-user.target
EOF
    fi
    
    chmod 644 "$SERVICE_FILE"
    systemctl daemon-reload
    log_info "Service installed"
}

# Enable and start the service
start_service() {
    log_info "Enabling and starting service..."
    
    systemctl enable "$SERVICE_NAME"
    systemctl start "$SERVICE_NAME"
    
    sleep 3
    
    # Verify service is running
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log_info "✅ Service $SERVICE_NAME is running successfully"
    else
        log_error "❌ Service $SERVICE_NAME failed to start"
        log_error "Check logs with: journalctl -u $SERVICE_NAME -n 50"
        exit 1
    fi
}

# Verify installation
verify_installation() {
    log_info "Verifying installation..."
    
    # Check binary exists
    if [ ! -f "$INSTALL_DIR/$BINARY_NAME" ]; then
        log_error "Binary not found at $INSTALL_DIR/$BINARY_NAME"
        return 1
    fi
    
    # Check service is active
    if ! systemctl is-active --quiet "$SERVICE_NAME"; then
        log_error "Service is not active"
        return 1
    fi
    
    # Check if NATS is accessible (if running locally)
    if command -v curl &> /dev/null; then
        if curl -s -f http://localhost:8222/varz &> /dev/null; then
            log_info "NATS server is accessible"
        else
            log_warn "NATS server is not accessible on localhost:8222"
            log_warn "Make sure NATS is installed and running"
        fi
    fi
    
    log_info "✅ Installation verified successfully"
}

# Rollback to previous version
rollback() {
    log_warn "Rolling back to previous version..."
    
    if [ -f "$BACKUP_DIR/${BINARY_NAME}.backup" ]; then
        systemctl stop "$SERVICE_NAME"
        cp "$BACKUP_DIR/${BINARY_NAME}.backup" "$INSTALL_DIR/$BINARY_NAME"
        chmod +x "$INSTALL_DIR/$BINARY_NAME"
        systemctl start "$SERVICE_NAME"
        log_info "✅ Rollback completed"
    else
        log_error "No backup found, cannot rollback"
        exit 1
    fi
}

# Main deployment function
deploy() {
    log_info "=========================================="
    log_info "  NimsForest Deployment Starting"
    log_info "=========================================="
    
    check_root
    create_user
    create_directories
    backup_current
    stop_service
    install_binary
    install_service
    start_service
    verify_installation
    
    log_info "=========================================="
    log_info "  ✅ Deployment Completed Successfully!"
    log_info "=========================================="
    log_info ""
    log_info "Service Status:"
    systemctl status "$SERVICE_NAME" --no-pager || true
    log_info ""
    log_info "Useful commands:"
    log_info "  - Check status:  sudo systemctl status $SERVICE_NAME"
    log_info "  - View logs:     sudo journalctl -u $SERVICE_NAME -f"
    log_info "  - Restart:       sudo systemctl restart $SERVICE_NAME"
    log_info "  - Stop:          sudo systemctl stop $SERVICE_NAME"
}

# Handle command line arguments
case "${1:-deploy}" in
    deploy)
        deploy
        ;;
    rollback)
        check_root
        rollback
        ;;
    verify)
        verify_installation
        ;;
    *)
        echo "Usage: $0 {deploy|rollback|verify}"
        exit 1
        ;;
esac
