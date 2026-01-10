# NimsForest Viewer Guide

## Overview

The NimsForest viewer is **decoupled from the core**. This means:

- The core (`forest`) has **zero graphics dependencies** (no ebiten, OpenGL, etc.)
- The viewer runs as a **separate process** that subscribes to NATS
- You can run the viewer on **any machine** that can reach the NATS cluster
- Multiple viewers can connect simultaneously

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     NATS CLUSTER                            │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  forest.viewmodel.state   ← Full state snapshots    │    │
│  │  forest.viewmodel.events  ← Real-time change events │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
         ▲                                    │
         │ Publishes                          │ Subscribes
         │                                    ▼
┌────────────────────┐              ┌────────────────────┐
│   FOREST CORE      │              │   EXTERNAL VIEWER  │
│                    │              │                    │
│   ViewmodelPublisher              │   - Ebiten GUI     │
│   publishes state  │              │   - Web dashboard  │
│   to NATS subjects │              │   - Smart TV app   │
│                    │              │   - CLI monitor    │
│   No graphics deps │              │   Has graphics deps│
└────────────────────┘              └────────────────────┘
```

## Enabling State Publishing

In your `forest.yaml`:

```yaml
viewer:
  enabled: true
  subject: forest.viewmodel.state      # State snapshots (default)
  event_subject: forest.viewmodel.events  # Change events (default)
  update_interval: 90     # Beats (90 = 1 second at 90Hz)
  only_on_change: true    # Only publish when state changes
```

When `enabled: true`, the forest publishes:

| Subject | Content | Frequency |
|---------|---------|-----------|
| `forest.viewmodel.state` | Full `PublishedState` JSON | Every `update_interval` or on change |
| `forest.viewmodel.events` | Individual `Event` JSON | On every state change |

## Published State Format

The `forest.viewmodel.state` subject receives JSON like:

```json
{
  "timestamp": "2026-01-10T12:00:00Z",
  "summary": {
    "land_count": 2,
    "manaland_count": 0,
    "total_ram": 34359738368,
    "total_cpu_cores": 8,
    "tree_count": 2,
    "treehouse_count": 1,
    "nim_count": 2,
    "occupancy": 15.5
  },
  "lands": [
    {
      "id": "node-1",
      "hostname": "forest-server-1",
      "ram_total": 17179869184,
      "cpu_cores": 4,
      "trees": [...],
      "treehouses": [...],
      "nims": [...]
    }
  ],
  "trees": [...],
  "treehouses": [...],
  "nims": [...]
}
```

## Building an External Viewer

### Minimal Go Example

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"

    "github.com/nats-io/nats.go"
)

// PublishedState matches the JSON from forest.viewmodel.state
type PublishedState struct {
    Timestamp  string  `json:"timestamp"`
    Summary    Summary `json:"summary"`
    // ... add other fields as needed
}

type Summary struct {
    LandCount      int     `json:"land_count"`
    TreeCount      int     `json:"tree_count"`
    TreehouseCount int     `json:"treehouse_count"`
    NimCount       int     `json:"nim_count"`
    Occupancy      float64 `json:"occupancy"`
}

func main() {
    // Connect to NATS (same cluster as forest)
    nc, err := nats.Connect("nats://localhost:4222")
    if err != nil {
        log.Fatal(err)
    }
    defer nc.Close()

    // Subscribe to state updates
    nc.Subscribe("forest.viewmodel.state", func(msg *nats.Msg) {
        var state PublishedState
        if err := json.Unmarshal(msg.Data, &state); err != nil {
            log.Printf("Failed to parse state: %v", err)
            return
        }

        fmt.Printf("Forest State at %s:\n", state.Timestamp)
        fmt.Printf("  Land: %d, Trees: %d, Nims: %d\n",
            state.Summary.LandCount,
            state.Summary.TreeCount,
            state.Summary.NimCount)
        fmt.Printf("  Occupancy: %.1f%%\n", state.Summary.Occupancy)
    })

    // Keep running
    select {}
}
```

### With Ebiten (GUI Viewer)

```go
package main

import (
    "encoding/json"
    "sync"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/nats-io/nats.go"
)

type Game struct {
    state PublishedState
    mu    sync.RWMutex
}

func (g *Game) Update() error {
    return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
    g.mu.RLock()
    defer g.mu.RUnlock()
    // Draw g.state to screen...
}

func (g *Game) Layout(w, h int) (int, int) {
    return 1920, 1080
}

func main() {
    game := &Game{}

    // Connect to NATS
    nc, _ := nats.Connect("nats://localhost:4222")
    nc.Subscribe("forest.viewmodel.state", func(msg *nats.Msg) {
        var state PublishedState
        json.Unmarshal(msg.Data, &state)
        
        game.mu.Lock()
        game.state = state
        game.mu.Unlock()
    })

    // Run Ebiten game loop
    ebiten.RunGame(game)
}
```

## Quick Testing with NATS CLI

Without writing any code, you can observe the viewmodel:

```bash
# Watch state updates
nats sub forest.viewmodel.state

# Watch individual events
nats sub forest.viewmodel.events

# Pretty-print state with jq
nats sub forest.viewmodel.state | jq .
```

## CLI Commands

The forest CLI provides viewmodel info commands:

```bash
# Show architecture overview
forest viewmodel summary

# Show how to connect
forest viewmodel print

# Show external viewer instructions
forest viewmodel viewer
```

## Event Types

Events published to `forest.viewmodel.events`:

| Event Type | When |
|------------|------|
| `land_added` | New node joins cluster |
| `land_removed` | Node leaves cluster |
| `land_updated` | Node resources change |
| `process_added` | Tree/Treehouse/Nim starts |
| `process_removed` | Tree/Treehouse/Nim stops |
| `process_updated` | Process state changes |

Event JSON format:

```json
{
  "type": "process_added",
  "timestamp": "2026-01-10T12:00:00Z",
  "land_id": "node-1",
  "process_id": "qualify-nim",
  "data": { ... }
}
```

## Multiple Viewers

You can run multiple viewers simultaneously:

- **Ops Dashboard**: Web app showing cluster health
- **Smart TV**: Visual representation in office
- **Alerting**: Service that watches for anomalies
- **Metrics**: Prometheus exporter

All subscribe to the same NATS subjects, all get the same data.

## Troubleshooting

### No state being published

1. Check `viewer.enabled: true` in `forest.yaml`
2. Verify forest is running: `forest status`
3. Check NATS connectivity: `nats sub test.subject`

### State updates are slow

- Decrease `update_interval` (lower = faster)
- Set `only_on_change: false` for constant updates

### Can't connect from remote viewer

- Ensure NATS is accessible from viewer machine
- Check firewall allows port 4222
- Use full URL: `nats://forest-server:4222`
