#!/bin/bash
# Quick script to stop NATS server

echo "🛑 Stopping NATS Server..."

if pgrep -x "nats-server" > /dev/null; then
    PID=$(pgrep -x "nats-server")
    echo "   Killing PID: $PID"
    pkill -x nats-server
    sleep 1
    
    if pgrep -x "nats-server" > /dev/null; then
        echo "   Force killing..."
        pkill -9 -x nats-server
        sleep 1
    fi
    
    if pgrep -x "nats-server" > /dev/null; then
        echo "❌ Failed to stop NATS server"
        exit 1
    else
        echo "✅ NATS Server stopped"
    fi
else
    echo "ℹ️  NATS server is not running"
fi
