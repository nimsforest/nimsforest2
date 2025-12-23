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

- **Go 1.22+** (automatically managed by go.mod)
- **Make** (standard on Linux/macOS, install on Windows via chocolatey or WSL)
- **NATS Server** (automatically installed by `make` commands)

## Quick Start

### 1. Clone the Repository

```bash
git clone <repository-url>
cd nimsforest
```

### 2. Run Setup (Recommended)

The setup command ensures your environment is fully configured:

```bash
make setup
```

This will:
- ✅ Verify Go installation (1.22+)
- ✅ Download all Go dependencies
- ✅ Create required project directory structure
- ✅ Install NATS server binary (if not present)
- ✅ Validate configuration files

### 3. Start NATS with JetStream

```bash
make start
```

This will automatically:
- Check if NATS is already running
- Install NATS server binary if not found (auto-detects OS/architecture)
- Start NATS with JetStream enabled
- Create data directory for persistence
- Display connection details and monitoring URLs

Connection details:
- Client connections on `localhost:4222`
- Monitoring UI on `http://localhost:8222`
- JetStream enabled with persistent storage

### 4. Verify NATS is Running

```bash
# Check NATS status with all details
make status

# Or check manually
curl http://localhost:8222/varz
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
├── Makefile            # Build and development commands
├── go.mod              # Go dependencies
└── README.md           # This file
```

## Development

### Available Make Commands

View all available commands:
```bash
make help
```

### Running Tests

```bash
# Run unit tests
make test

# Run integration tests (starts NATS if needed)
make test-integration

# Generate coverage report
make test-coverage
```

### Code Quality

```bash
# Format code
make fmt

# Run linter
make lint

# Run go vet
make vet

# Run all checks
make check
```

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all
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

## Managing NATS

### Stop NATS Server

```bash
make stop
```

### Restart NATS Server

```bash
make restart
```

### Check NATS Status

```bash
make status
```

## Troubleshooting

### NATS won't start

```bash
# Check NATS status
make status

# View NATS logs
tail -f /tmp/nats-server.log

# Verify NATS is installed
make verify

# Reinstall NATS if needed
make install-nats
```

### Connection refused errors

```bash
# Check if NATS is running
make status

# Restart NATS
make restart

# Verify connectivity
curl http://localhost:8222/varz
```

### Clear all JetStream data

```bash
# WARNING: This deletes all data
make clean-data
```

### Full cleanup and fresh start

```bash
# Stop NATS and clean everything
make clean-all

# Start fresh
make setup
make start
```

## Technology Stack

- **Language**: Go 1.23+
- **Messaging**: NATS Server v2.12.3 with JetStream (native binary)
- **Dependencies**: 
  - github.com/nats-io/nats.go v1.48.0
- **Infrastructure**: Native NATS binary managed via Make

## Documentation

For detailed implementation specifications, see:
- `Cursorinstructions.md` - Complete architecture and API specifications
- `TASK_BREAKDOWN.md` - Development task breakdown
- `PROGRESS.md` - Current development status

## License

[Add your license here]

## Contributing

[Add contribution guidelines here]
