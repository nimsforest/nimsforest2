# Staging Environment Setup

Quick guide to set up staging on Hetzner (or any Linux server).

## Prerequisites

- Hetzner Cloud account (or any VPS provider)
- GitHub CLI (`gh`) installed locally
- SSH access

---

## Step 1: Create Server

Create a server on [Hetzner Cloud](https://console.hetzner.cloud/):
- **Image:** Ubuntu 22.04
- **Type:** CPX11 (2 vCPU, 2GB RAM) - €4.51/month
- **Name:** nimsforest-staging

Copy the server IP address.

---

## Step 2: Setup Server

Run the setup script on your server:

```bash
# SSH to server
ssh root@YOUR_SERVER_IP

# Download and run setup script
wget https://raw.githubusercontent.com/YOUR_USERNAME/nimsforest/main/scripts/setup-server.sh
chmod +x setup-server.sh
sudo ./setup-server.sh
```

This installs:
- Go 1.24.0
- NATS Server with JetStream
- Firewall (UFW)
- fail2ban
- Directory structure

---

## Step 3: Configure GitHub Secrets

On your local machine where you have GitHub CLI:

```bash
# Generate SSH key for deployment
ssh-keygen -t ed25519 -C "github-staging" -f ~/.ssh/nimsforest_staging -N ""

# Copy public key to server
ssh-copy-id -i ~/.ssh/nimsforest_staging.pub root@YOUR_SERVER_IP

# Get server fingerprint
ssh-keyscan YOUR_SERVER_IP > /tmp/staging_known_hosts

# Set GitHub secrets
gh secret set STAGING_SSH_PRIVATE_KEY < ~/.ssh/nimsforest_staging
gh secret set STAGING_SSH_USER --body "root"
gh secret set STAGING_SSH_HOST --body "YOUR_SERVER_IP"
gh secret set STAGING_SSH_KNOWN_HOSTS < /tmp/staging_known_hosts
```

---

## Step 4: Deploy

```bash
git push origin main
```

GitHub Actions will automatically:
- Build your code
- Run tests
- Deploy to staging via SSH
- Restart service

---

## Monitoring

```bash
# Check service
ssh root@YOUR_SERVER_IP "sudo systemctl status nimsforest"

# View logs
ssh root@YOUR_SERVER_IP "sudo journalctl -u nimsforest -f"

# Check NATS
ssh root@YOUR_SERVER_IP "sudo systemctl status nats"
```

---

## Production Setup

Repeat the same process with:
- Server name: `nimsforest-production`
- Secrets prefix: `PRODUCTION_SSH_*`

Deploy to production via releases:
```bash
git tag -a v1.0.0 -m "Release 1.0.0"
git push origin v1.0.0
```

---

## How It Works

**Your workflow after setup:**
1. Make code changes
2. `git push origin main`
3. GitHub Actions automatically deploys to staging

**The deployment uses your existing Makefile:**
- `make deps` - Download dependencies
- `make build-deploy` - Build Linux binary
- `make deploy-package` - Create deployment package
- `make server-deploy` - Deploy on server

**Cost:** ~€5/month per server (Hetzner CPX11)

---

## Troubleshooting

### Service won't start
```bash
ssh root@YOUR_SERVER_IP
sudo systemctl status nimsforest
sudo journalctl -u nimsforest -n 100
```

### NATS issues
```bash
ssh root@YOUR_SERVER_IP
sudo systemctl status nats
sudo journalctl -u nats -n 100
curl http://localhost:8222/varz
```

### Deployment fails
```bash
gh run list --workflow=deploy.yml
gh run view --log
```

Check that all 4 secrets are set:
```bash
gh secret list | grep STAGING
```

---

## Private Repositories

If your repo is private, you can still use the wget method by:

1. **Making scripts/ directory public** (recommended)
2. **Or copying the script manually:**
   ```bash
   # View script locally
   cat scripts/setup-server.sh
   
   # SSH to server and create it
   ssh root@YOUR_SERVER_IP
   cat > setup-server.sh << 'EOF'
   # Paste script content here
   EOF
   chmod +x setup-server.sh
   sudo ./setup-server.sh
   ```

---

That's it! After initial setup, just `git push` to deploy.
