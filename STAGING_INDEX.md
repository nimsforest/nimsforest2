# Staging Environment Setup - Documentation Index

## 🚀 Start Here

Choose your preferred approach:

### For the Impatient (5 min)
👉 **[HETZNER_QUICKSTART.md](./HETZNER_QUICKSTART.md)** - Just give me the commands!

### For First-Timers (15 min)
👉 **[STAGING_SETUP_GUIDE.md](./STAGING_SETUP_GUIDE.md)** - Walk me through everything

### For Organized People (20 min)
👉 **[STAGING_SETUP_CHECKLIST.md](./STAGING_SETUP_CHECKLIST.md)** - Let me track my progress

---

## 📚 Complete Documentation

| Document | Purpose | Time | Audience |
|----------|---------|------|----------|
| **[HETZNER_QUICKSTART.md](./HETZNER_QUICKSTART.md)** | TL;DR version with just the commands | 5 min | Experienced users |
| **[STAGING_SETUP_GUIDE.md](./STAGING_SETUP_GUIDE.md)** | Complete step-by-step guide with explanations | 15 min | Everyone |
| **[STAGING_SETUP_CHECKLIST.md](./STAGING_SETUP_CHECKLIST.md)** | Interactive checklist to track progress | 20 min | First-timers |
| **[SETUP_SUMMARY.md](./SETUP_SUMMARY.md)** | Overview, commands reference, and next steps | 5 min | Quick reference |
| **[STAGING_SETUP_FLOW.md](./STAGING_SETUP_FLOW.md)** | Visual flowchart of the entire process | 2 min | Visual learners |
| **[STAGING_QUICK_REFERENCE.txt](./STAGING_QUICK_REFERENCE.txt)** | One-page quick reference card | 1 min | Print-friendly |
| **[WHERE_TO_RUN_COMMANDS.md](./WHERE_TO_RUN_COMMANDS.md)** | Where to run local vs server commands | 3 min | Beginners |
| **[PRIVATE_REPO_SETUP.md](./PRIVATE_REPO_SETUP.md)** | Setting up with private repositories | 5 min | Private repos |

---

## 🛠️ Scripts

| Script | Purpose | Run On |
|--------|---------|--------|
| **[scripts/setup-server.sh](./scripts/setup-server.sh)** | Install all server software | Server |
| **[scripts/setup-staging-local.sh](./scripts/setup-staging-local.sh)** | Configure local deployment (automated!) | Local machine |

---

## 📖 Related Documentation

### Deployment Docs (Already Existing)
- **[DEPLOYMENT_SSH.md](./DEPLOYMENT_SSH.md)** - Platform-agnostic SSH deployment
- **[CI_CD_SETUP.md](./CI_CD_SETUP.md)** - CI/CD pipeline documentation
- **[CI_CD.md](./CI_CD.md)** - Detailed CI/CD workflows

### Project Docs
- **[README.md](./README.md)** - Main project documentation (updated with staging setup)
- **[INDEX.md](./INDEX.md)** - Complete documentation index

---

## 🎯 Quick Decision Guide

**"I just want to deploy ASAP!"**
→ [HETZNER_QUICKSTART.md](./HETZNER_QUICKSTART.md)

**"I've never set up a server before"**
→ [STAGING_SETUP_GUIDE.md](./STAGING_SETUP_GUIDE.md)

**"I want to check off tasks as I go"**
→ [STAGING_SETUP_CHECKLIST.md](./STAGING_SETUP_CHECKLIST.md)

**"Show me what the process looks like"**
→ [STAGING_SETUP_FLOW.md](./STAGING_SETUP_FLOW.md)

**"I need a command reference"**
→ [SETUP_SUMMARY.md](./SETUP_SUMMARY.md) or [STAGING_QUICK_REFERENCE.txt](./STAGING_QUICK_REFERENCE.txt)

**"I'm using a different cloud provider"**
→ [DEPLOYMENT_SSH.md](./DEPLOYMENT_SSH.md)

**"My repository is private"**
→ [PRIVATE_REPO_SETUP.md](./PRIVATE_REPO_SETUP.md)

**"Where do I run these commands?"**
→ [WHERE_TO_RUN_COMMANDS.md](./WHERE_TO_RUN_COMMANDS.md)

**"Is this deployment method common? Can't I deploy directly from GitHub?"**
→ [DEPLOYMENT_METHODS_EXPLAINED.md](./DEPLOYMENT_METHODS_EXPLAINED.md) - You already do!

---

## ⚡ TL;DR - The Absolute Minimum

```bash
# 1. Create Hetzner server (web console)
#    https://console.hetzner.cloud/
#    Ubuntu 22.04, CPX11, copy IP

# 2. Setup server (for PRIVATE repos - use SCP)
scp scripts/setup-server.sh root@YOUR_IP:/tmp/
ssh root@YOUR_IP "cd /tmp && chmod +x setup-server.sh && sudo ./setup-server.sh"

# For PUBLIC repos, use wget:
# ssh root@YOUR_IP "wget https://raw.githubusercontent.com/USER/nimsforest/main/scripts/setup-server.sh && chmod +x setup-server.sh && sudo ./setup-server.sh"

# 3. Configure deployment (one command!)
./scripts/setup-staging-local.sh YOUR_IP

# 4. Deploy
git push origin main
```

---

## 📋 What You Need

### Before Starting
- [ ] Hetzner account (or other cloud provider)
- [ ] GitHub CLI (`gh`) installed
- [ ] SSH client (standard on Mac/Linux)
- [ ] Git installed
- [ ] 15 minutes of time
- [ ] €5/month budget (or use your own server for free)

### What You'll Create
- [x] Production-ready staging server
- [x] Automatic deployment pipeline
- [x] GitHub secrets configured
- [x] Monitoring and logging setup
- [x] Security hardening complete

---

## 🌟 Highlights

### One-Command Setup
Our automated script does everything:
```bash
./scripts/setup-staging-local.sh YOUR_IP
```

### Automatic Deployment
Once configured:
```bash
git push origin main  # Deploys automatically!
```

### Works Everywhere
Not just Hetzner:
- ✅ Hetzner Cloud (~€5/month)
- ✅ DigitalOcean (~$12/month)
- ✅ AWS EC2 / Lightsail
- ✅ Your own hardware ($0)
- ✅ Any Linux server with SSH

### Production-Ready
- ✅ Go + NATS infrastructure
- ✅ Firewall configured
- ✅ Automatic security updates
- ✅ Service monitoring
- ✅ Structured logging
- ✅ Rollback on failure

---

## 🎓 Learning Path

### Beginner
1. Read: [STAGING_SETUP_FLOW.md](./STAGING_SETUP_FLOW.md) - Understand the process
2. Follow: [STAGING_SETUP_GUIDE.md](./STAGING_SETUP_GUIDE.md) - Step by step
3. Use: [STAGING_SETUP_CHECKLIST.md](./STAGING_SETUP_CHECKLIST.md) - Track progress

### Intermediate
1. Read: [HETZNER_QUICKSTART.md](./HETZNER_QUICKSTART.md) - Get the commands
2. Run: `./scripts/setup-staging-local.sh YOUR_IP`
3. Deploy: `git push origin main`

### Advanced
1. Review: [DEPLOYMENT_SSH.md](./DEPLOYMENT_SSH.md) - Platform-agnostic docs
2. Customize: Adapt scripts for your infrastructure
3. Scale: Set up production, load balancers, monitoring

---

## 🔍 Troubleshooting

### Quick Fixes

**"Script fails"**
→ See troubleshooting in [STAGING_SETUP_GUIDE.md](./STAGING_SETUP_GUIDE.md#troubleshooting)

**"Deployment skipped"**
→ Check secrets: `gh secret list | grep STAGING`

**"Service won't start"**
→ Check NATS: `ssh root@SERVER "sudo systemctl status nats"`

### Full Troubleshooting Guides
- [STAGING_SETUP_GUIDE.md § Troubleshooting](./STAGING_SETUP_GUIDE.md#troubleshooting)
- [HETZNER_QUICKSTART.md § Troubleshooting](./HETZNER_QUICKSTART.md#troubleshooting)
- [SETUP_SUMMARY.md § Troubleshooting](./SETUP_SUMMARY.md#troubleshooting)

---

## 💬 Support

- **Documentation:** You're looking at it!
- **Issues:** Open a GitHub issue
- **Hetzner Docs:** https://docs.hetzner.com/cloud/
- **GitHub Actions:** https://docs.github.com/actions
- **NATS Docs:** https://docs.nats.io/

---

## 🎯 Next Steps After Staging

1. **Set up Production**
   - Follow same process with `PRODUCTION_*` secrets
   - Deploy via releases instead of pushes

2. **Add Monitoring**
   - UptimeRobot (uptime monitoring)
   - Netdata (server monitoring)
   - Better Stack (log aggregation)

3. **Configure Custom Domain**
   - Buy domain
   - Add DNS records
   - Update secrets with domain name

4. **Enable Backups**
   - Via Hetzner (automated)
   - Via custom script (free)

5. **Scale**
   - Add more servers
   - Set up load balancer
   - Enable auto-scaling

---

## 📊 Success Metrics

After setup, you should have:
- ✅ Server running Ubuntu 22.04 with Go + NATS
- ✅ Four GitHub secrets configured
- ✅ Automatic deployment on push to main
- ✅ Service running and accessible
- ✅ Logs viewable via journalctl
- ✅ Cost: ~€5/month (or $0 with own server)

---

## 🏆 Achievement Unlocked

Once complete, you'll have:
- 🎖️ Production-ready infrastructure
- 🎖️ Modern DevOps workflow
- 🎖️ Continuous deployment pipeline
- 🎖️ Professional monitoring setup
- 🎖️ Cost-effective hosting
- 🎖️ Scalable architecture

---

**🚀 Ready to start?**

Open [HETZNER_QUICKSTART.md](./HETZNER_QUICKSTART.md) and let's go!

---

## 📝 Document Versions

| Document | Size | Purpose |
|----------|------|---------|
| HETZNER_QUICKSTART.md | 7.0K | Quick start (5 min) |
| STAGING_SETUP_GUIDE.md | 13K | Complete guide (15 min) |
| STAGING_SETUP_CHECKLIST.md | 4.2K | Interactive checklist (20 min) |
| SETUP_SUMMARY.md | 12K | Reference & commands |
| STAGING_SETUP_FLOW.md | 9.8K | Visual flowchart |
| STAGING_QUICK_REFERENCE.txt | 3.4K | One-page reference |
| scripts/setup-staging-local.sh | 7.9K | Automated setup script |

**Total documentation:** ~57KB of comprehensive guides!

---

🌲 **NimsForest** - Event-Driven Organizational Orchestration
