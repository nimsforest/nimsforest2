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
- **Docker & Docker Compose** (for running NATS)

## Quick Start

### 1. Clone the Repository

```bash
git clone <repository-url>
cd nimsforest
```

### 2. Start NATS with JetStream

```bash
docker-compose up -d
```

This will start NATS with:
- Client connections on `localhost:4222`
- Monitoring UI on `http://localhost:8222`
- JetStream enabled with persistent storage

### 3. Verify NATS is Running

```bash
# Check container status
docker-compose ps

# Check NATS monitoring UI
curl http://localhost:8222/varz

# Or visit in browser:
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
├── docker-compose.yml  # NATS infrastructure
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

# Run integration tests (requires NATS)
docker-compose up -d
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
docker-compose logs nats
```

### Connection refused errors

```bash
# Ensure NATS container is running
docker-compose ps

# Restart NATS
docker-compose restart nats
```

### Clear all JetStream data

```bash
# Stop and remove volumes
docker-compose down -v

# Start fresh
docker-compose up -d
```

## Technology Stack

- **Language**: Go 1.23+
- **Messaging**: NATS Server with JetStream
- **Dependencies**: 
  - github.com/nats-io/nats.go v1.48.0
- **Infrastructure**: Docker Compose

## Documentation

For detailed implementation specifications, see:
- `Cursorinstructions.md` - Complete architecture and API specifications
- `TASK_BREAKDOWN.md` - Development task breakdown
- `PROGRESS.md` - Current development status

## License

[Add your license here]

## Contributing

[Add contribution guidelines here]
