#!/bin/bash
#
# Install Git hooks for NimsForest
# Usage: ./scripts/install-hooks.sh [--pre-commit-framework]
#
# Options:
#   --pre-commit-framework  Install using pre-commit framework (recommended)
#   (no args)               Install standalone bash hooks

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
GIT_HOOKS_DIR="$PROJECT_ROOT/.git/hooks"

echo -e "${BLUE}NimsForest Git Hooks Installer${NC}"
echo "================================="
echo ""

# Check if we're in a git repository
if [ ! -d "$PROJECT_ROOT/.git" ]; then
    echo -e "${RED}Error: Not a git repository${NC}"
    exit 1
fi

# Create hooks directory if it doesn't exist
mkdir -p "$GIT_HOOKS_DIR"

if [ "$1" = "--pre-commit-framework" ]; then
    # ─────────────────────────────────────────────────────────────────────────
    # Option 1: Install using pre-commit framework
    # ─────────────────────────────────────────────────────────────────────────
    echo -e "${BLUE}Installing using pre-commit framework...${NC}"
    echo ""

    # Check if pre-commit is installed
    if ! command -v pre-commit &> /dev/null; then
        echo -e "${YELLOW}pre-commit is not installed.${NC}"
        echo ""
        echo "Installing pre-commit..."

        # Try pip first, then pip3, then pipx
        if command -v pip &> /dev/null; then
            pip install pre-commit
        elif command -v pip3 &> /dev/null; then
            pip3 install pre-commit
        elif command -v pipx &> /dev/null; then
            pipx install pre-commit
        else
            echo -e "${RED}Error: pip/pip3/pipx not found${NC}"
            echo "Please install pre-commit manually: https://pre-commit.com/#install"
            exit 1
        fi
    fi

    # Check if .pre-commit-config.yaml exists
    if [ ! -f "$PROJECT_ROOT/.pre-commit-config.yaml" ]; then
        echo -e "${RED}Error: .pre-commit-config.yaml not found${NC}"
        exit 1
    fi

    # Install pre-commit hooks
    cd "$PROJECT_ROOT"
    pre-commit install
    pre-commit install --hook-type pre-push

    echo ""
    echo -e "${GREEN}✓ pre-commit hooks installed successfully!${NC}"
    echo ""
    echo "Hooks installed:"
    echo "  • pre-commit: Runs on every commit"
    echo "  • pre-push: Runs security scan before push"
    echo ""
    echo "Commands:"
    echo "  • pre-commit run --all-files  # Run all hooks on all files"
    echo "  • pre-commit run <hook-id>    # Run specific hook"
    echo "  • pre-commit autoupdate       # Update hook versions"
    echo ""

else
    # ─────────────────────────────────────────────────────────────────────────
    # Option 2: Install standalone bash hooks
    # ─────────────────────────────────────────────────────────────────────────
    echo -e "${BLUE}Installing standalone bash hooks...${NC}"
    echo ""

    # Install pre-commit hook
    PRE_COMMIT_HOOK="$GIT_HOOKS_DIR/pre-commit"
    if [ -f "$PRE_COMMIT_HOOK" ]; then
        echo -e "${YELLOW}Backing up existing pre-commit hook...${NC}"
        mv "$PRE_COMMIT_HOOK" "$PRE_COMMIT_HOOK.backup.$(date +%Y%m%d%H%M%S)"
    fi

    cp "$SCRIPT_DIR/pre-commit" "$PRE_COMMIT_HOOK"
    chmod +x "$PRE_COMMIT_HOOK"

    echo -e "${GREEN}✓ pre-commit hook installed${NC}"
    echo ""
    echo -e "${GREEN}Installation complete!${NC}"
    echo ""
    echo "The pre-commit hook will run automatically before each commit."
    echo ""
    echo "To bypass (use sparingly): git commit --no-verify"
    echo ""
fi

# ─────────────────────────────────────────────────────────────────────────────
# Verify required tools
# ─────────────────────────────────────────────────────────────────────────────
echo -e "${BLUE}Checking required tools...${NC}"
echo ""

check_tool() {
    local tool=$1
    local install_hint=$2
    if command -v "$tool" &> /dev/null; then
        echo -e "  ${GREEN}✓${NC} $tool"
    else
        echo -e "  ${YELLOW}⚠${NC} $tool not found - $install_hint"
    fi
}

check_tool "go" "Install from https://golang.org/dl/"
check_tool "golangci-lint" "Install from https://golangci-lint.run/usage/install/"
check_tool "gofmt" "Included with Go"

echo ""
echo -e "${GREEN}Done!${NC}"
