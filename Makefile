.PHONY: help setup deps install-nats start stop restart status test test-integration test-coverage build clean fmt lint vet docker-up docker-down docker-logs verify dirs

# Default target
.DEFAULT_GOAL := help

# Variables
BINARY_NAME := forest
NATS_VERSION := 2.12.3
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
	@echo "$(BLUE)🔨 Building $(BINARY_NAME)...$(NC)"
	@if [ -d cmd/forest ]; then \
		go build -o $(BINARY_NAME) ./cmd/forest; \
		echo "$(GREEN)✅ Built: $(BINARY_NAME)$(NC)"; \
	else \
		echo "$(YELLOW)⚠️  cmd/forest not found - this is a library project$(NC)"; \
		echo "$(BLUE)ℹ️  Running go build to verify compilation...$(NC)"; \
		go build ./...; \
		echo "$(GREEN)✅ All packages compile successfully$(NC)"; \
	fi

build-all: ## Build for all platforms
	@echo "$(BLUE)🔨 Building for all platforms...$(NC)"
	@GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64 ./cmd/forest
	@GOOS=linux GOARCH=arm64 go build -o $(BINARY_NAME)-linux-arm64 ./cmd/forest
	@GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME)-darwin-amd64 ./cmd/forest
	@GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME)-darwin-arm64 ./cmd/forest
	@echo "$(GREEN)✅ Built all platforms$(NC)"

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
