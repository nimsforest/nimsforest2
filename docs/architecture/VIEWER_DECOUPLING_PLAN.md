# Viewer Architecture: Decoupled from Core

## Status: ✅ IMPLEMENTED

The viewer (with ebiten/graphics dependencies) has been decoupled from the core. The forest core publishes state to NATS, and external viewers subscribe.

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
│   (no ebiten)      │              │   (has ebiten)     │
│                    │              │                    │
│   ViewmodelPublisher              │   Separate repo    │
│   in internal/viewmodel           │   or process       │
└────────────────────┘              └────────────────────┘
```

## Benefits

| Aspect | Result |
|--------|--------|
| Graphics deps in core | **None** |
| Core runs headless | **Yes** |
| Multiple viewers | **Yes** |
| Viewer on remote machine | **Yes** |
| Independent viewer updates | **Yes** |

## Implementation

### Core Components

- **`internal/viewmodel/publisher.go`** - Publishes state to NATS
- **`pkg/runtime/config.go`** - `ViewerConfig` for NATS subjects
- **`cmd/forest/viewmodel_cli.go`** - CLI help commands

### Configuration

```yaml
# forest.yaml
viewer:
  enabled: true
  subject: forest.viewmodel.state
  event_subject: forest.viewmodel.events
  update_interval: 90     # beats (1 second at 90Hz)
  only_on_change: true
```

### Published Data

State snapshots include:
- Summary statistics (counts, occupancy)
- All Land (nodes) with resources
- All Trees, Treehouses, Nims

Events include:
- `land_added`, `land_removed`, `land_updated`
- `process_added`, `process_removed`, `process_updated`

## For Viewer Developers

See **[docs/guides/VIEWER.md](../guides/VIEWER.md)** for:
- How to subscribe to state
- JSON schema for published data
- Example viewer code (Go, Ebiten)
- Testing with NATS CLI
