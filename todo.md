# Embedded NATS Implementation

## Design Note for Morpheus

### Architecture Decision: Mounted StorageBox

The StorageBox should be **mounted as a filesystem** on each machine rather than accessed via HTTP/WebDAV. This simplifies nimsforest significantly:

- **No HTTP client code** in nimsforest
- **No authentication handling** - mount handles it
- **No retry logic** - just read a file
- **No credentials in node-info.json** - cleaner separation

### Mount Setup (Morpheus responsibility)

Mount the StorageBox on each machine:

```bash
# Example using CIFS/SMB
mount -t cifs //uXXXXX.your-storagebox.de/backup /mnt/forest \
  -o user=uXXXXX,pass=PASSWORD,uid=root,gid=root

# Or add to /etc/fstab for persistence
//uXXXXX.your-storagebox.de/backup /mnt/forest cifs user=uXXXXX,pass=PASSWORD,uid=root,gid=root 0 0
```

### File Locations

| File | Path | Written by |
|------|------|------------|
| Node info | `/etc/morpheus/node-info.json` | Morpheus (per machine) |
| Registry | `/mnt/forest/registry.json` | Morpheus (shared) |
| nimsforest binary | `/opt/nimsforest/bin/forest` | Morpheus |
| JetStream data | `/var/lib/nimsforest/jetstream/` | nimsforest |

### What Morpheus Should Do

1. **Mount StorageBox** at `/mnt/forest/` on each machine

2. **Write `/etc/morpheus/node-info.json`** on each machine:
```json
{
  "forest_id": "forest-123",
  "node_id": "12345678"
}
```

3. **Write/update `/mnt/forest/registry.json`** (shared file):
```json
{
  "nodes": {
    "forest-123": [
      { "id": "12345678", "ip": "2a01:4f8:x:x::1", "forest_id": "forest-123" },
      { "id": "12345679", "ip": "2a01:4f8:x:x::2", "forest_id": "forest-123" },
      { "id": "12345680", "ip": "2a01:4f8:x:x::3", "forest_id": "forest-123" }
    ]
  }
}
```

4. **Create directories**:
   - `/var/lib/nimsforest/jetstream/`

5. **Deploy binary** to `/opt/nimsforest/bin/forest`

6. **Start service** via systemd

### What nimsforest Does on Startup

```
1. Read /etc/morpheus/node-info.json → get forest_id
2. Read /mnt/forest/registry.json → find peer IPs
3. Start embedded NATS with routes to peers
4. NATS cluster forms via gossip
5. Application starts
```

### Firewall Ports

- `6222` - NATS cluster communication (required)
- `4222` - NATS client (optional, for debugging)
- `8222` - NATS monitoring (optional)

---

## Implementation Tasks

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
