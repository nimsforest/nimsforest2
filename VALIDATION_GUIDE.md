# CI/CD Validation Guide

This guide will help you validate that the CI/CD pipeline works correctly before using it in production.

## Table of Contents

- [Local Validation](#local-validation)
- [GitHub Actions Validation](#github-actions-validation)
- [Debian Package Validation](#debian-package-validation)
- [End-to-End Test](#end-to-end-test)
- [Troubleshooting](#troubleshooting)

## Local Validation

### Step 1: Validate Make Commands

Test all Make commands work correctly on your system:

```bash
# Quick validation
make validate-quick

# Complete validation
make validate-all

# Or run individual checks:
make verify          # Check environment
make test            # Run tests
make lint            # Run linter
make fmt             # Format code
make build           # Build packages
make clean           # Clean up
```

### Step 2: Validate Workflow Syntax

Check that workflow files are valid YAML:

```bash
# Install yamllint if not present
pip install yamllint

# Validate workflow files
yamllint .github/workflows/ci.yml
yamllint .github/workflows/release.yml
yamllint .github/workflows/debian-package.yml
```

### Step 3: Test NATS Integration

```bash
# Start NATS
make start

# Run integration tests
make test-integration

# Check NATS status
make status

# Stop NATS
make stop
```

## GitHub Actions Validation

### Step 1: Push to a Test Branch

Create a test branch to trigger CI without affecting main:

```bash
# Create test branch
git checkout -b test/ci-validation

# Add a small test change
echo "# CI Test" >> TEST_CI.md
git add TEST_CI.md
git commit -m "test: validate CI pipeline"

# Push to GitHub
git push -u origin test/ci-validation
```

**Expected Results:**
- CI workflow should trigger automatically
- Go to: `https://github.com/yourusername/nimsforest/actions`
- You should see a workflow run in progress

**What to Check:**
- ✅ Test job completes (both Go 1.23.x and 1.24.x)
- ✅ Lint job completes
- ✅ Build job completes (amd64 and arm64)
- ✅ Integration test job completes
- ✅ Code coverage uploaded to Codecov (if configured)
- ✅ Build artifacts available for download

### Step 2: Create a Test Pull Request

```bash
# Create PR from test branch
gh pr create --title "Test: Validate CI/CD Pipeline" \
  --body "This PR tests the CI/CD pipeline. Will close without merging." \
  --base main --head test/ci-validation
```

**Expected Results:**
- CI checks appear on the PR
- All checks pass (green checkmarks)
- Codecov bot comments with coverage report (if configured)

### Step 3: Download and Test Build Artifacts

```bash
# List recent workflow runs
gh run list --limit 5

# Download artifacts from the latest run
gh run download <RUN_ID>

# Test the binaries
cd forest-linux-amd64
tar xzf forest-linux-amd64.tar.gz || ls -la
./forest --version || echo "Binary exists"
cd ..

# Clean up
rm -rf forest-linux-*
```

### Step 4: Test Release Workflow (Dry Run)

Create a test tag to validate the release process:

```bash
# Create a test release tag
git tag -a v0.0.1-test -m "Test release - do not use in production"

# Push the tag
git push origin v0.0.1-test
```

**Expected Results:**
1. **Release Workflow** runs:
   - Generates changelog
   - Creates GitHub release (draft or pre-release)
   - Builds Linux binaries (amd64 and arm64)
   - Uploads tar.gz files to release

2. **Debian Package Workflow** runs:
   - Builds .deb packages (amd64 and arm64)
   - Uploads .deb files to release

3. **CI Workflow** runs as validation

**What to Check:**
```bash
# View the test release
gh release view v0.0.1-test

# Expected assets:
# - forest-0.0.1-test-linux-amd64.tar.gz
# - forest-0.0.1-test-linux-arm64.tar.gz
# - nimsforest_0.0.1-test_amd64.deb
# - nimsforest_0.0.1-test_arm64.deb

# Download and test binary
gh release download v0.0.1-test

# Test the binary
tar xzf forest-0.0.1-test-linux-amd64.tar.gz
./forest --version || echo "Binary works"

# Clean up test release
gh release delete v0.0.1-test --yes
git push --delete origin v0.0.1-test
git tag -d v0.0.1-test
```

## Debian Package Validation

### On a Debian/Ubuntu System

```bash
# Download the test .deb package
gh release download v0.0.1-test --pattern "*.deb"

# Inspect the package
dpkg -I nimsforest_0.0.1-test_amd64.deb

# Expected:
# - Package: nimsforest
# - Version: 0.0.1-test
# - Architecture: amd64
# - Depends: libc6 (>= 2.34)

# List package contents
dpkg -c nimsforest_0.0.1-test_amd64.deb

# Expected files:
# - /usr/local/bin/forest
# - /usr/lib/systemd/system/nimsforest.service
# - /etc/nimsforest/
# - /var/lib/nimsforest/
# - /var/log/nimsforest/

# Test installation (requires sudo)
sudo dpkg -i nimsforest_0.0.1-test_amd64.deb

# Expected:
# - User 'forest' created
# - Directories created with correct permissions
# - Systemd service installed

# Verify service file
systemctl cat nimsforest

# Check service status (should be inactive)
systemctl status nimsforest

# Test uninstallation
sudo dpkg -r nimsforest
```

### Test systemd Service

```bash
# After installing the package
# Edit config if needed
sudo nano /etc/nimsforest/config.env

# Start NATS first (or point to existing NATS)
make start

# Start the service
sudo systemctl start nimsforest

# Check status
sudo systemctl status nimsforest

# View logs
sudo journalctl -u nimsforest -f

# Stop the service
sudo systemctl stop nimsforest

# Clean up
sudo dpkg -r nimsforest
```

## End-to-End Test

Complete validation from code to deployment:

```bash
# 1. Start fresh
git checkout main
git pull origin main

# 2. Create feature branch
git checkout -b test/complete-validation

# 3. Make a small change
echo "// Test change" >> internal/core/leaf.go
git add internal/core/leaf.go
git commit -m "test: complete CI/CD validation"

# 4. Run local checks
make test
make lint
make build

# 5. Push and create PR
git push -u origin test/complete-validation
gh pr create --title "Test: Complete CI/CD Validation" \
  --body "End-to-end validation of CI/CD pipeline" \
  --base main

# 6. Wait for CI to pass
gh pr checks

# 7. Merge PR (if all checks pass)
gh pr merge --squash

# 8. Create release
git checkout main
git pull origin main
git tag -a v0.0.2-test -m "Test release v0.0.2"
git push origin v0.0.2-test

# 9. Wait for release workflows
gh run watch

# 10. Verify release created
gh release view v0.0.2-test

# 11. Download and test
gh release download v0.0.2-test
tar xzf forest-0.0.2-test-linux-amd64.tar.gz
./forest --version

# 12. Clean up
gh release delete v0.0.2-test --yes
git push --delete origin v0.0.2-test
git tag -d v0.0.2-test
git branch -d test/complete-validation
git push origin --delete test/complete-validation
```

## Validation Checklist

Use this checklist to ensure everything works:

### Local Validation
- [ ] `make verify` passes
- [ ] `make test` passes (all tests green)
- [ ] `make lint` passes (no errors)
- [ ] `make fmt` shows no changes needed
- [ ] `make build` creates working binary
- [ ] `make test-integration` passes with NATS
- [ ] Workflow YAML files are valid

### CI Workflow
- [ ] Push to branch triggers CI
- [ ] Test job passes on Go 1.23.x
- [ ] Test job passes on Go 1.24.x
- [ ] Lint job passes
- [ ] Build job creates amd64 binary
- [ ] Build job creates arm64 binary
- [ ] Integration test job passes
- [ ] Code coverage uploaded (if configured)
- [ ] Build artifacts downloadable

### Release Workflow
- [ ] Tag push triggers release workflow
- [ ] Changelog generated correctly
- [ ] GitHub release created
- [ ] Linux amd64 binary uploaded
- [ ] Linux arm64 binary uploaded
- [ ] Binaries are executable
- [ ] Version injected correctly

### Debian Package Workflow
- [ ] Tag push triggers package workflow
- [ ] amd64 .deb package created
- [ ] arm64 .deb package created
- [ ] Package metadata correct
- [ ] Package contents include all files
- [ ] Package installs successfully
- [ ] Systemd service file present
- [ ] User/group created correctly
- [ ] Directories have correct permissions
- [ ] Service can start/stop
- [ ] Package uninstalls cleanly

### Pull Request Flow
- [ ] PR triggers CI checks
- [ ] Status checks appear on PR
- [ ] All checks pass
- [ ] Codecov comment appears (if configured)
- [ ] Can merge after checks pass

## Troubleshooting

### CI Fails on First Run

**Problem:** Workflow fails with permission errors

**Solution:**
```bash
# Check workflow permissions in GitHub
# Settings → Actions → General → Workflow permissions
# Enable "Read and write permissions"
```

### NATS Connection Fails in CI

**Problem:** Tests fail with "connection refused"

**Solution:**
- Check `.github/workflows/ci.yml` NATS installation step
- Ensure NATS starts before tests run
- Add longer sleep if needed: `sleep 3`

### Debian Package Won't Install

**Problem:** `dpkg` errors during installation

**Solution:**
```bash
# Check dependencies
dpkg -I package.deb | grep Depends

# Install missing dependencies
sudo apt-get install -f

# Check package structure
dpkg -c package.deb | grep -E "(bin|lib|etc)"
```

### Binary Not Executable

**Problem:** Downloaded binary won't run

**Solution:**
```bash
# Add execute permission
chmod +x forest

# Check binary
file forest
ldd forest  # Check dependencies
```

### Release Assets Not Uploading

**Problem:** Release created but no assets

**Solution:**
- Check GitHub Actions logs for upload errors
- Verify GITHUB_TOKEN has correct permissions
- Check asset paths in workflow file

### Codecov Upload Fails

**Problem:** Coverage upload fails

**Solution:**
```bash
# Codecov token required for private repos
# Add CODECOV_TOKEN secret to GitHub
gh secret set CODECOV_TOKEN --body "your-token"

# Public repos don't need token
# Check Codecov integration is enabled
```

## Quick Validation with Make

Run the automated validation:

```bash
# Quick validation (2-3 minutes)
make validate-quick

# Complete validation (5-10 minutes)
make validate-all

# Just the alias
make validate
```

## Next Steps After Validation

Once everything passes:

1. **Clean up test artifacts**:
   ```bash
   # Delete test branches
   git branch -D test/ci-validation test/complete-validation

   # Delete test releases
   gh release delete v0.0.1-test --yes
   gh release delete v0.0.2-test --yes

   # Delete test tags
   git tag -d v0.0.1-test v0.0.2-test
   git push origin --delete v0.0.1-test v0.0.2-test
   ```

2. **Update repository URLs**:
   - Replace `yourusername/nimsforest` with actual repo in all files

3. **Configure branch protection**:
   - Settings → Branches → Add rule for `main`
   - Require status checks to pass
   - Require code review

4. **Create first production release**:
   ```bash
   git tag -a v1.0.0 -m "Initial production release"
   git push origin v1.0.0
   ```

5. **Monitor the release**:
   - Watch GitHub Actions
   - Download and test artifacts
   - Install .deb package on production system

## Support

If validation fails:
- Review the [CI_CD.md](./CI_CD.md) troubleshooting section
- Check GitHub Actions logs
- Review workflow files for syntax errors
- Open an issue with error logs
