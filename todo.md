# Embedded NATS Implementation

## Overview

Embed NATS server directly into the nimsforest binary. When nimsforest starts:
1. Reads `/etc/morpheus/node-info.json` for forest ID and registry URL
2. Queries the StorageBox registry for peer nodes in the same forest
3. Starts embedded NATS server and connects to peers
4. Runs the nimsforest application

**Single binary deployment** - Morpheus only needs to provision the machine and deploy the nimsforest binary.

## How It Works

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Morpheus CLI                                 │
│  morpheus plant cloud medium                                         │
└─────────────────────────────────────────────────────────────────────┘
                                │
                                │ 1. Provisions 3 servers via Hetzner API
                                │ 2. Writes node-info.json to each server
                                │ 3. Registers nodes in StorageBox registry
                                │ 4. Deploys nimsforest binary
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    StorageBox Registry (WebDAV)                      │
│  https://uXXXXX.your-storagebox.de/morpheus/registry.json           │
│                                                                      │
│  {                                                                   │
│    "forests": {                                                      │
│      "forest-123": { "id": "forest-123", "size": "medium", ... }    │
│    },                                                                │
│    "nodes": {                                                        │
│      "forest-123": [                                                 │
│        { "id": "node-1", "ip": "2a01:...:1", "status": "active" },  │
│        { "id": "node-2", "ip": "2a01:...:2", "status": "active" },  │
│        { "id": "node-3", "ip": "2a01:...:3", "status": "active" }   │
│      ]                                                               │
│    }                                                                 │
│  }                                                                   │
└─────────────────────────────────────────────────────────────────────┘
                                │
          ┌─────────────────────┼─────────────────────┐
          │                     │                     │
          ▼                     ▼                     ▼
┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
│   Node 1        │   │   Node 2        │   │   Node 3        │
│                 │   │                 │   │                 │
│ /etc/morpheus/  │   │ /etc/morpheus/  │   │ /etc/morpheus/  │
│ node-info.json: │   │ node-info.json: │   │ node-info.json: │
│ {               │   │ {               │   │ {               │
│  "forest_id":   │   │  "forest_id":   │   │  "forest_id":   │
│   "forest-123", │   │   "forest-123", │   │   "forest-123", │
│  "registry_url":│   │  "registry_url":│   │  "registry_url":│
│   "https://..." │   │   "https://..." │   │   "https://..." │
│ }               │   │ }               │   │ }               │
│                 │   │                 │   │                 │
│ ┌─────────────┐ │   │ ┌─────────────┐ │   │ ┌─────────────┐ │
│ │ nimsforest  │◄┼───┼─┤ nimsforest  │◄┼───┼─┤ nimsforest  │ │
│ │ (embedded   │ │   │ │ (embedded   │ │   │ │ (embedded   │ │
│ │  NATS)      │ │   │ │  NATS)      │ │   │ │  NATS)      │ │
│ └─────────────┘ │   │ └─────────────┘ │   │ └─────────────┘ │
└─────────────────┘   └─────────────────┘   └─────────────────┘
        │                     │                     │
        └─────────────────────┴─────────────────────┘
                    NATS Cluster (full mesh)
```

## Startup Sequence

```
nimsforest starts
    │
    ├─► Read /etc/morpheus/node-info.json
    │   • forest_id: "forest-123"
    │   • registry_url: "https://uXXXXX.your-storagebox.de/morpheus/registry.json"
    │   • registry_username: "uXXXXX"
    │   • registry_password: "***"
    │
    ├─► Query StorageBox registry (WebDAV GET)
    │   • Get all nodes for forest-123
    │   • Filter out self (by IP or node ID)
    │   • Result: peer IPs [2a01:...:2, 2a01:...:3]
    │
    ├─► Start embedded NATS server
    │   • Cluster mode (always)
    │   • JetStream enabled
    │   • Cluster routes to peer IPs
    │
    ├─► Wait for cluster to form
    │   • First node: immediate (cluster of 1)
    │   • Later nodes: wait for peer connections
    │
    ├─► Initialize JetStream streams
    │   • Replication factor = min(cluster_size, 3)
    │
    └─► Start application (Trees, Nims, etc.)
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
      NodeName    string   // From node-info.json or hostname
      ClusterName string   // forest_id from node-info.json
      DataDir     string   // JetStream storage (default /var/lib/nimsforest/jetstream)
      ClientPort  int      // 4222
      ClusterPort int      // 6222
      MonitorPort int      // 8222
      Peers       []string // Peer IPs from registry query
  }

  type EmbeddedNATS struct { ... }

  func New(cfg Config) (*EmbeddedNATS, error)
  func (e *EmbeddedNATS) Start() error
  func (e *EmbeddedNATS) ClientConn() (*nats.Conn, error)  // In-process connection
  func (e *EmbeddedNATS) WaitForCluster(minPeers int, timeout time.Duration) error
  func (e *EmbeddedNATS) ClusterSize() int
  func (e *EmbeddedNATS) Shutdown() error
  ```

---

### Phase 2: Read Morpheus Node Info

#### 2.1 Create node info reader
- **File**: `internal/morpheus/nodeinfo.go` (new file)
- **Contents**:
  ```go
  package morpheus

  // NodeInfo matches /etc/morpheus/node-info.json structure
  type NodeInfo struct {
      ForestID         string `json:"forest_id"`
      NodeID           string `json:"node_id,omitempty"`
      Role             string `json:"role"`
      Provisioner      string `json:"provisioner"`
      ProvisionedAt    string `json:"provisioned_at"`
      RegistryURL      string `json:"registry_url"`
      RegistryUsername string `json:"registry_username,omitempty"`
      RegistryPassword string `json:"registry_password,omitempty"`
  }

  // LoadNodeInfo reads /etc/morpheus/node-info.json
  func LoadNodeInfo() (*NodeInfo, error)
  
  // LoadNodeInfoFrom reads from a custom path (for testing)
  func LoadNodeInfoFrom(path string) (*NodeInfo, error)
  ```

#### 2.2 Handle missing node-info.json (standalone mode)
- **Behavior**:
  - [ ] If file doesn't exist, run in standalone mode
  - [ ] Generate random forest_id
  - [ ] No registry query (no peers)
  - [ ] Log: "No morpheus node-info found, running standalone"

---

### Phase 3: Query StorageBox Registry

#### 3.1 Create registry client
- **File**: `internal/morpheus/registry.go` (new file)
- **Contents**:
  ```go
  package morpheus

  // RegistryClient queries the StorageBox registry
  type RegistryClient struct {
      URL      string
      Username string
      Password string
  }

  // Node represents a node in the registry
  type Node struct {
      ID       string `json:"id"`
      ForestID string `json:"forest_id"`
      IP       string `json:"ip"`
      Role     string `json:"role"`
      Status   string `json:"status"`
  }

  func NewRegistryClient(url, username, password string) *RegistryClient

  // GetPeers returns all nodes for a forest, excluding self
  func (c *RegistryClient) GetPeers(forestID, selfIP string) ([]Node, error)
  ```

#### 3.2 Implement WebDAV/HTTP fetch
- **Implementation**:
  - [ ] HTTP GET with basic auth
  - [ ] Parse JSON response
  - [ ] Filter nodes by forest_id
  - [ ] Exclude self (by IP match)
  - [ ] Return peer IPs for NATS routes

#### 3.3 Handle registry unavailability
- **Behavior**:
  - [ ] Retry with backoff (3 attempts)
  - [ ] If registry unreachable, start with no peers
  - [ ] Log warning: "Registry unavailable, starting without peers"
  - [ ] Continue to retry in background (for late peer discovery)

---

### Phase 4: Integrate in Main Application

#### 4.1 Update main.go startup sequence
- **File**: `cmd/forest/main.go`
- **New startup flow**:
  ```go
  func runForest() {
      // 1. Load node info (or use defaults for standalone)
      nodeInfo, err := morpheus.LoadNodeInfo()
      if err != nil {
          log.Printf("No morpheus config, running standalone: %v", err)
          nodeInfo = morpheus.DefaultNodeInfo()
      }
      
      // 2. Query registry for peers
      var peers []string
      if nodeInfo.RegistryURL != "" {
          client := morpheus.NewRegistryClient(
              nodeInfo.RegistryURL,
              nodeInfo.RegistryUsername,
              nodeInfo.RegistryPassword,
          )
          peerNodes, err := client.GetPeers(nodeInfo.ForestID, getLocalIP())
          if err != nil {
              log.Printf("Warning: Could not fetch peers: %v", err)
          }
          for _, p := range peerNodes {
              peers = append(peers, fmt.Sprintf("%s:6222", p.IP))
          }
      }
      
      // 3. Start embedded NATS
      natsServer := natsembed.New(natsembed.Config{
          NodeName:    hostname(),
          ClusterName: nodeInfo.ForestID,
          Peers:       peers,
          DataDir:     "/var/lib/nimsforest/jetstream",
      })
      natsServer.Start()
      
      // 4. Get in-process connection
      nc, _ := natsServer.ClientConn()
      js, _ := nc.JetStream()
      
      // 5. Initialize core components
      wind := core.NewWind(nc)
      river := core.NewRiver(js, natsServer.ClusterSize())
      // ... rest of initialization
  }
  ```

#### 4.2 Dynamic replication factor
- **Files**: `internal/core/river.go`, `internal/core/humus.go`, `internal/core/soil.go`
- **Changes**:
  - [ ] Accept cluster size as parameter
  - [ ] Set replication: `min(clusterSize, 3)`
  - [ ] Single node = R1, two nodes = R2, three+ = R3

---

### Phase 5: Background Peer Discovery

#### 5.1 Implement peer watcher
- **File**: `internal/morpheus/watcher.go` (new file)
- **Purpose**: Handle nodes that join after startup
- **Behavior**:
  - [ ] Poll registry every 30 seconds
  - [ ] Detect new nodes in same forest
  - [ ] Add NATS routes for new peers
  - [ ] Log: "Discovered new peer: node-4 at 2a01:...:4"

#### 5.2 Update node status in registry
- **File**: `internal/morpheus/registry.go`
- **Features**:
  - [ ] Update own status to "active" after NATS starts
  - [ ] Heartbeat to registry (optional, for health tracking)

---

### Phase 6: Update Systemd Service

#### 6.1 Simplify service file
- **File**: `scripts/systemd/nimsforest.service`
- **Changes**:
  ```ini
  [Unit]
  Description=NimsForest Event Orchestration System
  After=network-online.target
  Wants=network-online.target
  
  [Service]
  Type=simple
  User=root
  ExecStart=/opt/nimsforest/bin/forest
  Restart=always
  RestartSec=5
  
  # Data directory
  WorkingDirectory=/var/lib/nimsforest
  
  # Logging
  StandardOutput=journal
  StandardError=journal
  SyslogIdentifier=nimsforest
  
  [Install]
  WantedBy=multi-user.target
  ```
- **Removed**: NATS service dependency (now embedded)

---

### Phase 7: Health Endpoints

#### 7.1 Add health check endpoint
- **File**: `internal/httputil/health.go` (new file)
- **Endpoints**:
  - `GET /health` - overall status
  - `GET /health/cluster` - cluster info

- **Response**:
  ```json
  {
    "status": "healthy",
    "forest_id": "forest-123",
    "node_id": "node-1",
    "cluster": {
      "size": 3,
      "peers": ["node-2", "node-3"],
      "replication_factor": 3
    },
    "jetstream": {
      "streams": ["RIVER", "HUMUS"],
      "kv_buckets": ["SOIL"]
    }
  }
  ```

---

### Phase 8: Documentation

#### 8.1 Update README
- **File**: `README.md`
- **Changes**:
  - [ ] Document embedded NATS architecture
  - [ ] Explain Morpheus integration
  - [ ] Update quick start (no separate NATS)

#### 8.2 Document node-info.json format
- **File**: `docs/MORPHEUS_INTEGRATION.md` (new file)
- **Contents**:
  - [ ] node-info.json schema
  - [ ] Registry data format
  - [ ] Peer discovery process

---

### Phase 9: Testing

#### 9.1 Unit tests
- **Files**: 
  - `internal/natsembed/server_test.go`
  - `internal/morpheus/nodeinfo_test.go`
  - `internal/morpheus/registry_test.go`

#### 9.2 Integration tests
- **File**: `test/e2e/embedded_test.go`
- **Tests**:
  - [ ] Standalone mode (no node-info.json)
  - [ ] Single node with node-info
  - [ ] Mock registry with peers

---

## File Structure After Implementation

```
nimsforest/
├── cmd/forest/
│   └── main.go              # Updated startup with embedded NATS
├── internal/
│   ├── core/                # Existing core components
│   ├── natsembed/           # NEW: Embedded NATS server
│   │   ├── server.go
│   │   └── server_test.go
│   ├── morpheus/            # NEW: Morpheus integration
│   │   ├── nodeinfo.go      # Read /etc/morpheus/node-info.json
│   │   ├── registry.go      # Query StorageBox registry
│   │   ├── watcher.go       # Background peer discovery
│   │   └── *_test.go
│   └── httputil/
│       └── health.go        # NEW: Health endpoints
└── scripts/
    └── systemd/
        └── nimsforest.service  # Simplified (no NATS dependency)
```

---

## Configuration

### From /etc/morpheus/node-info.json (written by Morpheus)

```json
{
  "forest_id": "forest-1735234567",
  "node_id": "12345678",
  "role": "edge",
  "provisioner": "morpheus",
  "provisioned_at": "2025-12-26T10:30:00Z",
  "registry_url": "https://uXXXXX.your-storagebox.de/morpheus/registry.json",
  "registry_username": "uXXXXX",
  "registry_password": "your-password"
}
```

### Environment Variable Overrides (optional)

| Variable | Description | Default |
|----------|-------------|---------|
| `NIMSFOREST_NODE_INFO` | Custom path to node-info.json | `/etc/morpheus/node-info.json` |
| `NIMSFOREST_DATA_DIR` | JetStream data directory | `/var/lib/nimsforest/jetstream` |
| `NIMSFOREST_CLIENT_PORT` | NATS client port | `4222` |
| `NIMSFOREST_CLUSTER_PORT` | NATS cluster port | `6222` |
| `NIMSFOREST_MONITOR_PORT` | NATS monitor port | `8222` |

---

## Deployment Flow

### Morpheus Side (Already Done)

```bash
# User runs:
morpheus plant cloud medium

# Morpheus:
# 1. Creates 3 servers via Hetzner API
# 2. Registers nodes in StorageBox registry
# 3. Writes /etc/morpheus/node-info.json on each server
# 4. Deploys nimsforest binary to /opt/nimsforest/bin/
# 5. Starts nimsforest service
```

### NimsForest Side (What We Implement)

```
nimsforest starts
    │
    ├─► Reads /etc/morpheus/node-info.json
    │   forest_id: "forest-123"
    │   registry_url: "https://..."
    │
    ├─► Queries registry, gets peers
    │   Node 1 sees: [node-2, node-3]
    │   Node 2 sees: [node-1, node-3]
    │   Node 3 sees: [node-1, node-2]
    │
    ├─► Starts embedded NATS with peer routes
    │
    ├─► Cluster forms automatically
    │
    └─► Application starts
        "nimsforest is ready (cluster: 3 nodes, R3)"
```

---

## Implementation Priority

| Phase | Priority | Effort | Description |
|-------|----------|--------|-------------|
| 1 | Critical | 3-4h | Embed NATS server |
| 2 | Critical | 1-2h | Read node-info.json |
| 3 | Critical | 2-3h | Query StorageBox registry |
| 4 | Critical | 2h | Integrate in main.go |
| 5 | High | 2h | Background peer discovery |
| 6 | High | 0.5h | Update systemd service |
| 7 | Medium | 1-2h | Health endpoints |
| 8 | Medium | 1-2h | Documentation |
| 9 | Low | 2h | Testing |

**Total estimated effort**: 15-19 hours

---

## Notes

- **IPv6**: Hetzner uses IPv6-only by default, NATS supports IPv6
- **No GitHub downloads**: NATS is embedded, solves IPv4-only API issue
- **Morpheus unchanged**: nimsforest reads what Morpheus already writes
- **Graceful standalone**: Works without Morpheus for development
- **Dynamic scaling**: New nodes discovered via registry polling
