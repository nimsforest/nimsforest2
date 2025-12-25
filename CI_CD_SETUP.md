# CI/CD Setup Summary

## Overview

A comprehensive CI/CD pipeline has been added to NimsForest, optimized for Debian-based deployment targets using Make as the primary build system. The pipeline provides automated testing, building, releasing, and packaging for production use.

## What's Been Added

### GitHub Actions Workflows

#### 1. **CI Workflow** (`.github/workflows/ci.yml`)
- **Purpose**: Continuous Integration for every push and PR
- **Features**:
  - Multi-version Go testing (1.23.x, 1.24.x)
  - Automated NATS server setup and teardown
  - Race condition detection
  - Code coverage reporting to Codecov
  - Code linting with golangci-lint
  - Code formatting validation
  - Multi-platform builds (Linux, macOS × amd64, arm64)
  - Integration test suite
- **When it runs**: Push to `main` or `cursor/**` branches, all PRs

#### 2. **Release Workflow** (`.github/workflows/release.yml`)
- **Purpose**: Automated releases for version tags
- **Features**:
  - Automatic changelog generation from commits
  - Linux binary builds for Debian (amd64 and arm64)
  - Asset packaging (tar.gz)
  - Automated GitHub release creation
  - Version injection into binaries
- **When it runs**: Push tags matching `v*` (e.g., `v1.0.0`)

#### 3. **Debian Package Workflow** (`.github/workflows/debian-package.yml`)
- **Purpose**: Build native Debian packages
- **Features**:
  - Creates `.deb` packages for amd64 and arm64
  - Includes systemd service integration
  - Automatic user creation (forest)
  - Proper directory structure (/usr/local/bin, /var/lib, /var/log)
  - Post-installation scripts for service setup
  - Pre-removal scripts for cleanup
  - Attaches to GitHub releases
- **When it runs**: Version tags or manual dispatch

#### 4. **Hetzner Deployment Workflow** (`.github/workflows/deploy-hetzner.yml`)
- **Purpose**: Continuous deployment to Hetzner Cloud servers
- **Features**:
  - **Automatic staging deployment** on push to `main`
  - **Automatic production deployment** on release publication
  - Manual deployment trigger with environment selection
  - Zero-downtime deployment with automatic rollback
  - SSH-based secure deployment using Make commands
  - Service health verification
- **When it runs**: 
  - Push to `main` → Staging
  - Release published (`v*`) → Production
  - Manual trigger → Your choice

### Configuration Files

#### 1. **`.golangci.yml`**
Linter configuration with:
- 20+ enabled linters (errcheck, gosimple, govet, staticcheck, etc.)
- Cyclomatic complexity checks
- Code duplication detection
- Constant detection
- Security checks with gosec
- Test file exceptions

#### 2. **`.codecov.yml`**
Code coverage configuration:
- 75% target coverage for project
- 70% target coverage for patches
- Automatic PR comments with coverage diff
- GitHub checks integration
- Coverage annotations on PRs

### Documentation

#### 1. **`HETZNER_DEPLOYMENT.md`**
Complete Hetzner Cloud continuous deployment guide:
- Hetzner server setup and configuration
- GitHub Actions deployment workflow
- SSH key and secrets management
- Automatic deployment on release
- Manual deployment triggers
- Monitoring and management
- Rollback procedures
- Security best practices
- Cost optimization tips

#### 2. **`DEPLOYMENT.md`**
Comprehensive deployment guide covering:
- System requirements
- Debian package installation
- Binary installation
- Building from source with Make
- Systemd service configuration
- NATS server setup
- Production considerations
- Security best practices
- Monitoring and backup procedures
- Troubleshooting guide

#### 3. **`CI_CD.md`**
Complete CI/CD documentation:
- Workflow descriptions
- Configuration file explanations
- Usage instructions
- Secret management
- Troubleshooting guide
- Best practices
- Performance optimization tips

#### 4. **Updated `README.md`**
- CI/CD status badges
- Codecov badge
- Go Report Card badge
- License badge
- Quick deployment section with Hetzner CD
- Links to new documentation

## File Structure

```
.github/
└── workflows/
    ├── ci.yml                    # Main CI pipeline
    ├── release.yml               # Release automation
    ├── debian-package.yml        # Debian package builder
    └── deploy-hetzner.yml        # Hetzner CD pipeline

scripts/
├── deploy.sh                     # Deployment script for server
├── setup-hetzner-server.sh       # Initial server setup
└── systemd/
    └── nimsforest.service        # systemd service file

.codecov.yml                      # Codecov configuration
.golangci.yml                     # Linter configuration
HETZNER_DEPLOYMENT.md             # Hetzner CD guide
DEPLOYMENT.md                     # Deployment guide
CI_CD.md                          # CI/CD documentation
CI_CD_SETUP.md                    # This file
CI_CD_FILES_SUMMARY.txt           # Quick reference
README.md                         # Updated with badges
```

## Quick Start Guide

### For Users

**Install on Debian/Ubuntu:**
```bash
wget https://github.com/yourusername/nimsforest/releases/latest/download/nimsforest_VERSION_amd64.deb
sudo dpkg -i nimsforest_VERSION_amd64.deb
sudo systemctl start nimsforest
```

**Binary Installation:**
```bash
wget https://github.com/yourusername/nimsforest/releases/latest/download/forest-linux-amd64.tar.gz
tar xzf forest-linux-amd64.tar.gz
./forest
```

### For Developers

**Run CI checks locally:**
```bash
make validate-quick  # Quick validation
make test            # Run tests
make lint            # Run linter
make check           # All checks
```

**Build from source:**
```bash
make setup         # Complete environment setup
make build         # Build binary
```

**Create a release:**
```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

## Required GitHub Secrets

To enable all features, add these secrets to your GitHub repository:

### For Hetzner Continuous Deployment
- `HETZNER_SSH_PRIVATE_KEY` - SSH private key for deployment access
- `HETZNER_SSH_USER` - SSH user (typically `root`)
- `HETZNER_HOST` - Server IP address or hostname
- `HETZNER_KNOWN_HOSTS` - SSH host key fingerprint

### Optional but Recommended
- `CODECOV_TOKEN` - For coverage reporting (get from codecov.io)

### Adding Secrets
```bash
# Via GitHub CLI
gh secret set CODECOV_TOKEN --body "your-token"
gh secret set HETZNER_SSH_PRIVATE_KEY < ~/.ssh/deploy_key
gh secret set HETZNER_SSH_USER --body "root"
gh secret set HETZNER_HOST --body "YOUR_SERVER_IP"
gh secret set HETZNER_KNOWN_HOSTS < known_hosts
```

Or via GitHub web interface:
Settings → Secrets and variables → Actions → New repository secret

For complete Hetzner setup instructions, see **[HETZNER_DEPLOYMENT.md](./HETZNER_DEPLOYMENT.md)**.

## CI/CD Pipeline Flow

### On Every Push/PR
```
Push to main/cursor/** or PR
  ↓
CI Workflow Triggered
  ├── Test Job (parallel)
  │   ├── Go 1.23.x → Test → Upload Coverage
  │   └── Go 1.24.x → Test → Upload Coverage
  ├── Lint Job
  │   ├── gofmt check
  │   ├── go vet
  │   └── golangci-lint
  ├── Build Job (parallel)
  │   ├── linux/amd64 (Debian)
  │   └── linux/arm64 (Debian)
  └── Integration Test Job
      └── Full stack test with NATS
```

### On Version Tag (v*)
```
Push tag v1.0.0
  ↓
Two Workflows Triggered (parallel)
  ├── Release Workflow
  │   ├── Generate changelog
  │   ├── Create GitHub release
  │   ├── Build Linux binaries for Debian
  │   │   ├── amd64
  │   │   └── arm64
  │   └── Upload assets
  │
  ├── Debian Package Workflow
  │   ├── Build .deb for amd64
  │   ├── Build .deb for arm64
  │   └── Attach to release
  │
  └── CI Workflow (validation)
      └── Full test suite
```

## Benefits

### For Developers
- ✅ Automated testing catches bugs early
- ✅ Code quality enforcement via linting
- ✅ Coverage tracking prevents regressions
- ✅ Consistent builds across platforms
- ✅ Fast feedback on PRs
- ✅ Make-based workflow for consistency

### For Users
- ✅ Pre-built binaries for all platforms
- ✅ Native Debian packages with systemd
- ✅ Simple Make commands for building
- ✅ Automated changelogs for releases
- ✅ Verified and tested releases

### For Operations
- ✅ Easy installation via package manager
- ✅ Systemd integration for service management
- ✅ Make commands for all operations
- ✅ Proper logging and monitoring setup
- ✅ Graceful updates and rollbacks

## Platform Support

### Supported Platforms
- ✅ Debian 11 (Bullseye) and later
- ✅ Ubuntu 20.04 LTS and later
- ✅ Any Linux distribution (amd64, arm64)

### Build System
- ✅ Complete Make-based workflow
- ✅ Automated NATS server management
- ✅ Development environment setup
- ✅ Multi-platform builds

## Make Commands Reference

### Setup & Installation
```bash
make setup             # Complete environment setup
make deps              # Download Go dependencies
make install-nats      # Install NATS server
make verify            # Verify environment
```

### NATS Management
```bash
make start             # Start NATS with JetStream
make stop              # Stop NATS server
make restart           # Restart NATS
make status            # Check NATS status
```

### Testing
```bash
make test              # Run unit tests
make test-integration  # Run integration tests
make test-coverage     # Run tests with coverage
```

### Building
```bash
make build             # Build for current platform
make run               # Build and run
```

### Code Quality
```bash
make fmt               # Format code
make lint              # Run linter
make vet               # Run go vet
make check             # All checks
```

### Development
```bash
make dev               # Complete dev setup
make ci                # Run CI checks locally
```

## Monitoring CI/CD

### Status Pages
- **GitHub Actions**: `https://github.com/yourusername/nimsforest/actions`
- **Codecov**: `https://codecov.io/gh/yourusername/nimsforest`
- **Go Report**: `https://goreportcard.com/report/github.com/yourusername/nimsforest`

### Badges
All status badges are visible on the README:
- CI pipeline status
- Code coverage percentage
- Go Report Card grade
- License type

## Customization

### Add More Architectures
Edit `.github/workflows/ci.yml` or `release.yml`:
```yaml
strategy:
  matrix:
    goarch: [amd64, arm64, 386]  # Add more architectures if needed
```

### Change Coverage Target
Edit `.codecov.yml`:
```yaml
coverage:
  status:
    project:
      default:
        target: 80%  # Change from 75%
```

### Add More Linters
Edit `.golangci.yml`:
```yaml
linters:
  enable:
    - newlinter
```

## Next Steps

1. **Update Repository URLs**
   - Replace `yourusername/nimsforest` with your actual GitHub repository
   - Update in: README.md, CI_CD.md, DEPLOYMENT.md, workflow files

2. **Configure Secrets** (optional)
   - Add CODECOV_TOKEN for coverage reporting

3. **Test the Pipeline**
   ```bash
   # Make a change and push
   git add .
   git commit -m "test: verify CI pipeline"
   git push
   
   # Watch the Actions tab
   ```

4. **Create First Release**
   ```bash
   git tag -a v1.0.0 -m "Initial release"
   git push origin v1.0.0
   
   # Watch all workflows run
   # Download and test artifacts
   ```

5. **Set Up Branch Protection**
   - Require CI to pass before merging
   - Require code review
   - Configure in: Settings → Branches → Branch protection rules

## Troubleshooting

### CI Fails on NATS Connection
- Check if NATS installation step succeeded
- Verify NATS is running before tests
- Check network connectivity in runner

### Release Assets Not Uploading
- Verify GITHUB_TOKEN has correct permissions
- Check if release was created successfully
- Review asset path and naming

### Debian Package Issues
- Test package locally before pushing tag
- Verify dpkg-deb is installed
- Check package structure matches Debian standards

### Make Commands Failing
- Run `make verify` to check environment
- Ensure Go 1.22+ is installed
- Check NATS server installation
- Review Make output for specific errors

For more details, see:
- **[CI_CD.md](./CI_CD.md)** - Complete CI/CD documentation
- **[DEPLOYMENT.md](./DEPLOYMENT.md)** - Deployment troubleshooting
- **[Makefile](./Makefile)** - All available Make commands
- **GitHub Actions logs** - Real-time workflow execution details

## Support

- **Issues**: https://github.com/yourusername/nimsforest/issues
- **Discussions**: https://github.com/yourusername/nimsforest/discussions
- **Documentation**: https://github.com/yourusername/nimsforest

---

**CI/CD Setup Complete! 🎉**

The NimsForest project now has a production-ready CI/CD pipeline optimized for Debian deployments using Make as the build system.
