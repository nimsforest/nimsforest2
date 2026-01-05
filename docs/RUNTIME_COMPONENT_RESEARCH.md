# Runtime Component Addition

This document describes how NimsForest handles runtime component management.

## Architecture

NimsForest uses a **client-server pattern** (like Docker, Kubernetes, Consul):

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
│  │  GET  /health                - Health check      │ │
│  │  GET  /api/v1/status         - Get status        │ │
│  │  GET  /api/v1/treehouses     - List treehouses   │ │
│  │  POST /api/v1/treehouses     - Add treehouse     │ │
│  │  DELETE /api/v1/treehouses/x - Remove treehouse  │ │
│  │  GET  /api/v1/nims           - List nims         │ │
│  │  POST /api/v1/nims           - Add nim           │ │
│  │  DELETE /api/v1/nims/x       - Remove nim        │ │
│  │  POST /-/reload              - Reload config     │ │
│  └──────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────┘
                           ▲
                           │ HTTP (localhost only)
                           ▼
┌────────────────────────────────────────────────────────┐
│                 forest CLI (same binary)                │
│                                                        │
│  forest list                                           │
│  forest status                                         │
│  forest add treehouse scoring2 --config=...           │
│  forest remove nim qualifier                           │
│  forest reload                                         │
└────────────────────────────────────────────────────────┘
```

## CLI Commands

### Daemon Commands (start long-running process)

```bash
forest run              # Start with cluster config
forest standalone       # Start standalone mode (dev)
```

### Management Commands (talk to running daemon)

```bash
# List components
forest list                      # List all
forest list treehouses           # List treehouses only
forest list nims                 # List nims only

# Get status
forest status                    # Human-readable
forest status --json             # JSON output

# Add components
forest add treehouse <name> --subscribes=<subj> --publishes=<subj> --script=<path>
forest add treehouse --config=<path.yaml>
forest add nim <name> --subscribes=<subj> --publishes=<subj> --prompt=<path>
forest add nim --config=<path.yaml>

# Remove components
forest remove treehouse <name>
forest remove nim <name>

# Reload config from disk
forest reload
```

## Example Usage

```bash
# Terminal 1: Start the forest (systemd or manual)
$ forest standalone
🌲 NimsForest Standalone Mode
📡 Starting embedded NATS server...
✅ NATS server at nats://127.0.0.1:4222
🔧 Management API at http://127.0.0.1:8080
✅ Forest running!
   🏠 TreeHouse:scoring
   🧚 Nim:qualify

# Terminal 2 (or from morpheus): Manage components
$ forest list
TREEHOUSES:
  NAME      SUBSCRIBES        PUBLISHES      SCRIPT                           STATUS
  scoring   contact.created   lead.scored    ../scripts/treehouses/scoring.lua [running]

NIMS:
  NAME      SUBSCRIBES   PUBLISHES        PROMPT                      STATUS
  qualify   lead.scored  lead.qualified   ../scripts/nims/qualify.md  [running]

$ forest add treehouse rescore \
    --subscribes=contact.updated \
    --publishes=lead.rescored \
    --script=./scripts/treehouses/rescore.lua
✅ Added treehouse 'rescore'

$ forest remove treehouse rescore
✅ Removed treehouse 'rescore'

$ forest reload
✅ Configuration reloaded
```

## Configuration Files

### TreeHouse Config (YAML)

```yaml
# rescore.yaml
name: rescore
subscribes: contact.updated
publishes: lead.rescored
script: ./scripts/treehouses/rescore.lua
```

### Nim Config (YAML)

```yaml
# requalify.yaml
name: requalify
subscribes: lead.rescored
publishes: lead.requalified
prompt: ./scripts/nims/requalify.md
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `NIMSFOREST_API` | Management API address | `127.0.0.1:8080` |

## API Reference

### GET /health

Health check endpoint.

**Response:** `{"status": "ok"}`

### GET /api/v1/status

Get forest status.

**Response:**
```json
{
  "running": true,
  "treehouses": [...],
  "nims": [...],
  "config_path": "/path/to/forest.yaml"
}
```

### POST /api/v1/treehouses

Add a new treehouse.

**Request:**
```json
{
  "name": "scoring2",
  "subscribes": "contact.updated",
  "publishes": "lead.rescored",
  "script": "./rescore.lua"
}
```

### DELETE /api/v1/treehouses/{name}

Remove a treehouse by name.

### POST /api/v1/nims

Add a new nim.

**Request:**
```json
{
  "name": "qualify2",
  "subscribes": "lead.rescored",
  "publishes": "lead.requalified",
  "prompt": "./requalify.md"
}
```

### DELETE /api/v1/nims/{name}

Remove a nim by name.

### POST /-/reload

Reload configuration from disk.

## Security

- API binds to `127.0.0.1` only by default (localhost)
- No authentication required for localhost access
- For remote access, use SSH tunnel or VPN
