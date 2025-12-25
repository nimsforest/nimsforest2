# Deployment Quick Reference

## Automatic Deployments

```bash
# Staging (automatic)
git push origin main
# → Deploys to staging automatically

# Production (automatic)  
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
# → Deploys to production automatically
```

## Manual Deployment

### Using GitHub CLI
```bash
# Deploy to staging
gh workflow run deploy-hetzner.yml -f environment=staging

# Deploy to production
gh workflow run deploy-hetzner.yml -f environment=production
```

### Using Make
```bash
# Build and package
make deploy-package

# Deploy to server
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

- [ ] Create Hetzner servers (staging + production)
- [ ] Run setup script on each server
- [ ] Configure GitHub environments (staging, production)
- [ ] Add secrets to each environment
- [ ] Test staging deployment (push to main)
- [ ] Test production deployment (create release)

## GitHub Secrets (per environment)

```bash
HETZNER_SSH_PRIVATE_KEY    # SSH private key
HETZNER_SSH_USER           # Usually "root"
HETZNER_HOST               # Server IP address
HETZNER_KNOWN_HOSTS        # SSH fingerprint
```

## Troubleshooting

**Deployment fails**: Check GitHub Actions logs
```bash
gh run list --workflow=deploy-hetzner.yml
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

## Cost

| Item | Monthly Cost |
|------|--------------|
| Staging (CPX11) | €4.51 |
| Production (CPX11) | €4.51 |
| **Total** | **€9.02** |

Scale up production as needed (CPX21: €9.04, CPX31: €16.68)

## Documentation

- **[HETZNER_DEPLOYMENT.md](./HETZNER_DEPLOYMENT.md)** - Complete setup guide
- **[README.md](./README.md)** - Project overview
- **[Makefile](./Makefile)** - All Make targets

---

**Quick help**: See [HETZNER_DEPLOYMENT.md](./HETZNER_DEPLOYMENT.md) for detailed instructions
