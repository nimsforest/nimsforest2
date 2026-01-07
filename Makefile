.PHONY: help deps build build-go build-deploy web-build run test clean fmt vet lint

.DEFAULT_GOAL := help

# Variables
BINARY_NAME := forest
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

##@ General

help: ## Display this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make <target>\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  %-20s %s\n", $$1, $$2 } /^##@/ { printf "\n%s\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

deps: ## Download Go dependencies
	@go mod download
	@go mod tidy

test: ## Run all tests
	@go test -v -race ./...

fmt: ## Format Go code
	@go fmt ./...

vet: ## Run go vet
	@go vet ./...

lint: ## Run linter (requires golangci-lint)
	@golangci-lint run

##@ Building

web-build: ## Build the web frontend
	@if [ -d web ]; then cd web && npm install --silent && npm run build; fi

build: web-build ## Build the application (includes web frontend)
	@if [ -d cmd/forest ]; then \
		go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/forest; \
	else \
		go build ./...; \
	fi

build-go: ## Build only the Go binary (skip web frontend rebuild)
	@if [ -d cmd/forest ]; then \
		go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/forest; \
	else \
		go build ./...; \
	fi

build-deploy: web-build ## Build optimized binary for deployment (Linux AMD64)
	@GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BINARY_NAME) ./cmd/forest
	@chmod +x $(BINARY_NAME)

run: build ## Build and run the application
	@./$(BINARY_NAME)

##@ Cleanup

clean: ## Remove build artifacts
	@rm -f $(BINARY_NAME) $(BINARY_NAME)-* coverage.txt coverage.html *.test
