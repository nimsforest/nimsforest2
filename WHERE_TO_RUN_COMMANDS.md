# Where to Run Which Commands - Quick Reference

## 🚨 IMPORTANT: Know Where You Are!

During setup, you'll run commands in **two different places**. Here's a clear breakdown:

---

## 💻 ON YOUR LOCAL MACHINE (Laptop/Desktop)

**This is where you cloned the repository**

### When to use:
- Before connecting to server
- When running `scp`, `git`, `gh` commands
- When running the setup-staging-local.sh script

### Commands that run locally:

```bash
# Navigate to your project
cd ~/projects/nimsforest  # or wherever you cloned it

# Copy file to server (runs on local, transfers to remote)
scp scripts/setup-server.sh root@SERVER_IP:/tmp/

# SSH to server (connects you to remote)
ssh root@SERVER_IP

# Configure GitHub secrets (runs locally)
./scripts/setup-staging-local.sh SERVER_IP

# Git operations (run locally)
git push origin main
git commit -m "message"

# GitHub CLI (runs locally)
gh run watch
gh secret set NAME --body "value"

# Check local files
ls scripts/
cat scripts/setup-server.sh
```

---

## 🖥️ ON THE SERVER (Hetzner Remote Machine)

**This is the server you created on Hetzner**

### When to use:
- After running `ssh root@SERVER_IP`
- When checking services
- When running the setup-server.sh script
- When troubleshooting

### Commands that run on server:

```bash
# You'll see a prompt like: root@ubuntu-xxx:~#

# Run setup script (on server, after copying via SCP)
cd /tmp
chmod +x setup-server.sh
sudo ./setup-server.sh

# Check services (on server)
sudo systemctl status nats
sudo systemctl status nimsforest

# View logs (on server)
sudo journalctl -u nats -f
sudo journalctl -u nimsforest -f

# Check NATS monitoring (on server)
curl http://localhost:8222/varz

# Check system resources (on server)
free -h
df -h
top
```

---

## 🔄 Complete Flow with Locations

```
┌─────────────────────────────────────────────────────────────┐
│  STEP 1: CREATE SERVER (Web Browser)                       │
│  Location: https://console.hetzner.cloud/                  │
│  Result: Get SERVER_IP                                     │
└─────────────────────────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────────────────────────┐
│  STEP 2: COPY SETUP SCRIPT (Local Machine)                 │
│                                                             │
│  💻 Local Terminal:                                         │
│  $ cd ~/projects/nimsforest                                 │
│  $ scp scripts/setup-server.sh root@SERVER_IP:/tmp/        │
│                                                             │
│  This copies FROM your local machine TO the server         │
└─────────────────────────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────────────────────────┐
│  STEP 3: RUN SETUP SCRIPT (Remote Server)                  │
│                                                             │
│  💻 Local Terminal:                                         │
│  $ ssh root@SERVER_IP                                       │
│                                                             │
│  🖥️  Now on Server:                                         │
│  root@server:~# cd /tmp                                     │
│  root@server:~# chmod +x setup-server.sh                    │
│  root@server:~# sudo ./setup-server.sh                      │
│  root@server:~# exit                                        │
│                                                             │
│  Back to local machine                                     │
└─────────────────────────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────────────────────────┐
│  STEP 4: CONFIGURE DEPLOYMENT (Local Machine)              │
│                                                             │
│  💻 Local Terminal:                                         │
│  $ cd ~/projects/nimsforest                                 │
│  $ ./scripts/setup-staging-local.sh SERVER_IP              │
│                                                             │
│  This configures GitHub secrets (runs locally)             │
└─────────────────────────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────────────────────────┐
│  STEP 5: DEPLOY (Local Machine)                            │
│                                                             │
│  💻 Local Terminal:                                         │
│  $ cd ~/projects/nimsforest                                 │
│  $ git push origin main                                     │
│  $ gh run watch                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 🎯 Quick Decision Guide

**Where should I run this command?**

### Run on LOCAL machine if:
- ✅ Command starts with `git`
- ✅ Command starts with `gh`
- ✅ Command is `scp` (source is local)
- ✅ Command references `scripts/` or local files
- ✅ You're configuring secrets
- ✅ You're in your project directory

### Run on SERVER if:
- ✅ Command starts with `sudo systemctl`
- ✅ Command is `journalctl`
- ✅ You're checking NATS or services
- ✅ You're after running `ssh root@SERVER_IP`
- ✅ The prompt shows `root@server` or similar

---

## 🔍 How to Tell Where You Are

### Local Machine:
```bash
$ pwd
/home/yourname/projects/nimsforest

$ whoami
yourname

$ ls
README.md  scripts/  internal/  ...
```

### Remote Server:
```bash
root@ubuntu-xxx:~# pwd
/root

root@ubuntu-xxx:~# whoami
root

root@ubuntu-xxx:~# hostname
ubuntu-xxx
```

---

## 📝 Common Mistakes

### ❌ Wrong: Running scp on the server
```bash
# DON'T DO THIS (you're on the server, file doesn't exist here):
root@server:~# scp scripts/setup-server.sh root@localhost:/tmp/
```

### ✅ Right: Running scp on local machine
```bash
# DO THIS (you're on local machine, file exists here):
you@laptop:~/nimsforest$ scp scripts/setup-server.sh root@SERVER_IP:/tmp/
```

---

### ❌ Wrong: Running git push on server
```bash
# DON'T DO THIS (server doesn't have your git credentials):
root@server:~# git push origin main
```

### ✅ Right: Running git push on local machine
```bash
# DO THIS (your laptop has git configured):
you@laptop:~/nimsforest$ git push origin main
```

---

### ❌ Wrong: Checking server services from local machine
```bash
# DON'T DO THIS (NATS is running on server, not locally):
you@laptop:~/nimsforest$ systemctl status nats
```

### ✅ Right: SSH to server first, then check
```bash
# DO THIS:
you@laptop:~/nimsforest$ ssh root@SERVER_IP
root@server:~# systemctl status nats
```

---

## 💡 Pro Tips

### 1. Keep two terminals open:
```
Terminal 1: Local machine (for git, scp, gh commands)
Terminal 2: Server (ssh root@SERVER_IP) (for monitoring)
```

### 2. Use terminal tabs or tmux:
```
Tab 1: Your local nimsforest directory
Tab 2: SSH to staging server  
Tab 3: SSH to production server (later)
```

### 3. Use descriptive terminal titles:
```
Terminal 1: "nimsforest-local"
Terminal 2: "staging-server"
Terminal 3: "production-server"
```

### 4. Check where you are before running commands:
```bash
# Always verify:
pwd       # Where am I?
whoami    # Who am I?
hostname  # Which machine?
```

---

## 🆘 "I'm Confused, Where Am I?"

Run these commands to find out:

```bash
# Command 1: Check current directory
pwd

# If output is like: /home/yourname/... → You're on LOCAL
# If output is like: /root or /tmp     → You're on SERVER

# Command 2: Check hostname
hostname

# If output is your computer name → You're on LOCAL
# If output is like "ubuntu-xxx"  → You're on SERVER

# Command 3: Check if files exist
ls scripts/setup-server.sh

# If file exists → You're on LOCAL (in the project)
# If file not found → You're probably on SERVER
```

---

## 📋 Summary Table

| Action | Location | Command Example |
|--------|----------|----------------|
| Copy file to server | Local | `scp scripts/setup-server.sh root@IP:/tmp/` |
| Run setup script | Server | `sudo ./setup-server.sh` |
| Configure secrets | Local | `./scripts/setup-staging-local.sh IP` |
| Deploy code | Local | `git push origin main` |
| Watch deployment | Local | `gh run watch` |
| Check service | Server | `sudo systemctl status nimsforest` |
| View logs | Server | `sudo journalctl -u nimsforest -f` |
| SSH to server | Local | `ssh root@SERVER_IP` |
| Exit from server | Server | `exit` or `Ctrl+D` |

---

**Rule of thumb:** If it's about your code or Git, do it locally. If it's about services or the OS, do it on the server.
