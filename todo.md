# Embedded NATS Implementation

## Quick Summary for Morpheus

```
morpheus plant cloud medium
         │
         ▼
┌─────────────────────────────────────────────────────────┐
│ 1. Create servers (Hetzner API)                         │
│ 2. Mount StorageBox at /mnt/forest/ on each             │
│ 3. Write ALL nodes to /mnt/forest/registry.json         │  ◄── BEFORE starting services
│ 4. Write /etc/morpheus/node-info.json on each           │
│ 5. Create /var/lib/nimsforest/jetstream/ on each        │
│ 6. Deploy binary to /opt/nimsforest/bin/forest          │
│ 7. Start nimsforest service on all nodes                │
└─────────────────────────────────────────────────────────┘
         │
         ▼
   Cluster forms automatically (NATS handles it)
```

---

## Detailed Flow

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
  └─► Mount StorageBox on each server at /mnt/forest/

PHASE 2: Register All Nodes (BEFORE starting services)
──────────────────────────────────────────────────────
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

PHASE 3: Configure Each Node
────────────────────────────
On each server:
  ├─► Write /etc/morpheus/node-info.json
  │   { "forest_id": "forest-123", "node_id": "node-1" }
  │
  ├─► Create /var/lib/nimsforest/jetstream/
  │
  └─► Deploy /opt/nimsforest/bin/forest

PHASE 4: Start Services
───────────────────────
Start nimsforest service on all nodes (order doesn't matter)

PHASE 5: Cluster Forms Automatically  
────────────────────────────────────
node-1 starts:
  → Reads node-info.json → forest_id: "forest-123"
  → Reads registry.json → peers: node-2, node-3
  → Starts NATS with routes to [2a01:4f8:x:x::2]:6222, [2a01:4f8:x:x::3]:6222
  → Peers not up yet → connections fail (OK, NATS keeps retrying)
  → Runs as cluster of 1

node-2 starts:
  → Reads registry.json → peers: node-1, node-3
  → Starts NATS, connects to node-1 ✓
  → NATS gossip: nodes share peer info
  → Cluster is now 2 nodes

node-3 starts:
  → Reads registry.json → peers: node-1, node-2
  → Connects to both ✓
  → Full mesh formed
  → Cluster is now 3 nodes

DONE: 3-node NATS cluster, fully meshed, JetStream enabled
```

---

## File Formats

### /etc/morpheus/node-info.json (per machine)

```json
{
  "forest_id": "forest-123",
  "node_id": "node-1"
}
```

### /mnt/forest/registry.json (shared)

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

---

## Mount Setup

```bash
# Add to cloud-init or /etc/fstab
//uXXXXX.your-storagebox.de/backup /mnt/forest cifs user=uXXXXX,pass=PASSWORD,uid=root,gid=root 0 0
```

---

## Firewall Ports

| Port | Purpose | Required |
|------|---------|----------|
| 6222 | NATS cluster | Yes |
| 4222 | NATS client | Optional (debugging) |
| 8222 | NATS monitoring | Optional |

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
            peers = append(peers, "["+node.IP+"]:6222")  // IPv6 brackets
        }
    }
    return peers
}
```

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
