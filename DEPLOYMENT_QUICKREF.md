# Deployment Quick Reference

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

## Automatic Deployments

```bash
# Staging (automatic if secrets configured)
git push origin main
# → Deploys to staging server via SSH
# → Skips with warning if STAGING_* secrets not set

# Production (automatic if secrets configured)  
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
# → Deploys to production server via SSH
# → Skips with warning if PRODUCTION_* secrets not set
```

---

## Platform-Agnostic!

Works with **any Linux server** you can SSH into:
- Hetzner, DigitalOcean, AWS, Linode, Vultr
- Your own bare metal / VM / VPS
- Literally any Ubuntu/Debian server with SSH

**No cloud provider API needed** - just SSH!

---

## Manual Deployment

### Using GitHub CLI
```bash
# Deploy to staging
gh workflow run deploy.yml -f environment=staging

# Deploy to production
gh workflow run deploy.yml -f environment=production
```

### Using Make
```bash
# Build and package
make deploy-package

# Deploy to any server via SSH
scp nimsforest-deploy.tar.gz root@SERVER:/tmp/
ssh root@SERVER 'bash -s' < scripts/deploy.sh deploy
```

---

## Common Commands

```bash
# Check status
ssh root@SERVER "sudo systemctl status nimsforest"

# View logs
ssh root@SERVER "sudo journalctl -u nimsforest -f"

# Restart service
ssh root@SERVER "sudo systemctl restart nimsforest"

# Rollback
ssh root@SERVER 'bash -s' < scripts/deploy.sh rollback

# Verify deployment
ssh root@SERVER 'bash -s' < scripts/deploy.sh verify
```

---

## Make Commands

```bash
make build-deploy      # Build deployment binary
make deploy-package    # Create deployment package
make deploy-verify     # Verify deployment files
make help              # Show all commands
```

---

## Server Setup

### 1. Create Server
Any cloud provider or own hardware - just needs SSH access

### 2. Run Setup Script
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
- Automatic security updates

---

## Troubleshooting

**Deployment skipped with warning**:
```bash
⚠️  Staging secrets not configured - skipping deployment
```
→ Add the required `STAGING_*` or `PRODUCTION_*` secrets

**Deployment fails**: Check GitHub Actions logs
```bash
gh run list --workflow=deploy.yml
gh run view --log
```

**Service not running**: Check logs on server
```bash
ssh root@SERVER "sudo journalctl -u nimsforest -n 50"
```

**Need to rollback**: Use deployment script
```bash
ssh root@SERVER 'bash -s' < scripts/deploy.sh rollback
```

**SSH connection fails**: Verify secrets and server access
```bash
# Test SSH manually
ssh root@SERVER

# Check secrets are set
gh secret list
```

---

## Cost Comparison

| Provider | 2GB/2vCPU | Notes |
|----------|-----------|-------|
| **Hetzner** | €4.51/month | Best value |
| **DigitalOcean** | $12/month | Good reliability |
| **AWS EC2** | ~$15/month | Scalable |
| **Linode** | $12/month | Solid choice |
| **Your server** | $0 | Use existing hardware |

**For 2 environments** (staging + production):
- Hetzner: ~€9/month
- DigitalOcean: ~$24/month
- Your hardware: $0

---

## Documentation

- **[DEPLOYMENT_SSH.md](./DEPLOYMENT_SSH.md)** - Complete setup guide
- **[WHATS_NEW.md](./WHATS_NEW.md)** - What changed
- **[README.md](./README.md)** - Project overview
- **[Makefile](./Makefile)** - All Make targets

---

## Key Points

✅ **Platform-agnostic** - Works with any SSH-accessible Linux server  
✅ **No API tokens needed** - Just SSH keys  
✅ **Optional secrets** - Skips deployment with warning if not set  
✅ **Separate staging/production** - Different secret prefixes  
✅ **Automatic staging** - Push to main  
✅ **Automatic production** - Create release  
✅ **Cost-effective** - Use any provider or own hardware

---

**Quick help**: See [DEPLOYMENT_SSH.md](./DEPLOYMENT_SSH.md) for detailed instructions
