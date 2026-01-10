# Viewer Decoupling: Ebiten Removed from Core

## Status: ✅ IMPLEMENTED

**Ebiten has been successfully removed from the core.** The viewer now runs as a separate process that subscribes to NATS for state updates.

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

## What Was Implemented

### Changes Made

1. **Cleaned dependencies** (`go mod tidy`)
   - Removed orphaned ebiten dependencies from `go.sum`
   - go.sum reduced from 101 to 59 lines

2. **Created viewmodel publisher** (`internal/viewmodel/publisher.go`)
   - Publishes `World` state to NATS subject `forest.viewmodel.state`
   - Publishes events to `forest.viewmodel.events`
   - Supports periodic publishing and change-only publishing

3. **Updated main.go**
   - Replaced build-tag viewer with viewmodel publisher
   - No longer calls `startViewer()` - just publishes to NATS

4. **Updated config** (`pkg/runtime/config.go`, `config/forest.yaml`)
   - `ViewerConfig` now configures NATS subjects instead of graphics
   - Removed `web_addr`, `smarttv` fields (viewer handles these)

5. **Renamed/updated CLI** (`cmd/forest/viewmodel_cli.go`)
   - Removed build tag constraint (`//go:build !viewer`)
   - Updated help to explain external viewer architecture

## Verification ✅

- [x] `go mod graph | grep ebiten` returns nothing
- [x] `go build ./...` succeeds without graphics dependencies
- [x] All tests pass (`go test ./...`)
- [x] Core can run on headless servers

## Architecture Summary

| Aspect | Before | After |
|--------|--------|-------|
| Ebiten in core | In go.sum (orphaned) | **Removed** |
| Viewer integration | Build tag compile-time | **Runtime NATS subscription** |
| Deployment | Single binary with tag | **Separate processes** |
| go.sum lines | 101 | **59** |
| Graphics deps in core | Yes (transitive) | **None** |

## For External Viewer

The `nimsforestviewer` repository should be updated to:

```go
// Subscribe to state from any NATS-connected machine
nc, _ := nats.Connect(natsURL)
nc.Subscribe("forest.viewmodel.state", func(msg *nats.Msg) {
    var state viewmodel.PublishedState
    json.Unmarshal(msg.Data, &state)
    renderer.Update(state)
})
```
