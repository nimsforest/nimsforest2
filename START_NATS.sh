#!/bin/bash
# Quick script to start NATS server for development
# Alternative to docker-compose when Docker is not available
# Automatically installs NATS binary if not found

set -e  # Exit on error

NATS_VERSION="2.12.3"

# Function to detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case "$ARCH" in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        armv7l)
            ARCH="arm7"
            ;;
        *)
            echo "❌ Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    
    case "$OS" in
        linux|darwin)
            ;;
        *)
            echo "❌ Unsupported OS: $OS"
            exit 1
            ;;
    esac
    
    echo "${OS}-${ARCH}"
}

# Function to install NATS server
install_nats() {
    echo "📦 NATS server not found. Installing version ${NATS_VERSION}..."
    
    PLATFORM=$(detect_platform)
    DOWNLOAD_URL="https://github.com/nats-io/nats-server/releases/download/v${NATS_VERSION}/nats-server-v${NATS_VERSION}-${PLATFORM}.tar.gz"
    TEMP_DIR=$(mktemp -d)
    
    echo "   Platform: ${PLATFORM}"
    echo "   Downloading from: ${DOWNLOAD_URL}"
    
    cd "$TEMP_DIR"
    if curl -sSL "$DOWNLOAD_URL" -o nats-server.tar.gz; then
        tar -xzf nats-server.tar.gz
        
        # Find the nats-server binary
        BINARY=$(find . -name "nats-server" -type f | head -n 1)
        
        if [ -z "$BINARY" ]; then
            echo "❌ Failed to extract nats-server binary"
            rm -rf "$TEMP_DIR"
            exit 1
        fi
        
        # Try to install to /usr/local/bin first, fall back to ~/bin
        if sudo -n true 2>/dev/null && sudo mv "$BINARY" /usr/local/bin/nats-server 2>/dev/null; then
            echo "✅ Installed to /usr/local/bin/nats-server"
        else
            mkdir -p "$HOME/bin"
            mv "$BINARY" "$HOME/bin/nats-server"
            chmod +x "$HOME/bin/nats-server"
            export PATH="$HOME/bin:$PATH"
            echo "✅ Installed to $HOME/bin/nats-server"
            echo "   Added $HOME/bin to PATH for this session"
            echo "   Add 'export PATH=\"\$HOME/bin:\$PATH\"' to your ~/.bashrc or ~/.zshrc"
        fi
        
        rm -rf "$TEMP_DIR"
        
        # Verify installation
        if command -v nats-server > /dev/null; then
            echo "✅ NATS server $(nats-server --version) installed successfully"
        else
            echo "❌ Installation verification failed"
            exit 1
        fi
    else
        echo "❌ Failed to download NATS server"
        rm -rf "$TEMP_DIR"
        exit 1
    fi
}

echo "🚀 Starting NATS Server with JetStream..."

# Check if NATS is already running
if pgrep -x "nats-server" > /dev/null; then
    echo "⚠️  NATS server is already running!"
    echo "   PID: $(pgrep -x nats-server)"
    echo "   Use './STOP_NATS.sh' to stop it first"
    exit 1
fi

# Check if nats-server is installed, install if not
if ! command -v nats-server > /dev/null 2>&1; then
    install_nats
fi

# Create data directory if it doesn't exist
mkdir -p /tmp/nats-data

# Start NATS server
# Configuration matches docker-compose.yml:
# - JetStream enabled
# - Client port: 4222
# - Monitoring port: 8222
# - Data persistence: /tmp/nats-data
nats-server --jetstream --store_dir=/tmp/nats-data -p 4222 -m 8222 > /tmp/nats-server.log 2>&1 &

sleep 2

# Verify it started
if pgrep -x "nats-server" > /dev/null; then
    PID=$(pgrep -x "nats-server")
    echo "✅ NATS Server started successfully!"
    echo ""
    echo "   PID:           $PID"
    echo "   Client:        nats://localhost:4222"
    echo "   Monitoring:    http://localhost:8222"
    echo "   JetStream:     Enabled"
    echo "   Data:          /tmp/nats-data"
    echo "   Logs:          /tmp/nats-server.log"
    echo ""
    echo "📊 Quick checks:"
    echo "   • curl http://localhost:8222/varz"
    echo "   • curl http://localhost:8222/jsz"
    echo ""
    echo "🛑 To stop: ./STOP_NATS.sh or kill $PID"
else
    echo "❌ Failed to start NATS server"
    echo "   Check logs: cat /tmp/nats-server.log"
    exit 1
fi
