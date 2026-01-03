# Embedded NATS Implementation

## Note for Morpheus CLI

**nimsforest now embeds NATS** - no separate NATS installation needed.

### What Morpheus should do:

1. **Don't install NATS** - nimsforest binary includes it (you already don't, just confirming)

2. **Include registry credentials in node-info.json** - nimsforest needs to query the StorageBox registry to discover peers. Update `/etc/morpheus/node-info.json` to include:

```json
{
  "forest_id": "forest-123",
  "node_id": "12345678",
  "role": "edge",
  "provisioner": "morpheus",
  "provisioned_at": "2025-12-26T10:30:00Z",
  "registry_url": "https://uXXXXX.your-storagebox.de/morpheus/registry.json",
  "registry_username": "uXXXXX",
  "registry_password": "the-storagebox-password"
}
```

3. **Ensure nodes have IPs in registry** - nimsforest reads the registry to find peer IPs. The existing registry format works:
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

4. **Deploy nimsforest binary to** `/opt/nimsforest/bin/forest`

5. **Create data directory** `/var/lib/nimsforest/jetstream` (for JetStream storage)

6. **Start service** - nimsforest handles NATS startup internally

### What nimsforest does on startup:

```
1. Read /etc/morpheus/node-info.json
2. Query registry at registry_url (HTTP GET with basic auth)
3. Find other nodes in same forest_id
4. Start embedded NATS with routes to peer IPs
5. NATS cluster forms automatically
6. Application starts
```

### Ports needed (firewall):

- `4222` - NATS client (for debugging with nats cli)
- `6222` - NATS cluster (peer communication)
- `8222` - NATS monitoring (optional)

---

## Goal

Embed NATS into nimsforest binary. On startup:
1. Read `/etc/morpheus/node-info.json`
2. Query StorageBox registry for peers
3. Start embedded NATS with peer routes
4. Run application

## Why

- **Single binary** - no separate NATS download (solves IPv4-only GitHub API issue)
- **Simpler deployment** - Morpheus just deploys one binary
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

### 2. Read Morpheus Config & Query Peers

**File**: `internal/morpheus/morpheus.go`

```go
package morpheus

// NodeInfo from /etc/morpheus/node-info.json
type NodeInfo struct {
    ForestID         string `json:"forest_id"`
    RegistryURL      string `json:"registry_url"`
    RegistryUsername string `json:"registry_username"`
    RegistryPassword string `json:"registry_password"`
}

// Load reads node-info.json, returns nil if not found (standalone mode)
func Load() *NodeInfo

// GetPeers queries registry, returns peer IPs for this forest
func (n *NodeInfo) GetPeers(selfIP string) ([]string, error)
```

**Registry query**: Simple HTTP GET with basic auth, parse JSON, filter by forest_id.

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
        peers, _ = nodeInfo.GetPeers(getLocalIP())
    }
    
    // 3. Start embedded NATS
    ns, err := natsembed.New(natsembed.Config{
        NodeName:    hostname(),
        ClusterName: forestID,
        DataDir:     "/var/lib/nimsforest/jetstream",
        Peers:       peers,
    })
    ns.Start()
    defer ns.Shutdown()
    
    // 4. Get connection (in-process, no network)
    nc, _ := ns.ClientConn()
    js, _ := nc.JetStream()
    
    // 5. Rest unchanged - Wind, River, Trees, Nims...
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
After=network-online.target

[Service]
ExecStart=/opt/nimsforest/bin/forest
Restart=always
RestartSec=5
WorkingDirectory=/var/lib/nimsforest

[Install]
WantedBy=multi-user.target
```

Remove NATS dependency - it's embedded now.

---

## Not Needed

| What | Why not |
|------|---------|
| Background peer watcher | NATS gossip handles this after initial connect |
| Health endpoints | Can add later if needed |
| Env var overrides | Use node-info.json or defaults |
| Dynamic replication | Use R1 for now, NATS handles it |

---

## Estimated Effort

| Task | Hours |
|------|-------|
| Embed NATS server | 3-4h |
| Morpheus config + registry query | 2-3h |
| Wire up main.go | 1-2h |
| **Total** | **6-9h** |
