# Setting Up Staging Without Local Repo

If you don't have the repo cloned on your local machine, here are your options:

---

## Option 1: Create Script Directly on Server (Easiest!)

Just SSH to the server and paste the entire script:

```bash
# SSH to your server
ssh root@YOUR_SERVER_IP

# Create the setup script (paste this entire block)
cat > setup-server.sh << 'SETUP_SCRIPT_EOF'
#!/bin/bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

if [ "$EUID" -ne 0 ]; then 
    log_error "This script must be run as root"
    exit 1
fi

log_info "=========================================="
log_info "  Server Setup for NimsForest"
log_info "=========================================="

log_info "Updating system packages..."
apt-get update
apt-get upgrade -y

log_info "Installing essential packages..."
apt-get install -y curl wget git build-essential ufw fail2ban \
    unattended-upgrades apt-transport-https ca-certificates \
    software-properties-common

log_info "Configuring automatic security updates..."
dpkg-reconfigure -plow unattended-upgrades

log_info "Installing Go..."
GO_VERSION="1.24.0"
if ! command -v go &> /dev/null; then
    wget -q https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
    rm go${GO_VERSION}.linux-amd64.tar.gz
    
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

log_info "Installing NATS Server..."
if ! command -v nats-server &> /dev/null; then
    curl -sf https://binaries.nats.dev/nats-io/nats-server/v2@latest | sh
    mv nats-server /usr/local/bin/
    chmod +x /usr/local/bin/nats-server
    log_info "NATS Server installed: $(nats-server --version)"
else
    log_info "NATS Server already installed: $(nats-server --version)"
fi

log_info "Setting up NATS user and directories..."
if ! id "nats" &>/dev/null; then
    useradd -r -s /bin/false -d /var/lib/nats nats
fi
mkdir -p /var/lib/nats /var/log/nats
chown -R nats:nats /var/lib/nats /var/log/nats

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
StandardOutput=journal
StandardError=journal
SyslogIdentifier=nats-server
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable nats
systemctl start nats
sleep 3

if systemctl is-active --quiet nats; then
    log_info "✅ NATS Server is running"
else
    log_error "❌ NATS Server failed to start"
    journalctl -u nats -n 20
fi

log_info "Configuring firewall..."
ufw --force reset
ufw default deny incoming
ufw default allow outgoing
ufw allow ssh
ufw allow 22/tcp
ufw --force enable

log_info "Configuring fail2ban..."
systemctl enable fail2ban
systemctl start fail2ban

log_info "Creating deployment directories..."
mkdir -p /opt/nimsforest/backups
chmod 755 /opt/nimsforest

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

log_info "=========================================="
log_info "  ✅ Server Setup Completed!"
log_info "=========================================="
log_info "Next: Configure GitHub secrets for deployment"
SETUP_SCRIPT_EOF

# Make it executable and run
chmod +x setup-server.sh
sudo ./setup-server.sh
```

---

## Option 2: Clone Repo on Your Local Machine

If you want to follow the standard method:

```bash
# On your local machine:
cd ~
git clone https://github.com/YOUR_USERNAME/nimsforest.git
cd nimsforest

# Now you can use the standard method:
scp scripts/setup-server.sh root@YOUR_SERVER_IP:/tmp/
ssh root@YOUR_SERVER_IP "cd /tmp && chmod +x setup-server.sh && sudo ./setup-server.sh"
```

---

## Option 3: Clone Repo on the Server (Temporary)

```bash
# SSH to server
ssh root@YOUR_SERVER_IP

# Clone the repo (you'll need GitHub authentication)
cd /tmp
git clone https://github.com/YOUR_USERNAME/nimsforest.git
cd nimsforest

# Run the setup script
sudo ./scripts/setup-server.sh

# Clean up
cd /tmp
rm -rf nimsforest
```

**For private repos**, you'll need to authenticate:

### Using Personal Access Token:
```bash
git clone https://YOUR_USERNAME:YOUR_TOKEN@github.com/YOUR_USERNAME/nimsforest.git
```

### Using GitHub CLI:
```bash
# Install gh on server
curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null
sudo apt update
sudo apt install gh

# Login and clone
gh auth login
gh repo clone YOUR_USERNAME/nimsforest
cd nimsforest
sudo ./scripts/setup-server.sh
```

---

## Option 4: Use GitHub Raw URL (Public Repos Only)

If your repo becomes public:

```bash
ssh root@YOUR_SERVER_IP
wget https://raw.githubusercontent.com/YOUR_USERNAME/nimsforest/main/scripts/setup-server.sh
chmod +x setup-server.sh
sudo ./setup-server.sh
```

---

## Recommended: Option 1 (Paste Script Directly)

**This is the easiest since you don't have the repo locally:**

### Step-by-Step:

1. **Open this file in your editor:**
   ```bash
   cat scripts/setup-server.sh
   ```

2. **Copy the entire script content**

3. **SSH to your server:**
   ```bash
   ssh root@YOUR_SERVER_IP
   ```

4. **Create the script:**
   ```bash
   cat > setup-server.sh << 'EOF'
   # Paste the script content here (everything from #!/bin/bash to the end)
   EOF
   ```

5. **Run it:**
   ```bash
   chmod +x setup-server.sh
   sudo ./setup-server.sh
   ```

---

## After Server Setup

Once the server is set up (using any method above), configure GitHub secrets:

### Without Local Repo:

You'll need to run the GitHub secrets configuration from somewhere that has:
- GitHub CLI installed
- Can generate SSH keys

**Options:**

### A. Use GitHub Web UI:

```bash
# 1. Generate SSH key on server or any machine
ssh-keygen -t ed25519 -C "github-staging" -f nimsforest_staging_deploy -N ""

# 2. Copy public key to server
ssh-copy-id -i nimsforest_staging_deploy.pub root@YOUR_SERVER_IP

# 3. Get server fingerprint
ssh-keyscan YOUR_SERVER_IP > known_hosts_temp

# 4. Go to GitHub → Settings → Secrets → Actions → New secret
# Add these 4 secrets:
STAGING_SSH_PRIVATE_KEY     → content of nimsforest_staging_deploy
STAGING_SSH_USER            → root
STAGING_SSH_HOST            → YOUR_SERVER_IP
STAGING_SSH_KNOWN_HOSTS     → content of known_hosts_temp
```

### B. Clone Repo Locally (Once):

```bash
# On your local machine:
git clone https://github.com/YOUR_USERNAME/nimsforest.git
cd nimsforest
./scripts/setup-staging-local.sh YOUR_SERVER_IP

# Now you can delete the local repo if you want
cd ..
rm -rf nimsforest
```

---

## Complete Flow Without Local Repo

```bash
# 1. Create Hetzner server (web UI)
#    → https://console.hetzner.cloud/
#    → Copy IP: 1.2.3.4

# 2. SSH to server and create setup script
ssh root@1.2.3.4

cat > setup-server.sh << 'EOF'
#!/bin/bash
# ... paste the entire setup script content here ...
EOF

chmod +x setup-server.sh
sudo ./setup-server.sh
exit

# 3. Configure GitHub secrets (web UI)
#    → Generate SSH key anywhere
#    → Add 4 secrets to GitHub via web interface

# 4. Deploy
#    → git push from wherever your code is
#    → GitHub Actions deploys automatically
```

---

## Summary

**If you don't have the repo locally:**

✅ **Easiest:** SSH to server + paste script content (Option 1)  
✅ **Alternative:** Clone repo temporarily on server (Option 3)  
✅ **Long-term:** Clone repo on your local machine (Option 2)

**For GitHub secrets:**
- Use GitHub Web UI to add secrets manually
- Or clone repo once just to run the setup script
- After setup, you can work from anywhere (just `git push`)

---

## The Full Script Content

Here's the complete `setup-server.sh` content you can paste:

```bash
#!/bin/bash
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

if [ "$EUID" -ne 0 ]; then 
    log_error "This script must be run as root"
    exit 1
fi

log_info "=========================================="
log_info "  Server Setup for NimsForest"
log_info "=========================================="

log_info "Updating system packages..."
apt-get update && apt-get upgrade -y

log_info "Installing essential packages..."
apt-get install -y curl wget git build-essential ufw fail2ban \
    unattended-upgrades apt-transport-https ca-certificates \
    software-properties-common

dpkg-reconfigure -plow unattended-upgrades

log_info "Installing Go 1.24.0..."
if ! command -v go &> /dev/null; then
    wget -q https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz
    rm go1.24.0.linux-amd64.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile.d/go.sh
    source /etc/profile.d/go.sh
fi

log_info "Installing NATS Server..."
if ! command -v nats-server &> /dev/null; then
    curl -sf https://binaries.nats.dev/nats-io/nats-server/v2@latest | sh
    mv nats-server /usr/local/bin/ && chmod +x /usr/local/bin/nats-server
fi

log_info "Setting up NATS..."
id nats &>/dev/null || useradd -r -s /bin/false -d /var/lib/nats nats
mkdir -p /var/lib/nats /var/log/nats
chown -R nats:nats /var/lib/nats /var/log/nats

cat > /etc/systemd/system/nats.service << 'EOF'
[Unit]
Description=NATS Server
After=network.target

[Service]
Type=simple
User=nats
Group=nats
WorkingDirectory=/var/lib/nats
ExecStart=/usr/local/bin/nats-server --jetstream --store_dir=/var/lib/nats --http_port=8222 --port=4222
Restart=on-failure
LimitNOFILE=65536
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable nats
systemctl start nats
sleep 3

log_info "Configuring firewall..."
ufw --force reset
ufw default deny incoming
ufw default allow outgoing
ufw allow ssh
ufw allow 22/tcp
ufw --force enable

systemctl enable fail2ban
systemctl start fail2ban

mkdir -p /opt/nimsforest/backups
chmod 755 /opt/nimsforest

log_info "✅ Server Setup Complete!"
systemctl status nats --no-pager
```

Just copy this entire script and paste it into the `cat > setup-server.sh` command on your server!
