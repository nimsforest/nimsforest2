# Runtime Component Addition

This document describes how NimsForest handles runtime component management.

## Component Types

| Component | Watches | Purpose | Output |
|-----------|---------|---------|--------|
| **Tree** | River (JetStream) | Parse external data, filter | Structured Leaves |
| **TreeHouse** | Wind (pub/sub) | Transform, manipulate Leaves | Modified Leaves |
| **Nim** | Wind (pub/sub) | AI reasoning, decisions | Decision Leaves |

**Data flow:**
```
External Data → River → Trees → Wind → TreeHouses → Wind → Nims → Wind
```

## Architecture

NimsForest uses a **client-server pattern** (like Docker, Kubernetes, Consul):

```
┌────────────────────────────────────────────────────────┐
│                    forest run (daemon)                  │
│                                                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │    Trees     │  │  TreeHouses  │  │    Nims      │ │
│  │ (River→Wind) │  │ (Wind→Wind)  │  │ (Wind→Wind)  │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
│                                                        │
│  ┌──────────────────────────────────────────────────┐ │
│  │              Management API (:8080)               │ │
│  │  GET  /health                - Health check      │ │
│  │  GET  /api/v1/status         - Get status        │ │
│  │  GET  /api/v1/trees          - List trees        │ │
│  │  POST /api/v1/trees          - Add tree          │ │
│  │  DELETE /api/v1/trees/x      - Remove tree       │ │
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
│  forest add tree stripe --config=...                  │
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
forest list trees                # List trees only
forest list treehouses           # List treehouses only
forest list nims                 # List nims only

# Get status
forest status                    # Human-readable
forest status --json             # JSON output

# Add trees (River → Wind, parses external data)
forest add tree <name> --watches=<river-subj> --publishes=<wind-subj> --script=<path>
forest add tree --config=<path.yaml>

# Add treehouses (Wind → Wind, transforms Leaves)
forest add treehouse <name> --subscribes=<subj> --publishes=<subj> --script=<path>
forest add treehouse --config=<path.yaml>

# Add nims (Wind → Wind, AI-powered)
forest add nim <name> --subscribes=<subj> --publishes=<subj> --prompt=<path>
forest add nim --config=<path.yaml>

# Remove components
forest remove tree <name>
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
   🌳 Tree:stripe-parser
   🏠 TreeHouse:scoring
   🧚 Nim:qualify

# Terminal 2 (or from morpheus): Manage components
$ forest list
TREES:
  NAME           WATCHES                 PUBLISHES          SCRIPT              STATUS
  stripe-parser  river.stripe.webhook    payment.completed  ./parse_stripe.lua  [running]

TREEHOUSES:
  NAME      SUBSCRIBES        PUBLISHES      SCRIPT                           STATUS
  scoring   contact.created   lead.scored    ../scripts/treehouses/scoring.lua [running]

NIMS:
  NAME      SUBSCRIBES   PUBLISHES        PROMPT                      STATUS
  qualify   lead.scored  lead.qualified   ../scripts/nims/qualify.md  [running]

# Add a new tree (parses external data)
$ forest add tree salesforce \
    --watches=river.salesforce.webhook \
    --publishes=contact.created \
    --script=./scripts/trees/parse_salesforce.lua
✅ Added tree 'salesforce'

# Add a new treehouse (transforms internal Leaves)
$ forest add treehouse rescore \
    --subscribes=contact.updated \
    --publishes=lead.rescored \
    --script=./scripts/treehouses/rescore.lua
✅ Added treehouse 'rescore'

# Remove components
$ forest remove tree salesforce
✅ Removed tree 'salesforce'

$ forest reload
✅ Configuration reloaded
```

## Configuration Files

### forest.yaml (Main Config)

```yaml
trees:
  stripe-parser:
    watches: river.stripe.webhook
    publishes: payment.completed
    script: ./scripts/trees/parse_stripe.lua

treehouses:
  scoring:
    subscribes: contact.created
    publishes: lead.scored
    script: ./scripts/treehouses/scoring.lua

nims:
  qualify:
    subscribes: lead.scored
    publishes: lead.qualified
    prompt: ./scripts/nims/qualify.md
```

### Tree Config (YAML)

```yaml
# tree.yaml
name: salesforce-parser
watches: river.salesforce.webhook
publishes: contact.created
script: ./scripts/trees/parse_salesforce.lua
```

### TreeHouse Config (YAML)

```yaml
# treehouse.yaml
name: rescore
subscribes: contact.updated
publishes: lead.rescored
script: ./scripts/treehouses/rescore.lua
```

### Nim Config (YAML)

```yaml
# nim.yaml
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
  "trees": [...],
  "treehouses": [...],
  "nims": [...],
  "config_path": "/path/to/forest.yaml"
}
```

### POST /api/v1/trees

Add a new tree (River → Wind parser).

**Request:**
```json
{
  "name": "salesforce",
  "watches": "river.salesforce.webhook",
  "publishes": "contact.created",
  "script": "./parse_salesforce.lua"
}
```

### DELETE /api/v1/trees/{name}

Remove a tree by name.

### POST /api/v1/treehouses

Add a new treehouse (Wind → Wind transformer).

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

Add a new nim (Wind → Wind, AI-powered).

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
