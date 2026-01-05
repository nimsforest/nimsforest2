# NimsForest Viewmodel - Implementation Plan

## Phase 0: E2E Test (Define Success)

### Test: `test/e2e/viewmodel_test.go`
- Spin up embedded NATS cluster (2 nodes)
- Deploy a tree and a nim to different nodes
- Initialize viewmodel
- Assert: Territory contains 2 Land entries with correct RAM/CPU
- Assert: Tree appears on correct Land with ram_allocated
- Assert: Nim appears on correct Land with ram_allocated
- Assert: Occupancy calculations are correct

### Test: Event Updates
- With viewmodel running, start a new tree (creates subscriber)
- Assert: Viewmodel detects new subscription via Wind
- Assert: Territory updates without full rebuild
- Assert: New tree appears on correct Land
- Stop a tree (subscriber disappears)
- Assert: Tree removed from Land
- Remove a node from cluster
- Assert: Land removed from Territory

### Test: GPU Land
- Add node with GPU specs
- Assert: Land.HasGPU() returns true
- Assert: gpu_vram and gpu_tflops populated

This test defines the contract. Implementation is complete when it passes.

## Phase 1: Model Layer

### Define Core Structs
- `Land` struct: id, ram, cpu_cores, gpu_vram (optional), gpu_tflops (optional), trees[], treehouses[], nims[]
- `Tree`, `Treehouse`, `Nim` structs: id, ram_allocated, land_id
- `Territory` struct: collection of Land, helper methods for lookups

### Computed Properties
- `Land.Occupancy()` → sum of ram_allocated / total ram as percentage
- `Land.HasGPU()` → gpu_vram > 0

## Phase 2: Data Layer

### Embedded NATS Access
- nimsforest embeds NATS server → direct access to cluster state
- Use embedded server's `*server.Server` to query cluster info
- Access `Routez`, `Varz`, `Jsz` directly via server API (no HTTP round-trip)

### Cluster State Reader
- `ReadClusterState(server *nats.Server) → ClusterSnapshot`
- Get connected routes → peer nodes → Land entries
- Query JetStream for deployed processes metadata

### Mapper
- `BuildTerritory(ClusterSnapshot) → Territory`
- Map NATS node IDs to Land IDs
- Attach processes to their respective Land

## Phase 3: Event Subscription

### Internal Event Hooks
- Register callbacks on embedded NATS server for cluster events
- Use server's internal event system (route connect/disconnect)

### Process Detection via Wind/River
- No reserved subjects codex - detect organically
- Monitor Wind for new subscriptions appearing (subscriber count changes)
- Monitor River for new streams/consumers being created
- When new subscriber detected → infer tree/treehouse/nim from subject pattern
- When subscriber disappears → process removed
- Subject patterns already encode process type (trees/, nims/, etc.)

### Model Updater
- `ApplyEvent(territory, event) → updated territory`
- Incremental updates, no full rebuild
- Track which Land changed for partial re-render
- Infer process metadata (ram_allocated) from subject/stream config

## Phase 4: View Layer

### Grid Renderer
- Render Territory as grid of squares
- Square positioning algorithm (pack squares, larger ones first)
- Square size based on RAM capacity (define scale factor)

### Land Square Rendering
- Fill color shade based on cpu_cores (lighter = fewer, darker = more)
- Occupancy fill overlay (percentage of square filled)
- Border/outline for square boundary

### GPU Land Extension
- Detect `HasGPU()` lands
- Render vertical mana tube from center
- Tube diameter from gpu_vram
- Tube glow intensity from gpu_tflops

### Process Indicators
- Small markers on land squares for trees/treehouses/nims
- Position around mana tube if GPU land, else distributed on square

## Phase 5: Runtime Integration

### Viewmodel Controller
- Initialize: read embedded server state → build Territory → render
- Register event hooks → apply updates → re-render affected Land
- Handle cluster rejoin: full state read to rebuild Territory

### First Boot
- Start with empty territory
- As nodes appear via events, animate land squares appearing
- As processes deploy, show trees growing on land

## MVP Scope

**Include:**
- Land grid with size/shade encoding
- Occupancy percentage display
- Mana tubes for GPU nodes
- Initial load from NATS query
- Event-driven incremental updates

**Defer:**
- Animations/transitions
- External mana tethers visualization
- Soil/state accumulation visuals
- Process clustering around tubes
- Interactive elements (click, hover)

## File Structure

```
internal/viewmodel/
  model.go          # Land, Tree, Treehouse, Nim structs
  territory.go      # Territory collection + methods
  mapper.go         # NATS response → Territory
  updater.go        # Apply events to Territory
  events.go         # Event type definitions

internal/viewrender/
  grid.go           # Grid layout algorithm
  land.go           # Land square rendering
  tube.go           # Mana tube rendering
  renderer.go       # Main render orchestration
```

## Dependencies

- Access to embedded `*server.Server` from natsembed package
- Internal NATS client for JetStream subscriptions
- Terminal UI library (e.g., tcell, bubbletea) or web renderer (decision needed)
