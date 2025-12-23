#!/bin/bash
# NimsForest Environment Setup Script
# This script ensures your development environment is fully configured
# Run this after cloning the repository: ./setup.sh

set -e  # Exit on error

echo "🌲 NimsForest Environment Setup"
echo "================================"
echo ""

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print status messages
print_status() {
    echo -e "${GREEN}✅${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠️${NC}  $1"
}

print_error() {
    echo -e "${RED}❌${NC} $1"
}

print_info() {
    echo "ℹ️  $1"
}

# Step 1: Check Go installation
echo "📋 Step 1: Checking Go installation..."
if command -v go > /dev/null 2>&1; then
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    print_status "Go $GO_VERSION is installed"
    
    # Check if version is sufficient (1.22+)
    REQUIRED_VERSION="1.22.0"
    if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" = "$REQUIRED_VERSION" ]; then
        print_status "Go version meets requirements (>= 1.22)"
    else
        print_warning "Go version $GO_VERSION is below recommended 1.22+"
        print_info "Consider upgrading: https://go.dev/dl/"
    fi
else
    print_error "Go is not installed"
    echo "   Please install Go 1.22+ from https://go.dev/dl/"
    exit 1
fi
echo ""

# Step 2: Verify Go modules
echo "📋 Step 2: Verifying Go modules..."
if [ -f "go.mod" ]; then
    print_status "go.mod found"
    
    if go mod verify > /dev/null 2>&1; then
        print_status "Go modules verified"
    else
        print_warning "Go modules verification failed, running go mod tidy..."
        go mod tidy
        print_status "Go modules fixed"
    fi
else
    print_error "go.mod not found"
    exit 1
fi
echo ""

# Step 3: Download Go dependencies
echo "📋 Step 3: Downloading Go dependencies..."
go mod download
print_status "Dependencies downloaded"
echo ""

# Step 4: Verify directory structure
echo "📋 Step 4: Verifying project directory structure..."
REQUIRED_DIRS=(
    "cmd/forest"
    "internal/core"
    "internal/trees"
    "internal/nims"
    "internal/leaves"
)

MISSING_DIRS=()
for dir in "${REQUIRED_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        print_status "$dir exists"
    else
        print_warning "$dir missing, creating..."
        mkdir -p "$dir"
        MISSING_DIRS+=("$dir")
    fi
done

if [ ${#MISSING_DIRS[@]} -eq 0 ]; then
    print_status "All directories exist"
else
    print_status "Created ${#MISSING_DIRS[@]} missing directories"
fi
echo ""

# Step 5: Make scripts executable
echo "📋 Step 5: Ensuring scripts are executable..."
SCRIPTS=(
    "START_NATS.sh"
    "STOP_NATS.sh"
    "setup.sh"
)

for script in "${SCRIPTS[@]}"; do
    if [ -f "$script" ]; then
        chmod +x "$script"
        print_status "$script is executable"
    fi
done
echo ""

# Step 6: Check NATS installation
echo "📋 Step 6: Checking NATS server..."
if command -v nats-server > /dev/null 2>&1; then
    NATS_VERSION=$(nats-server --version 2>&1)
    print_status "NATS server is installed: $NATS_VERSION"
else
    print_warning "NATS server not found"
    print_info "NATS will be automatically installed when you run ./START_NATS.sh"
fi
echo ""

# Step 7: Verify NATS configuration
echo "📋 Step 7: Verifying NATS configuration..."
if [ -f "docker-compose.yml" ]; then
    print_status "docker-compose.yml exists"
fi

if [ -f "START_NATS.sh" ]; then
    print_status "START_NATS.sh exists"
fi

if [ -f "STOP_NATS.sh" ]; then
    print_status "STOP_NATS.sh exists"
fi
echo ""

# Step 8: Check for running NATS server
echo "📋 Step 8: Checking for running NATS server..."
if pgrep -x "nats-server" > /dev/null; then
    print_warning "NATS server is already running (PID: $(pgrep -x nats-server))"
    print_info "Use ./STOP_NATS.sh to stop it if needed"
else
    print_info "NATS server is not running"
    print_info "Start it with: ./START_NATS.sh"
fi
echo ""

# Step 9: Test configuration files
echo "📋 Step 9: Validating configuration files..."
CONFIG_FILES=(
    ".gitignore"
    "README.md"
    "go.mod"
    "go.sum"
)

for file in "${CONFIG_FILES[@]}"; do
    if [ -f "$file" ]; then
        print_status "$file exists"
    else
        print_warning "$file missing"
    fi
done
echo ""

# Final summary
echo "================================"
echo "🎉 Setup Complete!"
echo ""
echo "Next steps:"
echo "  1. Start NATS server:  ./START_NATS.sh"
echo "  2. Run tests:          go test ./..."
echo "  3. Start development:  Begin with Phase 2 tasks"
echo ""
echo "Quick reference:"
echo "  • NATS client:        nats://localhost:4222"
echo "  • NATS monitoring:    http://localhost:8222"
echo "  • Stop NATS:          ./STOP_NATS.sh"
echo "  • Documentation:      README.md"
echo ""
echo "For detailed setup instructions, see: README.md"
echo "For task breakdown, see: TASK_BREAKDOWN.md"
echo ""
