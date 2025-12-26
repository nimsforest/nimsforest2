#!/bin/bash
#
# Local Setup Script for Staging Environment
#
# This script helps you configure your local machine to deploy to staging.
# It will:
#   1. Generate SSH keys for deployment
#   2. Copy the public key to your server
#   3. Get the server's SSH fingerprint
#   4. Configure GitHub secrets
#
# Usage: ./scripts/setup-staging-local.sh [SERVER_IP]
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[✓]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[!]${NC} $1"; }
log_error() { echo -e "${RED}[✗]${NC} $1"; }
log_step() { echo -e "${BLUE}[→]${NC} $1"; }

# Banner
echo ""
echo "╔════════════════════════════════════════════════════╗"
echo "║                                                    ║"
echo "║     NimsForest Staging Setup (Local)               ║"
echo "║                                                    ║"
echo "╚════════════════════════════════════════════════════╝"
echo ""

# Check if SERVER_IP is provided
if [ -z "$1" ]; then
    log_error "Server IP address required!"
    echo ""
    echo "Usage: $0 SERVER_IP"
    echo ""
    echo "Example:"
    echo "  $0 123.456.789.012"
    echo ""
    exit 1
fi

SERVER_IP="$1"
SSH_KEY_PATH="$HOME/.ssh/nimsforest_staging_deploy"
KNOWN_HOSTS_PATH="/tmp/staging_known_hosts"

log_info "Server IP: $SERVER_IP"
echo ""

# Step 1: Generate SSH key
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
log_step "Step 1: Generate SSH Key for Deployment"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

if [ -f "$SSH_KEY_PATH" ]; then
    log_warn "SSH key already exists at: $SSH_KEY_PATH"
    read -p "Do you want to use the existing key? (y/n) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Generating new SSH key..."
        rm -f "$SSH_KEY_PATH" "$SSH_KEY_PATH.pub"
        ssh-keygen -t ed25519 -C "github-actions-staging" -f "$SSH_KEY_PATH" -N ""
        log_info "New SSH key generated!"
    fi
else
    log_info "Generating SSH key..."
    ssh-keygen -t ed25519 -C "github-actions-staging" -f "$SSH_KEY_PATH" -N ""
    log_info "SSH key generated!"
fi

echo ""
log_info "Private key: $SSH_KEY_PATH"
log_info "Public key:  $SSH_KEY_PATH.pub"
echo ""

# Step 2: Copy public key to server
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
log_step "Step 2: Copy Public Key to Server"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

log_info "Copying public key to root@$SERVER_IP..."
echo ""
log_warn "You may be prompted for the server's root password."
log_warn "If you added an SSH key during server creation, no password needed."
echo ""

if ssh-copy-id -i "$SSH_KEY_PATH.pub" "root@$SERVER_IP"; then
    log_info "Public key copied successfully!"
else
    log_error "Failed to copy public key to server"
    log_warn "Make sure you can SSH to the server with: ssh root@$SERVER_IP"
    exit 1
fi

echo ""

# Test SSH connection
log_step "Testing SSH connection..."
if ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no "root@$SERVER_IP" "echo 'Connection successful!'"; then
    log_info "SSH connection works!"
else
    log_error "SSH connection failed"
    exit 1
fi

echo ""

# Step 3: Get server's SSH fingerprint
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
log_step "Step 3: Get Server SSH Fingerprint"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

log_info "Retrieving SSH host keys..."
ssh-keyscan "$SERVER_IP" > "$KNOWN_HOSTS_PATH" 2>/dev/null

if [ -s "$KNOWN_HOSTS_PATH" ]; then
    log_info "SSH fingerprint saved to: $KNOWN_HOSTS_PATH"
else
    log_error "Failed to retrieve SSH fingerprint"
    exit 1
fi

echo ""

# Step 4: Configure GitHub secrets
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
log_step "Step 4: Configure GitHub Secrets"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    log_warn "GitHub CLI (gh) not found!"
    echo ""
    echo "Install it from: https://cli.github.com/"
    echo ""
    echo "Or use these commands manually:"
    echo ""
    echo "macOS:   brew install gh"
    echo "Linux:   See https://github.com/cli/cli/blob/trunk/docs/install_linux.md"
    echo ""
    log_warn "After installing gh, run:"
    echo "  gh auth login"
    echo "  $0 $SERVER_IP"
    echo ""
    exit 1
fi

# Check if logged in to GitHub
if ! gh auth status &> /dev/null; then
    log_warn "Not logged in to GitHub CLI"
    echo ""
    log_info "Running: gh auth login"
    echo ""
    gh auth login
fi

log_info "Setting GitHub secrets..."
echo ""

# Set secrets
gh secret set STAGING_SSH_PRIVATE_KEY < "$SSH_KEY_PATH"
log_info "✓ STAGING_SSH_PRIVATE_KEY set"

gh secret set STAGING_SSH_USER --body "root"
log_info "✓ STAGING_SSH_USER set"

gh secret set STAGING_SSH_HOST --body "$SERVER_IP"
log_info "✓ STAGING_SSH_HOST set"

gh secret set STAGING_SSH_KNOWN_HOSTS < "$KNOWN_HOSTS_PATH"
log_info "✓ STAGING_SSH_KNOWN_HOSTS set"

echo ""
log_info "All GitHub secrets configured!"
echo ""

# Verify secrets
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
log_step "Verification"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

log_info "GitHub secrets:"
gh secret list | grep "STAGING_SSH_" || echo "  (No staging secrets found)"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
log_info "🎉 Setup complete!"
echo ""
echo "Next steps:"
echo ""
echo "  1. Test deployment:"
echo "     git commit --allow-empty -m 'test: trigger staging deployment'"
echo "     git push origin main"
echo ""
echo "  2. Watch deployment:"
echo "     gh run watch"
echo ""
echo "  3. Check service on server:"
echo "     ssh -i $SSH_KEY_PATH root@$SERVER_IP 'sudo systemctl status nimsforest'"
echo ""
echo "  4. View logs:"
echo "     ssh -i $SSH_KEY_PATH root@$SERVER_IP 'sudo journalctl -u nimsforest -f'"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Save server info
SERVER_INFO_FILE="$HOME/.nimsforest_staging_info"
cat > "$SERVER_INFO_FILE" << EOF
# NimsForest Staging Server Info
# Generated: $(date)

SERVER_IP=$SERVER_IP
SSH_KEY=$SSH_KEY_PATH
SSH_USER=root

# Quick commands:
# SSH:  ssh -i $SSH_KEY_PATH root@$SERVER_IP
# Logs: ssh -i $SSH_KEY_PATH root@$SERVER_IP 'sudo journalctl -u nimsforest -f'
# Status: ssh -i $SSH_KEY_PATH root@$SERVER_IP 'sudo systemctl status nimsforest'
EOF

log_info "Server info saved to: $SERVER_INFO_FILE"
echo ""
