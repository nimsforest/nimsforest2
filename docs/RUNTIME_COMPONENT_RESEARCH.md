# Runtime Component Addition - Research & Design

## Problem Statement

How can we add/remove Trees, TreeHouses, and Nims at runtime without restarting NimsForest?

The challenge: The main `forest run` process blocks waiting for shutdown signals. CLI commands need to communicate with this running process.

## How Other Tools Solve This

### 1. Docker (Unix Socket)

```bash
# Daemon
dockerd  # Listens on /var/run/docker.sock

# CLI (separate process)
docker run nginx  # Connects to socket, sends command
```

### 2. Kubernetes (HTTP API)

```bash
# Daemon (API server)
kube-apiserver  # Listens on :6443

# CLI (separate process)  
kubectl apply -f pod.yaml  # HTTP POST to API server
```

### 3. Consul (HTTP API)

```bash
# Daemon
consul agent -server  # HTTP API on :8500

# CLI (separate process)
consul services register service.json  # HTTP PUT
```

### 4. Prometheus (SIGHUP + HTTP)

```bash
# Reload via signal
kill -HUP $(pidof prometheus)

# Or via HTTP
curl -X POST http://localhost:9090/-/reload
```

## Recommended Design for NimsForest

### Architecture

```
┌────────────────────────────────────────────────────────┐
│                    forest run (daemon)                  │
│                                                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │  TreeHouses  │  │    Nims      │  │    Trees     │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
│                                                        │
│  ┌──────────────────────────────────────────────────┐ │
│  │              Management API (:8080)               │ │
│  │  POST /api/v1/treehouses     - Add treehouse     │ │
│  │  DELETE /api/v1/treehouses/x - Remove treehouse  │ │
│  │  POST /api/v1/nims           - Add nim           │ │
│  │  DELETE /api/v1/nims/x       - Remove nim        │ │
│  │  POST /-/reload              - Reload config     │ │
│  │  GET /api/v1/status          - Get status        │ │
│  └──────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────┘
                           ▲
                           │ HTTP
                           ▼
┌────────────────────────────────────────────────────────┐
│                 forest CLI (client)                     │
│                                                        │
│  forest add treehouse scoring2 --subscribes=... ...   │
│  forest remove nim qualifier                           │
│  forest list                                           │
│  forest reload                                         │
│  forest status                                         │
└────────────────────────────────────────────────────────┘
```

### CLI Commands

```bash
# Start daemon (existing)
forest run
forest standalone

# Management commands (NEW - talks to running daemon)
forest add treehouse <name> --subscribes=<subj> --publishes=<subj> --script=<path>
forest add nim <name> --subscribes=<subj> --publishes=<subj> --prompt=<path>
forest remove treehouse <name>
forest remove nim <name>
forest list [treehouses|nims|all]
forest reload                    # Reload config file
forest status                    # Show running components

# Direct config reload via signal (also works)
kill -HUP $(pidof forest)
```

### Alternative: Use NATS as Control Plane

Since NimsForest already has NATS, we could use NATS subjects for control:

```
forest.control.add.treehouse    - Add treehouse
forest.control.remove.treehouse - Remove treehouse  
forest.control.add.nim          - Add nim
forest.control.remove.nim       - Remove nim
forest.control.reload           - Reload config
forest.control.status           - Request status
forest.control.status.response  - Status response
```

**Pros:**
- No additional port needed
- Works across cluster nodes
- Reuses existing infrastructure

**Cons:**
- Requires NATS connection for CLI
- More complex than HTTP

## Implementation Plan

### Phase 1: Core Runtime Methods
Add to `pkg/runtime/forest.go`:
- `AddTreeHouse(name string, cfg TreeHouseConfig) error`
- `RemoveTreeHouse(name string) error`
- `AddNim(name string, cfg NimConfig) error`
- `RemoveNim(name string) error`
- `Reload(cfg *Config) error`
- `ListTreeHouses() []TreeHouseInfo`
- `ListNims() []NimInfo`
- `Status() ForestStatus`

### Phase 2: Management API Server
Add `pkg/runtime/api.go`:
- HTTP server on configurable port (default 8080)
- REST endpoints for all operations
- JSON request/response

### Phase 3: CLI Client
Update `cmd/forest/main.go`:
- Add subcommands that call the API
- Use same binary for daemon and client
- Auto-detect if daemon is running

### Phase 4: SIGHUP Support
- Signal handler for config reload
- Graceful component replacement

## File Structure

```
pkg/runtime/
├── forest.go        # Add runtime methods
├── api.go           # NEW: HTTP management API
├── api_handlers.go  # NEW: HTTP handlers
└── ...

cmd/forest/
├── main.go          # Route to daemon or client mode
├── daemon.go        # Daemon (forest run)
├── client.go        # NEW: CLI client commands
└── ...
```

## Example Usage

```bash
# Terminal 1: Start the forest
$ forest standalone
🌲 NimsForest Standalone Mode
📡 Starting embedded NATS server...
✅ NATS server at nats://127.0.0.1:4222
🔧 Management API at http://127.0.0.1:8080
✅ Forest running!
   🏠 TreeHouse:scoring
   🧚 Nim:qualify

# Terminal 2: Add a new component
$ forest add treehouse rescore \
    --subscribes contact.updated \
    --publishes lead.rescored \
    --script ./scripts/treehouses/rescore.lua
✅ Added treehouse 'rescore'

$ forest list
TREEHOUSES:
  scoring   contact.created → lead.scored     [running]
  rescore   contact.updated → lead.rescored   [running]

NIMS:
  qualify   lead.scored → lead.qualified      [running]

$ forest remove treehouse rescore
✅ Removed treehouse 'rescore'

# Or reload entire config
$ forest reload
✅ Config reloaded (added: 1 treehouse, removed: 0)
```

## Security Considerations

1. **Bind to localhost only** by default (127.0.0.1:8080)
2. **Optional authentication** for production
3. **Rate limiting** on API endpoints
4. **Audit logging** for all changes
