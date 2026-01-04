# Using Pastebin for Private Repo

Since your repo is private, use pastebin to share the setup script.

## Step 1: Copy the Script

Copy the entire content of `scripts/setup-server.sh` (shown below).

## Step 2: Create Pastebin

Go to one of these services:
- **pastebin.com** - Public, simple
- **gist.github.com** - GitHub gists (can be private!)
- **paste.ubuntu.com** - Ubuntu pastebin
- **dpaste.com** - Simple pastebin

Paste the script and get the URL.

## Step 3: Use on Server

```bash
# SSH to your server
ssh root@YOUR_SERVER_IP

# Download from pastebin
wget https://pastebin.com/raw/YOUR_PASTE_ID -O setup-server.sh

# OR for gist:
wget https://gist.githubusercontent.com/USERNAME/GIST_ID/raw/setup-server.sh -O setup-server.sh

# Make executable and run
chmod +x setup-server.sh
sudo ./setup-server.sh
```

## Example with pastebin.com

1. Go to https://pastebin.com/
2. Paste the script
3. Set "Paste Expiration": Never (or as needed)
4. Set "Paste Exposure": Unlisted (so it's not public but accessible via URL)
5. Click "Create New Paste"
6. Get the RAW URL (click "raw" button)
7. Use: `wget https://pastebin.com/raw/XXXXX -O setup-server.sh`

## Example with GitHub Gist

1. Go to https://gist.github.com/
2. Paste the script
3. Set filename: `setup-server.sh`
4. Choose "Create secret gist" (or public)
5. Click "Create gist"
6. Click "Raw" button to get the raw URL
7. Use: `wget https://gist.githubusercontent.com/USERNAME/ID/raw/setup-server.sh`

## Alternative: Copy/Paste Directly

If you prefer not to use pastebin:

```bash
# SSH to server
ssh root@YOUR_SERVER_IP

# Create the script
cat > setup-server.sh << 'EOF'
# Paste the entire script here
EOF

chmod +x setup-server.sh
sudo ./setup-server.sh
```

---

## The Complete Script to Copy:

```bash
#!/bin/bash
#
# Initial Server Setup for NimsForest
#
# This script prepares any fresh Linux server for NimsForest deployment
# Works with any Ubuntu/Debian server from any provider
# Run this once when setting up a new server

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    log_error "This script must be run as root"
    exit 1
fi

log_info "=========================================="
log_info "  Server Setup for NimsForest"
log_info "=========================================="

# Update system
log_info "Updating system packages..."
apt-get update
apt-get upgrade -y

# Install essential packages
log_info "Installing essential packages..."
apt-get install -y \
    curl \
    wget \
    git \
    build-essential \
    ufw \
    fail2ban \
    unattended-upgrades \
    apt-transport-https \
    ca-certificates \
    software-properties-common

# Configure automatic security updates
log_info "Configuring automatic security updates..."
dpkg-reconfigure -plow unattended-upgrades

# Install Go (latest version)
log_info "Installing Go..."
GO_VERSION="1.24.0"
if ! command -v go &> /dev/null; then
    wget -q https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
    rm go${GO_VERSION}.linux-amd64.tar.gz

    # Add Go to PATH for all users
    cat >> /etc/profile.d/go.sh << 'EOF'
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
EOF

    source /etc/profile.d/go.sh
    log_info "Go installed: $(go version)"
else
    log_info "Go already installed: $(go version)"
fi

# Install NATS Server
log_info "Installing NATS Server..."
if ! command -v nats-server &> /dev/null; then
    curl -sf https://binaries.nats.dev/nats-io/nats-server/v2@latest | sh
    mv nats-server /usr/local/bin/
    chmod +x /usr/local/bin/nats-server
    log_info "NATS Server installed: $(nats-server --version)"
else
    log_info "NATS Server already installed: $(nats-server --version)"
fi

# Create NATS user and directories
log_info "Setting up NATS user and directories..."
if ! id "nats" &>/dev/null; then
    useradd -r -s /bin/false -d /var/lib/nats nats
fi
mkdir -p /var/lib/nats
mkdir -p /var/log/nats
chown -R nats:nats /var/lib/nats
chown -R nats:nats /var/log/nats

# Install NATS systemd service
log_info "Installing NATS systemd service..."
cat > /etc/systemd/system/nats.service << 'EOF'
[Unit]
Description=NATS Server
After=network.target
Documentation=https://docs.nats.io

[Service]
Type=simple
User=nats
Group=nats
WorkingDirectory=/var/lib/nats
ExecStart=/usr/local/bin/nats-server \
    --jetstream \
    --store_dir=/var/lib/nats \
    --http_port=8222 \
    --port=4222 \
    --max_payload=8MB \
    --max_connections=10000
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=nats-server

# Security
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable nats
systemctl start nats

# Wait for NATS to start
sleep 3

# Verify NATS is running
if systemctl is-active --quiet nats; then
    log_info "✅ NATS Server is running"
    if command -v curl &> /dev/null; then
        curl -s http://localhost:8222/varz | head -n 5 || true
    fi
else
    log_error "❌ NATS Server failed to start"
    journalctl -u nats -n 20
fi

# Configure firewall
log_info "Configuring firewall..."
ufw --force reset
ufw default deny incoming
ufw default allow outgoing
ufw allow ssh
ufw allow 22/tcp
# Only allow NATS ports from localhost (application will connect locally)
# If you need external access to NATS, uncomment:
# ufw allow 4222/tcp  # NATS client port
# ufw allow 8222/tcp  # NATS monitoring port
ufw --force enable
log_info "Firewall configured"

# Configure fail2ban for SSH protection
log_info "Configuring fail2ban..."
systemctl enable fail2ban
systemctl start fail2ban

# Create deployment directory
log_info "Creating deployment directories..."
mkdir -p /opt/nimsforest/backups
chmod 755 /opt/nimsforest

# Set up log rotation
log_info "Setting up log rotation..."
cat > /etc/logrotate.d/nimsforest << 'EOF'
/var/log/nimsforest/*.log {
    daily
    rotate 14
    compress
    delaycompress
    notifempty
    missingok
    create 0644 forest forest
    sharedscripts
    postrotate
        systemctl reload nimsforest >/dev/null 2>&1 || true
    endscript
}
EOF

# Display system information
log_info "=========================================="
log_info "  ✅ Server Setup Completed!"
log_info "=========================================="
log_info ""
log_info "System Information:"
log_info "  - Hostname:    $(hostname)"
log_info "  - OS:          $(lsb_release -ds 2>/dev/null || cat /etc/os-release | grep PRETTY_NAME | cut -d'"' -f2)"
log_info "  - Kernel:      $(uname -r)"
log_info "  - Go:          $(go version | cut -d' ' -f3)"
log_info "  - NATS:        $(nats-server --version 2>&1 | head -n1)"
log_info ""
log_info "Services Status:"
systemctl status nats --no-pager || true
log_info ""
log_info "Next Steps:"
log_info "  1. Configure GitHub Actions secrets for deployment"
log_info "  2. Create a release or trigger manual deployment"
log_info "  3. Monitor deployment via GitHub Actions"
log_info ""
log_info "Useful commands:"
log_info "  - NATS status:     sudo systemctl status nats"
log_info "  - NATS logs:       sudo journalctl -u nats -f"
log_info "  - NATS monitoring: curl http://localhost:8222/varz"
log_info "  - Firewall status: sudo ufw status"
log_info ""
```

Copy everything above and paste it into pastebin!
