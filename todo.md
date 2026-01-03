# Embedded NATS Implementation

## Overview

Embed NATS server directly into the nimsforest binary. Starting nimsforest automatically:
1. Starts an embedded NATS server with JetStream
2. Joins the cluster (if peers configured)
3. Runs the nimsforest application

**Single binary, single process, single deployment artifact.**

## Why Embedded?

| Before (Separate) | After (Embedded) |
|-------------------|------------------|
| Download NATS separately | Single binary |
| GitHub API IPv4 issues | No external downloads |
| Two systemd services | One service |
| Version mismatches possible | Always compatible |
| Two things to monitor | One process |
| Network connection to NATS | In-process (faster) |

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      nimsforest binary                           │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                  Embedded NATS Server                       │ │
│  │  • JetStream enabled                                       │ │
│  │  • Cluster port 6222 (peer communication)                  │ │
│  │  • Client port 4222 (optional, for nats cli debugging)     │ │
│  │  • Monitoring port 8222 (optional, for metrics)            │ │
│  └────────────────────────────────────────────────────────────┘ │
│                              ▲                                   │
│                              │ in-process connection             │
│                              ▼                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                 nimsforest application                      │ │
│  │  Wind ─► River ─► Trees ─► Leaves ─► Nims ─► Humus/Soil    │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                              │
                    Cluster port 6222
                              │
              ┌───────────────┴───────────────┐
              ▼                               ▼
   ┌─────────────────────┐         ┌─────────────────────┐
   │ nimsforest (node 2) │ ◄─────► │ nimsforest (node 3) │
   │  embedded NATS      │         │  embedded NATS      │
   └─────────────────────┘         └─────────────────────┘

        JetStream streams replicated across all nodes
```

## Deployment Model

Morpheus provisions a machine and runs:
```bash
nimsforest --cluster-peers "192.168.1.11:6222,192.168.1.12:6222"
```

That's it. No NATS download, no separate service, no coordination.

---

## Tasks

### Phase 1: Embed NATS Server

#### 1.1 Add nats-server dependency
- **File**: `go.mod`
- **Changes**:
  - [ ] Add `github.com/nats-io/nats-server/v2` dependency
  - [ ] Run `go mod tidy`

#### 1.2 Create embedded server wrapper
- **File**: `internal/natsembed/server.go` (new file)
- **Contents**:
  ```go
  package natsembed
  
  type Config struct {
      NodeName     string   // Unique node identifier
      ClusterPeers []string // Other nodes in cluster (host:port)
      DataDir      string   // JetStream storage directory
      ClientPort   int      // Client connections (default 4222)
      ClusterPort  int      // Cluster communication (default 6222)
      MonitorPort  int      // HTTP monitoring (default 8222, 0 to disable)
  }
  
  type EmbeddedNATS struct {
      server *server.Server
      config Config
  }
  
  func New(cfg Config) (*EmbeddedNATS, error)
  func (e *EmbeddedNATS) Start() error
  func (e *EmbeddedNATS) ClientURL() string      // For in-process connection
  func (e *EmbeddedNATS) WaitForReady() error    // Block until server ready
  func (e *EmbeddedNATS) WaitForCluster() error  // Block until cluster formed
  func (e *EmbeddedNATS) Shutdown() error
  ```
- **Implementation**:
  - [ ] Configure server options from Config
  - [ ] Enable JetStream with configured storage
  - [ ] Configure cluster routes from peers
  - [ ] Handle graceful shutdown

#### 1.3 Add cluster formation logic
- **File**: `internal/natsembed/cluster.go` (new file)
- **Contents**:
  - [ ] Parse peer addresses from comma-separated string
  - [ ] Convert to NATS route URLs (`nats-route://host:port`)
  - [ ] Validate peer addresses
  - [ ] Health check for cluster connectivity

---

### Phase 2: Update Main Application

#### 2.1 Update main.go to start embedded NATS
- **File**: `cmd/forest/main.go`
- **Changes**:
  - [ ] Add command-line flags for cluster configuration
  - [ ] Start embedded NATS before application
  - [ ] Connect to embedded server (in-process URL)
  - [ ] Proper shutdown order (app first, then NATS)
  - [ ] Remove external NATS connection code

#### 2.2 Add configuration via environment variables
- **File**: `cmd/forest/main.go`
- **Environment variables**:
  - [ ] `NATS_NODE_NAME` - node identifier (default: hostname)
  - [ ] `NATS_CLUSTER_PEERS` - comma-separated peer list
  - [ ] `NATS_DATA_DIR` - JetStream storage (default: `/var/lib/nimsforest/jetstream`)
  - [ ] `NATS_CLIENT_PORT` - client port (default: 4222)
  - [ ] `NATS_CLUSTER_PORT` - cluster port (default: 6222)
  - [ ] `NATS_MONITOR_PORT` - monitoring port (default: 8222, 0 to disable)

#### 2.3 Add command-line flags (alternative to env vars)
- **File**: `cmd/forest/main.go`
- **Flags**:
  - [ ] `--node-name` 
  - [ ] `--cluster-peers`
  - [ ] `--data-dir`
  - [ ] `--client-port`
  - [ ] `--cluster-port`
  - [ ] `--monitor-port`

---

### Phase 3: Startup Modes

#### 3.1 Implement standalone mode (single node)
- **Behavior**:
  - [ ] No cluster peers configured
  - [ ] JetStream with single replica (R1)
  - [ ] Good for development and small deployments
  - [ ] Default mode if no peers specified

#### 3.2 Implement cluster mode
- **Behavior**:
  - [ ] Cluster peers configured
  - [ ] Wait for minimum peers before marking ready
  - [ ] JetStream with replication (R3 for 3+ nodes)
  - [ ] Automatic stream configuration for replication

#### 3.3 Update stream creation for replication
- **Files**: `internal/core/river.go`, `internal/core/humus.go`, `internal/core/soil.go`
- **Changes**:
  - [ ] Accept replication factor as parameter
  - [ ] Configure streams with appropriate replicas
  - [ ] Handle single-node vs cluster automatically

---

### Phase 4: Systemd Integration

#### 4.1 Update systemd service file
- **File**: `scripts/systemd/nimsforest.service`
- **Changes**:
  - [ ] Remove NATS service dependency (`Wants=`, `After=nats.service`)
  - [ ] Add environment variables for cluster config
  - [ ] Update `ExecStart` to include flags if needed
  - [ ] Ensure proper data directory permissions

#### 4.2 Create environment file template
- **File**: `scripts/systemd/nimsforest.env.template` (new file)
- **Contents**:
  ```bash
  # Node identification
  NATS_NODE_NAME=nimsforest-1
  
  # Cluster peers (comma-separated, empty for standalone)
  NATS_CLUSTER_PEERS=192.168.1.11:6222,192.168.1.12:6222
  
  # Storage
  NATS_DATA_DIR=/var/lib/nimsforest/jetstream
  
  # Ports (defaults shown)
  # NATS_CLIENT_PORT=4222
  # NATS_CLUSTER_PORT=6222
  # NATS_MONITOR_PORT=8222
  ```

---

### Phase 5: Setup Script Updates

#### 5.1 Simplify setup script
- **File**: `scripts/setup-server.sh`
- **Changes**:
  - [ ] Remove NATS server installation steps
  - [ ] Remove NATS systemd service setup
  - [ ] Keep nimsforest binary installation
  - [ ] Add cluster peer configuration prompts
  - [ ] Generate environment file from template

#### 5.2 Create Morpheus-friendly setup
- **File**: `scripts/morpheus-setup.sh` (new file)
- **Features**:
  - [ ] Accept parameters for automated setup
  - [ ] `--node-name` (required)
  - [ ] `--cluster-peers` (optional, standalone if empty)
  - [ ] `--data-dir` (optional)
  - [ ] Non-interactive mode for automation
  - [ ] Verify setup and report status

---

### Phase 6: Health & Observability

#### 6.1 Add health endpoint
- **File**: `internal/httputil/health.go` (new file)
- **Endpoints**:
  - [ ] `GET /health` - overall health
  - [ ] `GET /health/nats` - NATS server status
  - [ ] `GET /health/cluster` - cluster membership
  - [ ] `GET /health/jetstream` - JetStream status

#### 6.2 Expose NATS monitoring
- **Built-in NATS endpoints** (on monitor port):
  - `/varz` - server statistics
  - `/jsz` - JetStream statistics  
  - `/routez` - cluster route information
  - `/healthz` - NATS health check

---

### Phase 7: Documentation

#### 7.1 Update README
- **File**: `README.md`
- **Changes**:
  - [ ] Document embedded NATS architecture
  - [ ] Update quick start (no separate NATS setup)
  - [ ] Document cluster configuration
  - [ ] Update environment variables list

#### 7.2 Create cluster deployment guide
- **File**: `docs/CLUSTER_DEPLOYMENT.md` (new file)
- **Contents**:
  - [ ] Architecture overview
  - [ ] Standalone vs cluster mode
  - [ ] Step-by-step cluster setup
  - [ ] Morpheus integration guide
  - [ ] Troubleshooting

#### 7.3 Update help output
- **File**: `cmd/forest/main.go`
- **Changes**:
  - [ ] Update `printHelp()` with new flags
  - [ ] Document cluster configuration
  - [ ] Show examples for standalone and cluster modes

---

### Phase 8: Testing

#### 8.1 Unit tests for embedded NATS
- **File**: `internal/natsembed/server_test.go` (new file)
- **Tests**:
  - [ ] Test server startup and shutdown
  - [ ] Test configuration parsing
  - [ ] Test cluster route parsing

#### 8.2 Integration tests
- **File**: `test/e2e/embedded_test.go` (new file)
- **Tests**:
  - [ ] Test standalone mode operation
  - [ ] Test JetStream operations
  - [ ] Test graceful shutdown

---

## Environment Variables (Final)

| Variable | Default | Description |
|----------|---------|-------------|
| `NATS_NODE_NAME` | hostname | Unique node identifier |
| `NATS_CLUSTER_PEERS` | (empty) | Comma-separated `host:port` list |
| `NATS_DATA_DIR` | `/var/lib/nimsforest/jetstream` | JetStream storage |
| `NATS_CLIENT_PORT` | `4222` | Client connection port |
| `NATS_CLUSTER_PORT` | `6222` | Cluster communication port |
| `NATS_MONITOR_PORT` | `8222` | HTTP monitoring (0 to disable) |

## Command-Line Flags

```
nimsforest [flags] [command]

Flags:
  --node-name string       Node identifier (default: hostname)
  --cluster-peers string   Comma-separated peer addresses (host:port)
  --data-dir string        JetStream data directory
  --client-port int        NATS client port (default 4222)
  --cluster-port int       NATS cluster port (default 6222)
  --monitor-port int       NATS monitor port (default 8222, 0 to disable)

Commands:
  run, start      Start nimsforest (default)
  version         Show version
  update          Update to latest version
  help            Show help
```

---

## Deployment Examples

### Standalone (Development / Single Node)

```bash
# Just run it - no configuration needed
nimsforest

# Or with custom data directory
nimsforest --data-dir /data/nimsforest
```

### Cluster (Production)

```bash
# Node 1 (192.168.1.10)
nimsforest --node-name node-1 --cluster-peers "192.168.1.11:6222,192.168.1.12:6222"

# Node 2 (192.168.1.11)
nimsforest --node-name node-2 --cluster-peers "192.168.1.10:6222,192.168.1.12:6222"

# Node 3 (192.168.1.12)
nimsforest --node-name node-3 --cluster-peers "192.168.1.10:6222,192.168.1.11:6222"
```

### Morpheus Provisioning

```bash
# Morpheus can run this on each provisioned machine
nimsforest \
  --node-name "${HOSTNAME}" \
  --cluster-peers "${CLUSTER_PEER_LIST}" \
  --data-dir /var/lib/nimsforest/jetstream
```

---

## Implementation Priority

| Phase | Priority | Effort | Description |
|-------|----------|--------|-------------|
| 1 | Critical | 3-4h | Embed NATS server |
| 2 | Critical | 2-3h | Update main application |
| 3 | High | 2h | Startup modes (standalone/cluster) |
| 4 | High | 1h | Systemd updates |
| 5 | Medium | 1-2h | Setup script simplification |
| 6 | Medium | 2h | Health endpoints |
| 7 | Medium | 2h | Documentation |
| 8 | Low | 2h | Testing |

**Total estimated effort**: 15-18 hours

---

## Migration Path

For existing deployments with separate NATS:

1. Stop nimsforest service
2. Stop NATS service
3. Deploy new nimsforest binary
4. Configure cluster peers (if clustered)
5. Start nimsforest (embedded NATS starts automatically)
6. Remove old NATS service (optional cleanup)

Data migration:
- JetStream data can be preserved if using same data directory
- Or start fresh (streams will be recreated automatically)

---

## Notes

- Binary size increases by ~15-20MB (NATS server code)
- Single process simplifies monitoring and restarts
- In-process connection is faster than network
- External NATS CLI tools still work (connect to client port)
- Cluster formation is automatic when peers are reachable
- JetStream replication factor should match cluster size (R1 for standalone, R3 for 3+ nodes)
