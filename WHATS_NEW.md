# What's New: Platform-Agnostic SSH Deployment

## ✅ You Were Right!

The deployment is completely **platform-agnostic** - it just uses SSH. Works with **any Linux server** you can SSH into.

---

## 🔄 What Changed

### 1. Renamed Everything from "Hetzner" to "SSH"

**Files**:
- `deploy-hetzner.yml` → `deploy.yml`
- `HETZNER_DEPLOYMENT.md` → `DEPLOYMENT_SSH.md`

**GitHub Secrets** (now generic):
- `HETZNER_SSH_PRIVATE_KEY` → `SSH_PRIVATE_KEY`
- `HETZNER_SSH_USER` → `SSH_USER`
- `HETZNER_HOST` → `SSH_HOST`
- `HETZNER_KNOWN_HOSTS` → `SSH_KNOWN_HOSTS`

### 2. Simplified Documentation

**Before**: 9 files, 2,900+ lines  
**After**: 3 files, 953 lines (67% reduction)

**Essential docs**:
- **[DEPLOYMENT_QUICKREF.md](./deployment/DEPLOYMENT_QUICKREF.md)** - Quick commands
- **[DEPLOYMENT_SSH.md](./deployment/DEPLOYMENT_SSH.md)** - Complete guide
- **[DEPLOYMENT_CHANGES.md](./deployment/DEPLOYMENT_CHANGES.md)** - What changed

### 3. Added Automatic Staging

- Push to `main` → Auto-deploys to staging
- Create release → Auto-deploys to production

---

## 🌍 Works With Any Linux Server

### Cloud Providers
- ✅ **Hetzner Cloud** (~€5/month)
- ✅ **DigitalOcean** (~$12/month)
- ✅ **AWS EC2** (~$15/month)
- ✅ **Linode** / Vultr / OVH
- ✅ **Your own server** ($0)

### Requirements
- SSH access
- Ubuntu/Debian (or similar)
- 2GB RAM minimum
- **No cloud provider API needed!**

---

## 🔑 What You Need

**Just SSH!** No cloud provider credentials.

### Per Environment:

**Staging** (optional - skips if not set):
```bash
STAGING_SSH_PRIVATE_KEY      # SSH key
STAGING_SSH_USER             # Usually "root"
STAGING_SSH_HOST             # Server IP
STAGING_SSH_KNOWN_HOSTS      # Fingerprint
```

**Production** (optional - skips if not set):
```bash
PRODUCTION_SSH_PRIVATE_KEY   # SSH key
PRODUCTION_SSH_USER          # Usually "root"
PRODUCTION_SSH_HOST          # Server IP
PRODUCTION_SSH_KNOWN_HOSTS   # Fingerprint
```

### Setup:

```bash
# Generate keys
ssh-keygen -t ed25519 -f ~/.ssh/deploy_staging -N ""
ssh-keygen -t ed25519 -f ~/.ssh/deploy_prod -N ""

# Copy to servers
ssh-copy-id -i ~/.ssh/deploy_staging.pub root@STAGING_IP
ssh-copy-id -i ~/.ssh/deploy_prod.pub root@PROD_IP

# Add to GitHub (staging - note the STAGING_ prefix!)
gh secret set STAGING_SSH_PRIVATE_KEY < ~/.ssh/deploy_staging
gh secret set STAGING_SSH_USER --body "root"
gh secret set STAGING_SSH_HOST --body "STAGING_IP"
gh secret set STAGING_SSH_KNOWN_HOSTS < <(ssh-keyscan STAGING_IP)

# Add to GitHub (production - note the PRODUCTION_ prefix!)
gh secret set PRODUCTION_SSH_PRIVATE_KEY < ~/.ssh/deploy_prod
gh secret set PRODUCTION_SSH_USER --body "root"
gh secret set PRODUCTION_SSH_HOST --body "PROD_IP"
gh secret set PRODUCTION_SSH_KNOWN_HOSTS < <(ssh-keyscan PROD_IP)
```

**⚠️ Secrets are optional** - deployment will skip with a warning if not configured!

---

## ⚡ Daily Workflow

```bash
# Deploy to staging (automatic)
git push origin main

# Deploy to production (automatic)
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

**That's it!** Works with any server.

---

## 💰 Cost Comparison

| Provider | Server Type | Monthly Cost |
|----------|-------------|--------------|
| **Hetzner** | CPX11 | €4.51 |
| **DigitalOcean** | Basic Droplet | $12 |
| **AWS** | t3.small | ~$15 |
| **Linode** | Nanode | $12 |
| **Your server** | Any | $0 |

**For 2 environments** (staging + production):
- Hetzner: ~€9/month
- Your hardware: Free!

---

## 📚 Quick Reference

### Automatic Deployments
```bash
git push origin main          # → Staging
git tag v1.0.0 && git push origin v1.0.0  # → Production
```

### Manual Deployment
```bash
make deploy-package
scp nimsforest-deploy.tar.gz root@SERVER:/tmp/
ssh root@SERVER 'bash -s' < scripts/deploy.sh deploy
```

### Common Commands
```bash
ssh root@SERVER "sudo systemctl status nimsforest"
ssh root@SERVER "sudo journalctl -u nimsforest -f"
ssh root@SERVER 'bash -s' < scripts/deploy.sh rollback
```

---

## 📖 Documentation

**Start here**: [DEPLOYMENT_QUICKREF.md](./deployment/DEPLOYMENT_QUICKREF.md)  
**Complete guide**: [DEPLOYMENT_SSH.md](./deployment/DEPLOYMENT_SSH.md)  
**What changed**: [DEPLOYMENT_CHANGES.md](./deployment/DEPLOYMENT_CHANGES.md)

---

## ✅ Key Points

1. **Platform-agnostic** - Works with ANY SSH-accessible Linux server
2. **No API tokens** - Just SSH keys
3. **Automatic staging** - Push to main
4. **Automatic production** - Create release
5. **Simple docs** - 67% reduction in documentation
6. **Cost-effective** - Use any provider or own hardware

---

## 🎯 Next Steps

1. Choose your server(s) - any cloud provider or own hardware
2. Run setup script on each server
3. Configure GitHub secrets (just SSH keys)
4. Push to main to test staging
5. Create release to test production

See [DEPLOYMENT_SSH.md](./deployment/DEPLOYMENT_SSH.md) for detailed instructions.

---

**Happy Deploying to Any Server! 🚀**
