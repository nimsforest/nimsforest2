#!/bin/bash
# Quick script to start NATS server for development
# Alternative to docker-compose when Docker is not available

echo "🚀 Starting NATS Server with JetStream..."

# Check if NATS is already running
if pgrep -x "nats-server" > /dev/null; then
    echo "⚠️  NATS server is already running!"
    echo "   PID: $(pgrep -x nats-server)"
    echo "   Use './STOP_NATS.sh' to stop it first"
    exit 1
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
