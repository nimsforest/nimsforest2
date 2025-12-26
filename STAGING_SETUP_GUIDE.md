# Staging Environment Setup Guide - Hetzner

This guide will walk you through setting up a staging server on Hetzner Cloud and configuring automatic deployment.

## Overview

**What you'll do:**
1. Create a Hetzner Cloud account and server (~10 min)
2. Set up the server with required software (~5 min)
3. Configure GitHub secrets for automatic deployment (~5 min)
4. Test the deployment pipeline (~2 min)

**Cost:** ~€4.51/month for a CPX11 server (2 vCPU, 2GB RAM)

---

## Step 1: Create Hetzner Account

If you don't have a Hetzner account:

1. Go to [Hetzner Cloud Console](https://console.hetzner.cloud/)
2. Sign up for an account
3. Verify your email
4. Add a payment method (credit card or PayPal)

---

## Step 2: Create the Staging Server

### Option A: Via Web Console (Easier)

1. **Log in** to [Hetzner Cloud Console](https://console.hetzner.cloud/)

2. **Create a new project** (or select existing):
   - Click "New Project"
   - Name: `nimsforest` or `nimsforest-staging`

3. **Click "Add Server"**

4. **Choose Location:**
   - Select closest to you (e.g., Nuremberg, Helsinki, Ashburn)

5. **Choose Image:**
   - Select: **Ubuntu 22.04**

6. **Choose Type:**
   - Select: **CPX11** (2 vCPU, 2GB RAM, 40GB disk)
   - Cost: €4.51/month

7. **Add SSH Key** (recommended):
   - Click "Add SSH Key"
   - Paste your public key: `cat ~/.ssh/id_ed25519.pub` or `cat ~/.ssh/id_rsa.pub`
   - Or create one: `ssh-keygen -t ed25519 -C "your_email@example.com"`
   - Name it: `my-laptop`

8. **Server Name:**
   - Name: `nimsforest-staging`

9. **Click "Create & Buy"**

10. **Wait 30-60 seconds** for server to be created

11. **Copy the IP address** - you'll need this!

### Option B: Via Hetzner CLI (Advanced)

```bash
# Install Hetzner CLI
brew install hcloud  # macOS
# OR
wget https://github.com/hetznercloud/cli/releases/latest/download/hcloud-linux-amd64.tar.gz
tar xzf hcloud-linux-amd64.tar.gz
sudo mv hcloud /usr/local/bin/

# Create API token in Hetzner Console: Project → Security → API Tokens
hcloud context create nimsforest

# Create SSH key (if you don't have one)
ssh-keygen -t ed25519 -C "your_email@example.com" -f ~/.ssh/hetzner_deploy

# Upload SSH key to Hetzner
hcloud ssh-key create --name my-laptop --public-key-from-file ~/.ssh/id_ed25519.pub

# Create server
hcloud server create \
  --name nimsforest-staging \
  --type cpx11 \
  --image ubuntu-22.04 \
  --ssh-key my-laptop \
  --location nbg1

# Get server IP
hcloud server describe nimsforest-staging
```

---

## Step 3: Initial Server Setup

Now that you have a server, let's set it up with all required software.

### 3.1 SSH into your server

```bash
# Replace YOUR_SERVER_IP with the IP from Hetzner
ssh root@YOUR_SERVER_IP

# If you get a warning about host authenticity, type 'yes'
```

### 3.2 Download and run the setup script

```bash
# On the server, run:
wget https://raw.githubusercontent.com/YOUR_GITHUB_USERNAME/nimsforest/main/scripts/setup-server.sh

# Make it executable
chmod +x setup-server.sh

# Run it
sudo ./setup-server.sh
```

**This script will:**
- ✅ Update system packages
- ✅ Install Go (1.24.0)
- ✅ Install NATS Server with JetStream
- ✅ Configure firewall (UFW)
- ✅ Set up fail2ban for SSH protection
- ✅ Configure automatic security updates
- ✅ Create necessary directories
- ✅ Set up log rotation
- ✅ Start NATS server

**Time:** ~3-5 minutes

### 3.3 Verify setup

```bash
# Check NATS is running
sudo systemctl status nats

# Should show: "Active: active (running)"

# Check NATS monitoring
curl http://localhost:8222/varz

# Should return JSON with server info

# You can now exit the server
exit
```

---

## Step 4: Configure GitHub Secrets

Now let's configure GitHub to automatically deploy to your staging server.

### 4.1 Generate deployment SSH key

On your **local machine** (not the server):

```bash
# Generate a dedicated SSH key for GitHub Actions deployments
ssh-keygen -t ed25519 -C "github-actions-staging" -f ~/.ssh/nimsforest_staging_deploy

# This creates two files:
#   ~/.ssh/nimsforest_staging_deploy      (private key - keep secret!)
#   ~/.ssh/nimsforest_staging_deploy.pub  (public key)
```

### 4.2 Add public key to the server

```bash
# Copy the public key to your staging server
# Replace YOUR_SERVER_IP with your actual IP
ssh-copy-id -i ~/.ssh/nimsforest_staging_deploy.pub root@YOUR_SERVER_IP

# Test the key works
ssh -i ~/.ssh/nimsforest_staging_deploy root@YOUR_SERVER_IP "echo 'SSH key works!'"
# Should print: SSH key works!
```

### 4.3 Get the server's SSH fingerprint

```bash
# Get the server's SSH host key
ssh-keyscan YOUR_SERVER_IP > /tmp/staging_known_hosts

# View it (should show a few lines starting with YOUR_SERVER_IP)
cat /tmp/staging_known_hosts
```

### 4.4 Add secrets to GitHub

**Option A: Via GitHub CLI (Recommended)**

```bash
# Install GitHub CLI if you don't have it
brew install gh  # macOS
# OR: https://cli.github.com/

# Login to GitHub
gh auth login

# Navigate to your project
cd /path/to/nimsforest

# Add the four required secrets
gh secret set STAGING_SSH_PRIVATE_KEY < ~/.ssh/nimsforest_staging_deploy
gh secret set STAGING_SSH_USER --body "root"
gh secret set STAGING_SSH_HOST --body "YOUR_SERVER_IP"
gh secret set STAGING_SSH_KNOWN_HOSTS < /tmp/staging_known_hosts

# Verify secrets are set
gh secret list
```

**Option B: Via GitHub Web Interface**

1. Go to your repository on GitHub
2. Click **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret**

Add these four secrets:

| Secret Name | Value | How to Get |
|------------|-------|------------|
| `STAGING_SSH_PRIVATE_KEY` | Private key content | `cat ~/.ssh/nimsforest_staging_deploy` |
| `STAGING_SSH_USER` | `root` | Use this exact value |
| `STAGING_SSH_HOST` | Your server IP | From Hetzner console |
| `STAGING_SSH_KNOWN_HOSTS` | Host fingerprint | `cat /tmp/staging_known_hosts` |

**Important:** Copy the entire private key including the lines:
```
-----BEGIN OPENSSH PRIVATE KEY-----
...
-----END OPENSSH PRIVATE KEY-----
```

---

## Step 5: Test the Deployment

Now let's trigger a deployment to make sure everything works!

### 5.1 Commit and push to main

```bash
# Make a small change (or just push current state)
git add .
git commit -m "feat: set up staging environment"
git push origin main
```

### 5.2 Watch the deployment

**Option A: Via GitHub Web**

1. Go to your repository on GitHub
2. Click **Actions** tab
3. You should see a workflow running: "Deploy via SSH"
4. Click on it to see live logs

**Option B: Via GitHub CLI**

```bash
# Watch the workflow run
gh run watch

# Or list recent runs
gh run list --workflow=deploy.yml

# View logs for a specific run
gh run view --log
```

### 5.3 Verify deployment succeeded

You should see output like:

```
✅ Deployment to STAGING completed successfully!
```

Check the service is running on the server:

```bash
# SSH to your server
ssh -i ~/.ssh/nimsforest_staging_deploy root@YOUR_SERVER_IP

# Check NimsForest service status
sudo systemctl status nimsforest

# Should show: "Active: active (running)"

# View logs
sudo journalctl -u nimsforest -n 50 --no-pager

# Exit
exit
```

---

## 🎉 Success!

Your staging environment is now set up and configured for automatic deployment!

### What happens now?

- **Every push to `main`** → Automatically deploys to staging
- **Every release created** → Automatically deploys to production (when configured)
- **Manual trigger** → Deploy anytime via GitHub Actions UI

---

## Common Commands

### On your local machine:

```bash
# Trigger manual deployment
gh workflow run deploy.yml --ref main -f environment=staging

# Watch deployment
gh run watch

# SSH to staging server
ssh -i ~/.ssh/nimsforest_staging_deploy root@YOUR_SERVER_IP

# View deployment logs
gh run view --log
```

### On the staging server:

```bash
# Check NimsForest service
sudo systemctl status nimsforest
sudo journalctl -u nimsforest -f

# Check NATS service
sudo systemctl status nats
sudo journalctl -u nats -f

# Check NATS monitoring
curl http://localhost:8222/varz

# Restart services
sudo systemctl restart nimsforest
sudo systemctl restart nats

# View server resources
free -h
df -h
top
```

---

## Troubleshooting

### Issue: "Permission denied (publickey)"

**Solution:**
```bash
# Make sure the SSH key is added to the server
ssh-copy-id -i ~/.ssh/nimsforest_staging_deploy.pub root@YOUR_SERVER_IP

# Test connection
ssh -i ~/.ssh/nimsforest_staging_deploy root@YOUR_SERVER_IP
```

### Issue: "NATS connection refused"

**Solution:**
```bash
# SSH to server and check NATS
ssh root@YOUR_SERVER_IP
sudo systemctl status nats
sudo systemctl restart nats
curl http://localhost:8222/varz
```

### Issue: "Deployment workflow skipped"

**Reason:** Secrets not configured properly

**Solution:**
```bash
# Verify all 4 secrets are set
gh secret list

# Should show:
# STAGING_SSH_PRIVATE_KEY
# STAGING_SSH_USER
# STAGING_SSH_HOST
# STAGING_SSH_KNOWN_HOSTS

# Re-add if missing
```

### Issue: "Service failed to start"

**Solution:**
```bash
# SSH to server
ssh root@YOUR_SERVER_IP

# Check service status and logs
sudo systemctl status nimsforest
sudo journalctl -u nimsforest -n 100 --no-pager

# Check if binary exists and is executable
ls -la /usr/local/bin/forest

# Try running manually to see errors
/usr/local/bin/forest
```

---

## Next Steps

### Set up Production Environment

Once staging is working, repeat this process for production:

1. Create another Hetzner server named `nimsforest-production`
2. Run the setup script on it
3. Generate new SSH keys: `ssh-keygen -t ed25519 -f ~/.ssh/nimsforest_production_deploy`
4. Add production secrets with `PRODUCTION_` prefix:
   - `PRODUCTION_SSH_PRIVATE_KEY`
   - `PRODUCTION_SSH_USER`
   - `PRODUCTION_SSH_HOST`
   - `PRODUCTION_SSH_KNOWN_HOSTS`
5. Create a release to deploy to production:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

### Set up Custom Domain

1. Buy a domain (e.g., from Namecheap, Google Domains)
2. Add DNS A records pointing to your server IPs:
   ```
   staging.yourdomain.com → STAGING_IP
   app.yourdomain.com     → PRODUCTION_IP
   ```
3. Update `STAGING_SSH_HOST` and `PRODUCTION_SSH_HOST` secrets with domain names

### Set up Monitoring

Consider adding:
- **Uptime monitoring**: [UptimeRobot](https://uptimerobot.com/) (free tier available)
- **Server monitoring**: [Netdata](https://www.netdata.cloud/) (free, easy to install)
- **Log aggregation**: [Better Stack](https://betterstack.com/) (free tier available)

### Set up Backups

```bash
# SSH to server
ssh root@YOUR_SERVER_IP

# Create backup script
sudo nano /usr/local/bin/backup-nimsforest.sh
```

Add:
```bash
#!/bin/bash
BACKUP_DIR=/opt/backups
mkdir -p $BACKUP_DIR
tar czf $BACKUP_DIR/nats-data-$(date +%Y%m%d-%H%M%S).tar.gz /var/lib/nats
# Keep only last 7 days
find $BACKUP_DIR -name "nats-data-*.tar.gz" -mtime +7 -delete
```

```bash
# Make executable
sudo chmod +x /usr/local/bin/backup-nimsforest.sh

# Add to crontab (daily at 2 AM)
sudo crontab -e
# Add: 0 2 * * * /usr/local/bin/backup-nimsforest.sh
```

---

## Cost Estimate

### Hetzner Cloud Pricing

| Item | Cost |
|------|------|
| CPX11 Server (Staging) | €4.51/month |
| CPX11 Server (Production) | €4.51/month |
| Backups (optional, 20GB) | ~€0.02/month |
| **Total** | **~€9/month** |

**Note:** First month often has promotional pricing. Check Hetzner's current offers.

---

## Security Checklist

- ✅ Dedicated SSH keys for deployment (not your personal key)
- ✅ SSH key authentication only (no password login)
- ✅ Firewall (UFW) configured
- ✅ Fail2ban active for SSH protection
- ✅ Automatic security updates enabled
- ✅ GitHub secrets properly configured
- ✅ NATS not exposed to internet (local only)
- ⬜ Consider adding SSL/TLS if exposing services publicly
- ⬜ Consider setting up VPN for server access
- ⬜ Consider enabling NATS authentication for production

---

## Additional Resources

- **Hetzner Cloud Docs**: https://docs.hetzner.com/cloud/
- **Hetzner CLI**: https://github.com/hetznercloud/cli
- **NATS Documentation**: https://docs.nats.io/
- **GitHub Actions Docs**: https://docs.github.com/en/actions
- **NimsForest Deployment Docs**: [DEPLOYMENT_SSH.md](./DEPLOYMENT_SSH.md)

---

## Support

If you encounter issues:

1. Check the [Troubleshooting](#troubleshooting) section above
2. Review GitHub Actions logs: Repository → Actions → Select failed run
3. Check server logs: `ssh root@SERVER "sudo journalctl -u nimsforest -n 100"`
4. Open an issue on GitHub with logs and error messages

---

**🌲 Your NimsForest staging environment is ready!** 🌲
