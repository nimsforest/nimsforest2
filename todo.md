# Embedded NATS Implementation

## Quick Summary for Morpheus

```
For each node:
  1. Create server
  2. Mount StorageBox at /mnt/forest/
  3. Add node to /mnt/forest/registry.json
  4. Write /etc/morpheus/node-info.json
  5. Create /var/lib/nimsforest/jetstream/
  6. Deploy binary to /opt/nimsforest/bin/forest
  7. Start nimsforest service

NATS gossip handles cluster formation automatically.
```

---

## How Clustering Works

**Registry is just for initial peer discovery.** NATS gossip handles everything else.

```
Node-1 starts:
  → Reads registry → no other nodes yet
  → Starts as cluster of 1

Node-2 starts:
  → Reads registry → sees node-1
  → Connects to node-1
  → Cluster is now 2 nodes

Node-3 starts:
  → Reads registry → sees node-1, node-2 (or just one of them)
  → Connects to at least one peer
  → NATS gossip propagates info to all
  → Cluster is now 3 nodes

Node-4 added later:
  → Reads registry → sees existing nodes
  → Connects to any one of them
  → NATS gossip tells everyone about node-4
  → Cluster is now 4 nodes
```

**Key point:** A new node only needs to find ONE existing peer. NATS gossip spreads the rest.

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
| 4222 | NATS client | Optional |
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
            peers = append(peers, "["+node.IP+"]:6222")
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

---

## Archive: Humus → Soil → SQLite

### Overview

State flows through three layers:

```
Humus (events) → Decomposer → Soil (active state) → Archive (SQLite)
                                                         ↓
                                              Network drive for
                                              offline access
```

- **Soil**: Active state, requires runtime
- **Archive**: Inactive/historical data, always accessible (SQLite on network drive)

### ✅ Completed

- [x] `internal/core/archive.go` - SQLite archive component
- [x] `Archive.Store(key, data, archivedBy)` - Store entity
- [x] `Archive.Get(key)` - Retrieve entity
- [x] `Archive.List()` - List all keys
- [x] `Archive.Delete(key)` - Remove entity
- [x] `ArchiveFromSoil(soil, archive, key, node, delete)` - Move from Soil to Archive
- [x] Tests for archive functionality

### 🔲 TODO: Archival Policy

**Problem**: When should data be moved from Soil to Archive?

Options to implement:

1. **Manual archival** - Nim explicitly archives when task/entity is "done"
   ```go
   // In a nim handler
   if task.Status == "completed" && task.CompletedAt.Before(thirtyDaysAgo) {
       core.ArchiveFromSoil(soil, archive, taskKey, nodeName, true)
   }
   ```

2. **Time-based archival** - Background worker archives old data
   ```go
   // Archiver checks entities older than X days
   type Archiver struct {
       soil       *Soil
       archive    *Archive
       maxAge     time.Duration  // e.g., 30 days
       checkEvery time.Duration  // e.g., 1 hour
   }
   ```

3. **Entity-type rules** - Different retention per entity type
   ```go
   // Config-driven
   archivalRules:
     tasks/*: 30d
     orders/*: 90d
     customers/*: never  # always active
   ```

4. **Capacity-based** - Archive when Soil reaches size limit

### 🔲 TODO: Implementation Tasks

| Task | Priority | Notes |
|------|----------|-------|
| Define archival trigger (manual vs automatic) | High | Architectural decision |
| Add `archived_at` / `status` field to entities | High | Need to know what's archivable |
| Create `Archiver` worker (if automatic) | Medium | Runs periodically |
| Add archival rules config | Medium | Per entity-type retention |
| Expose archive via API for offline queries | Low | REST/CLI to query SQLite |

### Questions to Answer

1. **Who decides when to archive?** The nim (business logic) or a system worker?
2. **What makes an entity "inactive"?** Status field? Last modified time? Both?
3. **Should archived data be queryable from nims?** Or only offline via SQLite tools?
4. **Retention period?** How long before completed tasks move to archive?
