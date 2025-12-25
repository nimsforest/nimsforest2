# Make-Based Deployment Guide

This guide explains how to use Make commands for building and deploying NimsForest.

## Overview

The NimsForest project uses Make as the primary build system for consistency and simplicity. All deployment operations can be performed using Make commands instead of manual shell commands.

## Available Deployment Make Targets

### Core Deployment Commands

```bash
make build-deploy      # Build optimized binary for deployment
make deploy-package    # Create complete deployment package
make deploy-verify     # Verify all deployment files exist
```

### Standard Build Commands

```bash
make build            # Build for current platform
make build-all        # Build for all platforms (Linux, macOS, AMD64, ARM64)
make deps             # Download Go dependencies
```

## Usage Examples

### 1. Build Deployment Binary

Build an optimized Linux AMD64 binary ready for production:

```bash
make build-deploy
```

This creates:
- **Binary**: `forest` (Linux AMD64, stripped, optimized)
- **Size**: ~30% smaller than debug build
- **Flags**: `-ldflags="-s -w"` removes debug info and symbols

### 2. Create Deployment Package

Create a complete deployment package with all necessary files:

```bash
make deploy-package
```

This creates `nimsforest-deploy.tar.gz` containing:
- `forest` binary
- `deploy.sh` script
- `nimsforest.service` systemd service file

### 3. Verify Deployment Files

Check that all required deployment files exist:

```bash
make deploy-verify
```

Verifies:
- ✅ `scripts/deploy.sh`
- ✅ `scripts/setup-hetzner-server.sh`
- ✅ `scripts/systemd/nimsforest.service`
- ✅ `.github/workflows/deploy-hetzner.yml`

## Complete Deployment Workflow

### Local to Server Deployment

```bash
# 1. Build and package
make deploy-package

# 2. Copy to server
scp nimsforest-deploy.tar.gz root@YOUR_SERVER_IP:/tmp/

# 3. Deploy on server
ssh root@YOUR_SERVER_IP 'bash -s' < scripts/deploy.sh deploy

# 4. Verify deployment
ssh root@YOUR_SERVER_IP 'bash -s' < scripts/deploy.sh verify
```

### GitHub Actions Integration

The GitHub Actions workflow uses Make commands automatically:

```yaml
- name: Download dependencies
  run: make deps

- name: Build deployment binary
  run: make build-deploy

- name: Create deployment package
  run: make deploy-package
```

## Comparison: Make vs Shell Commands

### Building Binary

**Using Make** (recommended):
```bash
make build-deploy
```

**Using Shell**:
```bash
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o forest ./cmd/forest
chmod +x forest
```

### Creating Package

**Using Make** (recommended):
```bash
make deploy-package
```

**Using Shell**:
```bash
mkdir -p deploy
cp forest deploy/
cp scripts/deploy.sh deploy/
cp scripts/systemd/nimsforest.service deploy/
tar czf nimsforest-deploy.tar.gz deploy/
rm -rf deploy/
```

### Running Tests

**Using Make** (recommended):
```bash
make test
make test-integration
make test-coverage
```

**Using Shell**:
```bash
go test -v -race -short ./...
go test -v -race ./...
go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
```

## Benefits of Using Make

### Consistency
- Same commands work on all platforms
- Reduces human error
- Standard interface for all operations

### Simplicity
- Single command instead of multiple steps
- No need to remember complex flags
- Self-documenting with `make help`

### Integration
- Used by CI/CD workflows
- Matches project conventions
- Easy to extend and modify

### Portability
- Works on Linux, macOS, Windows (with make)
- No shell-specific syntax
- Clear dependency management

## Make Command Reference

### Build & Package
```bash
make deps              # Download dependencies
make build             # Build for current platform
make build-deploy      # Build optimized deployment binary
make build-all         # Build for all platforms
make deploy-package    # Create deployment package
```

### Testing
```bash
make test              # Run unit tests
make test-integration  # Run integration tests
make test-coverage     # Run with coverage report
```

### Code Quality
```bash
make fmt               # Format code
make lint              # Run linter
make vet               # Run go vet
make check             # Run all checks
```

### NATS Management
```bash
make install-nats      # Install NATS server
make start             # Start NATS with JetStream
make stop              # Stop NATS
make restart           # Restart NATS
make status            # Check NATS status
```

### Deployment
```bash
make deploy-verify     # Verify deployment files
make deploy-package    # Create deployment package
make build-deploy      # Build deployment binary
```

### Validation
```bash
make validate          # Run all validations
make validate-quick    # Quick validation
make verify            # Verify environment
```

### Cleanup
```bash
make clean             # Remove build artifacts
make clean-data        # Remove NATS data
make clean-all         # Remove everything
```

## CI/CD Integration

### GitHub Actions Workflow

The deployment workflow (`.github/workflows/deploy-hetzner.yml`) uses Make:

```yaml
steps:
  - name: Download dependencies
    run: make deps
  
  - name: Build deployment binary
    run: make build-deploy
  
  - name: Create deployment package
    run: make deploy-package
```

### Local CI Simulation

Run the same checks as CI locally:

```bash
make ci
```

This runs:
1. `make deps` - Download dependencies
2. `make verify` - Verify environment
3. `make test` - Run tests
4. `make vet` - Run go vet

## Makefile Structure

The Makefile is organized into sections:

```makefile
##@ General
help                   # Display help

##@ Setup & Installation
setup                  # Complete environment setup
deps                   # Download dependencies
install-nats           # Install NATS server
verify                 # Verify environment

##@ NATS Server Management
start                  # Start NATS
stop                   # Stop NATS
restart                # Restart NATS
status                 # Check status

##@ Testing
test                   # Run tests
test-integration       # Integration tests
test-coverage          # With coverage

##@ Building
build                  # Build application
build-all              # All platforms
build-deploy           # Deployment binary
run                    # Build and run

##@ Code Quality
fmt                    # Format code
lint                   # Run linter
vet                    # Run go vet
check                  # All checks

##@ Deployment
deploy-package         # Create package
deploy-verify          # Verify files

##@ Cleanup
clean                  # Remove artifacts
clean-data             # Remove NATS data
clean-all              # Remove everything

##@ Development Workflow
dev                    # Complete dev setup
ci                     # Run CI checks

##@ Validation
validate               # Run all validations
validate-quick         # Quick validation
```

## Extending the Makefile

### Adding New Targets

To add a new deployment-related target:

```makefile
##@ Deployment

deploy-backup: ## Backup current deployment
	@echo "$(BLUE)📦 Creating backup...$(NC)"
	@ssh root@$(SERVER) 'tar czf /backup/forest-$(date +%Y%m%d).tar.gz /usr/local/bin/forest'
	@echo "$(GREEN)✅ Backup created$(NC)"
```

### Using Variables

The Makefile defines useful variables:

```makefile
BINARY_NAME := forest
NATS_VERSION := 2.12.3
NATS_PORT := 4222
NATS_MONITOR_PORT := 8222
GO_PACKAGES := ./...
```

Use them in targets:

```makefile
my-target:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) ./cmd/forest
```

## Best Practices

### 1. Always Use Make Commands

**Good**:
```bash
make build-deploy
make deploy-package
```

**Avoid**:
```bash
GOOS=linux GOARCH=amd64 go build -o forest ./cmd/forest
tar czf package.tar.gz forest
```

### 2. Check Dependencies First

```bash
make verify          # Check environment
make deploy-verify   # Check deployment files
```

### 3. Use Make for CI/CD

In GitHub Actions:
```yaml
run: make build-deploy
```

Instead of:
```yaml
run: GOOS=linux GOARCH=amd64 go build -o forest ./cmd/forest
```

### 4. Leverage Make Help

```bash
make help            # See all commands
make                 # Same as make help (default target)
```

### 5. Combine Related Commands

```bash
make deps verify build test   # Chain multiple targets
```

## Troubleshooting

### Make Command Not Found

**Linux/macOS**:
```bash
# Ubuntu/Debian
sudo apt-get install make

# macOS
xcode-select --install
# or
brew install make
```

**Windows**:
```bash
# Using Chocolatey
choco install make

# Using WSL (recommended)
wsl --install
```

### Permission Denied

If scripts aren't executable:
```bash
chmod +x scripts/*.sh
```

Or let Make handle it:
```bash
make deploy-package  # Automatically sets permissions
```

### Build Fails

1. Check Go version:
   ```bash
   make verify
   ```

2. Clean and rebuild:
   ```bash
   make clean
   make deps
   make build-deploy
   ```

3. Check for missing dependencies:
   ```bash
   go mod download
   go mod tidy
   ```

## Quick Reference

### Most Common Commands

```bash
# Development
make setup           # First-time setup
make dev             # Complete dev environment
make test            # Run tests
make build           # Build locally

# Deployment
make build-deploy    # Build for deployment
make deploy-package  # Create deployment package
make deploy-verify   # Verify files

# Maintenance
make clean           # Clean build artifacts
make verify          # Verify environment
make help            # Show all commands
```

### One-Liner Deployment

```bash
make deploy-package && scp nimsforest-deploy.tar.gz root@SERVER:/tmp/ && ssh root@SERVER 'bash -s' < scripts/deploy.sh deploy
```

## Related Documentation

- **[Makefile](../Makefile)** - Complete Makefile with all targets
- **[HETZNER_DEPLOYMENT.md](./HETZNER_DEPLOYMENT.md)** - Deployment guide
- **[CI_CD.md](./CI_CD.md)** - CI/CD documentation
- **[README.md](./README.md)** - Project overview

---

**🎯 Always prefer Make commands for consistency and reliability!**
