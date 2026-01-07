# Deployment Quick Reference

## All Using Make! No Shell Scripts

Everything uses Make commands - both locally and on the server.

---

## GitHub Secrets Required

**Staging** (auto-deploys on push to main):
```bash
STAGING_SSH_PRIVATE_KEY      # SSH key for staging server
STAGING_SSH_USER             # Usually "root"
STAGING_SSH_HOST             # Staging server IP
STAGING_SSH_KNOWN_HOSTS      # SSH fingerprint
```

**Production** (auto-deploys on release):
```bash
PRODUCTION_SSH_PRIVATE_KEY   # SSH key for production server
PRODUCTION_SSH_USER          # Usually "root"  
PRODUCTION_SSH_HOST          # Production server IP
PRODUCTION_SSH_KNOWN_HOSTS   # SSH fingerprint
```

**⚠️ Secrets are optional** - deployment skips with warning if not configured

---

## Setup Secrets

```bash
# Generate SSH keys
ssh-keygen -t ed25519 -f ~/.ssh/deploy_staging -N ""
ssh-keygen -t ed25519 -f ~/.ssh/deploy_prod -N ""

# Copy to servers
ssh-copy-id -i ~/.ssh/deploy_staging.pub root@STAGING_IP
ssh-copy-id -i ~/.ssh/deploy_prod.pub root@PROD_IP

# Add to GitHub (staging)
gh secret set STAGING_SSH_PRIVATE_KEY < ~/.ssh/deploy_staging
gh secret set STAGING_SSH_USER --body "root"
gh secret set STAGING_SSH_HOST --body "STAGING_IP"
gh secret set STAGING_SSH_KNOWN_HOSTS < <(ssh-keyscan STAGING_IP)

# Add to GitHub (production)
gh secret set PRODUCTION_SSH_PRIVATE_KEY < ~/.ssh/deploy_prod
gh secret set PRODUCTION_SSH_USER --body "root"
gh secret set PRODUCTION_SSH_HOST --body "PROD_IP"
gh secret set PRODUCTION_SSH_KNOWN_HOSTS < <(ssh-keyscan PROD_IP)
```

---

## Automatic Deployments (Make-based)

```bash
# Staging (automatic if secrets configured)
git push origin main
# → Runs: make deps, make build-deploy, make deploy-package
# → On server: make server-deploy

# Production (automatic if secrets configured)  
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
# → Runs: make deps, make build-deploy, make deploy-package
# → On server: make server-deploy
```

---

## Manual Deployment (All Make)

### Local Build
```bash
make deploy-package   # Builds and creates package with Makefile
```

### Deploy to Server
```bash
# Copy to server
scp nimsforest-deploy.tar.gz root@SERVER:/tmp/

# Deploy using Make on server
ssh root@SERVER << 'EOF'
  cd /tmp
  tar xzf nimsforest-deploy.tar.gz
  cd deploy
  sudo make server-deploy
  cd /tmp && rm -rf deploy nimsforest-deploy.tar.gz
EOF
```

---

## Server Commands (Make-based)

All operations on the server use Make:

```bash
# On server after extracting package
sudo make server-deploy      # Complete deployment
sudo make server-verify       # Verify deployment
sudo make server-rollback     # Rollback to previous version
sudo make server-status       # Check service status
sudo make server-logs         # View logs
sudo make server-restart      # Restart service
sudo make server-stop         # Stop service
sudo make server-start        # Start service
```

Individual steps (if needed):
```bash
sudo make server-create-user      # Create service user
sudo make server-create-dirs      # Create directories
sudo make server-backup           # Backup current binary
sudo make server-install-binary   # Install binary
sudo make server-install-service  # Install systemd service
```

---

## Local Make Commands

```bash
make build-deploy      # Build optimized deployment binary
make deploy-package    # Create deployment package (includes Makefile!)
make deploy-verify     # Verify deployment files
make help              # Show all commands
```

---

## Server Setup (One-Time)

```bash
ssh root@SERVER_IP
wget https://raw.githubusercontent.com/youruser/nimsforest/main/scripts/setup-server.sh
chmod +x setup-server.sh
sudo ./setup-server.sh
```

This installs:
- Go (latest)
- NATS Server with JetStream  
- Firewall (UFW)
- fail2ban
- Make (if not present)

---

## Platform-Agnostic!

Works with **any Linux server** with SSH + Make:
- Hetzner, DigitalOcean, AWS, Linode, Vultr
- Your own bare metal / VM / VPS
- Any Ubuntu/Debian server

**No cloud provider API needed** - just SSH!

---

## Troubleshooting

**Check what Make targets are available on server**:
```bash
ssh root@SERVER "cd /tmp/deploy && make help"
```

**View deployment logs**:
```bash
ssh root@SERVER "sudo make server-logs"
```

**Service not running**:
```bash
ssh root@SERVER "sudo make server-status"
ssh root@SERVER "sudo journalctl -u nimsforest -n 50"
```

**Rollback deployment**:
```bash
ssh root@SERVER << 'EOF'
  cd /tmp/deploy
  sudo make server-rollback
EOF
```

**Deployment skipped with warning**:
```
⚠️  Staging secrets not configured - skipping deployment
```
→ Add the required `STAGING_*` or `PRODUCTION_*` secrets

---

## Why Make Instead of Shell Scripts?

✅ **Consistent** - Same tool everywhere  
✅ **Self-documenting** - `make help` shows all commands  
✅ **Modular** - Each operation is a separate target  
✅ **Dependencies** - Make handles task dependencies  
✅ **Cross-platform** - Works on any Unix system  
✅ **No extra scripts** - Makefile does everything

---

## Cost Comparison

| Provider | 2GB/2vCPU | Notes |
|----------|-----------|-------|
| **Hetzner** | €4.51/month | Best value |
| **DigitalOcean** | $12/month | Good reliability |
| **AWS EC2** | ~$15/month | Scalable |
| **Your server** | $0 | Use existing hardware |

**For 2 environments**: Hetzner ~€9/month, Your hardware $0

---

## Documentation

- **[DEPLOYMENT_SSH.md](./DEPLOYMENT_SSH.md)** - Complete setup guide
- **[README.md](./README.md)** - Project overview
- **[Makefile](./Makefile)** - All Make targets

---

## Key Points

✅ **Everything uses Make** - No shell scripts for deployment  
✅ **Platform-agnostic** - Any SSH-accessible Linux server  
✅ **No API tokens** - Just SSH keys  
✅ **Optional secrets** - Skips with warning if not set  
✅ **Separate environments** - STAGING_* and PRODUCTION_* prefixes  
✅ **Automatic deployments** - Push to main or create release

---

**Quick help**: Run `make help` locally or `sudo make help` on server
