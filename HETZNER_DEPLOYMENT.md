# Hetzner Deployment Guide

Complete guide for deploying NimsForest to Hetzner Cloud with automatic CD.

## Quick Summary

- **Staging**: Auto-deploys on push to `main`
- **Production**: Auto-deploys on release (tag `v*`)
- **Manual**: Trigger via GitHub Actions UI
- **Cost**: ~€5/month per environment

## Table of Contents

- [Quick Start](#quick-start)
- [Initial Server Setup](#initial-server-setup)
- [GitHub Configuration](#github-configuration)
- [Deployment Workflow](#deployment-workflow)
- [Manual Deployment](#manual-deployment)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)

---

## Quick Start

### 1. Create Servers (10 min)

```bash
# Staging server
hcloud server create --name nimsforest-staging --type cpx11 --image ubuntu-22.04

# Production server (when ready)
hcloud server create --name nimsforest-prod --type cpx11 --image ubuntu-22.04
```

### 2. Setup Servers (5 min each)

```bash
# For each server
ssh root@SERVER_IP
wget https://raw.githubusercontent.com/youruser/nimsforest/main/scripts/setup-hetzner-server.sh
chmod +x setup-hetzner-server.sh
sudo ./setup-hetzner-server.sh
```

### 3. Configure GitHub Secrets (5 min)

```bash
# Generate keys
ssh-keygen -t ed25519 -f ~/.ssh/deploy_staging
ssh-keygen -t ed25519 -f ~/.ssh/deploy_prod

# Copy to servers
ssh-copy-id -i ~/.ssh/deploy_staging.pub root@STAGING_IP
ssh-copy-id -i ~/.ssh/deploy_prod.pub root@PROD_IP

# Add to GitHub (for staging)
gh secret set HETZNER_SSH_PRIVATE_KEY --env staging < ~/.ssh/deploy_staging
gh secret set HETZNER_SSH_USER --env staging --body "root"
gh secret set HETZNER_HOST --env staging --body "STAGING_IP"
gh secret set HETZNER_KNOWN_HOSTS --env staging < <(ssh-keyscan STAGING_IP)

# Add to GitHub (for production)
gh secret set HETZNER_SSH_PRIVATE_KEY --env production < ~/.ssh/deploy_prod
gh secret set HETZNER_SSH_USER --env production --body "root"
gh secret set HETZNER_HOST --env production --body "PROD_IP"
gh secret set HETZNER_KNOWN_HOSTS --env production < <(ssh-keyscan PROD_IP)
```

### 4. Deploy!

```bash
# Staging: Just push to main
git push origin main
# → Auto-deploys to staging

# Production: Create a release
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
# → Auto-deploys to production
```

**Done!** 🎉

---

## Deployment Triggers

| Event | Environment | When |
|-------|-------------|------|
| Push to `main` | **Staging** | Every commit to main branch |
| Release created (`v*`) | **Production** | When you publish a release |
| Manual trigger | **Your choice** | Via GitHub Actions UI |

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    GitHub Repository                         │
│  ┌──────────────┐      ┌──────────────┐                    │
│  │   Release    │  or  │   Manual     │                    │
│  │   Created    │      │   Trigger    │                    │
│  └──────┬───────┘      └──────┬───────┘                    │
│         │                     │                             │
│         └─────────┬───────────┘                             │
│                   ↓                                          │
│         ┌──────────────────┐                                │
│         │  GitHub Actions  │                                │
│         │  - Build binary  │                                │
│         │  - Package       │                                │
│         │  - Deploy        │                                │
│         └────────┬─────────┘                                │
└──────────────────┼──────────────────────────────────────────┘
                   │ SSH
                   ↓
         ┌──────────────────┐
         │  Hetzner Server  │
         │                  │
         │  ┌────────────┐  │
         │  │   NATS     │  │ (JetStream enabled)
         │  └────────────┘  │
         │  ┌────────────┐  │
         │  │ NimsForest │  │ (systemd service)
         │  └────────────┘  │
         └──────────────────┘
```

## Server Requirements

**Minimum per environment**:
- Ubuntu 22.04 or Debian 11+
- 2GB RAM, 2 vCPU (Hetzner CPX11)
- Public IP address

**Recommended**: Start with CPX11 (~€4.51/month) for each environment

## Initial Server Setup

### Step 1: Create Hetzner Server

#### Via Hetzner Cloud Console:

1. Log in to [Hetzner Cloud Console](https://console.hetzner.cloud/)
2. Create a new project (or select existing)
3. Click "Add Server"
4. Choose:
   - Location: Select nearest to your users
   - Image: Ubuntu 22.04 or Debian 11
   - Type: CPX11 or higher
   - SSH Key: Add your SSH public key
5. Click "Create & Buy"

#### Via Hetzner CLI (hcloud):

```bash
# Install hcloud CLI
brew install hcloud  # macOS
# or download from: https://github.com/hetznercloud/cli

# Login
hcloud context create nimsforest

# Create server
hcloud server create \
  --name nimsforest-prod \
  --type cpx11 \
  --image ubuntu-22.04 \
  --ssh-key your-key-name \
  --location nbg1
```

### Step 2: Initial Server Configuration

Once your server is created, run the initial setup script:

```bash
# SSH into your server
ssh root@YOUR_SERVER_IP

# Download and run setup script
wget https://raw.githubusercontent.com/yourusername/nimsforest/main/scripts/setup-hetzner-server.sh
chmod +x setup-hetzner-server.sh
sudo ./setup-hetzner-server.sh
```

This script will:
- ✅ Update system packages
- ✅ Install Go (latest version)
- ✅ Install and configure NATS Server with JetStream
- ✅ Configure firewall (UFW)
- ✅ Set up fail2ban for SSH protection
- ✅ Configure automatic security updates
- ✅ Create necessary directories
- ✅ Set up log rotation

**Manual Setup (if you prefer):**

See the script content in `scripts/setup-hetzner-server.sh` for step-by-step manual instructions.

### Step 3: Verify Server Setup

After setup completes, verify everything is working:

```bash
# Check Go installation
go version

# Check NATS is running
sudo systemctl status nats
curl http://localhost:8222/varz

# Check firewall
sudo ufw status

# Check available resources
free -h
df -h
```

## GitHub Configuration

### Step 1: Generate SSH Key for Deployment

On your **local machine**, generate a dedicated SSH key for deployments:

```bash
# Generate SSH key (no passphrase for automation)
ssh-keygen -t ed25519 -C "github-actions-deploy" -f ~/.ssh/nimsforest_deploy
```

This creates:
- `~/.ssh/nimsforest_deploy` (private key - keep secret!)
- `~/.ssh/nimsforest_deploy.pub` (public key)

### Step 2: Add Public Key to Server

Copy the public key to your Hetzner server:

```bash
# Copy public key to server
ssh-copy-id -i ~/.ssh/nimsforest_deploy.pub root@YOUR_SERVER_IP

# Or manually:
cat ~/.ssh/nimsforest_deploy.pub
# Then on server: echo "PUBLIC_KEY_CONTENT" >> ~/.ssh/authorized_keys
```

### Step 3: Get Server's SSH Host Key

```bash
# Get the server's SSH host key fingerprint
ssh-keyscan YOUR_SERVER_IP > known_hosts_temp
cat known_hosts_temp
```

### Step 4: Configure GitHub Secrets

Add the following secrets to your GitHub repository:

**Go to**: Repository → Settings → Secrets and variables → Actions → New repository secret

Add these secrets:

| Secret Name | Value | How to Get |
|------------|-------|------------|
| `HETZNER_SSH_PRIVATE_KEY` | Private key content | `cat ~/.ssh/nimsforest_deploy` |
| `HETZNER_SSH_USER` | `root` | Default for Hetzner servers |
| `HETZNER_HOST` | Your server IP | From Hetzner console |
| `HETZNER_KNOWN_HOSTS` | Host key fingerprint | From `ssh-keyscan` output |

#### Adding Secrets via GitHub CLI:

```bash
# Install GitHub CLI
brew install gh  # macOS
# or: https://cli.github.com/

# Login
gh auth login

# Add secrets
gh secret set HETZNER_SSH_PRIVATE_KEY < ~/.ssh/nimsforest_deploy
gh secret set HETZNER_SSH_USER --body "root"
gh secret set HETZNER_HOST --body "YOUR_SERVER_IP"
gh secret set HETZNER_KNOWN_HOSTS < known_hosts_temp
```

#### Adding Secrets via Web Interface:

1. Copy private key:
   ```bash
   cat ~/.ssh/nimsforest_deploy
   # Copy the entire output including BEGIN and END lines
   ```

2. Go to GitHub:
   - Repository → Settings → Secrets and variables → Actions
   - Click "New repository secret"
   - Name: `HETZNER_SSH_PRIVATE_KEY`
   - Value: Paste the private key
   - Click "Add secret"

3. Repeat for other secrets

### Step 5: Set Up GitHub Environments (Optional but Recommended)

For production protection:

1. Go to: Repository → Settings → Environments
2. Click "New environment"
3. Name: `production`
4. Configure:
   - ✅ Required reviewers: Add yourself or team members
   - ✅ Wait timer: 5 minutes (optional)
   - Environment secrets: Add server-specific secrets here

## Deployment Workflow

### Automatic Deployment on Release

When you create a new release, deployment happens automatically:

```bash
# Create and push a new version tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# GitHub will automatically:
# 1. Run CI tests
# 2. Build release binaries
# 3. Create GitHub release
# 4. Deploy to Hetzner (via deploy-hetzner.yml workflow)
```

### Manual Deployment

You can also trigger deployment manually:

#### Via GitHub Web Interface:

1. Go to: Repository → Actions
2. Select "Deploy to Hetzner" workflow
3. Click "Run workflow"
4. Select branch: `main`
5. Select environment: `production` or `staging`
6. Click "Run workflow"

#### Via GitHub CLI:

```bash
# Trigger manual deployment
gh workflow run deploy-hetzner.yml \
  --ref main \
  -f environment=production

# Watch the deployment
gh run watch
```

### Deployment Steps

The deployment workflow performs these steps:

1. **Build**: Compile Go binary for Linux AMD64
2. **Package**: Create deployment tarball with binary and scripts
3. **Upload**: Copy package to server via SCP
4. **Deploy**: 
   - Stop existing service
   - Backup current binary
   - Install new binary
   - Update systemd service
   - Start service
5. **Verify**: Check service is running correctly
6. **Rollback**: Automatic rollback if deployment fails

## Manual Deployment

If you need to deploy manually without GitHub Actions:

### Option 1: Direct Deployment from Local Machine (Using Make)

```bash
# Build and package using Make
make deploy-package

# Copy to server
scp nimsforest-deploy.tar.gz root@YOUR_SERVER_IP:/tmp/

# Deploy on server
ssh root@YOUR_SERVER_IP 'bash -s' < scripts/deploy.sh deploy

# Verify deployment
ssh root@YOUR_SERVER_IP 'bash -s' < scripts/deploy.sh verify
```

### Option 1b: Manual Shell Commands

```bash
# Build binary
GOOS=linux GOARCH=amd64 go build -o forest ./cmd/forest

# Copy to server
scp forest root@YOUR_SERVER_IP:/tmp/

# Deploy on server
ssh root@YOUR_SERVER_IP << 'EOF'
  cd /tmp
  sudo systemctl stop nimsforest
  sudo cp /usr/local/bin/forest /opt/nimsforest/backups/forest.backup
  sudo cp forest /usr/local/bin/forest
  sudo chmod +x /usr/local/bin/forest
  sudo systemctl start nimsforest
  sudo systemctl status nimsforest
EOF
```

### Option 2: Build on Server (Using Make)

```bash
# SSH to server
ssh root@YOUR_SERVER_IP

# Clone repository (if not already)
git clone https://github.com/yourusername/nimsforest.git
cd nimsforest

# Or update existing
cd nimsforest
git pull origin main

# Build optimized deployment binary
make build-deploy

# Use deployment script
sudo systemctl stop nimsforest
sudo cp /usr/local/bin/forest /opt/nimsforest/backups/forest.backup
sudo cp forest /usr/local/bin/forest
sudo systemctl start nimsforest

# Verify
make verify
```

## Monitoring and Management

### Check Service Status

```bash
# Via SSH
ssh root@YOUR_SERVER_IP

# Service status
sudo systemctl status nimsforest

# View logs (live)
sudo journalctl -u nimsforest -f

# View last 100 lines
sudo journalctl -u nimsforest -n 100

# Check resource usage
top -u forest
```

### NATS Monitoring

```bash
# Check NATS status
sudo systemctl status nats

# NATS monitoring endpoint
curl http://localhost:8222/varz

# JetStream info
curl http://localhost:8222/jsz

# View NATS logs
sudo journalctl -u nats -f
```

### Log Files

```bash
# Application logs
sudo journalctl -u nimsforest --since "1 hour ago"

# NATS logs
sudo journalctl -u nats --since "1 hour ago"

# System logs
sudo tail -f /var/log/syslog
```

### Managing the Service

```bash
# Start service
sudo systemctl start nimsforest

# Stop service
sudo systemctl stop nimsforest

# Restart service
sudo systemctl restart nimsforest

# Reload configuration (if supported)
sudo systemctl reload nimsforest

# Enable auto-start on boot
sudo systemctl enable nimsforest

# Disable auto-start
sudo systemctl disable nimsforest
```

## Troubleshooting

### Service Won't Start

```bash
# Check service status and errors
sudo systemctl status nimsforest
sudo journalctl -u nimsforest -n 50 --no-pager

# Check binary is executable
ls -la /usr/local/bin/forest

# Try running manually to see errors
sudo -u forest /usr/local/bin/forest
```

### NATS Connection Issues

```bash
# Verify NATS is running
sudo systemctl status nats
curl http://localhost:8222/varz

# Check NATS logs
sudo journalctl -u nats -n 50

# Restart NATS
sudo systemctl restart nats

# Test connectivity
nc -zv localhost 4222
```

### Deployment Failed

```bash
# Check GitHub Actions logs
gh run list --workflow=deploy-hetzner.yml
gh run view <run-id> --log

# Check SSH connectivity from local machine
ssh -i ~/.ssh/nimsforest_deploy root@YOUR_SERVER_IP

# Verify secrets are set correctly
gh secret list
```

### Rollback to Previous Version

#### Automatic Rollback (if deployment fails):

The workflow automatically rolls back if verification fails.

#### Manual Rollback:

```bash
# Method 1: Using deployment script (recommended)
ssh root@YOUR_SERVER_IP 'bash -s' < scripts/deploy.sh rollback

# Method 2: SSH and run on server
ssh root@YOUR_SERVER_IP

# Check available backups
ls -lah /opt/nimsforest/backups/

# Rollback manually
sudo systemctl stop nimsforest
sudo cp /opt/nimsforest/backups/forest.backup /usr/local/bin/forest
sudo chmod +x /usr/local/bin/forest
sudo systemctl start nimsforest
```

### High Memory Usage

```bash
# Check memory usage
free -h
top -u forest

# Restart service to clear memory
sudo systemctl restart nimsforest

# Check for memory leaks in logs
sudo journalctl -u nimsforest | grep -i "memory\|oom"
```

### Disk Space Issues

```bash
# Check disk usage
df -h

# Clean old logs
sudo journalctl --vacuum-time=7d
sudo journalctl --vacuum-size=1G

# Clean old backups
sudo find /opt/nimsforest/backups -mtime +30 -delete

# Clean NATS data (WARNING: deletes all messages)
# sudo systemctl stop nats
# sudo rm -rf /var/lib/nats/*
# sudo systemctl start nats
```

## Security Best Practices

### 1. SSH Hardening

```bash
# Disable password authentication
sudo nano /etc/ssh/sshd_config
# Set: PasswordAuthentication no
# Set: PermitRootLogin prohibit-password
sudo systemctl restart sshd

# Use fail2ban to prevent brute force
sudo systemctl status fail2ban
```

### 2. Firewall Configuration

```bash
# Review firewall rules
sudo ufw status verbose

# Allow only necessary ports
sudo ufw allow 22/tcp   # SSH
# Don't expose NATS to internet unless needed
# If needed, use authentication and TLS
```

### 3. Keep System Updated

```bash
# Update system packages
sudo apt update
sudo apt upgrade -y

# Enable automatic security updates (already done by setup script)
sudo dpkg-reconfigure -plow unattended-upgrades
```

### 4. NATS Security

For production, configure NATS authentication:

```bash
# Edit NATS config
sudo nano /etc/nats/nats.conf

# Add authentication
authorization {
  user: nimsforest
  password: "CHANGE_THIS_PASSWORD"
}

# Update service environment
sudo nano /etc/systemd/system/nimsforest.service
# Change: Environment="NATS_URL=nats://nimsforest:PASSWORD@localhost:4222"

sudo systemctl daemon-reload
sudo systemctl restart nats
sudo systemctl restart nimsforest
```

### 5. Regular Backups

```bash
# Backup NATS data
sudo tar czf /backup/nats-$(date +%Y%m%d).tar.gz /var/lib/nats

# Backup binary and config
sudo tar czf /backup/nimsforest-$(date +%Y%m%d).tar.gz \
  /usr/local/bin/forest \
  /etc/systemd/system/nimsforest.service

# Automate with cron
sudo crontab -e
# Add: 0 2 * * * /usr/local/bin/backup-nimsforest.sh
```

### 6. Monitoring and Alerts

Consider setting up:
- **Uptime monitoring**: UptimeRobot, Pingdom
- **Server monitoring**: Prometheus + Grafana, Netdata
- **Log aggregation**: ELK stack, Loki
- **Alerting**: AlertManager, PagerDuty

## Cost Optimization

### Hetzner Cloud Pricing (as of 2024)

| Resource | Cost |
|----------|------|
| CPX11 (2 vCPU, 2GB RAM) | ~€4.51/month |
| CPX21 (3 vCPU, 4GB RAM) | ~€9.04/month |
| 40GB Disk | Included |
| Traffic | 20TB included, then €1/TB |

### Tips to Reduce Costs

1. **Right-size**: Start with CPX11 for testing, scale up as needed
2. **Snapshots**: Use Hetzner snapshots for backups (€0.01/GB/month)
3. **Floating IP**: Use if you need to switch servers (€1/month)
4. **ARM servers**: Consider CAX instances (cheaper, same performance)

## Advanced Configuration

### Multiple Environments

Set up separate servers for staging and production:

```yaml
# In deploy-hetzner.yml
environments:
  staging:
    HETZNER_HOST: staging.example.com
  production:
    HETZNER_HOST: prod.example.com
```

### Load Balancer Setup

For high availability, use Hetzner Load Balancer:

```bash
# Create load balancer via Hetzner CLI
hcloud load-balancer create \
  --name nimsforest-lb \
  --type lb11 \
  --location nbg1

# Add servers
hcloud load-balancer add-target nimsforest-lb \
  --server nimsforest-prod-1

hcloud load-balancer add-target nimsforest-lb \
  --server nimsforest-prod-2
```

### Custom Domain Setup

```bash
# On Hetzner Cloud
hcloud server describe nimsforest-prod

# Update DNS records (in your DNS provider)
# A record: nimsforest.example.com → SERVER_IP

# Optional: Set up reverse DNS in Hetzner console
```

## Support and Resources

### Documentation

- [NimsForest README](./README.md)
- [General Deployment Guide](./DEPLOYMENT.md)
- [CI/CD Documentation](./CI_CD.md)

### External Resources

- [Hetzner Cloud Docs](https://docs.hetzner.com/cloud/)
- [Hetzner Cloud CLI](https://github.com/hetznercloud/cli)
- [NATS Documentation](https://docs.nats.io/)
- [GitHub Actions Docs](https://docs.github.com/en/actions)

### Getting Help

- **GitHub Issues**: https://github.com/yourusername/nimsforest/issues
- **Hetzner Support**: https://www.hetzner.com/support
- **NATS Community**: https://slack.nats.io/

---

## Quick Reference

### Essential Commands

```bash
# Build and deploy using Make
make deploy-package
scp nimsforest-deploy.tar.gz root@SERVER:/tmp/
ssh root@SERVER 'bash -s' < scripts/deploy.sh deploy

# Verify deployment
ssh root@SERVER 'bash -s' < scripts/deploy.sh verify

# Check service
ssh root@SERVER 'sudo systemctl status nimsforest'

# View logs
ssh root@SERVER 'sudo journalctl -u nimsforest -f'

# Restart service
ssh root@SERVER 'sudo systemctl restart nimsforest'

# Rollback
ssh root@SERVER 'bash -s' < scripts/deploy.sh rollback
```

### GitHub Actions Commands

```bash
# Trigger deployment
gh workflow run deploy-hetzner.yml -f environment=production

# Watch deployment
gh run watch

# View logs
gh run view --log
```

---

**🚀 Happy Deploying!**
