# Deployment Quick Reference

## Automatic Deployments

```bash
# Staging (automatic)
git push origin main
# → Deploys to staging server via SSH

# Production (automatic)  
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
# → Deploys to production server via SSH
```

## Platform-Agnostic!

Works with **any Linux server** you can SSH into:
- Hetzner, DigitalOcean, AWS, Linode, Vultr
- Your own bare metal / VM / VPS
- Literally any Ubuntu/Debian server with SSH

**No cloud provider API needed** - just SSH!

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

## Make Commands

```bash
make build-deploy      # Build deployment binary
make deploy-package    # Create deployment package
make deploy-verify     # Verify deployment files
make help              # Show all commands
```

## Setup Checklist

- [ ] Create Linux servers (any cloud provider or own hardware)
- [ ] Run setup script on each server
- [ ] Configure GitHub environments (staging, production)
- [ ] Add SSH secrets to each environment
- [ ] Test staging deployment (push to main)
- [ ] Test production deployment (create release)

## GitHub Secrets (per environment)

**Generic names - works with any provider!**

```bash
SSH_PRIVATE_KEY      # SSH private key for deployment
SSH_USER             # Usually "root" or your user
SSH_HOST             # Server IP or hostname
SSH_KNOWN_HOSTS      # SSH fingerprint (from ssh-keyscan)
```

### Setting Secrets

```bash
# Generate keys
ssh-keygen -t ed25519 -f ~/.ssh/deploy_staging -N ""
ssh-keygen -t ed25519 -f ~/.ssh/deploy_prod -N ""

# Add to servers
ssh-copy-id -i ~/.ssh/deploy_staging.pub root@STAGING_IP
ssh-copy-id -i ~/.ssh/deploy_prod.pub root@PROD_IP

# Add to GitHub (staging)
gh secret set SSH_PRIVATE_KEY --env staging < ~/.ssh/deploy_staging
gh secret set SSH_USER --env staging --body "root"
gh secret set SSH_HOST --env staging --body "STAGING_IP"
gh secret set SSH_KNOWN_HOSTS --env staging < <(ssh-keyscan STAGING_IP)

# Add to GitHub (production)
gh secret set SSH_PRIVATE_KEY --env production < ~/.ssh/deploy_prod
gh secret set SSH_USER --env production --body "root"
gh secret set SSH_HOST --env production --body "PROD_IP"
gh secret set SSH_KNOWN_HOSTS --env production < <(ssh-keyscan PROD_IP)
```

## Server Providers

### Hetzner Cloud (Recommended - Best Price)
```bash
hcloud server create --name nimsforest-staging --type cpx11 --image ubuntu-22-04
# €4.51/month
```

### DigitalOcean
```bash
doctl compute droplet create nimsforest-staging --size s-1vcpu-2gb --image ubuntu-22-04-x64
# $12/month
```

### AWS EC2
```bash
aws ec2 run-instances --image-id ami-0c55b159cbfafe1f0 --instance-type t3.small
# ~$15/month
```

### Your Own Server
```bash
# Just point SSH to your server!
# No special setup needed
```

## Troubleshooting

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
gh secret list --env staging
```

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

## Documentation

- **[DEPLOYMENT_SSH.md](./DEPLOYMENT_SSH.md)** - Complete setup guide
- **[DEPLOYMENT_CHANGES.md](./DEPLOYMENT_CHANGES.md)** - What changed
- **[README.md](./README.md)** - Project overview
- **[Makefile](./Makefile)** - All Make targets

## Key Points

✅ **Platform-agnostic** - Works with any SSH-accessible Linux server  
✅ **No API tokens needed** - Just SSH keys  
✅ **Automatic staging** - Push to main  
✅ **Automatic production** - Create release  
✅ **Cost-effective** - Use any provider or own hardware  
✅ **Simple** - Just SSH and Make commands

---

**Quick help**: See [DEPLOYMENT_SSH.md](./DEPLOYMENT_SSH.md) for detailed instructions
