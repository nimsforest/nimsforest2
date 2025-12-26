# Staging Setup Flow

## Visual Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    STAGING SETUP PROCESS                        │
└─────────────────────────────────────────────────────────────────┘


STEP 1: CREATE HETZNER SERVER (5 min)
═══════════════════════════════════════════════════════════════════

    🌐 https://console.hetzner.cloud/
         │
         ├─→ Create Project: "nimsforest"
         ├─→ Add Server
         │    ├─ Location: Nearest to you
         │    ├─ Image: Ubuntu 22.04
         │    ├─ Type: CPX11 (2 vCPU, 2GB RAM)
         │    ├─ Name: nimsforest-staging
         │    └─ SSH Key: Add your public key (optional)
         │
         └─→ ✅ Copy Server IP: XXX.XXX.XXX.XXX


STEP 2: SETUP SERVER SOFTWARE (3-5 min)
═══════════════════════════════════════════════════════════════════

    💻 Your Local Machine
         │
         │  ssh root@SERVER_IP
         ├────────────────────────────────────────┐
         │                                        │
         │                                        ↓
         │                         🖥️  Hetzner Server (Ubuntu 22.04)
         │                                │
         │  wget setup-server.sh          │
         ├───────────────────────────────→│
         │                                │
         │  chmod +x setup-server.sh      │
         ├───────────────────────────────→│
         │                                │
         │  sudo ./setup-server.sh        │
         ├───────────────────────────────→│
         │                                │
         │                                ├─→ Update packages
         │                                ├─→ Install Go 1.24.0
         │                                ├─→ Install NATS Server
         │                                ├─→ Configure firewall (UFW)
         │                                ├─→ Setup fail2ban
         │                                ├─→ Start NATS service
         │                                ├─→ Create directories
         │                                └─→ Setup log rotation
         │                                │
         │  ✅ Setup complete!             │
         ├────────────────────────────────┤
         │                                │
         │  exit                          │
         └────────────────────────────────┘


STEP 3: CONFIGURE LOCAL DEPLOYMENT (1 min - AUTOMATED!)
═══════════════════════════════════════════════════════════════════

    💻 Your Local Machine
         │
         │  ./scripts/setup-staging-local.sh SERVER_IP
         ├───────────────────────────────────────────────────┐
         │                                                   │
         │  1. Generate SSH Keys                             │
         ├─→ ssh-keygen -t ed25519                          │
         │   ~/.ssh/nimsforest_staging_deploy                │
         │   ~/.ssh/nimsforest_staging_deploy.pub            │
         │                                                   │
         │  2. Copy Public Key to Server                     │
         ├─→ ssh-copy-id root@SERVER_IP ───────────────────→│ 🖥️  Server
         │                                                   │  ✅ Key added
         │                                                   │
         │  3. Get Server SSH Fingerprint                    │
         ├─→ ssh-keyscan SERVER_IP                          │
         │   /tmp/staging_known_hosts                        │
         │                                                   │
         │  4. Configure GitHub Secrets                      │
         ├─→ gh secret set STAGING_SSH_PRIVATE_KEY         │
         ├─→ gh secret set STAGING_SSH_USER                │
         ├─→ gh secret set STAGING_SSH_HOST                │
         └─→ gh secret set STAGING_SSH_KNOWN_HOSTS         │
              │
              ↓
         🔐 GitHub Repository
              ├─ STAGING_SSH_PRIVATE_KEY    ✅
              ├─ STAGING_SSH_USER            ✅
              ├─ STAGING_SSH_HOST            ✅
              └─ STAGING_SSH_KNOWN_HOSTS     ✅


STEP 4: DEPLOY! (2 min)
═══════════════════════════════════════════════════════════════════

    💻 Your Local Machine
         │
         │  git push origin main
         ├─────────────────────────────────┐
         │                                 │
         │                                 ↓
         │                        🔄 GitHub Actions
         │                                 │
         │                                 ├─→ Checkout code
         │                                 ├─→ Setup Go
         │                                 ├─→ Build binary (Linux)
         │                                 ├─→ Create package
         │                                 │
         │                                 │  SSH Deployment
         │                                 ├──────────────────┐
         │                                 │                  │
         │                                 │                  ↓
         │                                 │         🖥️  Hetzner Server
         │                                 │                  │
         │                                 │  scp package     │
         │                                 ├─────────────────→│
         │                                 │                  │
         │                                 │  ssh deploy      │
         │                                 ├─────────────────→│
         │                                 │                  ├─→ Stop service
         │                                 │                  ├─→ Backup binary
         │                                 │                  ├─→ Install new binary
         │                                 │                  ├─→ Start service
         │                                 │                  └─→ Verify running
         │                                 │                  │
         │                                 │  ✅ Deployed!     │
         │                                 ├──────────────────┘
         │                                 │
         │  ✅ Deployment successful!      │
         └─────────────────────────────────┘


VERIFICATION
═══════════════════════════════════════════════════════════════════

    💻 Your Local Machine
         │
         │  gh run watch
         ├─→ Watch deployment progress
         │
         │  ssh root@SERVER_IP
         ├────────────────────────────────┐
         │                                │
         │                                ↓
         │                       🖥️  Hetzner Server
         │                                │
         │  sudo systemctl status nimsforest
         ├───────────────────────────────→│
         │  ✅ Active: active (running)   │
         │                                │
         │  sudo journalctl -u nimsforest -f
         ├───────────────────────────────→│
         │  📝 Live logs streaming...     │
         └────────────────────────────────┘


ONGOING DEPLOYMENT FLOW
═══════════════════════════════════════════════════════════════════

    💻 Developer
         │
         │  git commit -m "feat: new feature"
         │  git push origin main
         │
         ↓
    🔄 GitHub Actions (Automatic)
         │
         ├─→ Run tests
         ├─→ Build binary
         ├─→ Deploy to staging
         │
         ↓
    🖥️  Staging Server
         │
         ├─→ Service updated
         └─→ Running new version
         
    ✅ Deploy complete! (~2 minutes)


ARCHITECTURE AFTER SETUP
═══════════════════════════════════════════════════════════════════

    💻 Local Development
         ↕ git push
    🔄 GitHub Actions
         ↕ SSH
    🖥️  Hetzner Server (€4.51/mo)
         ├─ NATS Server (JetStream)
         ├─ NimsForest Service (systemd)
         ├─ Firewall (UFW)
         ├─ Security (fail2ban)
         └─ Logs (journald)


TIME BREAKDOWN
═══════════════════════════════════════════════════════════════════

    Step 1: Create server         → 5 min  (via web console)
    Step 2: Setup server           → 3-5 min (automated script)
    Step 3: Configure deployment   → 1 min  (automated script)
    Step 4: First deployment       → 2 min  (automatic)
    
    Total: ~15 minutes to full production-ready staging!


WHAT YOU GET
═══════════════════════════════════════════════════════════════════

    ✅ Secure Linux server
    ✅ Go + NATS infrastructure
    ✅ Automatic deployments
    ✅ Service monitoring
    ✅ Professional DevOps workflow
    ✅ Cost: ~€5/month
```

---

## Quick Commands After Setup

### Deploy
```bash
git push origin main          # Deploy to staging
gh run watch                  # Watch deployment
```

### Monitor
```bash
gh run list                   # List deployments
ssh root@SERVER "sudo systemctl status nimsforest"
ssh root@SERVER "sudo journalctl -u nimsforest -f"
```

### Troubleshoot
```bash
gh run view --log             # View deployment logs
ssh root@SERVER               # SSH to server
sudo systemctl restart nimsforest  # Restart service
```

---

## Next: Set Up Production

Repeat the entire flow with:
- Server name: `nimsforest-production`
- Secrets prefix: `PRODUCTION_*` instead of `STAGING_*`
- Deploy trigger: Release (not push to main)

```bash
git tag -a v1.0.0 -m "Release 1.0.0"
git push origin v1.0.0
# → Automatically deploys to production
```

---

**🌲 Ready to start? Open [HETZNER_QUICKSTART.md](./HETZNER_QUICKSTART.md)!**
