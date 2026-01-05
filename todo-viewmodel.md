# NimsForest Viewmodel - Implementation Plan

## Phase 1: Model Layer

### Define Core Structs
- `Land` struct: id, ram, cpu_cores, gpu_vram (optional), gpu_tflops (optional), trees[], treehouses[], nims[]
- `Tree`, `Treehouse`, `Nim` structs: id, ram_allocated, land_id
- `Territory` struct: collection of Land, helper methods for lookups

### Computed Properties
- `Land.Occupancy()` → sum of ram_allocated / total ram as percentage
- `Land.HasGPU()` → gpu_vram > 0

## Phase 2: Data Layer

### NATS HTTP Query
- Function to query NATS monitoring endpoint for cluster state
- Parse node list → Land entries
- Parse running processes → Trees/Treehouses/Nims with land assignments

### Mapper
- `BuildTerritory(natsResponse) → Territory`
- Map NATS node IDs to Land IDs
- Attach processes to their respective Land

## Phase 3: Event Subscription

### NATS JetStream Subscription
- Subscribe to cluster events stream
- Event types to handle:
  - `node.joined` / `node.left`
  - `tree.deployed` / `tree.removed`
  - `treehouse.deployed` / `treehouse.removed`
  - `nim.deployed` / `nim.removed`

### Model Updater
- `ApplyEvent(territory, event) → updated territory`
- Incremental updates, no full rebuild
- Track which Land changed for partial re-render

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
- Initialize: query NATS → build Territory → render
- Subscribe to events → apply updates → re-render affected Land
- Handle reconnect: full query to rebuild state

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

- NATS client for HTTP queries and JetStream subscription
- Terminal UI library (e.g., tcell, bubbletea) or web renderer (decision needed)
