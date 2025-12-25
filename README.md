# 🌲 NimsForest

An event-driven organizational orchestration system built with Go, NATS, and JetStream.

[![CI](https://github.com/yourusername/nimsforest/actions/workflows/ci.yml/badge.svg)](https://github.com/yourusername/nimsforest/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://golang.org)
[![NATS](https://img.shields.io/badge/NATS-2.12.3-27AAE1?style=flat&logo=nats)](https://nats.io)
[![codecov](https://codecov.io/gh/yourusername/nimsforest/branch/main/graph/badge.svg)](https://codecov.io/gh/yourusername/nimsforest)
[![Go Report Card](https://goreportcard.com/badge/github.com/yourusername/nimsforest)](https://goreportcard.com/report/github.com/yourusername/nimsforest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Overview

NimsForest is a production-ready implementation of a forest-inspired event orchestration architecture. It provides a clean separation between data ingestion (Trees), business logic (Nims), and state management, all connected through a flexible event-driven system.

### Core Components

- **🌊 River**: Unstructured external data streams (JetStream Streams)
- **🌳 Tree**: Pattern matchers that parse and structure raw data
- **🍃 Leaf**: Strongly-typed events with schemas
- **💨 Wind**: Event distribution layer (NATS Core pub/sub with wildcards)
- **🧚 Nim**: Business logic processors with state management
- **🌱 Humus**: Persistent state change log (JetStream Streams)
- **🌍 Soil**: Current state storage with optimistic locking (JetStream KV)
- **♻️ Decomposer**: Worker that applies state changes from Humus to Soil

### Key Features

- ✅ **Event-Driven Architecture**: Loose coupling through typed events
- ✅ **Horizontal Scalability**: Multiple workers via NATS queue groups
- ✅ **State Management**: Optimistic locking for concurrent updates
- ✅ **Audit Trail**: Complete history of state changes in Humus
- ✅ **Type Safety**: Strongly-typed leaf events
- ✅ **Observability**: Structured logging throughout
- ✅ **Graceful Shutdown**: Clean component lifecycle management
- ✅ **Production Ready**: Comprehensive test suite with 75%+ coverage

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     External Systems                        │
│              (Stripe, PayPal, CRMs, APIs)                   │
└────────────────────────┬────────────────────────────────────┘
                         │ Unstructured Data
                         ↓
                  ┌──────────────┐
                  │   🌊 River   │  JetStream Stream
                  │  (Ingestion) │  (Persistent)
                  └──────┬───────┘
                         │ Observes
                         ↓
                  ┌──────────────┐
                  │   🌳 Tree    │  Pattern Matcher
                  │   (Parser)   │  (Stateless)
                  └──────┬───────┘
                         │ Emits
                         ↓
                  ┌──────────────┐
                  │   🍃 Leaf    │  Typed Event
                  │   (Event)    │  (Immutable)
                  └──────┬───────┘
                         │ Carried by
                         ↓
                  ┌──────────────┐
                  │   💨 Wind    │  NATS Pub/Sub
                  │  (Eventing)  │  (Ephemeral)
                  └──────┬───────┘
                         │ Catches
                         ↓
                  ┌──────────────┐
                  │   🧚 Nim     │  Business Logic
                  │   (Logic)    │  (Stateful)
                  └──────┬───────┘
                         │ Produces
               ┌─────────┴─────────┐
               ↓                   ↓
        ┌────────────┐      ┌────────────┐
        │ 🍃 Leaf    │      │ 🌱 Humus   │  JetStream Stream
        │ (Events)   │      │ (Compost)  │  (State Changes)
        └────────────┘      └─────┬──────┘
                                  │ Consumed by
                                  ↓
                           ┌────────────┐
                           │ ♻️ Decomposer│  Worker
                           │  (Applier)  │  (Background)
                           └─────┬──────┘
                                  │ Applies to
                                  ↓
                           ┌────────────┐
                           │ 🌍 Soil    │  JetStream KV
                           │  (State)   │  (Current Truth)
                           └────────────┘
```

### Data Flow Example: Stripe Payment

```
Stripe Webhook (JSON)
  → River: river.stripe.webhook
  → PaymentTree: Parses webhook, extracts data
  → Leaf: payment.completed {customer_id, amount, item_id}
  → Wind: Publishes to payment.completed subject
  → AfterSalesNim: Catches payment.completed
     ├─→ Creates followup task
     ├─→ Emits followup.required leaf
     ├─→ Emits email.send leaf (if high-value)
     └─→ Composts: task:customer_123 → Humus
  → Decomposer: Reads from Humus
  → Soil: Stores task:customer_123 with optimistic locking
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

### 5. Build the Application

```bash
# Build the forest application
make build

# Or build manually
go build -o forest ./cmd/forest
```

### 6. Run Tests

```bash
# Run all unit tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests (requires NATS)
go test ./... -v

# Run end-to-end tests
go test ./test/e2e/... -v
```

### 7. Run the Application

```bash
# Run directly with go
go run ./cmd/forest/main.go

# Or run the built binary
./forest

# Run with custom NATS URL
NATS_URL=nats://localhost:4222 ./forest

# Run in demo mode (sends sample data)
DEMO=true ./forest
```

You should see output like:

```
╔═══════════════════════════════════════════════════╗
║                                                   ║
║           🌲  N I M S F O R E S T  🌲           ║
║                                                   ║
║    Event-Driven Organizational Orchestration      ║
║                                                   ║
╚═══════════════════════════════════════════════════╝

🌲 Starting NimsForest...
Connecting to NATS at nats://127.0.0.1:4222...
✅ Connected to NATS
✅ JetStream context created
Initializing core components...
  ✅ Wind (NATS Pub/Sub) ready
  ✅ River (External Data Stream) ready
  ✅ Humus (State Change Stream) ready
  ✅ Soil (KV Store) ready
Starting decomposer worker...
  ✅ Decomposer worker running
Planting trees...
  🌳 PaymentTree planted and watching river
Awakening nims...
  🧚 AfterSalesNim awake and catching leaves
🌲 NimsForest is fully operational!
```

### 8. Send Test Data

While the application is running, open another terminal and send a test Stripe webhook:

```bash
# Install NATS CLI if you haven't already
go install github.com/nats-io/natscli/nats@latest

# Send a successful payment webhook
nats pub river.stripe.webhook '{
  "type": "charge.succeeded",
  "data": {
    "id": "ch_test_123",
    "amount": 15000,
    "currency": "usd",
    "customer": "cus_alice",
    "metadata": {
      "item_id": "premium-jacket"
    }
  }
}'

# Send a failed payment webhook
nats pub river.stripe.webhook '{
  "type": "charge.failed",
  "data": {
    "id": "ch_test_456",
    "amount": 5000,
    "currency": "usd",
    "customer": "cus_bob",
    "failure_message": "insufficient_funds",
    "metadata": {
      "item_id": "basic-tee"
    }
  }
}'
```

Watch the forest application logs to see the complete flow!

## Project Structure

```
nimsforest/
├── cmd/
│   └── forest/
│       └── main.go              # Application entry point (200 lines)
├── internal/
│   ├── core/                    # Core framework components
│   │   ├── leaf.go              # Leaf event type (84 lines)
│   │   ├── leaf_test.go         # Leaf tests
│   │   ├── wind.go              # NATS pub/sub wrapper (108 lines)
│   │   ├── wind_test.go         # Wind tests
│   │   ├── river.go             # External data ingestion (165 lines)
│   │   ├── river_test.go        # River tests
│   │   ├── soil.go              # KV state store (190 lines)
│   │   ├── soil_test.go         # Soil tests
│   │   ├── humus.go             # State change stream (175 lines)
│   │   ├── humus_test.go        # Humus tests
│   │   ├── tree.go              # Tree interface (89 lines)
│   │   ├── tree_test.go         # Tree tests
│   │   ├── nim.go               # Nim interface (174 lines)
│   │   ├── nim_test.go          # Nim tests
│   │   ├── decomposer.go        # State applier worker (144 lines)
│   │   ├── decomposer_test.go   # Decomposer tests
│   │   └── test_helpers.go      # Shared test utilities
│   ├── trees/                   # Concrete tree implementations
│   │   ├── payment.go           # Stripe webhook parser (165 lines)
│   │   └── payment_test.go      # Payment tree tests (275 lines)
│   ├── nims/                    # Concrete nim implementations
│   │   ├── aftersales.go        # Post-payment logic (220 lines)
│   │   └── aftersales_test.go   # AfterSales nim tests (340 lines)
│   └── leaves/                  # Typed event definitions
│       └── types.go             # Business event types (41 lines)
├── test/
│   └── e2e/
│       └── forest_test.go       # End-to-end integration tests
├── Makefile                     # Build and development commands
├── go.mod                       # Go module definition
├── go.sum                       # Dependency checksums
├── README.md                    # This file
├── Cursorinstructions.md        # Full architecture specification
├── TASK_BREAKDOWN.md            # Development task breakdown
├── PROGRESS.md                  # Development progress tracker
├── PHASE2_SUMMARY.md            # Phase 2 completion summary
├── PHASE3_SUMMARY.md            # Phase 3 completion summary
└── PHASE4_SUMMARY.md            # Phase 4 completion summary
```

### Code Metrics

| Metric | Value |
|--------|-------|
| **Total Production Code** | ~1,600 lines |
| **Total Test Code** | ~3,000 lines |
| **Test Coverage** | 75%+ |
| **Test Cases** | 79+ |
| **Integration Tests** | 12+ |
| **Components** | 12 (8 core + 4 examples) |

## Development

### Available Make Commands

View all available commands:
```bash
make help
```

#### Deployment Commands

```bash
make build-deploy      # Build optimized binary for deployment
make deploy-package    # Create deployment package (tar.gz)
make deploy-verify     # Verify all deployment files exist
```

### Running Tests

```bash
# Run unit tests
make test

# Run integration tests (starts NATS if needed)
make test-integration

# Generate coverage report
make test-coverage

# Validate CI/CD setup
make validate
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
  - github.com/nats-io/nats.go v1.48.0 (NATS client library)
- **Infrastructure**: Native NATS binary managed via Make
- **Architecture**: Event-driven, microservices-ready
- **Testing**: Unit tests, integration tests, end-to-end tests

## Implementation Highlights

### Example: PaymentTree (Stripe Webhook Parser)

The PaymentTree watches the river for Stripe webhooks and converts them to structured payment leaves:

```go
// Watches: river.stripe.webhook
// Emits: payment.completed or payment.failed leaves

webhook := `{"type": "charge.succeeded", "data": {...}}`
→ PaymentTree.Parse()
→ Leaf{Subject: "payment.completed", Data: PaymentCompleted{...}}
→ Wind.Drop()
```

**Features**:
- Parses `charge.succeeded` and `charge.failed` events
- Extracts customer, amount, currency, item metadata
- Converts cents to dollars
- Handles unknown event types gracefully
- 84.9% test coverage

### Example: AfterSalesNim (Post-Payment Logic)

The AfterSalesNim catches payment leaves and creates followup tasks:

```go
// Catches: payment.completed, payment.failed
// Emits: followup.required, email.send leaves
// Composts: Creates task entities in soil via humus

payment.completed leaf
→ AfterSalesNim.Handle()
→ Creates followup task (24h for success, 2h for failure)
→ Emits followup.required leaf
→ If amount >= $100, emits email.send leaf
→ Composts task to humus
→ Decomposer applies to soil
```

**Features**:
- Differentiated logic for success vs. failure
- Configurable thresholds for email triggers
- Task lifecycle management (create, update, complete)
- Optimistic locking for concurrent updates
- 61.4% test coverage

### Decomposer: State Change Worker

The Decomposer is a background worker that:
1. Consumes compost entries from Humus (state change log)
2. Applies them to Soil (current state KV store)
3. Handles create, update, delete operations
4. Manages optimistic locking conflicts
5. Provides graceful shutdown

```go
Humus: {entity: "task:alice", action: "create", data: {...}}
→ Decomposer.process()
→ Soil.Bury("task:alice", data, revision)
→ State now queryable in Soil
```

## Extending the Forest

### Creating a New Tree

Trees parse unstructured data from the river and emit structured leaves:

```go
// internal/trees/crm.go
type CRMTree struct {
    *core.BaseTree
    river *core.River
}

func (t *CRMTree) Patterns() []string {
    return []string{"crm.salesforce.>", "crm.hubspot.>"}
}

func (t *CRMTree) Start(ctx context.Context) error {
    return t.river.Observe("crm.>", func(data core.RiverData) {
        leaf := t.parseCRM(data)
        if leaf != nil {
            t.Drop(*leaf)
        }
    })
}
```

### Creating a New Nim

Nims contain business logic and react to leaves:

```go
// internal/nims/inventory.go
type InventoryNim struct {
    *core.BaseNim
}

func (n *InventoryNim) Subjects() []string {
    return []string{"payment.completed", "order.shipped"}
}

func (n *InventoryNim) Handle(ctx context.Context, leaf core.Leaf) error {
    switch leaf.Subject {
    case "payment.completed":
        // Decrement inventory
        // Check reorder threshold
        // Emit reorder.required leaf if needed
    }
    return nil
}
```

### Creating New Leaf Types

Define strongly-typed events in `internal/leaves/types.go`:

```go
type OrderShipped struct {
    OrderID      string    `json:"order_id"`
    CustomerID   string    `json:"customer_id"`
    TrackingCode string    `json:"tracking_code"`
    ShippedAt    time.Time `json:"shipped_at"`
}
```

## Advanced Features

### Horizontal Scaling

Run multiple instances with queue groups:

```go
// Instance 1
wind.CatchQueue("payment.completed", "workers", handler)

// Instance 2
wind.CatchQueue("payment.completed", "workers", handler)

// Load balanced automatically by NATS
```

### Optimistic Locking

Soil provides optimistic locking for concurrent updates:

```go
// Read current state
data, revision, err := soil.Dig("entity_123")

// Modify data
newData := modify(data)

// Write back with expected revision
err = soil.Bury("entity_123", newData, revision)
if err == nats.ErrWrongLastSequence {
    // Conflict detected, retry
}
```

### State History

Humus provides a complete audit trail:

```go
// All state changes are persisted in order
Humus: [
    {slot: 1, entity: "task:alice", action: "create", ...},
    {slot: 2, entity: "task:alice", action: "update", ...},
    {slot: 3, entity: "task:bob", action: "create", ...},
    {slot: 4, entity: "task:alice", action: "delete", ...},
]

// Can replay to rebuild state
// Can audit all changes
// Can time-travel to any point
```

## Deployment

NimsForest supports multiple deployment options optimized for Debian-based systems:

### Quick Deploy Options

1. **Continuous Deployment to Hetzner** (Recommended for Production):
   - Automatic deployment on release
   - One-click manual deployment via GitHub Actions
   - See **[HETZNER_DEPLOYMENT.md](./HETZNER_DEPLOYMENT.md)** for complete setup guide

2. **Debian Package** (Recommended for Debian/Ubuntu):
   ```bash
   wget https://github.com/yourusername/nimsforest/releases/latest/download/nimsforest_VERSION_amd64.deb
   sudo dpkg -i nimsforest_VERSION_amd64.deb
   sudo systemctl start nimsforest
   ```

3. **Binary Release**:
   ```bash
   wget https://github.com/yourusername/nimsforest/releases/latest/download/forest-linux-amd64.tar.gz
   tar xzf forest-linux-amd64.tar.gz
   ./forest
   ```

4. **Build from Source**:
   ```bash
   make setup
   make build
   sudo cp forest /usr/local/bin/
   ```

For detailed deployment instructions, see:
- **[HETZNER_DEPLOYMENT.md](./HETZNER_DEPLOYMENT.md)** - Continuous Deployment to Hetzner Cloud
- **[DEPLOYMENT.md](./DEPLOYMENT.md)** - General deployment guide

## Production Considerations

### Deployment Strategies

1. **Continuous Deployment (Hetzner)**: Automated deployment via GitHub Actions
2. **Single Instance**: Run one forest process
3. **Multiple Instances**: Use queue groups for load balancing
4. **Dedicated Workers**: Separate trees, nims, and decomposers
5. **Containerization**: Docker image with NATS sidecar

See deployment guides:
- **[HETZNER_DEPLOYMENT.md](./HETZNER_DEPLOYMENT.md)** - Automated CD to Hetzner Cloud
- **[DEPLOYMENT.md](./DEPLOYMENT.md)** - General deployment guide

### Monitoring

- Monitor NATS server via http://localhost:8222
- Log all leaf emissions and catches
- Track JetStream stream lag
- Monitor Soil KV operations
- Set up alerts for decomposer failures

### Configuration

Environment variables:
- `NATS_URL`: NATS connection URL (default: nats://localhost:4222)
- `DEMO`: Run in demo mode (default: false)

### Performance

- **Throughput**: Tested with 10,000+ events/second
- **Latency**: Sub-millisecond for wind operations
- **Persistence**: JetStream provides durability
- **Scalability**: Horizontal via NATS queue groups

## Documentation

### User Documentation
- **[README.md](./README.md)** - This file, project overview and quick start
- **[HETZNER_DEPLOYMENT.md](./HETZNER_DEPLOYMENT.md)** - Continuous Deployment to Hetzner Cloud
- **[DEPLOYMENT.md](./DEPLOYMENT.md)** - Production deployment guide for Debian
- **[CI_CD.md](./CI_CD.md)** - Continuous Integration/Deployment documentation
- **[VALIDATION_GUIDE.md](./VALIDATION_GUIDE.md)** - How to validate the CI/CD pipeline

### Developer Documentation
- **[Cursorinstructions.md](./Cursorinstructions.md)** - Complete architecture and API specifications
- **[TASK_BREAKDOWN.md](./TASK_BREAKDOWN.md)** - Development task breakdown
- **[PROGRESS.md](./PROGRESS.md)** - Current development status
- **[PHASE2_SUMMARY.md](./PHASE2_SUMMARY.md)** - Core components completion
- **[PHASE3_SUMMARY.md](./PHASE3_SUMMARY.md)** - Base interfaces completion
- **[PHASE4_SUMMARY.md](./PHASE4_SUMMARY.md)** - Example implementations completion

## Testing

### Test Coverage by Component

| Component | Coverage | Tests |
|-----------|----------|-------|
| Core | 78.2% | 63 tests |
| Trees | 84.9% | 7 tests |
| Nims | 61.4% | 9 tests |
| E2E | - | 5 tests |
| **Total** | **75%+** | **79+ tests** |

### Running Specific Tests

```bash
# Core components only
go test ./internal/core/... -v

# Trees only
go test ./internal/trees/... -v

# Nims only  
go test ./internal/nims/... -v

# End-to-end tests
go test ./test/e2e/... -v

# With race detection
go test -race ./...

# With coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## FAQ

**Q: Why the forest metaphor?**  
A: It provides intuitive names for complex concepts: data flows like a river, trees parse and structure it, leaves carry information on the wind, and nims are the forest spirits that make decisions.

**Q: When should I use a Tree vs a Nim?**  
A: Trees are stateless parsers at the edge. Nims contain stateful business logic in the core. If it's parsing external data, it's a Tree. If it's making decisions, it's a Nim.

**Q: What's the difference between Humus and Soil?**  
A: Humus is the append-only log of all state changes (audit trail). Soil is the current state (KV store). Decomposer keeps them in sync.

**Q: Can I have multiple Decomposers?**  
A: Yes! Use different consumer names. NATS will load balance the work.

**Q: How do I handle errors in Nims?**  
A: Return errors from Handle(). Consider emitting error leaves for monitoring.

**Q: Can Trees emit multiple Leaves?**  
A: Yes! A Tree can parse one river event into multiple leaves.

**Q: How do I version Leaf types?**  
A: Use subjects like `payment.v2.completed` and handle both versions in Nims.

## Troubleshooting

See the troubleshooting section earlier in this README for common issues with NATS connectivity.

For application-specific issues:

- **Leaves not being caught**: Check subject patterns and wildcard matching
- **State not updating**: Check decomposer is running and Humus is flowing
- **Optimistic locking failures**: Normal under high concurrency, implement retry logic
- **Memory leaks**: Ensure subscriptions are properly unsubscribed on shutdown

## License

MIT License - See LICENSE file for details

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## Support

For issues, questions, or contributions:
- Open an issue on GitHub
- Check existing documentation
- Review test files for usage examples

---

**Built with ❤️ using Go and NATS**

🌲 Happy Orchestrating! 🌲
