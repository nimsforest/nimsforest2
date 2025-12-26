# Staging Setup Checklist

Use this checklist to track your progress setting up the staging environment.

## ☐ Step 1: Hetzner Account & Server (10 min)

- [ ] Create Hetzner account at https://console.hetzner.cloud/
- [ ] Create new project: `nimsforest`
- [ ] Create server with:
  - [ ] Location: (choose nearest)
  - [ ] Image: Ubuntu 22.04
  - [ ] Type: CPX11 (2 vCPU, 2GB RAM)
  - [ ] Name: `nimsforest-staging`
- [ ] Add SSH key to server
- [ ] Server created successfully
- [ ] **Copy server IP:** `___________________`

## ☐ Step 2: Server Setup (5 min)

- [ ] SSH to server: `ssh root@YOUR_SERVER_IP`
- [ ] Download setup script:
  ```bash
  wget https://raw.githubusercontent.com/YOUR_USERNAME/nimsforest/main/scripts/setup-server.sh
  ```
- [ ] Make executable: `chmod +x setup-server.sh`
- [ ] Run setup: `sudo ./setup-server.sh`
- [ ] Verify NATS is running: `sudo systemctl status nats`
- [ ] Verify NATS monitoring: `curl http://localhost:8222/varz`
- [ ] Exit server: `exit`

## ☐ Step 3: SSH Keys for Deployment (5 min)

On your local machine:

- [ ] Generate deployment key:
  ```bash
  ssh-keygen -t ed25519 -C "github-actions-staging" -f ~/.ssh/nimsforest_staging_deploy
  ```
- [ ] Copy public key to server:
  ```bash
  ssh-copy-id -i ~/.ssh/nimsforest_staging_deploy.pub root@YOUR_SERVER_IP
  ```
- [ ] Test connection:
  ```bash
  ssh -i ~/.ssh/nimsforest_staging_deploy root@YOUR_SERVER_IP "echo 'Works!'"
  ```
- [ ] Get server fingerprint:
  ```bash
  ssh-keyscan YOUR_SERVER_IP > /tmp/staging_known_hosts
  ```

## ☐ Step 4: GitHub Secrets (5 min)

Choose one method:

### Method A: GitHub CLI (Recommended)

- [ ] Install GitHub CLI: `brew install gh` (or visit https://cli.github.com/)
- [ ] Login: `gh auth login`
- [ ] Navigate to project: `cd /path/to/nimsforest`
- [ ] Set secrets:
  ```bash
  gh secret set STAGING_SSH_PRIVATE_KEY < ~/.ssh/nimsforest_staging_deploy
  gh secret set STAGING_SSH_USER --body "root"
  gh secret set STAGING_SSH_HOST --body "YOUR_SERVER_IP"
  gh secret set STAGING_SSH_KNOWN_HOSTS < /tmp/staging_known_hosts
  ```
- [ ] Verify: `gh secret list`

### Method B: GitHub Web UI

- [ ] Go to repository → Settings → Secrets and variables → Actions
- [ ] Add secret: `STAGING_SSH_PRIVATE_KEY`
  - Value: `cat ~/.ssh/nimsforest_staging_deploy` (entire content)
- [ ] Add secret: `STAGING_SSH_USER`
  - Value: `root`
- [ ] Add secret: `STAGING_SSH_HOST`
  - Value: `YOUR_SERVER_IP`
- [ ] Add secret: `STAGING_SSH_KNOWN_HOSTS`
  - Value: `cat /tmp/staging_known_hosts`

## ☐ Step 5: Test Deployment (5 min)

- [ ] Commit changes:
  ```bash
  git add .
  git commit -m "feat: configure staging environment"
  git push origin main
  ```
- [ ] Watch deployment:
  - [ ] GitHub: Repository → Actions tab
  - [ ] OR: `gh run watch`
- [ ] Deployment succeeded: ✅
- [ ] Verify on server:
  ```bash
  ssh -i ~/.ssh/nimsforest_staging_deploy root@YOUR_SERVER_IP
  sudo systemctl status nimsforest
  sudo journalctl -u nimsforest -n 20
  ```

## ☐ Step 6: Verification

- [ ] Service is running on server
- [ ] NATS is running on server
- [ ] Automatic deployments working on push to main
- [ ] Server IP saved in password manager/notes
- [ ] SSH key backed up securely

---

## 🎉 Staging Complete!

Your staging environment is fully operational!

### Quick Commands Reference

```bash
# Deploy to staging
git push origin main

# Watch deployment
gh run watch

# SSH to staging
ssh -i ~/.ssh/nimsforest_staging_deploy root@YOUR_SERVER_IP

# Check service
ssh root@YOUR_SERVER_IP "sudo systemctl status nimsforest"

# View logs
ssh root@YOUR_SERVER_IP "sudo journalctl -u nimsforest -f"

# Manual deployment
gh workflow run deploy.yml --ref main -f environment=staging
```

---

## 📝 Server Details

Keep this information in a safe place:

```
Server Name: nimsforest-staging
Provider: Hetzner Cloud
IP Address: _____________________
SSH User: root
SSH Key: ~/.ssh/nimsforest_staging_deploy
Location: _____________________
Server Type: CPX11 (2 vCPU, 2GB RAM)
Cost: ~€4.51/month
```

---

## Next Steps

- [ ] Set up monitoring (optional)
- [ ] Set up backups (optional)
- [ ] Configure custom domain (optional)
- [ ] Set up production environment (when ready)
- [ ] Document any custom configurations
