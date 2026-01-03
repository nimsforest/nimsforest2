# Embedded NATS Implementation

## Design Note for Morpheus

### Architecture Decision: Mounted StorageBox

The StorageBox is **mounted as a filesystem** on each machine. nimsforest reads it as a local file - no HTTP client needed.

### Deployment Flow

When user runs `morpheus plant cloud medium` (3 nodes):

```
PHASE 1: Provision Infrastructure
──────────────────────────────────
Morpheus:
  ├─► Create 3 Hetzner servers
  │   • node-1: 2a01:4f8:x:x::1
  │   • node-2: 2a01:4f8:x:x::2  
  │   • node-3: 2a01:4f8:x:x::3
  │
  ├─► Mount StorageBox on each server at /mnt/forest/
  │
  └─► Register ALL nodes in registry FIRST (before starting any service)
      Write to /mnt/forest/registry.json:
      {
        "nodes": {
          "forest-123": [
            { "id": "node-1", "ip": "2a01:4f8:x:x::1" },
            { "id": "node-2", "ip": "2a01:4f8:x:x::2" },
            { "id": "node-3", "ip": "2a01:4f8:x:x::3" }
          ]
        }
      }

PHASE 2: Configure Each Node
────────────────────────────
On each server:
  ├─► Write /etc/morpheus/node-info.json
  │   { "forest_id": "forest-123", "node_id": "node-1" }
  │
  ├─► Create /var/lib/nimsforest/jetstream/
  │
  └─► Deploy /opt/nimsforest/bin/forest

PHASE 3: Start Services
───────────────────────
Start nimsforest service on all nodes (order doesn't matter)

PHASE 4: Cluster Forms Automatically  
────────────────────────────────────
node-1 starts:
  → Reads registry, sees node-2, node-3
  → Starts NATS, tries to connect to peers
  → Peers not up yet, connections fail (OK - NATS retries)
  → Runs as cluster of 1

node-2 starts:
  → Reads registry, sees node-1, node-3
  → Starts NATS, connects to node-1 ✓
  → NATS gossip: both nodes now know each other
  → Cluster is 2 nodes

node-3 starts:
  → Reads registry, sees node-1, node-2
  → Starts NATS, connects to both ✓
  → Full mesh formed
  → Cluster is 3 nodes

RESULT: 3-node NATS cluster with JetStream, fully meshed
```

### Important: Registry Before Service

**Register ALL nodes in registry.json BEFORE starting nimsforest on ANY node.**

This ensures every node knows about all peers from startup. NATS handles connection retries automatically.

### File Locations

| File | Path | Written by |
|------|------|------------|
| Node info | `/etc/morpheus/node-info.json` | Morpheus (per machine) |
| Registry | `/mnt/forest/registry.json` | Morpheus (shared) |
| Binary | `/opt/nimsforest/bin/forest` | Morpheus |
| JetStream data | `/var/lib/nimsforest/jetstream/` | nimsforest |

### Mount Setup

```bash
# Mount StorageBox (add to cloud-init or /etc/fstab)
//uXXXXX.your-storagebox.de/backup /mnt/forest cifs user=uXXXXX,pass=PASSWORD,uid=root,gid=root 0 0
```

### node-info.json Format

```json
{
  "forest_id": "forest-123",
  "node_id": "12345678"
}
```

### registry.json Format

```json
{
  "nodes": {
    "forest-123": [
      { "id": "node-1", "ip": "2a01:4f8:x:x::1", "forest_id": "forest-123" },
      { "id": "node-2", "ip": "2a01:4f8:x:x::2", "forest_id": "forest-123" },
      { "id": "node-3", "ip": "2a01:4f8:x:x::3", "forest_id": "forest-123" }
    ]
  }
}
```

### Firewall Ports

- `6222` - NATS cluster (required)
- `4222` - NATS client (optional)
- `8222` - NATS monitoring (optional)

---

## nimsforest Implementation Tasks

### 1. Embed NATS Server

**File**: `internal/natsembed/server.go`

```go
package natsembed

import (
    "github.com/nats-io/nats-server/v2/server"
    "github.com/nats-io/nats.go"
)

type Server struct {
    ns *server.Server
}

type Config struct {
    NodeName    string   // hostname
    ClusterName string   // forest_id
    DataDir     string   // /var/lib/nimsforest/jetstream
    Peers       []string // ["[2a01::1]:6222", "[2a01::2]:6222"]
}

func New(cfg Config) (*Server, error)
func (s *Server) Start() error
func (s *Server) ClientConn() (*nats.Conn, error)
func (s *Server) Shutdown()
```

**Add dependency**:
```bash
go get github.com/nats-io/nats-server/v2
```

---

### 2. Read Config Files

**File**: `internal/morpheus/morpheus.go`

```go
package morpheus

import (
    "encoding/json"
    "os"
)

const (
    NodeInfoPath = "/etc/morpheus/node-info.json"
    RegistryPath = "/mnt/forest/registry.json"
)

type NodeInfo struct {
    ForestID string `json:"forest_id"`
    NodeID   string `json:"node_id"`
}

type Registry struct {
    Nodes map[string][]Node `json:"nodes"`
}

type Node struct {
    ID       string `json:"id"`
    IP       string `json:"ip"`
    ForestID string `json:"forest_id"`
}

func Load() *NodeInfo {
    data, err := os.ReadFile(NodeInfoPath)
    if err != nil {
        return nil
    }
    var info NodeInfo
    json.Unmarshal(data, &info)
    return &info
}

func GetPeers(forestID, selfIP string) []string {
    data, _ := os.ReadFile(RegistryPath)
    var reg Registry
    json.Unmarshal(data, &reg)
    
    var peers []string
    for _, node := range reg.Nodes[forestID] {
        if node.IP != selfIP {
            peers = append(peers, node.IP+":6222")
        }
    }
    return peers
}
```

---

### 3. Update main.go

**File**: `cmd/forest/main.go`

```go
func runForest() {
    nodeInfo := morpheus.Load()
    
    var peers []string
    forestID := "standalone"
    if nodeInfo != nil {
        forestID = nodeInfo.ForestID
        peers = morpheus.GetPeers(forestID, getLocalIP())
    }
    
    ns, _ := natsembed.New(natsembed.Config{
        NodeName:    hostname(),
        ClusterName: forestID,
        DataDir:     "/var/lib/nimsforest/jetstream",
        Peers:       peers,
    })
    ns.Start()
    defer ns.Shutdown()
    
    nc, _ := ns.ClientConn()
    js, _ := nc.JetStream()
    
    wind := core.NewWind(nc)
    river, _ := core.NewRiver(js)
    // ... rest unchanged
}
```

---

### 4. Update systemd service

**File**: `scripts/systemd/nimsforest.service`

```ini
[Unit]
Description=NimsForest
After=network-online.target mnt-forest.mount

[Service]
ExecStart=/opt/nimsforest/bin/forest
Restart=always
RestartSec=5
WorkingDirectory=/var/lib/nimsforest

[Install]
WantedBy=multi-user.target
```

---

## Estimated Effort

| Task | Hours |
|------|-------|
| Embed NATS server | 3-4h |
| Read config files | 1h |
| Wire up main.go | 1h |
| **Total** | **5-6h** |
