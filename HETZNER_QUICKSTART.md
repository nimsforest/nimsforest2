# Hetzner Staging Setup - Quick Start

The fastest way to get a staging server running on Hetzner.

## TL;DR - 5 Minute Setup

```bash
# 1. Create Hetzner server (via web console)
#    → https://console.hetzner.cloud/
#    → Ubuntu 22.04, CPX11, name: nimsforest-staging
#    → Copy the IP address

# 2. Set up the server
ssh root@YOUR_SERVER_IP
wget https://raw.githubusercontent.com/YOUR_USERNAME/nimsforest/main/scripts/setup-server.sh
chmod +x setup-server.sh && sudo ./setup-server.sh
# Wait 3-5 minutes for setup to complete
exit

# 3. Configure local deployment (one command!)
./scripts/setup-staging-local.sh YOUR_SERVER_IP

# 4. Deploy!
git push origin main
gh run watch
```

Done! 🎉

---

## Detailed Instructions

### Option 1: Automated Script (Recommended)

This script does steps 2-4 automatically:

```bash
# After creating your Hetzner server:
./scripts/setup-staging-local.sh YOUR_SERVER_IP
```

**What it does:**
- ✅ Generates SSH keys for deployment
- ✅ Copies public key to server
- ✅ Gets server SSH fingerprint
- ✅ Configures all 4 GitHub secrets
- ✅ Verifies everything works

### Option 2: Manual Setup

Follow the comprehensive guide: **[STAGING_SETUP_GUIDE.md](./STAGING_SETUP_GUIDE.md)**

Includes:
- Step-by-step instructions with screenshots
- Troubleshooting section
- Security best practices
- Cost breakdown
- Next steps (production, monitoring, backups)

### Option 3: Interactive Checklist

Use the checklist to track your progress: **[STAGING_SETUP_CHECKLIST.md](./STAGING_SETUP_CHECKLIST.md)**

Perfect for:
- First-time setup
- Team member onboarding
- Documentation purposes

---

## Prerequisites

- **Hetzner account** (free to create)
- **GitHub CLI** (`gh`) - [Install here](https://cli.github.com/)
- **SSH client** (standard on Mac/Linux)
- **Git** (for pushing code)

---

## What You Get

After setup:

✅ **Automatic Deployments**
- Push to `main` → Auto-deploy to staging
- Create release → Auto-deploy to production

✅ **Production-Ready Server**
- Ubuntu 22.04 LTS
- Go 1.24.0
- NATS Server with JetStream
- Firewall configured (UFW)
- SSH hardened (fail2ban)
- Automatic security updates

✅ **Monitoring & Management**
- systemd service management
- journald logging
- NATS monitoring UI (port 8222)

---

## Cost

**Staging Server:** €4.51/month (CPX11 - 2 vCPU, 2GB RAM)

Optional:
- **Production Server:** +€4.51/month
- **Backups:** +€0.02/month
- **Load Balancer:** +€5.39/month (if needed)

**Total for staging + production:** ~€9/month

---

## Quick Commands

```bash
# Create server (after logging into Hetzner)
hcloud server create --name nimsforest-staging --type cpx11 --image ubuntu-22.04

# Setup (automated)
./scripts/setup-staging-local.sh YOUR_SERVER_IP

# Deploy
git push origin main

# Watch deployment
gh run watch

# SSH to server
ssh -i ~/.ssh/nimsforest_staging_deploy root@YOUR_SERVER_IP

# Check service
ssh root@YOUR_SERVER_IP "sudo systemctl status nimsforest"

# View logs
ssh root@YOUR_SERVER_IP "sudo journalctl -u nimsforest -f"

# Manual deployment
gh workflow run deploy.yml --ref main -f environment=staging
```

---

## Files in This Setup

| File | Purpose |
|------|---------|
| **[HETZNER_QUICKSTART.md](./HETZNER_QUICKSTART.md)** | This file - quick reference |
| **[STAGING_SETUP_GUIDE.md](./STAGING_SETUP_GUIDE.md)** | Comprehensive step-by-step guide |
| **[STAGING_SETUP_CHECKLIST.md](./STAGING_SETUP_CHECKLIST.md)** | Interactive checklist |
| **[scripts/setup-server.sh](./scripts/setup-server.sh)** | Server setup script (runs on server) |
| **[scripts/setup-staging-local.sh](./scripts/setup-staging-local.sh)** | Local setup script (runs on your machine) |
| **[DEPLOYMENT_SSH.md](./DEPLOYMENT_SSH.md)** | Platform-agnostic SSH deployment docs |

---

## Troubleshooting

### Script fails with "SSH connection failed"

```bash
# Test basic SSH access first
ssh root@YOUR_SERVER_IP

# If password prompt, add your SSH key to Hetzner server creation
# Or manually copy it:
ssh-copy-id root@YOUR_SERVER_IP
```

### "gh command not found"

```bash
# Install GitHub CLI
brew install gh  # macOS
# OR visit: https://cli.github.com/

# Login
gh auth login
```

### Deployment skipped with "secrets not configured"

```bash
# Verify secrets are set
gh secret list

# Should show all 4 staging secrets:
# - STAGING_SSH_PRIVATE_KEY
# - STAGING_SSH_USER
# - STAGING_SSH_HOST
# - STAGING_SSH_KNOWN_HOSTS

# Re-run setup if missing
./scripts/setup-staging-local.sh YOUR_SERVER_IP
```

### Service won't start on server

```bash
# SSH to server
ssh root@YOUR_SERVER_IP

# Check NATS is running
sudo systemctl status nats

# If not running, restart it
sudo systemctl restart nats

# Check NimsForest logs
sudo journalctl -u nimsforest -n 100 --no-pager

# Try running binary manually to see errors
/usr/local/bin/forest
```

---

## Next Steps

### 1. Set Up Production

Repeat the process for production:

```bash
# Create production server
hcloud server create --name nimsforest-production --type cpx11 --image ubuntu-22.04

# Generate separate keys
ssh-keygen -t ed25519 -f ~/.ssh/nimsforest_production_deploy -N ""

# Setup production (change script for PRODUCTION_ prefix)
# Or manually add secrets with PRODUCTION_ prefix
gh secret set PRODUCTION_SSH_PRIVATE_KEY < ~/.ssh/nimsforest_production_deploy
gh secret set PRODUCTION_SSH_USER --body "root"
gh secret set PRODUCTION_SSH_HOST --body "PROD_SERVER_IP"
gh secret set PRODUCTION_SSH_KNOWN_HOSTS < <(ssh-keyscan PROD_SERVER_IP)

# Deploy with a release
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

### 2. Add Custom Domain

```bash
# Buy domain (namecheap, google domains, etc.)

# Add DNS A records:
# staging.yourdomain.com → STAGING_IP
# app.yourdomain.com     → PRODUCTION_IP

# Update GitHub secrets with domain names
gh secret set STAGING_SSH_HOST --body "staging.yourdomain.com"
gh secret set PRODUCTION_SSH_HOST --body "app.yourdomain.com"
```

### 3. Set Up Monitoring

Free options:
- **Uptime:** [UptimeRobot](https://uptimerobot.com/)
- **Server:** [Netdata](https://www.netdata.cloud/)
- **Logs:** [Better Stack](https://betterstack.com/)

### 4. Enable Backups

```bash
# Via Hetzner (costs €0.02/GB/month)
hcloud server enable-backup nimsforest-staging

# Or via script on server (free)
# See STAGING_SETUP_GUIDE.md "Set up Backups" section
```

---

## Support

- **Issues:** See [STAGING_SETUP_GUIDE.md](./STAGING_SETUP_GUIDE.md) troubleshooting section
- **Hetzner:** https://docs.hetzner.com/cloud/
- **GitHub Actions:** https://docs.github.com/actions

---

## Security Notes

✅ **What we do:**
- Dedicated SSH keys (not your personal key)
- Key-based auth only (no passwords)
- Firewall configured
- fail2ban active
- Automatic security updates
- Secrets stored in GitHub (encrypted)

⚠️ **What you should do:**
- Keep SSH keys secure
- Don't commit keys to git
- Use strong passwords for Hetzner account
- Enable 2FA on GitHub and Hetzner
- Regularly check server logs
- Keep server software updated

---

**🚀 Ready to deploy? Let's go!**

```bash
./scripts/setup-staging-local.sh YOUR_SERVER_IP
```
