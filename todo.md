# Embedded NATS Implementation

## Note for Morpheus CLI

**nimsforest now embeds NATS** - no separate NATS installation needed.

### What Morpheus should do:

1. **Mount StorageBox** on each machine at `/mnt/forest/` - nimsforest reads the registry as a local file

2. **Write node-info.json** to `/etc/morpheus/node-info.json`:
```json
{
  "forest_id": "forest-123",
  "node_id": "12345678",
  "role": "edge",
  "provisioner": "morpheus",
  "provisioned_at": "2025-12-26T10:30:00Z"
}
```
(No registry credentials needed - StorageBox is mounted)

3. **Ensure registry.json exists** at `/mnt/forest/registry.json` with node IPs:
```json
{
  "nodes": {
    "forest-123": [
      { "id": "node-1", "ip": "2a01:4f8:...", "forest_id": "forest-123" },
      { "id": "node-2", "ip": "2a01:4f8:...", "forest_id": "forest-123" }
    ]
  }
}
```

4. **Deploy nimsforest binary** to `/opt/nimsforest/bin/forest`

5. **Create data directory** `/var/lib/nimsforest/jetstream`

6. **Start service**

### What nimsforest does on startup:

```
1. Read /etc/morpheus/node-info.json → get forest_id
2. Read /mnt/forest/registry.json → find peer IPs in same forest
3. Start embedded NATS with routes to peers
4. Cluster forms automatically
5. Application starts
```

### Ports needed (firewall):

- `6222` - NATS cluster (peer communication) - **required**
- `4222` - NATS client (optional, for nats cli debugging)
- `8222` - NATS monitoring (optional)

---

## Goal

Embed NATS into nimsforest binary. On startup:
1. Read `/etc/morpheus/node-info.json` for forest_id
2. Read `/mnt/forest/registry.json` for peer IPs
3. Start embedded NATS with peer routes
4. Run application

## Why

- **Single binary** - no separate NATS download (solves IPv4-only GitHub API issue)
- **Simpler deployment** - Morpheus just deploys one binary
- **No HTTP client** - registry is a mounted file, just read it
- **Auto-clustering** - NATS gossip handles peer discovery after initial connection

---

## Tasks

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
    NodeInfoPath  = "/etc/morpheus/node-info.json"
    RegistryPath  = "/mnt/forest/registry.json"
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

// Load reads node-info.json, returns nil if not found
func Load() *NodeInfo {
    data, err := os.ReadFile(NodeInfoPath)
    if err != nil {
        return nil
    }
    var info NodeInfo
    json.Unmarshal(data, &info)
    return &info
}

// GetPeers reads registry.json and returns peer IPs for this forest
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
    // 1. Load morpheus config (nil = standalone)
    nodeInfo := morpheus.Load()
    
    // 2. Determine peers
    var peers []string
    forestID := "standalone"
    if nodeInfo != nil {
        forestID = nodeInfo.ForestID
        peers = morpheus.GetPeers(forestID, getLocalIP())
    }
    
    // 3. Start embedded NATS
    ns, _ := natsembed.New(natsembed.Config{
        NodeName:    hostname(),
        ClusterName: forestID,
        DataDir:     "/var/lib/nimsforest/jetstream",
        Peers:       peers,
    })
    ns.Start()
    defer ns.Shutdown()
    
    // 4. Get connection (in-process)
    nc, _ := ns.ClientConn()
    js, _ := nc.JetStream()
    
    // 5. Rest unchanged
    wind := core.NewWind(nc)
    river, _ := core.NewRiver(js)
    // ...
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
