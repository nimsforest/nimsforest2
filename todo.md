# Embedded NATS Implementation

## Overview

Embed NATS server directly into the nimsforest binary. Every nimsforest instance:
1. Starts an embedded NATS server with JetStream
2. Registers itself in the Morpheus registry
3. Discovers and connects to other peers from the registry
4. Runs the nimsforest application

**Always cluster mode** - a single node is simply a cluster of one, ready for peers to join.

## Architecture

```
                         Morpheus Registry
                    ┌─────────────────────────┐
                    │  nimsforest-1: host1:6222│
                    │  nimsforest-2: host2:6222│
                    │  nimsforest-3: host3:6222│
                    └────────────┬────────────┘
                          register│& discover
              ┌──────────────────┼──────────────────┐
              │                  │                  │
              ▼                  ▼                  ▼
┌─────────────────────┐ ┌─────────────────────┐ ┌─────────────────────┐
│   nimsforest (1)    │ │   nimsforest (2)    │ │   nimsforest (3)    │
│  ┌───────────────┐  │ │  ┌───────────────┐  │ │  ┌───────────────┐  │
│  │ Embedded NATS │◄─┼─┼─►│ Embedded NATS │◄─┼─┼─►│ Embedded NATS │  │
│  │  JetStream    │  │ │  │  JetStream    │  │ │  │  JetStream    │  │
│  └───────────────┘  │ │  └───────────────┘  │ │  └───────────────┘  │
│  ┌───────────────┐  │ │  ┌───────────────┐  │ │  ┌───────────────┐  │
│  │  Application  │  │ │  │  Application  │  │ │  │  Application  │  │
│  └───────────────┘  │ │  └───────────────┘  │ │  └───────────────┘  │
└─────────────────────┘ └─────────────────────┘ └─────────────────────┘

        First node starts as cluster of 1, others join via registry
```

## Startup Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                        nimsforest startup                            │
├─────────────────────────────────────────────────────────────────────┤
│  1. Start embedded NATS server (cluster mode)                        │
│  2. Register self in Morpheus registry                               │
│  3. Query registry for existing peers                                │
│  4. Connect to discovered peers (if any)                             │
│  5. Wait for NATS cluster to stabilize                               │
│  6. Initialize JetStream streams (with appropriate replication)      │
│  7. Start nimsforest application (Trees, Nims, etc.)                 │
│  8. Ready to serve                                                   │
└─────────────────────────────────────────────────────────────────────┘

First node: Steps 3-4 find no peers → cluster of 1 → R1 replication
Later nodes: Steps 3-4 find peers → join cluster → R{n} replication
```

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
      NodeName    string   // Unique node identifier
      ClusterName string   // Cluster name (default: "nimsforest")
      DataDir     string   // JetStream storage directory
      ClientPort  int      // Client connections (default 4222)
      ClusterPort int      // Cluster communication (default 6222)
      MonitorPort int      // HTTP monitoring (default 8222, 0 to disable)
      Peers       []string // Initial peers (can be empty for first node)
  }

  type EmbeddedNATS struct { ... }

  func New(cfg Config) (*EmbeddedNATS, error)
  func (e *EmbeddedNATS) Start() error
  func (e *EmbeddedNATS) ClientConn() (*nats.Conn, error)  // In-process connection
  func (e *EmbeddedNATS) AddRoute(peer string) error       // Dynamic peer addition
  func (e *EmbeddedNATS) ClusterSize() int                 // Current cluster size
  func (e *EmbeddedNATS) Shutdown() error
  ```
- **Implementation**:
  - [ ] Always configure as cluster mode (even with 0 peers)
  - [ ] Enable JetStream
  - [ ] Support dynamic route addition (for registry-discovered peers)

#### 1.3 Dynamic peer management
- **File**: `internal/natsembed/peers.go` (new file)
- **Features**:
  - [ ] Add peer route at runtime
  - [ ] Handle peer connection/disconnection events
  - [ ] Report cluster membership changes

---

### Phase 2: Registry Integration

#### 2.1 Define registry interface
- **File**: `internal/registry/registry.go` (new file)
- **Interface**:
  ```go
  package registry

  type NodeInfo struct {
      Name        string    // Node identifier
      ClusterAddr string    // host:port for cluster communication
      RegisteredAt time.Time
  }

  type Registry interface {
      // Register this node in the registry
      Register(ctx context.Context, info NodeInfo) error
      
      // Deregister this node (on shutdown)
      Deregister(ctx context.Context) error
      
      // Get all registered peers (excludes self)
      GetPeers(ctx context.Context) ([]NodeInfo, error)
      
      // Watch for peer changes (new nodes joining/leaving)
      WatchPeers(ctx context.Context) (<-chan PeerEvent, error)
  }

  type PeerEvent struct {
      Type string   // "added" or "removed"
      Node NodeInfo
  }
  ```

#### 2.2 Implement Morpheus registry client
- **File**: `internal/registry/morpheus.go` (new file)
- **Implementation**:
  - [ ] Connect to Morpheus registry API
  - [ ] Register node on startup
  - [ ] Deregister on shutdown
  - [ ] Query existing peers
  - [ ] Watch for peer changes
  - [ ] Handle registry connection failures gracefully

#### 2.3 Add registry configuration
- **Environment variables**:
  - [ ] `REGISTRY_URL` - Morpheus registry endpoint
  - [ ] `REGISTRY_SERVICE_NAME` - Service name to register under (default: "nimsforest")
  - [ ] `REGISTRY_HEARTBEAT_INTERVAL` - Keep-alive interval (default: 10s)

---

### Phase 3: Cluster Coordination

#### 3.1 Create cluster manager
- **File**: `internal/cluster/manager.go` (new file)
- **Responsibilities**:
  ```go
  package cluster

  type Manager struct { ... }

  func NewManager(nats *natsembed.EmbeddedNATS, reg registry.Registry) *Manager
  
  // Start the cluster manager
  func (m *Manager) Start(ctx context.Context) error
  
  // Called when registry reports new peer
  func (m *Manager) OnPeerDiscovered(peer registry.NodeInfo)
  
  // Called when registry reports peer gone
  func (m *Manager) OnPeerLost(peer registry.NodeInfo)
  
  // Get current cluster size
  func (m *Manager) ClusterSize() int
  
  // Get recommended replication factor
  func (m *Manager) ReplicationFactor() int  // min(clusterSize, 3)
  ```

#### 3.2 Implement peer discovery loop
- **File**: `internal/cluster/discovery.go` (new file)
- **Behavior**:
  - [ ] On startup: query registry, connect to all existing peers
  - [ ] Watch registry for changes
  - [ ] Add routes when new peers join
  - [ ] Handle peer departures gracefully
  - [ ] Log cluster membership changes

#### 3.3 Dynamic replication adjustment
- **Files**: `internal/core/river.go`, `internal/core/humus.go`, `internal/core/soil.go`
- **Changes**:
  - [ ] Accept cluster manager as dependency
  - [ ] Query cluster size when creating streams
  - [ ] Set replication factor: `min(clusterSize, 3)`
  - [ ] Single node = R1, two nodes = R2, three+ = R3

---

### Phase 4: Update Main Application

#### 4.1 Integrate all components in main.go
- **File**: `cmd/forest/main.go`
- **Startup sequence**:
  ```go
  func runForest() {
      // 1. Parse configuration
      cfg := loadConfig()
      
      // 2. Start embedded NATS (cluster mode, even if first node)
      nats := natsembed.New(cfg.NATS)
      nats.Start()
      
      // 3. Connect to registry
      reg := registry.NewMorpheus(cfg.Registry)
      
      // 4. Start cluster manager (handles peer discovery)
      cluster := cluster.NewManager(nats, reg)
      cluster.Start(ctx)
      
      // 5. Register self in registry
      reg.Register(ctx, nodeInfo)
      
      // 6. Initialize core components with cluster-aware replication
      river := core.NewRiver(nats.JetStream(), cluster.ReplicationFactor())
      // ... etc
      
      // 7. Start application
      // ... trees, nims, etc
      
      // 8. Wait for shutdown
      <-sigChan
      
      // 9. Cleanup
      reg.Deregister(ctx)
      nats.Shutdown()
  }
  ```

#### 4.2 Update configuration loading
- **File**: `cmd/forest/config.go` (new file)
- **Sources** (in priority order):
  - [ ] Command-line flags
  - [ ] Environment variables
  - [ ] Config file (optional)
  - [ ] Defaults

---

### Phase 5: Systemd Integration

#### 5.1 Update systemd service file
- **File**: `scripts/systemd/nimsforest.service`
- **Changes**:
  - [ ] Remove NATS service dependencies
  - [ ] Add registry configuration
  - [ ] Ensure proper shutdown for deregistration

#### 5.2 Create environment file template
- **File**: `scripts/systemd/nimsforest.env.template`
- **Contents**:
  ```bash
  # Registry (Morpheus)
  REGISTRY_URL=http://morpheus-registry:8500
  REGISTRY_SERVICE_NAME=nimsforest
  
  # Node identification (typically set by Morpheus)
  NATS_NODE_NAME=${HOSTNAME}
  
  # Storage
  NATS_DATA_DIR=/var/lib/nimsforest/jetstream
  
  # Ports
  NATS_CLIENT_PORT=4222
  NATS_CLUSTER_PORT=6222
  NATS_MONITOR_PORT=8222
  ```

---

### Phase 6: Health & Observability

#### 6.1 Add health endpoints
- **File**: `internal/httputil/health.go` (new file)
- **Endpoints**:
  - [ ] `GET /health` - overall health
  - [ ] `GET /health/cluster` - cluster status with peer list
  - [ ] `GET /health/registry` - registry connection status
  
- **Response example**:
  ```json
  {
    "status": "healthy",
    "node": "nimsforest-1",
    "cluster": {
      "size": 3,
      "peers": ["nimsforest-2", "nimsforest-3"],
      "replication_factor": 3
    },
    "registry": {
      "connected": true,
      "url": "http://morpheus-registry:8500"
    },
    "jetstream": {
      "streams": ["RIVER", "HUMUS"],
      "kv_buckets": ["SOIL"]
    }
  }
  ```

#### 6.2 Logging for cluster events
- **Log events**:
  - [ ] "Starting as cluster node {name}"
  - [ ] "Registered in registry at {url}"
  - [ ] "Discovered peer {name} at {addr}"
  - [ ] "Peer {name} joined cluster (size now {n})"
  - [ ] "Peer {name} left cluster (size now {n})"
  - [ ] "Replication factor adjusted to {n}"

---

### Phase 7: Documentation

#### 7.1 Update README
- **File**: `README.md`
- **Changes**:
  - [ ] Document embedded architecture
  - [ ] Document registry integration
  - [ ] Explain cluster formation
  - [ ] Update configuration reference

#### 7.2 Create deployment guide
- **File**: `docs/DEPLOYMENT.md` (new file)
- **Contents**:
  - [ ] Morpheus integration overview
  - [ ] Registry requirements
  - [ ] Single node deployment (cluster of 1)
  - [ ] Multi-node cluster deployment
  - [ ] Scaling up (adding nodes)
  - [ ] Scaling down (removing nodes)
  - [ ] Troubleshooting

---

### Phase 8: Testing

#### 8.1 Unit tests
- **Files**: `internal/natsembed/*_test.go`, `internal/cluster/*_test.go`
- **Tests**:
  - [ ] Embedded NATS startup/shutdown
  - [ ] Peer addition/removal
  - [ ] Replication factor calculation

#### 8.2 Integration tests
- **File**: `test/e2e/cluster_test.go`
- **Tests**:
  - [ ] Single node operation
  - [ ] Two nodes joining
  - [ ] Three nodes with R3 replication
  - [ ] Node departure handling

#### 8.3 Mock registry for testing
- **File**: `internal/registry/mock.go`
- **Features**:
  - [ ] In-memory registry for tests
  - [ ] Simulate peer events

---

## Configuration Reference

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `REGISTRY_URL` | (required) | Morpheus registry endpoint |
| `REGISTRY_SERVICE_NAME` | `nimsforest` | Service name in registry |
| `REGISTRY_HEARTBEAT_INTERVAL` | `10s` | Registry heartbeat interval |
| `NATS_NODE_NAME` | hostname | Unique node identifier |
| `NATS_DATA_DIR` | `/var/lib/nimsforest/jetstream` | JetStream storage |
| `NATS_CLIENT_PORT` | `4222` | NATS client port |
| `NATS_CLUSTER_PORT` | `6222` | NATS cluster port |
| `NATS_MONITOR_PORT` | `8222` | NATS monitoring port |

### Command-Line Flags

```
nimsforest [flags]

Registry:
  --registry-url string       Morpheus registry URL (required)
  --registry-service string   Service name (default "nimsforest")

Node:
  --node-name string          Node identifier (default: hostname)
  --data-dir string           JetStream data directory
  --client-port int           NATS client port (default 4222)
  --cluster-port int          NATS cluster port (default 6222)
  --monitor-port int          NATS monitor port (default 8222)
```

---

## Deployment Examples

### First Node (Cluster of 1)

```bash
# Morpheus provisions first machine and runs:
nimsforest --registry-url http://morpheus-registry:8500

# Output:
# Starting nimsforest...
# Starting embedded NATS cluster node: nimsforest-1
# Registered in registry at http://morpheus-registry:8500
# No existing peers found - starting as cluster of 1
# JetStream initialized with replication factor 1
# nimsforest is ready
```

### Second Node Joins

```bash
# Morpheus provisions second machine and runs:
nimsforest --registry-url http://morpheus-registry:8500

# Output:
# Starting nimsforest...
# Starting embedded NATS cluster node: nimsforest-2
# Registered in registry at http://morpheus-registry:8500
# Discovered peer: nimsforest-1 at 192.168.1.10:6222
# Connected to peer nimsforest-1
# Cluster size: 2, replication factor: 2
# nimsforest is ready

# Meanwhile, on node 1:
# Peer nimsforest-2 joined cluster (size now 2)
# Replication factor adjusted to 2
```

### Third Node Joins

```bash
# Morpheus provisions third machine and runs:
nimsforest --registry-url http://morpheus-registry:8500

# Output:
# Starting nimsforest...
# Discovered peers: nimsforest-1, nimsforest-2
# Connected to 2 peers
# Cluster size: 3, replication factor: 3
# nimsforest is ready

# On all nodes:
# Cluster size: 3, replication factor: 3
```

---

## Implementation Priority

| Phase | Priority | Effort | Description |
|-------|----------|--------|-------------|
| 1 | Critical | 3-4h | Embed NATS server |
| 2 | Critical | 3-4h | Registry integration |
| 3 | Critical | 2-3h | Cluster coordination |
| 4 | High | 2h | Main application integration |
| 5 | High | 1h | Systemd updates |
| 6 | Medium | 2h | Health endpoints |
| 7 | Medium | 2h | Documentation |
| 8 | Low | 2-3h | Testing |

**Total estimated effort**: 17-21 hours

---

## Registry API Requirements

nimsforest needs the Morpheus registry to support:

1. **Service Registration**
   - Register: `POST /v1/services/{service_name}/nodes`
   - Body: `{"name": "node-1", "address": "192.168.1.10", "port": 6222}`

2. **Service Discovery**
   - List: `GET /v1/services/{service_name}/nodes`
   - Returns: `[{"name": "node-1", "address": "...", "port": ...}, ...]`

3. **Health/Heartbeat**
   - Heartbeat: `PUT /v1/services/{service_name}/nodes/{node_name}/heartbeat`
   - TTL-based expiry if heartbeat stops

4. **Watch (optional, for efficiency)**
   - Watch: `GET /v1/services/{service_name}/nodes?watch=true`
   - Long-poll or SSE for change notifications

If Morpheus uses a different API, the `registry.Morpheus` implementation will adapt to it.

---

## Notes

- **Always cluster mode**: Removes conditional logic, simpler code
- **First node is not special**: It just happens to have no peers initially
- **Dynamic scaling**: Nodes join/leave via registry, NATS handles the rest
- **Replication scales with cluster**: R1 → R2 → R3 as nodes join
- **Graceful degradation**: If registry is temporarily unavailable, existing cluster continues working
- **No manual peer configuration**: Registry is the source of truth
