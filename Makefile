.PHONY: help setup deps install-nats start stop restart status test test-integration test-coverage build clean fmt lint vet docker-up docker-down docker-logs verify dirs

# Default target
.DEFAULT_GOAL := help

# Variables
BINARY_NAME := forest
NATS_VERSION := 2.12.3

# Get version from git tags, fallback to "dev" if no tags
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
NATS_PORT := 4222
NATS_MONITOR_PORT := 8222
NATS_DATA_DIR := /tmp/nats-data
NATS_LOG_FILE := /tmp/nats-server.log
GO_FILES := $(shell find . -name '*.go' -type f -not -path "./vendor/*")
GO_PACKAGES := ./...

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

# Detect OS and architecture
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

ifeq ($(UNAME_S),Linux)
	OS := linux
endif
ifeq ($(UNAME_S),Darwin)
	OS := darwin
endif

ifeq ($(UNAME_M),x86_64)
	ARCH := amd64
endif
ifeq ($(UNAME_M),aarch64)
	ARCH := arm64
endif
ifeq ($(UNAME_M),arm64)
	ARCH := arm64
endif
ifeq ($(UNAME_M),armv7l)
	ARCH := arm7
endif

PLATFORM := $(OS)-$(ARCH)
NATS_URL := https://github.com/nats-io/nats-server/releases/download/v$(NATS_VERSION)/nats-server-v$(NATS_VERSION)-$(PLATFORM).tar.gz

##@ General

help: ## Display this help message
	@echo "$(BLUE)NimsForest - Available Make Targets$(NC)"
	@echo "======================================"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make $(BLUE)<target>$(NC)\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  $(BLUE)%-20s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n$(YELLOW)%s$(NC)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo ""

##@ Setup & Installation

setup: deps dirs install-nats verify ## Complete environment setup (recommended for first-time setup)
	@echo "$(GREEN)✅ Setup complete!$(NC)"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Start NATS:  make start"
	@echo "  2. Run tests:   make test"
	@echo "  3. Build:       make build"
	@echo ""

deps: ## Download Go dependencies
	@echo "$(BLUE)📦 Downloading Go dependencies...$(NC)"
	@go mod download
	@go mod tidy
	@echo "$(GREEN)✅ Dependencies downloaded$(NC)"

dirs: ## Create required project directories
	@echo "$(BLUE)📁 Creating project directories...$(NC)"
	@mkdir -p cmd/forest
	@mkdir -p internal/core
	@mkdir -p internal/trees
	@mkdir -p internal/nims
	@mkdir -p internal/leaves
	@echo "$(GREEN)✅ Directories created$(NC)"

install-nats: ## Install NATS server binary (if not already installed)
	@if command -v nats-server > /dev/null 2>&1; then \
		echo "$(GREEN)✅ NATS server already installed: $$(nats-server --version)$(NC)"; \
	else \
		echo "$(BLUE)📦 Installing NATS server v$(NATS_VERSION)...$(NC)"; \
		echo "   Platform: $(PLATFORM)"; \
		TEMP_DIR=$$(mktemp -d); \
		cd $$TEMP_DIR && \
		curl -sSL $(NATS_URL) -o nats-server.tar.gz && \
		tar -xzf nats-server.tar.gz && \
		BINARY=$$(find . -name "nats-server" -type f | head -n 1); \
		if [ -z "$$BINARY" ]; then \
			echo "$(RED)❌ Failed to extract nats-server binary$(NC)"; \
			rm -rf $$TEMP_DIR; \
			exit 1; \
		fi; \
		if sudo -n true 2>/dev/null && sudo mv $$BINARY /usr/local/bin/nats-server 2>/dev/null; then \
			echo "$(GREEN)✅ Installed to /usr/local/bin/nats-server$(NC)"; \
		else \
			mkdir -p $$HOME/bin; \
			mv $$BINARY $$HOME/bin/nats-server; \
			chmod +x $$HOME/bin/nats-server; \
			echo "$(GREEN)✅ Installed to $$HOME/bin/nats-server$(NC)"; \
			echo "$(YELLOW)⚠️  Add $$HOME/bin to your PATH if not already done$(NC)"; \
		fi; \
		rm -rf $$TEMP_DIR; \
		nats-server --version; \
	fi

verify: ## Verify environment setup
	@echo "$(BLUE)🔍 Verifying environment...$(NC)"
	@echo -n "  Go version:     "
	@go version | awk '{print $$3}' || (echo "$(RED)❌ Go not found$(NC)" && exit 1)
	@echo -n "  NATS server:    "
	@if command -v nats-server > /dev/null 2>&1; then \
		nats-server --version; \
	else \
		echo "$(YELLOW)⚠️  Not installed (run: make install-nats)$(NC)"; \
	fi
	@echo -n "  Go modules:     "
	@if go mod verify > /dev/null 2>&1; then \
		echo "$(GREEN)✅ Verified$(NC)"; \
	else \
		echo "$(RED)❌ Failed$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)✅ Environment verified$(NC)"

##@ NATS Server Management

start: install-nats ## Start NATS server with JetStream
	@# Check if NATS is actually running (not just a zombie process)
	@if curl -s http://localhost:$(NATS_MONITOR_PORT)/varz > /dev/null 2>&1; then \
		echo "$(YELLOW)⚠️  NATS server is already running$(NC)"; \
		echo "   Use 'make stop' to stop it first"; \
		exit 1; \
	fi
	@# Clean up any zombie processes
	@pkill -9 -x nats-server 2>/dev/null || true
	@sleep 1
	@echo "$(BLUE)🚀 Starting NATS Server with JetStream...$(NC)"
	@mkdir -p $(NATS_DATA_DIR)
	@nats-server --jetstream --store_dir=$(NATS_DATA_DIR) -p $(NATS_PORT) -m $(NATS_MONITOR_PORT) > $(NATS_LOG_FILE) 2>&1 &
	@sleep 2
	@if pgrep -x nats-server > /dev/null 2>&1; then \
		echo "$(GREEN)✅ NATS Server started successfully!$(NC)"; \
		echo ""; \
		echo "   PID:           $$(pgrep -x nats-server)"; \
		echo "   Client:        nats://localhost:$(NATS_PORT)"; \
		echo "   Monitoring:    http://localhost:$(NATS_MONITOR_PORT)"; \
		echo "   JetStream:     Enabled"; \
		echo "   Data:          $(NATS_DATA_DIR)"; \
		echo "   Logs:          $(NATS_LOG_FILE)"; \
		echo ""; \
		echo "$(BLUE)📊 Quick checks:$(NC)"; \
		echo "   • make status"; \
		echo "   • curl http://localhost:$(NATS_MONITOR_PORT)/varz"; \
		echo "   • curl http://localhost:$(NATS_MONITOR_PORT)/jsz"; \
	else \
		echo "$(RED)❌ Failed to start NATS server$(NC)"; \
		echo "   Check logs: cat $(NATS_LOG_FILE)"; \
		exit 1; \
	fi

stop: ## Stop NATS server
	@echo "$(BLUE)🛑 Stopping NATS Server...$(NC)"
	@# Check if NATS is actually responsive
	@if curl -s http://localhost:$(NATS_MONITOR_PORT)/varz > /dev/null 2>&1; then \
		PID=$$(pgrep -x nats-server | head -1); \
		echo "   Killing PID: $$PID"; \
		pkill -TERM -x nats-server 2>/dev/null || true; \
		sleep 2; \
	fi
	@# Force kill any remaining processes (including zombies)
	@if pgrep -x nats-server > /dev/null 2>&1; then \
		echo "   Force killing remaining processes..."; \
		pkill -9 -x nats-server 2>/dev/null || true; \
		sleep 1; \
	fi
	@# Verify stopped
	@if curl -s http://localhost:$(NATS_MONITOR_PORT)/varz > /dev/null 2>&1; then \
		echo "$(RED)❌ NATS server still responding$(NC)"; \
		exit 1; \
	else \
		echo "$(GREEN)✅ NATS Server stopped$(NC)"; \
	fi

restart: stop start ## Restart NATS server

status: ## Check NATS server status
	@echo "$(BLUE)📊 NATS Server Status$(NC)"
	@echo "====================="
	@if pgrep -x nats-server > /dev/null 2>&1; then \
		echo "Status:        $(GREEN)Running$(NC)"; \
		echo "PID:           $$(pgrep -x nats-server)"; \
		echo "Client Port:   $(NATS_PORT)"; \
		echo "Monitor Port:  $(NATS_MONITOR_PORT)"; \
		echo ""; \
		echo "$(BLUE)Server Info:$(NC)"; \
		curl -s http://localhost:$(NATS_MONITOR_PORT)/varz | head -20 || echo "$(YELLOW)⚠️  Cannot connect to monitoring port$(NC)"; \
	else \
		echo "Status:        $(RED)Not Running$(NC)"; \
		echo ""; \
		echo "Start with:    make start"; \
	fi

##@ Testing

test: ## Run all unit tests
	@echo "$(BLUE)🧪 Running tests...$(NC)"
	@go test -v -race -short $(GO_PACKAGES)

test-e2emvp: ## Run MVP E2E test (embedded NATS, mock brain)
	@echo "$(BLUE)🧪 Running MVP E2E tests...$(NC)"
	@cd test/e2emvp && go test -v -race ./...

test-integration: ## Run integration tests (requires NATS running)
	@echo "$(BLUE)🧪 Running integration tests...$(NC)"
	@if ! pgrep -x nats-server > /dev/null 2>&1; then \
		echo "$(YELLOW)⚠️  NATS server not running, starting it...$(NC)"; \
		$(MAKE) start; \
	fi
	@if [ -f test_nats_connection.go ]; then \
		echo "$(BLUE)Running NATS connection test...$(NC)"; \
		go run test_nats_connection.go; \
	fi
	@go test -v -race $(GO_PACKAGES)

test-coverage: ## Run tests with coverage report
	@echo "$(BLUE)🧪 Running tests with coverage...$(NC)"
	@go test -v -race -coverprofile=coverage.txt -covermode=atomic $(GO_PACKAGES)
	@go tool cover -html=coverage.txt -o coverage.html
	@echo "$(GREEN)✅ Coverage report generated: coverage.html$(NC)"

##@ Building

build: ## Build the application
	@echo "$(BLUE)🔨 Building $(BINARY_NAME) version $(VERSION)...$(NC)"
	@if [ -d cmd/forest ]; then \
		go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/forest; \
		echo "$(GREEN)✅ Built: $(BINARY_NAME) ($(VERSION))$(NC)"; \
	else \
		echo "$(YELLOW)⚠️  cmd/forest not found - this is a library project$(NC)"; \
		echo "$(BLUE)ℹ️  Running go build to verify compilation...$(NC)"; \
		go build ./...; \
		echo "$(GREEN)✅ All packages compile successfully$(NC)"; \
	fi

build-all: ## Build for all platforms
	@echo "$(BLUE)🔨 Building for all platforms (version $(VERSION))...$(NC)"
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 ./cmd/forest
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 ./cmd/forest
	@GOOS=linux GOARCH=arm GOARM=7 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-arm ./cmd/forest
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 ./cmd/forest
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 ./cmd/forest
	@echo "$(GREEN)✅ Built all platforms$(NC)"

build-deploy: ## Build optimized binary for deployment (Linux AMD64)
	@echo "$(BLUE)🔨 Building deployment binary (version $(VERSION))...$(NC)"
	@if [ -d cmd/forest ]; then \
		GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BINARY_NAME) ./cmd/forest; \
		chmod +x $(BINARY_NAME); \
		echo "$(GREEN)✅ Deployment binary ready: $(BINARY_NAME) ($(VERSION))$(NC)"; \
	else \
		echo "$(YELLOW)⚠️  cmd/forest not found - this is a library project$(NC)"; \
		echo "$(BLUE)ℹ️  Verifying all packages compile for linux/amd64...$(NC)"; \
		GOOS=linux GOARCH=amd64 go build ./...; \
		echo "$(GREEN)✅ All packages compile successfully for deployment$(NC)"; \
	fi

run: build ## Build and run the application
	@echo "$(BLUE)▶️  Running $(BINARY_NAME)...$(NC)"
	@./$(BINARY_NAME)

##@ Code Quality

fmt: ## Format Go code
	@echo "$(BLUE)📝 Formatting code...$(NC)"
	@go fmt $(GO_PACKAGES)
	@echo "$(GREEN)✅ Code formatted$(NC)"

lint: ## Run linter (requires golangci-lint)
	@echo "$(BLUE)🔍 Running linter...$(NC)"
	@if command -v golangci-lint > /dev/null 2>&1; then \
		golangci-lint run; \
		echo "$(GREEN)✅ Linting complete$(NC)"; \
	else \
		echo "$(YELLOW)⚠️  golangci-lint not installed$(NC)"; \
		echo "   Install: https://golangci-lint.run/usage/install/"; \
	fi

vet: ## Run go vet
	@echo "$(BLUE)🔍 Running go vet...$(NC)"
	@go vet $(GO_PACKAGES)
	@echo "$(GREEN)✅ Vet complete$(NC)"

check: fmt vet lint ## Run all code quality checks

##@ Git Hooks

install-hooks: ## Install Git pre-commit hooks (requires: pip install pre-commit)
	@echo "$(BLUE)🔗 Installing Git hooks...$(NC)"
	@if command -v pre-commit > /dev/null 2>&1; then \
		pre-commit install; \
		pre-commit install --hook-type pre-push; \
		echo "$(GREEN)✅ Git hooks installed$(NC)"; \
	else \
		echo "$(RED)❌ pre-commit not found$(NC)"; \
		echo "   Install with: pip install pre-commit"; \
		exit 1; \
	fi

uninstall-hooks: ## Remove Git hooks
	@echo "$(BLUE)🔗 Removing Git hooks...$(NC)"
	@if command -v pre-commit > /dev/null 2>&1; then \
		pre-commit uninstall 2>/dev/null || true; \
		pre-commit uninstall --hook-type pre-push 2>/dev/null || true; \
	fi
	@rm -f .git/hooks/pre-commit .git/hooks/pre-push
	@echo "$(GREEN)✅ Git hooks removed$(NC)"

run-hooks: ## Run pre-commit checks manually (without committing)
	@echo "$(BLUE)🔍 Running pre-commit checks...$(NC)"
	@if command -v pre-commit > /dev/null 2>&1; then \
		pre-commit run --all-files; \
	else \
		echo "$(RED)❌ pre-commit not found$(NC)"; \
		echo "   Install with: pip install pre-commit"; \
		exit 1; \
	fi

##@ Cleanup

clean: ## Remove build artifacts and temporary files
	@echo "$(BLUE)🧹 Cleaning...$(NC)"
	@rm -f $(BINARY_NAME)
	@rm -f $(BINARY_NAME)-*
	@rm -f coverage.txt coverage.html
	@rm -f *.test
	@echo "$(GREEN)✅ Cleaned build artifacts$(NC)"

clean-data: ## Remove NATS data directory (WARNING: deletes all JetStream data)
	@echo "$(YELLOW)⚠️  This will delete all JetStream data!$(NC)"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		if pgrep -x nats-server > /dev/null 2>&1; then \
			echo "$(RED)❌ Stop NATS server first: make stop$(NC)"; \
			exit 1; \
		fi; \
		echo "$(BLUE)🧹 Removing NATS data...$(NC)"; \
		rm -rf $(NATS_DATA_DIR); \
		echo "$(GREEN)✅ NATS data removed$(NC)"; \
	else \
		echo "Cancelled."; \
	fi

clean-all: clean clean-data ## Remove all build artifacts and data

##@ Development Workflow

dev: setup start test ## Complete development setup and validation
	@echo "$(GREEN)✅ Development environment ready!$(NC)"
	@echo ""
	@echo "You can now start developing. Quick commands:"
	@echo "  • make status     - Check NATS status"
	@echo "  • make test       - Run tests"
	@echo "  • make build      - Build the application"
	@echo "  • make stop       - Stop NATS"
	@echo ""

ci: deps verify test vet ## Run CI checks (used in continuous integration)
	@echo "$(GREEN)✅ All CI checks passed$(NC)"

##@ Validation

validate-quick: verify test ## Quick validation (prerequisites, modules, tests)
	@echo "$(GREEN)✅ Quick validation passed$(NC)"

validate-build: build ## Validate build process
	@echo "$(BLUE)🔍 Validating build...$(NC)"
	@if [ -f $(BINARY_NAME) ]; then \
		echo "$(GREEN)✅ Binary created: $(BINARY_NAME)$(NC)"; \
		if [ -x $(BINARY_NAME) ]; then \
			echo "$(GREEN)✅ Binary is executable$(NC)"; \
		else \
			echo "$(RED)❌ Binary is not executable$(NC)"; \
			exit 1; \
		fi; \
	else \
		echo "$(YELLOW)⚠️  No binary (library project)$(NC)"; \
		echo "$(GREEN)✅ All packages compile successfully$(NC)"; \
	fi

validate-workflows: ## Validate GitHub Actions workflow files
	@echo "$(BLUE)🔍 Validating workflow files...$(NC)"
	@if [ -f .github/workflows/ci.yml ] && [ -f .github/workflows/release.yml ] && [ -f .github/workflows/debian-package.yml ]; then \
		echo "$(GREEN)✅ All workflow files present$(NC)"; \
	else \
		echo "$(RED)❌ Missing workflow files$(NC)"; \
		exit 1; \
	fi
	@if command -v yamllint > /dev/null 2>&1; then \
		for file in .github/workflows/*.yml; do \
			if yamllint $$file > /dev/null 2>&1; then \
				echo "$(GREEN)✅ $$(basename $$file) syntax valid$(NC)"; \
			else \
				echo "$(RED)❌ $$(basename $$file) syntax invalid$(NC)"; \
				yamllint $$file; \
				exit 1; \
			fi; \
		done; \
	else \
		echo "$(YELLOW)⚠️  yamllint not installed, skipping syntax check$(NC)"; \
	fi

validate-docs: ## Validate documentation files exist
	@echo "$(BLUE)🔍 Checking documentation...$(NC)"
	@MISSING=0; \
	for doc in README.md DEPLOYMENT.md CI_CD.md CI_CD_SETUP.md VALIDATION_GUIDE.md Makefile; do \
		if [ -f $$doc ]; then \
			echo "$(GREEN)✅ $$doc$(NC)"; \
		else \
			echo "$(RED)❌ $$doc missing$(NC)"; \
			MISSING=1; \
		fi; \
	done; \
	if [ $$MISSING -eq 1 ]; then exit 1; fi

validate-config: ## Validate configuration files
	@echo "$(BLUE)🔍 Checking configuration files...$(NC)"
	@MISSING=0; \
	for conf in .golangci.yml .codecov.yml; do \
		if [ -f $$conf ]; then \
			echo "$(GREEN)✅ $$conf$(NC)"; \
		else \
			echo "$(RED)❌ $$conf missing$(NC)"; \
			MISSING=1; \
		fi; \
	done; \
	if [ $$MISSING -eq 1 ]; then exit 1; fi

validate-nats: start ## Validate NATS integration
	@echo "$(BLUE)🔍 Validating NATS integration...$(NC)"
	@sleep 1
	@if curl -f http://localhost:$(NATS_MONITOR_PORT)/varz > /dev/null 2>&1; then \
		echo "$(GREEN)✅ NATS is running and responding$(NC)"; \
	else \
		echo "$(RED)❌ NATS is not responding$(NC)"; \
		exit 1; \
	fi
	@$(MAKE) stop

validate-all: validate-quick validate-build validate-workflows validate-docs validate-config ## Run all validations
	@echo ""
	@echo "$(GREEN)╔═══════════════════════════════════════════════════════════════════════╗$(NC)"
	@echo "$(GREEN)║                                                                       ║$(NC)"
	@echo "$(GREEN)║                 ✅ ALL VALIDATIONS PASSED! ✅                        ║$(NC)"
	@echo "$(GREEN)║                                                                       ║$(NC)"
	@echo "$(GREEN)║              Your CI/CD pipeline is ready to use!                    ║$(NC)"
	@echo "$(GREEN)║                                                                       ║$(NC)"
	@echo "$(GREEN)╚═══════════════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Push to GitHub to trigger CI"
	@echo "  2. Create a test tag: git tag v0.0.1-test"
	@echo "  3. Review VALIDATION_GUIDE.md for detailed testing"

validate: validate-all ## Alias for validate-all

##@ Deployment

deploy-package: build-deploy ## Create deployment package
	@echo "$(BLUE)📦 Creating deployment package...$(NC)"
	@mkdir -p deploy
	@if [ -f $(BINARY_NAME) ]; then \
		cp $(BINARY_NAME) deploy/; \
	fi
	@cp Makefile deploy/ 2>/dev/null || echo "$(YELLOW)⚠️  Makefile not copied$(NC)"
	@cp scripts/systemd/nimsforest.service deploy/ 2>/dev/null || echo "$(YELLOW)⚠️  service file not found$(NC)"
	@tar czf nimsforest-deploy.tar.gz deploy/
	@rm -rf deploy/
	@echo "$(GREEN)✅ Deployment package created: nimsforest-deploy.tar.gz$(NC)"

deploy-verify: ## Verify deployment files exist
	@echo "$(BLUE)🔍 Verifying deployment files...$(NC)"
	@if [ -f Makefile ]; then \
		echo "$(GREEN)✅ Makefile$(NC)"; \
	else \
		echo "$(RED)❌ Makefile missing$(NC)"; \
		exit 1; \
	fi
	@if [ -f scripts/setup-server.sh ]; then \
		echo "$(GREEN)✅ setup-server.sh$(NC)"; \
	else \
		echo "$(RED)❌ setup-server.sh missing$(NC)"; \
		exit 1; \
	fi
	@if [ -f scripts/systemd/nimsforest.service ]; then \
		echo "$(GREEN)✅ nimsforest.service$(NC)"; \
	else \
		echo "$(RED)❌ nimsforest.service missing$(NC)"; \
		exit 1; \
	fi
	@if [ -f .github/workflows/deploy.yml ]; then \
		echo "$(GREEN)✅ deploy.yml$(NC)"; \
	else \
		echo "$(RED)❌ deploy.yml missing$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)✅ All deployment files present$(NC)"

##@ Server Operations (run on server)

SERVER_BINARY := /usr/local/bin/forest
SERVER_SERVICE := nimsforest
SERVER_USER := forest
SERVER_DATA_DIR := /var/lib/nimsforest
SERVER_LOG_DIR := /var/log/nimsforest
SERVER_BACKUP_DIR := /opt/nimsforest/backups
SERVER_SERVICE_FILE := /etc/systemd/system/$(SERVER_SERVICE).service

server-deploy: ## Deploy on server (run this on the server)
	@echo "$(BLUE)🚀 Starting server deployment...$(NC)"
	@if [ "$$(id -u)" -ne 0 ]; then \
		echo "$(RED)❌ Must run as root$(NC)"; \
		exit 1; \
	fi
	@$(MAKE) server-create-user
	@$(MAKE) server-create-dirs
	@$(MAKE) server-backup
	@$(MAKE) server-stop
	@$(MAKE) server-install-binary
	@$(MAKE) server-install-service
	@$(MAKE) server-start
	@$(MAKE) server-verify
	@echo "$(GREEN)✅ Deployment completed successfully!$(NC)"

server-create-user: ## Create service user
	@if ! id $(SERVER_USER) &>/dev/null; then \
		echo "$(BLUE)Creating user $(SERVER_USER)...$(NC)"; \
		useradd -r -s /bin/false -d $(SERVER_DATA_DIR) $(SERVER_USER); \
	fi

server-create-dirs: ## Create required directories
	@echo "$(BLUE)Creating directories...$(NC)"
	@mkdir -p $(SERVER_DATA_DIR) $(SERVER_LOG_DIR) $(SERVER_BACKUP_DIR)
	@chown -R $(SERVER_USER):$(SERVER_USER) $(SERVER_DATA_DIR) $(SERVER_LOG_DIR)

server-backup: ## Backup current binary
	@if [ -f $(SERVER_BINARY) ]; then \
		echo "$(BLUE)Backing up current binary...$(NC)"; \
		cp $(SERVER_BINARY) $(SERVER_BACKUP_DIR)/forest.backup.$$(date +%Y%m%d_%H%M%S); \
		cp $(SERVER_BINARY) $(SERVER_BACKUP_DIR)/forest.backup; \
	fi

server-stop: ## Stop service
	@echo "$(BLUE)Stopping service...$(NC)"
	@systemctl stop $(SERVER_SERVICE) 2>/dev/null || true

server-install-binary: ## Install binary
	@if [ -f ./$(BINARY_NAME) ]; then \
		echo "$(BLUE)Installing binary...$(NC)"; \
		cp ./$(BINARY_NAME) $(SERVER_BINARY); \
		chmod +x $(SERVER_BINARY); \
		chown root:root $(SERVER_BINARY); \
	else \
		echo "$(RED)❌ Binary not found$(NC)"; \
		exit 1; \
	fi

server-install-service: ## Install systemd service
	@echo "$(BLUE)Installing systemd service...$(NC)"
	@if [ -f ./nimsforest.service ]; then \
		cp ./nimsforest.service $(SERVER_SERVICE_FILE); \
	elif [ -f ../scripts/systemd/nimsforest.service ]; then \
		cp ../scripts/systemd/nimsforest.service $(SERVER_SERVICE_FILE); \
	else \
		echo "$(YELLOW)Creating default service file...$(NC)"; \
		printf '[Unit]\nDescription=NimsForest Event Orchestration System\nAfter=network.target nats.service\nWants=nats.service\n\n[Service]\nType=simple\nUser=%s\nGroup=%s\nWorkingDirectory=%s\nEnvironment="NATS_URL=nats://localhost:4222"\nExecStart=%s\nRestart=on-failure\nRestartSec=10\nStandardOutput=journal\nStandardError=journal\nSyslogIdentifier=nimsforest\n\nNoNewPrivileges=true\nPrivateTmp=true\nProtectSystem=strict\nProtectHome=true\nReadWritePaths=%s %s\n\n[Install]\nWantedBy=multi-user.target\n' \
			"$(SERVER_USER)" "$(SERVER_USER)" "$(SERVER_DATA_DIR)" "$(SERVER_BINARY)" "$(SERVER_DATA_DIR)" "$(SERVER_LOG_DIR)" \
			> $(SERVER_SERVICE_FILE); \
	fi
	@chmod 644 $(SERVER_SERVICE_FILE)
	@systemctl daemon-reload

server-start: ## Start service
	@echo "$(BLUE)Starting service...$(NC)"
	@systemctl enable $(SERVER_SERVICE)
	@systemctl start $(SERVER_SERVICE)
	@sleep 2

server-verify: ## Verify deployment
	@echo "$(BLUE)Verifying deployment...$(NC)"
	@if systemctl is-active --quiet $(SERVER_SERVICE); then \
		echo "$(GREEN)✅ Service is running$(NC)"; \
		systemctl status $(SERVER_SERVICE) --no-pager || true; \
	else \
		echo "$(RED)❌ Service failed to start$(NC)"; \
		journalctl -u $(SERVER_SERVICE) -n 20 --no-pager; \
		exit 1; \
	fi

server-rollback: ## Rollback to previous version
	@echo "$(YELLOW)⚠️  Rolling back...$(NC)"
	@if [ -f $(SERVER_BACKUP_DIR)/forest.backup ]; then \
		systemctl stop $(SERVER_SERVICE); \
		cp $(SERVER_BACKUP_DIR)/forest.backup $(SERVER_BINARY); \
		chmod +x $(SERVER_BINARY); \
		systemctl start $(SERVER_SERVICE); \
		echo "$(GREEN)✅ Rollback complete$(NC)"; \
	else \
		echo "$(RED)❌ No backup found$(NC)"; \
		exit 1; \
	fi

server-status: ## Check service status
	@systemctl status $(SERVER_SERVICE) --no-pager

server-logs: ## View service logs
	@journalctl -u $(SERVER_SERVICE) -f

server-restart: ## Restart service
	@systemctl restart $(SERVER_SERVICE)
