# Staging Environment Setup - Summary

## What Was Created

I've created a complete staging environment setup for you to deploy NimsForest on Hetzner Cloud (or any other provider).

### New Files Created

1. **[HETZNER_QUICKSTART.md](./HETZNER_QUICKSTART.md)**
   - Fastest way to get started (TL;DR version)
   - Quick reference for common commands
   - Troubleshooting tips
   
2. **[STAGING_SETUP_GUIDE.md](./STAGING_SETUP_GUIDE.md)**
   - Comprehensive step-by-step guide
   - Detailed instructions with explanations
   - Security best practices
   - Cost breakdown
   - Monitoring and backup setup
   - Next steps (production, custom domain, etc.)

3. **[STAGING_SETUP_CHECKLIST.md](./STAGING_SETUP_CHECKLIST.md)**
   - Interactive checklist format
   - Track your progress as you go
   - Quick command reference
   - Server details template

4. **[scripts/setup-staging-local.sh](./scripts/setup-staging-local.sh)**
   - Automated local configuration script
   - Generates SSH keys
   - Copies keys to server
   - Configures GitHub secrets
   - Verifies everything works

5. **Updated [README.md](./README.md)**
   - Added quick start section for staging setup
   - Links to all new guides

### Existing Files (Already Available)

- **[scripts/setup-server.sh](./scripts/setup-server.sh)** - Server setup script (runs on server)
- **[.github/workflows/deploy.yml](./.github/workflows/deploy.yml)** - Deployment workflow
- **[DEPLOYMENT_SSH.md](./DEPLOYMENT_SSH.md)** - Platform-agnostic deployment docs

---

## Quick Start Guide

### The Fastest Way (5 Minutes)

```bash
# Step 1: Create Hetzner server
# Go to: https://console.hetzner.cloud/
# Create: Ubuntu 22.04, CPX11, name it "nimsforest-staging"
# Copy the IP address

# Step 2: Setup server
ssh root@YOUR_SERVER_IP
wget https://raw.githubusercontent.com/YOUR_USERNAME/nimsforest/main/scripts/setup-server.sh
chmod +x setup-server.sh && sudo ./setup-server.sh
# Wait 3-5 minutes, then exit
exit

# Step 3: Configure local deployment (automated!)
./scripts/setup-staging-local.sh YOUR_SERVER_IP

# Step 4: Deploy!
git push origin main
gh run watch
```

### What Each Step Does

**Step 1: Create Server**
- Creates a virtual server on Hetzner (~€4.51/month)
- Ubuntu 22.04 with 2GB RAM, 2 vCPU, 40GB disk

**Step 2: Setup Server**
- Installs Go, NATS Server, and all dependencies
- Configures firewall and security
- Starts NATS with JetStream
- Sets up directories and logging

**Step 3: Configure Local**
- Generates SSH keys for deployment
- Copies public key to server
- Gets server SSH fingerprint
- Configures all 4 GitHub secrets automatically

**Step 4: Deploy**
- Pushes code to GitHub
- Triggers deployment workflow
- Builds application
- Deploys to staging server
- Starts systemd service

---

## GitHub Secrets Required

The setup script will configure these automatically:

| Secret | Description |
|--------|-------------|
| `STAGING_SSH_PRIVATE_KEY` | SSH private key for deployment |
| `STAGING_SSH_USER` | SSH user (usually `root`) |
| `STAGING_SSH_HOST` | Server IP or hostname |
| `STAGING_SSH_KNOWN_HOSTS` | Server SSH fingerprint |

For production, use the same pattern with `PRODUCTION_` prefix.

---

## Automatic Deployment

Once configured, deployment is automatic:

- **Push to `main`** → Deploys to staging
- **Create release** → Deploys to production
- **Manual trigger** → Deploy via GitHub Actions UI

---

## Cost Breakdown

### Hetzner Cloud

| Item | Cost |
|------|------|
| Staging Server (CPX11) | €4.51/month |
| Production Server (CPX11) | €4.51/month |
| Backups (optional) | ~€0.02/month |
| **Total for both environments** | **~€9/month** |

### Free Alternatives

The setup works with any Linux server via SSH:
- Your own hardware (cost: $0)
- DigitalOcean ($12/month for 2GB)
- AWS Lightsail (~$10/month)
- Linode ($12/month)
- Any VPS provider

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                  Your Local Machine                      │
│  ┌────────────────────────────────────────────────────┐ │
│  │  git push origin main                              │ │
│  └────────────────────┬───────────────────────────────┘ │
└─────────────────────────┼───────────────────────────────┘
                          │
                          ↓
┌─────────────────────────────────────────────────────────┐
│                    GitHub Actions                        │
│  ┌────────────────────────────────────────────────────┐ │
│  │  1. Build Go binary (Linux)                        │ │
│  │  2. Create deployment package                      │ │
│  │  3. Copy to server via SCP                         │ │
│  │  4. Deploy via SSH                                 │ │
│  └────────────────────┬───────────────────────────────┘ │
└─────────────────────────┼───────────────────────────────┘
                          │ SSH
                          ↓
┌─────────────────────────────────────────────────────────┐
│               Hetzner Server (Staging)                   │
│  ┌────────────────────────────────────────────────────┐ │
│  │  NATS Server (JetStream)     :4222, :8222         │ │
│  │  NimsForest Service          systemd               │ │
│  │  Firewall                    UFW                   │ │
│  │  Security                    fail2ban              │ │
│  │  Logs                        journald              │ │
│  └────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

---

## Next Steps After Setup

### 1. Monitor Your Deployment

```bash
# Watch deployment in real-time
gh run watch

# Check service status
ssh -i ~/.ssh/nimsforest_staging_deploy root@YOUR_SERVER_IP \
  "sudo systemctl status nimsforest"

# View logs
ssh -i ~/.ssh/nimsforest_staging_deploy root@YOUR_SERVER_IP \
  "sudo journalctl -u nimsforest -f"
```

### 2. Set Up Production

Repeat the process for production:
- Create another server named `nimsforest-production`
- Use `PRODUCTION_` prefix for secrets
- Deploy via releases instead of pushes

### 3. Add Custom Domain

```bash
# Buy domain and add DNS A records:
# staging.yourdomain.com → STAGING_IP
# app.yourdomain.com     → PRODUCTION_IP

# Update secrets
gh secret set STAGING_SSH_HOST --body "staging.yourdomain.com"
```

### 4. Set Up Monitoring

Free options:
- [UptimeRobot](https://uptimerobot.com/) - uptime monitoring
- [Netdata](https://www.netdata.cloud/) - server monitoring
- [Better Stack](https://betterstack.com/) - log aggregation

### 5. Enable Backups

```bash
# Via Hetzner (automated)
hcloud server enable-backup nimsforest-staging

# Or create custom backup script (see STAGING_SETUP_GUIDE.md)
```

---

## Troubleshooting

### Setup Script Fails

**Problem:** `ssh: connect to host X port 22: Connection refused`

**Solution:**
```bash
# Wait 30 seconds for server to fully boot
sleep 30

# Try again
ssh root@YOUR_SERVER_IP
```

### Deployment Fails

**Problem:** "Secrets not configured"

**Solution:**
```bash
# Verify secrets are set
gh secret list | grep STAGING

# Should show 4 secrets
# If not, re-run:
./scripts/setup-staging-local.sh YOUR_SERVER_IP
```

### Service Won't Start

**Problem:** NimsForest service fails to start

**Solution:**
```bash
# SSH to server
ssh root@YOUR_SERVER_IP

# Check NATS is running
sudo systemctl status nats
sudo systemctl restart nats

# Check NimsForest logs
sudo journalctl -u nimsforest -n 100 --no-pager

# Check binary exists
ls -la /usr/local/bin/forest
```

### More Help

See the troubleshooting sections in:
- [STAGING_SETUP_GUIDE.md](./STAGING_SETUP_GUIDE.md#troubleshooting)
- [HETZNER_QUICKSTART.md](./HETZNER_QUICKSTART.md#troubleshooting)

---

## Commands Reference

### Local Machine

```bash
# Deploy to staging
git push origin main

# Deploy to production
git tag -a v1.0.0 -m "Release 1.0.0"
git push origin v1.0.0

# Watch deployment
gh run watch

# View recent deployments
gh run list --workflow=deploy.yml

# Manual deployment
gh workflow run deploy.yml --ref main -f environment=staging

# SSH to staging
ssh -i ~/.ssh/nimsforest_staging_deploy root@YOUR_SERVER_IP

# SSH to production
ssh -i ~/.ssh/nimsforest_production_deploy root@YOUR_SERVER_IP
```

### On Server

```bash
# Check services
sudo systemctl status nimsforest
sudo systemctl status nats

# View logs
sudo journalctl -u nimsforest -f
sudo journalctl -u nats -f

# Restart services
sudo systemctl restart nimsforest
sudo systemctl restart nats

# Check NATS monitoring
curl http://localhost:8222/varz

# Check resources
free -h
df -h
top
```

---

## Security Checklist

- ✅ Dedicated SSH keys (not personal keys)
- ✅ Key-based authentication only
- ✅ Firewall configured (UFW)
- ✅ fail2ban active
- ✅ Automatic security updates
- ✅ GitHub secrets encrypted
- ✅ NATS not exposed to internet
- ⬜ 2FA enabled on GitHub
- ⬜ 2FA enabled on Hetzner
- ⬜ Regular security audits
- ⬜ Backup strategy implemented

---

## File Structure

```
nimsforest/
├── HETZNER_QUICKSTART.md          # Quick start guide (you are here)
├── STAGING_SETUP_GUIDE.md         # Comprehensive guide
├── STAGING_SETUP_CHECKLIST.md     # Interactive checklist
├── SETUP_SUMMARY.md               # This summary
├── DEPLOYMENT_SSH.md              # Platform-agnostic docs
├── scripts/
│   ├── setup-server.sh            # Server setup (runs on server)
│   └── setup-staging-local.sh     # Local setup (runs locally)
└── .github/
    └── workflows/
        └── deploy.yml             # Deployment workflow
```

---

## Support

- 📖 **Documentation:** All guides in this repository
- 🐛 **Issues:** Open a GitHub issue
- 💬 **Hetzner:** https://docs.hetzner.com/cloud/
- 💬 **GitHub Actions:** https://docs.github.com/actions

---

## What You've Accomplished

After completing this setup, you'll have:

✅ **Production-ready infrastructure**
- Secure Linux server
- NATS messaging system
- Automatic deployments
- Service monitoring

✅ **Modern DevOps practices**
- Infrastructure as Code
- Continuous Deployment
- Automated testing (via CI)
- Version control integration

✅ **Cost-effective hosting**
- ~€5/month for staging
- Scales to production easily
- No vendor lock-in

✅ **Professional workflow**
- Push to deploy
- Automatic rollbacks on failure
- Service health checks
- Structured logging

---

## Ready to Start?

Pick your path:

1. **Fastest (5 min):** Follow [HETZNER_QUICKSTART.md](./HETZNER_QUICKSTART.md)
2. **Guided (15 min):** Follow [STAGING_SETUP_GUIDE.md](./STAGING_SETUP_GUIDE.md)
3. **Track Progress (20 min):** Use [STAGING_SETUP_CHECKLIST.md](./STAGING_SETUP_CHECKLIST.md)

---

**🚀 Let's deploy!**

```bash
./scripts/setup-staging-local.sh YOUR_SERVER_IP
```
