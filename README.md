# NimsForest

An event-driven organizational orchestration system built with Go, NATS, and JetStream.

## Overview

NimsForest is a prototype implementation of a forest-inspired event orchestration architecture where:

- **River**: Unstructured external data streams (JetStream)
- **Tree**: Pattern matchers that parse and structure data
- **Leaf**: Structured events with schemas
- **Wind**: Event distribution layer (NATS Core pub/sub)
- **Nim**: Business logic processors
- **Humus**: Persistent state changes (JetStream)
- **Soil**: Current state storage (JetStream KV)

## Architecture

```
river (webhooks, APIs, raw data)
    ↓
tree (parse, structure)
    ↓
leaf (named event: "payment.completed")
    ↓
wind (carries leaf)
    ↓
nim (business logic)
    ↓
leaf (wind) and/or compost (humus)
    ↓
soil (current state)
```

## Prerequisites

- **Go 1.23+** (automatically managed by go.mod)
- **NATS Server** (native binary, no Docker required)
  - Automatically installed via `START_NATS.sh` or manually downloadable
  - Docker Compose optional for production deployments

## Quick Start

### 1. Clone the Repository

```bash
git clone <repository-url>
cd nimsforest
```

### 2. Run Setup Script (Recommended)

The setup script ensures your environment is fully configured:

```bash
./setup.sh
```

This will:
- ✅ Verify Go installation (1.22+)
- ✅ Download all Go dependencies
- ✅ Ensure project directory structure exists
- ✅ Make all scripts executable
- ✅ Check/install NATS server if needed
- ✅ Validate configuration files

**Note**: The setup script is idempotent - safe to run multiple times.

### 3. Start NATS with JetStream

**Primary Approach: Native Binary (No Docker Required)**
```bash
./START_NATS.sh
```

The START_NATS.sh script will automatically:
- Check if NATS is already running
- Install NATS server binary if not found (auto-detects OS/architecture)
- Start NATS with JetStream enabled
- Create data directory for persistence
- Display connection details and monitoring URLs

**Alternative: Docker Compose (Optional for Production)**
```bash
docker-compose up -d
```

Both approaches provide identical functionality:
- Client connections on `localhost:4222`
- Monitoring UI on `http://localhost:8222`
- JetStream enabled with persistent storage

**Note**: The native binary is the default development approach. Docker Compose is kept for production deployments. See `INFRASTRUCTURE_VERIFICATION.md` for details.

### 4. Verify NATS is Running

```bash
# Check NATS server process
ps aux | grep nats-server

# Check NATS monitoring UI
curl http://localhost:8222/varz

# Check JetStream status
curl http://localhost:8222/jsz

# Or visit monitoring UI in browser:
# http://localhost:8222
```

### 4. Install Go Dependencies

```bash
go mod download
go mod tidy
```

### 5. Build the Application (Once Implemented)

```bash
go build -o forest ./cmd/forest
```

### 6. Run the Application (Once Implemented)

```bash
./forest
```

## Project Structure

```
nimsforest/
├── cmd/
│   └── forest/          # Main application entry point
├── internal/
│   ├── core/           # Core components (Wind, River, Soil, Humus)
│   ├── trees/          # Tree implementations (parsers)
│   ├── nims/           # Nim implementations (business logic)
│   └── leaves/         # Leaf type definitions
├── START_NATS.sh       # Start NATS server (primary)
├── STOP_NATS.sh        # Stop NATS server
├── docker-compose.yml  # NATS infrastructure (optional)
├── go.mod              # Go dependencies
└── README.md           # This file
```

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test ./... -cover

# Run integration tests (requires NATS running)
./START_NATS.sh  # Start if not already running
go test ./... -tags=integration

# Run with race detection
go test ./... -race
```

### Code Formatting

```bash
go fmt ./...
```

### Linting

```bash
golangci-lint run
```

## NATS Connection Details

### Connection String

```go
nats.Connect("nats://localhost:4222")
```

### JetStream Setup

```go
nc, _ := nats.Connect("nats://localhost:4222")
js, _ := nc.JetStream()
```

### Monitoring

- **Monitoring UI**: http://localhost:8222
- **Varz endpoint**: http://localhost:8222/varz
- **Connz endpoint**: http://localhost:8222/connz
- **Jsz endpoint**: http://localhost:8222/jsz (JetStream info)

## Stopping NATS

**Native Binary (Primary):**
```bash
./STOP_NATS.sh
```

**Docker Compose (If Using):**
```bash
# Stop and remove containers
docker-compose down

# Stop and remove containers + volumes (clears all data)
docker-compose down -v
```

## Troubleshooting

### NATS won't start

```bash
# Check if port 4222 or 8222 is already in use
lsof -i :4222
lsof -i :8222

# View NATS logs
tail -f /tmp/nats-server.log

# Check if NATS binary is installed
which nats-server
nats-server --version
```

### Connection refused errors

```bash
# Ensure NATS server is running
ps aux | grep nats-server

# Restart NATS
./STOP_NATS.sh
./START_NATS.sh

# Verify connectivity
nc -zv localhost 4222
curl http://localhost:8222/varz
```

### Clear all JetStream data

```bash
# Stop NATS
./STOP_NATS.sh

# Remove data directory
rm -rf /tmp/nats-data

# Start fresh
./START_NATS.sh
```

## Technology Stack

- **Language**: Go 1.23+
- **Messaging**: NATS Server v2.12.3 with JetStream (native binary)
- **Dependencies**: 
  - github.com/nats-io/nats.go v1.48.0
- **Infrastructure**: Native NATS binary (Docker Compose optional)

## Documentation

For detailed implementation specifications, see:
- `Cursorinstructions.md` - Complete architecture and API specifications
- `TASK_BREAKDOWN.md` - Development task breakdown
- `PROGRESS.md` - Current development status

## License

[Add your license here]

## Contributing

[Add contribution guidelines here]
