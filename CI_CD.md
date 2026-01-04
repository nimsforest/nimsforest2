# CI/CD Documentation

This document describes the Continuous Integration and Continuous Deployment (CI/CD) pipeline for NimsForest.

## Overview

NimsForest uses GitHub Actions for automated testing, building, and releasing. The pipeline is optimized for Debian-based systems and includes:

- ✅ Automated testing on multiple Go versions
- ✅ Code linting and formatting checks
- ✅ Code coverage reporting via Codecov
- ✅ Multi-platform binary builds
- ✅ Automated Debian package creation
- ✅ Continuous deployment to Hetzner Cloud
- ✅ Docker image building and publishing
- ✅ Release automation with changelog generation

## Workflows

### 1. CI Workflow (`.github/workflows/ci.yml`)

**Triggers**: Push to `main` or `cursor/**` branches, Pull Requests

**Jobs**:

#### Test
- Runs on: `ubuntu-latest`
- Go versions: `1.23.x`, `1.24.x`
- Steps:
  1. Checkout code
  2. Set up Go with caching
  3. Install NATS Server
  4. Download dependencies
  5. Start NATS with JetStream
  6. Run tests with race detector and coverage
  7. Upload coverage to Codecov
  8. Stop NATS Server

#### Lint
- Runs on: `ubuntu-latest`
- Go version: `1.24.x`
- Steps:
  1. Checkout code
  2. Set up Go
  3. Check code formatting with `gofmt`
  4. Run `go vet`
  5. Install and run `golangci-lint`

#### Build
- Runs on: `ubuntu-latest`
- Matrix: `linux` × `amd64/arm64`
- Steps:
  1. Checkout code
  2. Set up Go
  3. Build binary for Linux (Debian)
  4. Upload build artifacts (7 day retention)

#### Integration Test
- Runs on: `ubuntu-latest`
- Depends on: `test`, `lint`
- Steps:
  1. Checkout code
  2. Set up Go
  3. Install NATS
  4. Run integration tests via `make test-integration`
  5. Clean up NATS

### 2. Release Workflow (`.github/workflows/release.yml`)

**Triggers**: Tags matching `v*` (e.g., `v1.0.0`)

**Jobs**:

#### Create Release
- Generates changelog from git commits
- Creates GitHub release with notes

#### Build and Upload Assets
- Matrix: Linux architectures
  - linux/amd64 (for Debian amd64)
  - linux/arm64 (for Debian arm64)
- Creates tarballs (`.tar.gz`)
- Uploads all assets to GitHub release


### 3. Debian Package Workflow (`.github/workflows/debian-package.yml`)

**Triggers**: Tags matching `v*`, Manual dispatch

**Jobs**:

#### Build Debian Package
- Matrix: `amd64`, `arm64`
- Steps:
  1. Checkout code
  2. Set up Go
  3. Extract version from tag
  4. Create Debian package structure
  5. Build binary for target architecture
  6. Generate control file with metadata
  7. Create systemd service file

### 4. Hetzner Deployment Workflow (`.github/workflows/deploy-hetzner.yml`)

**Triggers**:
- Push to `main` → **Staging** (automatic)
- Release published (`v*`) → **Production** (automatic)
- Manual dispatch → **Your choice**

**Jobs**:

#### Deploy
- Uses Make commands: `make deps`, `make build-deploy`, `make deploy-package`
- Environment auto-selected based on trigger
- Steps:
  1. Checkout code and setup Go
  2. Build deployment binary with Make
  3. Create deployment package with Make
  4. Copy package to server via SCP
  5. Deploy via SSH script invocation
  6. Verify service health
  7. Auto-rollback on failure

**Key Features**:
- **Automatic staging on every push to main**
- **Automatic production on release**
- Make-based for consistency
- Zero-downtime deployment
- Automatic rollback

Setup: [DEPLOYMENT_SSH.md](./DEPLOYMENT_SSH.md) | Quick ref: [DEPLOYMENT_QUICKREF.md](./DEPLOYMENT_QUICKREF.md)
  8. Create postinst/prerm scripts
  9. Build `.deb` package
  10. Upload artifact
  11. Attach to GitHub release

**Package Contents**:
- Binary: `/usr/local/bin/forest`
- Systemd service: `/usr/lib/systemd/system/nimsforest.service`
- Directories: `/etc/nimsforest`, `/var/lib/nimsforest`, `/var/log/nimsforest`
- User: `forest` system user (created automatically)

## Configuration Files

### `.golangci.yml`

Configures `golangci-lint` with:
- Multiple enabled linters (errcheck, gosimple, govet, staticcheck, etc.)
- Cyclomatic complexity threshold: 15
- Duplication threshold: 100 lines
- Test file exclusions for certain linters
- US locale for spell checking

### `.codecov.yml`

Configures Codecov with:
- Target coverage: 75% (project), 70% (patch)
- Precision: 2 decimal places
- Automatic PR comments with coverage diff
- GitHub checks integration


## Usage

### Running CI Locally

Simulate CI checks locally before pushing:

```bash
# Run tests
make test

# Run linting
make lint

# Run all checks
make check

# Format code
make fmt

# Run vet
make vet
```

### Creating a Release

1. **Commit all changes** and ensure CI passes:
   ```bash
   git add .
   git commit -m "Prepare release v1.0.0"
   git push
   ```

2. **Create and push a tag**:
   ```bash
   git tag -a v1.0.0 -m "Release version 1.0.0"
   git push origin v1.0.0
   ```

3. **Wait for workflows** to complete:
   - CI workflow validates the tag
   - Release workflow creates GitHub release
   - Debian package workflow builds `.deb` files

4. **Verify release**:
   - Check GitHub Releases page
   - Download and test binaries
   - Test Debian package installation

### Manual Workflow Dispatch

Some workflows support manual triggering:

```bash
# Trigger via GitHub CLI
gh workflow run debian-package.yml

# Or via GitHub web interface:
# Actions → Select workflow → Run workflow
```

## Secrets Configuration

### Required Secrets

Add these to your GitHub repository settings (Settings → Secrets and variables → Actions):

#### For Codecov (Optional but Recommended)
- `CODECOV_TOKEN`: Codecov upload token
  - Get from: https://codecov.io/gh/yourusername/nimsforest/settings

### Setting Secrets

Via GitHub CLI:
```bash
gh secret set CODECOV_TOKEN --body "your-token-here"
```

Via GitHub Web:
1. Go to repository settings
2. Navigate to Secrets and variables → Actions
3. Click "New repository secret"
4. Enter name and value
5. Click "Add secret"

## Status Badges

Add these badges to your README to show CI/CD status:

```markdown
[![CI](https://github.com/yourusername/nimsforest/actions/workflows/ci.yml/badge.svg)](https://github.com/yourusername/nimsforest/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/yourusername/nimsforest/branch/main/graph/badge.svg)](https://codecov.io/gh/yourusername/nimsforest)
[![Go Report Card](https://goreportcard.com/badge/github.com/yourusername/nimsforest)](https://goreportcard.com/report/github.com/yourusername/nimsforest)
```

## Troubleshooting

### CI Failures

#### Tests Failing
```bash
# Run tests locally with verbose output
go test -v -race ./...

# Check NATS connection
curl http://localhost:8222/varz

# Review test logs in GitHub Actions
```

#### Linter Failures
```bash
# Run golangci-lint locally
golangci-lint run

# Auto-fix issues
golangci-lint run --fix

# Format code
gofmt -s -w .
```

#### Build Failures
```bash
# Test build for specific platform
GOOS=linux GOARCH=amd64 go build -v ./cmd/forest

# Check for missing dependencies
go mod verify
go mod download
```

### Release Issues

#### Tag Not Building
- Ensure tag follows `v*` pattern (e.g., `v1.0.0`)
- Check that workflows are enabled in repository settings
- Verify branch protection rules don't block tags

#### Asset Upload Failing
- Check GitHub token permissions
- Ensure release was created successfully
- Review workflow logs for errors

### Debian Package Issues

#### Package Won't Install
```bash
# Check package contents
dpkg -c nimsforest_1.0.0_amd64.deb

# Install with verbose output
sudo dpkg -i --debug=10 nimsforest_1.0.0_amd64.deb

# Check dependencies
dpkg -I nimsforest_1.0.0_amd64.deb
```

#### Service Won't Start
```bash
# Check systemd service
sudo systemctl status nimsforest

# View logs
sudo journalctl -u nimsforest -n 50

# Test binary directly
sudo -u forest /usr/local/bin/forest
```

## Performance Optimization

### Caching

The CI workflows use Go module caching to speed up builds:

```yaml
- uses: actions/setup-go@v5
  with:
    go-version: '1.24.x'
    cache: true  # Enables automatic Go module caching
```

### Matrix Parallelization

Tests run in parallel across multiple Go versions:

```yaml
strategy:
  matrix:
    go-version: ['1.23.x', '1.24.x']
```

### Artifact Retention

Build artifacts are kept for 7 days to save storage:

```yaml
- uses: actions/upload-artifact@v4
  with:
    retention-days: 7
```

## Best Practices

1. **Always run tests locally** before pushing:
   ```bash
   make test
   make lint
   ```

2. **Keep workflows fast**:
   - Use caching for dependencies
   - Run expensive jobs only on main branch
   - Parallelize where possible

3. **Semantic versioning**:
   - Major: Breaking changes (v2.0.0)
   - Minor: New features (v1.1.0)
   - Patch: Bug fixes (v1.0.1)

4. **Write good commit messages**:
   - Used in automated changelogs
   - Follow conventional commits format

5. **Test releases**:
   - Test on actual Debian system
   - Verify all platforms work
   - Check Docker image runs correctly

6. **Monitor coverage**:
   - Keep project coverage above 75%
   - Review coverage reports in PRs
   - Add tests for new features

## Continuous Improvement

### Adding New Linters

Edit `.golangci.yml`:

```yaml
linters:
  enable:
    - newlinter
```

### Adding New Build Targets

Edit `.github/workflows/ci.yml`:

```yaml
strategy:
  matrix:
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64, 386]
```

### Custom Workflow

Create `.github/workflows/custom.yml`:

```yaml
name: Custom Workflow

on:
  schedule:
    - cron: '0 0 * * 0'  # Weekly

jobs:
  custom-job:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      # Your custom steps
```

## Support

For CI/CD issues:
- Check workflow logs in GitHub Actions tab
- Review this documentation
- Open an issue with workflow logs attached
- Tag with `ci/cd` label
