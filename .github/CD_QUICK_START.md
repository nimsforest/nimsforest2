# Continuous Deployment Quick Start

## 🚀 Get Started in 3 Steps

### Step 1: Set Up Your Hetzner Server (10 minutes)

```bash
# 1. Create a Hetzner Cloud server at https://console.hetzner.cloud/
#    - Choose Ubuntu 22.04 or Debian 11+
#    - Select CPX11 or higher (~€4.51/month)
#    - Add your SSH key

# 2. SSH to your new server
ssh root@YOUR_SERVER_IP

# 3. Run the setup script
wget https://raw.githubusercontent.com/yourusername/nimsforest/main/scripts/setup-hetzner-server.sh
chmod +x setup-hetzner-server.sh
sudo ./setup-hetzner-server.sh

# ✅ Server is now ready for deployments!
```

### Step 2: Configure GitHub Secrets (5 minutes)

```bash
# 1. Generate a deployment SSH key
ssh-keygen -t ed25519 -C "github-deploy" -f ~/.ssh/nimsforest_deploy

# 2. Copy the public key to your server
ssh-copy-id -i ~/.ssh/nimsforest_deploy.pub root@YOUR_SERVER_IP

# 3. Get the server's host key
ssh-keyscan YOUR_SERVER_IP > known_hosts

# 4. Add secrets to GitHub (replace YOUR_SERVER_IP)
gh secret set HETZNER_SSH_PRIVATE_KEY < ~/.ssh/nimsforest_deploy
gh secret set HETZNER_SSH_USER --body "root"
gh secret set HETZNER_HOST --body "YOUR_SERVER_IP"
gh secret set HETZNER_KNOWN_HOSTS < known_hosts

# ✅ GitHub is now configured for deployment!
```

**Don't have GitHub CLI?**

Add secrets via web interface:
1. Go to: Repository → Settings → Secrets and variables → Actions
2. Click "New repository secret"
3. Add each secret:
   - `HETZNER_SSH_PRIVATE_KEY`: Content of `~/.ssh/nimsforest_deploy`
   - `HETZNER_SSH_USER`: `root`
   - `HETZNER_HOST`: Your server IP
   - `HETZNER_KNOWN_HOSTS`: Content of `known_hosts` file

### Step 3: Deploy! (1 command)

```bash
# Create and push a release tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# 🎉 Watch it deploy automatically at:
# https://github.com/yourusername/nimsforest/actions
```

## ✨ That's It!

Your application is now deployed and running on Hetzner!

## What Just Happened?

1. ✅ GitHub Actions built your application
2. ✅ Created a release with binaries
3. ✅ Deployed to your Hetzner server
4. ✅ Started the service with systemd
5. ✅ Verified everything is working

## Verify It's Running

```bash
# Check service status
ssh root@YOUR_SERVER_IP "sudo systemctl status nimsforest"

# View live logs
ssh root@YOUR_SERVER_IP "sudo journalctl -u nimsforest -f"

# Check NATS is running
ssh root@YOUR_SERVER_IP "curl http://localhost:8222/varz"
```

## Deploy Again

```bash
# Method 1: Create a new release
git tag -a v1.0.1 -m "Release v1.0.1"
git push origin v1.0.1

# Method 2: Manual deployment via GitHub Actions
gh workflow run deploy-hetzner.yml -f environment=production
```

## Common Commands

```bash
# Restart the service
ssh root@YOUR_SERVER_IP "sudo systemctl restart nimsforest"

# View logs
ssh root@YOUR_SERVER_IP "sudo journalctl -u nimsforest -n 100"

# Check NATS status
ssh root@YOUR_SERVER_IP "sudo systemctl status nats"

# Rollback to previous version
ssh root@YOUR_SERVER_IP 'bash -s' < scripts/deploy.sh rollback

# Verify deployment
ssh root@YOUR_SERVER_IP 'bash -s' < scripts/deploy.sh verify
```

## Using Make Commands

```bash
# Build deployment package locally
make deploy-package

# Deploy to server
scp nimsforest-deploy.tar.gz root@YOUR_SERVER_IP:/tmp/
ssh root@YOUR_SERVER_IP 'bash -s' < scripts/deploy.sh deploy

# Verify all deployment files
make deploy-verify
```

## Need Help?

- **Full Guide**: [HETZNER_DEPLOYMENT.md](../HETZNER_DEPLOYMENT.md)
- **Troubleshooting**: See HETZNER_DEPLOYMENT.md troubleshooting section
- **GitHub Issues**: https://github.com/yourusername/nimsforest/issues

## Next Steps

1. **Set up monitoring**: Use UptimeRobot or similar
2. **Configure alerts**: Get notified of issues
3. **Add staging environment**: Test before production
4. **Set up backups**: Regular NATS data backups
5. **Custom domain**: Point your domain to the server

## Cost Breakdown

| Item | Monthly Cost |
|------|--------------|
| Hetzner CPX11 Server | €4.51 |
| Traffic (20TB included) | €0.00 |
| **Total** | **€4.51** |

**Compare to AWS/Azure**: Save 70-80% on hosting costs! 💰

---

**🎉 Congratulations! You now have continuous deployment!**

Every time you create a release, your application automatically deploys to Hetzner with zero downtime.
