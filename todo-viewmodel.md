# NimsForest Viewmodel - Implementation Plan

## Phase 0: E2E Test (Define Success)

### Test: `test/e2e/viewmodel_test.go`
- Spin up embedded NATS cluster (minimum 2 nodes)
- Deploy a tree on node 1, a nim on node 2
- Initialize viewmodel
- Assert: Territory contains 2 Land entries with correct RAM/CPU
- Assert: Tree appears on correct Land with ram_allocated
- Assert: Nim appears on correct Land with ram_allocated
- Assert: Occupancy calculations are correct per Land

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

### Test: CLI Print
- Run `nimsforest viewmodel print`
- Assert: Output lists all Land entries with specs
- Assert: Processes nested under their Land
- Assert: GPU land shows vram

### Test: CLI Summary
- Run `nimsforest viewmodel summary`
- Assert: Output shows total counts
- Assert: Counts match actual deployed resources

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
- Infer process metadata (ram_allocated) from subject/stream config

## Phase 4: CLI Integration

### Command: `nimsforest viewmodel print`
- Reads current Territory state from embedded server
- Prints full nested list to stdout
- Prints once and exits

### Command: `nimsforest viewmodel summary`
- Reads current Territory state from embedded server
- Prints counts only

### Summary Output Format
```
Capacity:
  Land: 2 (ram: 32GB, cpu: 8 cores)
  Manaland: 1 (ram: 32GB, cpu: 8 cores, vram: 24GB)
  Total: 3 land (ram: 64GB, cpu: 16 cores, vram: 24GB)

Usage:
  Trees: 2 (ram: 8GB)
  Treehouses: 1 (ram: 1GB)
  Nims: 1 (ram: 2GB)
  Total: 4 processes (ram: 11GB, 17% of capacity)
```

### Print Output Format
```
Territory: 3 land

Land: node-abc (ram: 16GB, cpu: 4, occupancy: 38%)
  Trees:
    - payment-processor (ram: 4GB)
  Treehouses:
    - scoring (ram: 1GB)
  Nims:
    - qualify (ram: 2GB)

Land: node-xyz (ram: 16GB, cpu: 4, occupancy: 25%)
  Trees:
    - router (ram: 4GB)
  Treehouses: (none)
  Nims: (none)

Land: node-gpu (ram: 32GB, cpu: 8, gpu: 24GB vram, occupancy: 0%)
  Trees: (none)
  Treehouses: (none)
  Nims: (none)
```

## MVP Scope

**Include:**
- Territory model with Land/Tree/Nim structs
- Read cluster state from embedded NATS
- Detect processes via Wind/River subscriptions
- `nimsforest viewmodel print` outputs text list

**Defer:**
- Visual grid rendering (ASCII or GUI)
- Live updating display
- Animations/transitions

## File Structure

```
cmd/forest/
  viewmodel.go      # CLI: nimsforest viewmodel print

internal/viewmodel/
  model.go          # Land, Tree, Treehouse, Nim structs
  territory.go      # Territory collection + methods
  mapper.go         # Cluster state → Territory
  updater.go        # Apply events to Territory
  detector.go       # Wind/River subscription detection
  print.go          # Simple text list output
```

## Dependencies

- Access to embedded `*server.Server` from natsembed package
- Internal NATS client for JetStream subscriptions
