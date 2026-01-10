# Plan: Decoupling Ebiten/Viewer from Core

## Executive Summary

**Yes, we can omit ebiten from the core.** The viewer can be hooked in after as a separate, independently-deployed component rather than a compile-time dependency.

## Current State Analysis

### What We Have

1. **Build-tag based conditional compilation** (already in place):
   - `cmd/forest/viewer_disabled.go` with `//go:build !viewer` - default no-op stub
   - No `viewer_enabled.go` exists in this repo (it's meant to be external)

2. **External viewer package** exists at `github.com/nimsforest/nimsforestviewer`

3. **Orphaned dependencies in go.sum**:
   ```
   github.com/hajimehoshi/ebiten/v2 v2.6.6
   github.com/ebitengine/purego v0.6.0
   github.com/nimsforest/nimsforestviewer v0.0.0-20260109190140-6d4b7829f83f
   ```
   These are NOT imported by any Go files in the core repo.

4. **ViewerConfig** defined in `pkg/runtime/config.go` - configuration structure exists

5. **Integration point** in `cmd/forest/main.go:412`:
   ```go
   if runtimeConfig.Viewer != nil && runtimeConfig.Viewer.Enabled {
       startViewer(ctx, ns.InternalServer(), runtimeConfig.Viewer, wind)
   }
   ```

## Problem Statement

- The `go.sum` contains ebiten dependencies even though no code imports them
- This bloats the dependency graph unnecessarily
- Ebiten has heavy system dependencies (OpenGL, etc.) that aren't needed for headless server operation
- The viewer is optional functionality that most deployments won't use

## Proposed Architecture: Hook-In After (Not With)

### Option A: Separate Process Architecture (Recommended)

The viewer runs as a **completely separate process** that connects to the same NATS cluster:

```
┌─────────────────────────────────────────────────────────────┐
│                     NATS CLUSTER                            │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Subject: forest.viewmodel.state                     │    │
│  │  Subject: forest.viewmodel.events                    │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
         ▲                                    │
         │ Publish state                      │ Subscribe to state
         │                                    ▼
┌────────────────────┐              ┌────────────────────┐
│   FOREST CORE      │              │   VIEWER (separate)│
│   (no ebiten)      │              │   - Ebiten GUI     │
│                    │              │   - Web API        │
│   - Lua runtime    │              │   - Smart TV       │
│   - AI Nims        │              │                    │
│   - Trees          │              │   Independent repo:│
│   - TreeHouses     │              │   nimsforestviewer │
└────────────────────┘              └────────────────────┘
```

**Benefits:**
- Zero coupling - core has no viewer dependency
- Viewer can be updated independently
- Can run multiple viewer instances
- Can run viewer on different machines
- Core can run headless on servers without GPU

**Implementation:**
1. Core publishes viewmodel state to `forest.viewmodel.>` subjects
2. Viewer subscribes and renders
3. No build tags needed - just run two processes

### Option B: Plugin via Shared Library (Alternative)

Use Go plugins (`-buildmode=plugin`) to load viewer at runtime:

```go
// In core - check for plugin at startup
if plugin, err := plugin.Open("viewer.so"); err == nil {
    startFn, _ := plugin.Lookup("StartViewer")
    startFn.(func(...))(ctx, ns, cfg, wind)
}
```

**Benefits:**
- Single process
- Dynamic loading
- No compile-time dependency

**Drawbacks:**
- Plugin support varies by platform
- More complex deployment

### Option C: Keep Build Tags but Clean Dependencies (Quick Fix)

Keep the current architecture but properly isolate:

1. Remove orphaned ebiten deps from `go.sum`
2. The `viewer_enabled.go` stays in the external `nimsforestviewer` repo
3. Users who want viewer: `go install github.com/nimsforest/nimsforestviewer/cmd/forest-with-viewer`

## Recommended Implementation: Option A

### Step 1: Clean Core (Immediate)

```bash
# Remove orphaned dependencies
go mod tidy
```

Verify `go.sum` no longer contains ebiten after tidy.

### Step 2: Add Viewmodel Publisher to Core

Create `internal/viewmodel/publisher.go`:

```go
package viewmodel

import (
    "encoding/json"
    "github.com/nats-io/nats.go"
)

type Publisher struct {
    nc *nats.Conn
}

// State represents the forest state for visualization
type State struct {
    Trees      []TreeState      `json:"trees"`
    TreeHouses []TreeHouseState `json:"treehouses"`
    Nims       []NimState       `json:"nims"`
    Events     []RecentEvent    `json:"recent_events"`
    Stats      Stats            `json:"stats"`
}

func (p *Publisher) PublishState(state State) error {
    data, _ := json.Marshal(state)
    return p.nc.Publish("forest.viewmodel.state", data)
}
```

### Step 3: Create Standalone Viewer Repository

The `nimsforestviewer` repo becomes the standalone viewer:

```go
// cmd/viewer/main.go
func main() {
    nc, _ := nats.Connect(natsURL)
    
    // Subscribe to state updates
    nc.Subscribe("forest.viewmodel.state", func(msg *nats.Msg) {
        var state viewmodel.State
        json.Unmarshal(msg.Data, &state)
        renderer.Update(state)
    })
    
    // Run ebiten game loop
    ebiten.RunGame(game)
}
```

### Step 4: Update Configuration

`forest.yaml` becomes:

```yaml
viewer:
  enabled: true
  # Instead of running inline, just enable state publishing
  publish_subject: forest.viewmodel.state
  publish_interval: 90  # beats (1 second at 90Hz)
```

The viewer reads from the NATS subject - no coupling required.

## Migration Path

1. **Phase 1** (This PR): Document the plan, clean `go.sum`
2. **Phase 2**: Add viewmodel state publisher to core
3. **Phase 3**: Update nimsforestviewer to be standalone NATS subscriber
4. **Phase 4**: Remove `viewer_disabled.go` - no longer needed

## Verification Checklist

After implementation:

- [ ] `go mod graph | grep ebiten` returns nothing
- [ ] Core builds without any graphics dependencies
- [ ] Core can run on headless servers
- [ ] Viewer can connect and visualize from separate process
- [ ] `go build ./...` (without tags) produces working binary

## Summary

| Aspect | Current | After Change |
|--------|---------|--------------|
| Ebiten in core | In go.sum (orphaned) | Removed |
| Viewer integration | Build tag compile-time | Runtime NATS subscription |
| Deployment | Single binary with tag | Two binaries or containers |
| Core dependencies | ~15 indirect | ~10 indirect |
| Viewer can run on | Same machine | Any machine in cluster |
