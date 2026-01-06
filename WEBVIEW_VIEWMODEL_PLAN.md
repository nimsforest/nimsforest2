# NimsForest Viewmodel Webview Integration Plan

## Overview

This plan leverages the [pogicity-demo](https://github.com/twofactor/pogicity-demo) isometric city builder engine to create a visual webview for the NimsForest viewmodel. The viewmodel's cluster state (Land, Trees, Treehouses, Nims) will be rendered as an interactive isometric world.

## Concept Mapping

| Viewmodel Concept | Isometric Visual | pogicity Equivalent |
|-------------------|------------------|---------------------|
| `LandViewModel` (regular node) | Green terrain tile | Grass tile |
| `LandViewModel` (Manaland/GPU) | Purple/blue glowing terrain | Snow tile (custom) |
| `TreeViewModel` | Tree sprite on land | Building (tree asset) |
| `TreehouseViewModel` | Small cabin sprite | Building (house asset) |
| `NimViewModel` | Factory/workshop sprite | Building (commercial) |
| `World` | Full isometric grid | Grid state |

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Web Browser                               │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                    Next.js Application                       ││
│  │  ┌──────────────────┐    ┌──────────────────────────────┐  ││
│  │  │   React State    │◄──►│    Phaser 3 Scene            │  ││
│  │  │   (WorldState)   │    │    (Isometric Renderer)      │  ││
│  │  └────────┬─────────┘    └──────────────────────────────┘  ││
│  └───────────┼─────────────────────────────────────────────────┘│
│              │ SSE/WebSocket                                     │
└──────────────┼───────────────────────────────────────────────────┘
               │
┌──────────────▼───────────────────────────────────────────────────┐
│                      Go Backend (forest)                          │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │   HTTP API Server                                            │ │
│  │   - GET  /api/viewmodel      → World JSON                   │ │
│  │   - GET  /api/viewmodel/sse  → Server-Sent Events stream    │ │
│  │   - POST /api/viewmodel/action → Trigger actions            │ │
│  └──────────────────────────────────────────────────────────────┘ │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │   Viewmodel Package (existing)                               │ │
│  │   - World, LandViewModel, TreeViewModel, etc.                │ │
│  │   - Event system with callbacks                              │ │
│  └─────────────────────────────────────────────────────────────┘ │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │   Embedded NATS Server                                       │ │
│  └─────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

## Implementation Phases

### Phase 1: Backend API Layer

#### 1.1 JSON Serialization for Viewmodel

Add JSON tags and serialization methods to existing viewmodel structs (already present, verify completeness).

**File:** `internal/viewmodel/api.go` (new)

```go
package viewmodel

import (
    "encoding/json"
)

// WorldJSON is a JSON-friendly representation of the World.
type WorldJSON struct {
    Lands       []LandJSON `json:"lands"`
    Summary     Summary    `json:"summary"`
    UpdatedAt   int64      `json:"updated_at"` // Unix timestamp
}

// LandJSON is a JSON-friendly representation of LandViewModel.
type LandJSON struct {
    ID           string             `json:"id"`
    Hostname     string             `json:"hostname"`
    RAMTotal     uint64             `json:"ram_total"`
    RAMAllocated uint64             `json:"ram_allocated"`
    CPUCores     int                `json:"cpu_cores"`
    GPUVram      uint64             `json:"gpu_vram,omitempty"`
    GPUTflops    float64            `json:"gpu_tflops,omitempty"`
    Occupancy    float64            `json:"occupancy"`
    IsManaland   bool               `json:"is_manaland"`
    GridX        int                `json:"grid_x"` // Position in isometric grid
    GridY        int                `json:"grid_y"`
    Trees        []TreeJSON         `json:"trees"`
    Treehouses   []TreehouseJSON    `json:"treehouses"`
    Nims         []NimJSON          `json:"nims"`
}

// TreeJSON, TreehouseJSON, NimJSON - similar structure...

// ToWorldJSON converts World to WorldJSON for API response.
func (w *World) ToWorldJSON() WorldJSON { ... }
```

#### 1.2 HTTP API Server

**File:** `internal/webview/server.go` (new)

```go
package webview

import (
    "encoding/json"
    "net/http"
    "github.com/yourusername/nimsforest/internal/viewmodel"
)

type Server struct {
    vm       *viewmodel.ViewModel
    mux      *http.ServeMux
    sseClients map[chan Event]bool
}

// Routes:
// GET  /api/viewmodel       - Full world state
// GET  /api/viewmodel/sse   - Server-Sent Events for live updates
// GET  /                    - Serve static web assets
```

#### 1.3 Server-Sent Events for Live Updates

```go
// SSE handler streams viewmodel changes to connected clients
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    events := make(chan Event)
    s.sseClients[events] = true
    defer delete(s.sseClients, events)
    
    for event := range events {
        fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, event.Data)
        w.(http.Flusher).Flush()
    }
}
```

#### 1.4 CLI Integration

**File:** `cmd/forest/webview.go` (new)

```go
// handleWebview handles the 'webview' command
func handleWebview(args []string) {
    // Parse flags: --port, --open-browser
    // Start HTTP server
    // Optionally open browser
}
```

**Usage:**
```bash
forest viewmodel webview           # Start webview on :8080
forest viewmodel webview --port 3000
forest viewmodel webview --open    # Auto-open browser
```

---

### Phase 2: Frontend - Fork/Adapt pogicity-demo

#### 2.1 Project Setup

Create `web/` directory in workspace:

```
web/
├── app/
│   ├── components/
│   │   ├── game/
│   │   │   ├── phaser/
│   │   │   │   ├── ForestScene.ts      # Main Phaser scene (adapted from MainScene)
│   │   │   │   ├── PhaserGame.tsx      # React wrapper
│   │   │   │   └── gameConfig.ts
│   │   │   ├── ForestBoard.tsx         # Main component (adapted from GameBoard)
│   │   │   ├── types.ts                # TypeScript types for viewmodel
│   │   │   └── landUtils.ts            # Land positioning utilities
│   │   └── ui/
│   │       ├── Sidebar.tsx             # Info panel showing stats
│   │       ├── ProcessDetails.tsx      # Details for selected process
│   │       └── Summary.tsx             # World summary stats
│   ├── data/
│   │   └── assets.ts                   # Asset registry (Land, Tree, etc.)
│   ├── hooks/
│   │   └── useViewmodel.ts             # Hook for fetching/subscribing to viewmodel
│   ├── globals.css
│   ├── layout.tsx
│   └── page.tsx
├── public/
│   ├── Land/                           # Land tile sprites
│   │   ├── 1x1land.png
│   │   └── 1x1manaland.png
│   ├── Processes/                      # Process sprites
│   │   ├── tree.png
│   │   ├── treehouse.png
│   │   └── nim.png
│   └── UI/                             # UI elements
├── package.json
├── tsconfig.json
├── next.config.ts
└── tailwind.config.ts
```

#### 2.2 Type Definitions

**File:** `web/app/components/game/types.ts`

```typescript
// Mirrors Go viewmodel types

export interface ProcessBase {
  id: string;
  name: string;
  type: 'tree' | 'treehouse' | 'nim';
  ram_allocated: number;
  land_id: string;
  started_at: string;
}

export interface TreeProcess extends ProcessBase {
  type: 'tree';
  subjects: string[];
}

export interface TreehouseProcess extends ProcessBase {
  type: 'treehouse';
  script_path: string;
}

export interface NimProcess extends ProcessBase {
  type: 'nim';
  ai_enabled: boolean;
  model?: string;
}

export type Process = TreeProcess | TreehouseProcess | NimProcess;

export interface Land {
  id: string;
  hostname: string;
  ram_total: number;
  ram_allocated: number;
  cpu_cores: number;
  gpu_vram?: number;
  gpu_tflops?: number;
  occupancy: number;
  is_manaland: boolean;
  grid_x: number;  // Assigned position in isometric grid
  grid_y: number;
  trees: TreeProcess[];
  treehouses: TreehouseProcess[];
  nims: NimProcess[];
}

export interface WorldSummary {
  land_count: number;
  manaland_count: number;
  total_ram: number;
  total_cpu_cores: number;
  total_mana_vram: number;
  tree_count: number;
  treehouse_count: number;
  nim_count: number;
  total_ram_allocated: number;
  occupancy: number;
}

export interface World {
  lands: Land[];
  summary: WorldSummary;
  updated_at: number;
}

// Visual state
export interface SelectedEntity {
  type: 'land' | 'tree' | 'treehouse' | 'nim';
  id: string;
  landId?: string;
}

// Isometric constants (same as pogicity)
export const TILE_WIDTH = 44;
export const TILE_HEIGHT = 22;

// Grid to isometric conversion
export function gridToIso(gridX: number, gridY: number): { x: number; y: number } {
  return {
    x: (gridX - gridY) * (TILE_WIDTH / 2),
    y: (gridX + gridY) * (TILE_HEIGHT / 2),
  };
}
```

#### 2.3 Viewmodel Hook

**File:** `web/app/hooks/useViewmodel.ts`

```typescript
import { useState, useEffect, useCallback } from 'react';
import { World, Land } from '../components/game/types';

export function useViewmodel(apiBase: string = '/api') {
  const [world, setWorld] = useState<World | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  const [connected, setConnected] = useState(false);

  // Initial fetch
  useEffect(() => {
    fetch(`${apiBase}/viewmodel`)
      .then(res => res.json())
      .then(data => {
        setWorld(data);
        setLoading(false);
      })
      .catch(err => {
        setError(err);
        setLoading(false);
      });
  }, [apiBase]);

  // SSE subscription for live updates
  useEffect(() => {
    const eventSource = new EventSource(`${apiBase}/viewmodel/sse`);
    
    eventSource.onopen = () => setConnected(true);
    eventSource.onerror = () => setConnected(false);
    
    eventSource.addEventListener('land_added', (e) => {
      const land: Land = JSON.parse(e.data);
      setWorld(prev => prev ? {
        ...prev,
        lands: [...prev.lands, land]
      } : null);
    });
    
    eventSource.addEventListener('land_removed', (e) => {
      const { id } = JSON.parse(e.data);
      setWorld(prev => prev ? {
        ...prev,
        lands: prev.lands.filter(l => l.id !== id)
      } : null);
    });
    
    eventSource.addEventListener('process_added', (e) => {
      const { land_id, process } = JSON.parse(e.data);
      setWorld(prev => {
        if (!prev) return null;
        return {
          ...prev,
          lands: prev.lands.map(l => 
            l.id === land_id 
              ? addProcessToLand(l, process)
              : l
          )
        };
      });
    });
    
    // ... more event handlers
    
    return () => eventSource.close();
  }, [apiBase]);

  return { world, loading, error, connected };
}
```

#### 2.4 Main Phaser Scene

**File:** `web/app/components/game/phaser/ForestScene.ts`

```typescript
import Phaser from 'phaser';
import { World, Land, Process, TILE_WIDTH, TILE_HEIGHT, gridToIso } from '../types';

export class ForestScene extends Phaser.Scene {
  private world: World | null = null;
  private landSprites: Map<string, Phaser.GameObjects.Image> = new Map();
  private processSprites: Map<string, Phaser.GameObjects.Image> = new Map();
  private selectedEntity: string | null = null;
  
  // Callbacks to React
  private onEntitySelect: ((entity: { type: string; id: string }) => void) | null = null;
  
  constructor() {
    super({ key: 'ForestScene' });
  }
  
  preload(): void {
    // Load land tiles
    this.load.image('land', '/Land/1x1land.png');
    this.load.image('manaland', '/Land/1x1manaland.png');
    
    // Load process sprites
    this.load.image('tree', '/Processes/tree.png');
    this.load.image('treehouse', '/Processes/treehouse.png');
    this.load.image('nim', '/Processes/nim.png');
    this.load.image('nim_ai', '/Processes/nim_ai.png');
    
    // Load UI elements
    this.load.image('highlight', '/UI/highlight.png');
  }
  
  create(): void {
    // Set up camera controls (pan, zoom)
    this.setupCamera();
    
    // Set up input handlers
    this.input.on('pointerdown', this.handleClick, this);
    
    // Initial render if world is set
    if (this.world) {
      this.renderWorld();
    }
  }
  
  // Called from React when world updates
  updateWorld(newWorld: World): void {
    const oldWorld = this.world;
    this.world = newWorld;
    
    if (!oldWorld) {
      // First render
      this.renderWorld();
    } else {
      // Incremental update
      this.updateLands(oldWorld.lands, newWorld.lands);
    }
  }
  
  private renderWorld(): void {
    if (!this.world) return;
    
    // Clear existing sprites
    this.landSprites.forEach(sprite => sprite.destroy());
    this.processSprites.forEach(sprite => sprite.destroy());
    this.landSprites.clear();
    this.processSprites.clear();
    
    // Render each land
    for (const land of this.world.lands) {
      this.renderLand(land);
    }
    
    // Sort by depth for proper isometric rendering
    this.children.sort('depth');
  }
  
  private renderLand(land: Land): void {
    const { x, y } = gridToIso(land.grid_x, land.grid_y);
    
    // Land tile
    const tileKey = land.is_manaland ? 'manaland' : 'land';
    const landSprite = this.add.image(x, y, tileKey);
    landSprite.setDepth(land.grid_x + land.grid_y);
    landSprite.setInteractive();
    landSprite.setData('landId', land.id);
    this.landSprites.set(land.id, landSprite);
    
    // Render processes on this land
    this.renderProcesses(land);
  }
  
  private renderProcesses(land: Land): void {
    const { x, y } = gridToIso(land.grid_x, land.grid_y);
    const processes: Process[] = [
      ...land.trees,
      ...land.treehouses,
      ...land.nims
    ];
    
    // Stack processes vertically on the land tile
    let offsetY = -20; // Start above the land tile
    for (const proc of processes) {
      const spriteKey = this.getProcessSpriteKey(proc);
      const sprite = this.add.image(x, y + offsetY, spriteKey);
      sprite.setDepth(land.grid_x + land.grid_y + 0.1);
      sprite.setInteractive();
      sprite.setData('processId', proc.id);
      sprite.setData('landId', land.id);
      this.processSprites.set(proc.id, sprite);
      offsetY -= 15; // Stack upward
    }
  }
  
  private getProcessSpriteKey(proc: Process): string {
    switch (proc.type) {
      case 'tree': return 'tree';
      case 'treehouse': return 'treehouse';
      case 'nim': return proc.ai_enabled ? 'nim_ai' : 'nim';
    }
  }
  
  // ... more methods for camera, selection, animations
}
```

#### 2.5 React Component

**File:** `web/app/components/game/ForestBoard.tsx`

```tsx
'use client';

import { useState, useRef, useEffect } from 'react';
import dynamic from 'next/dynamic';
import { useViewmodel } from '@/app/hooks/useViewmodel';
import Sidebar from '../ui/Sidebar';
import Summary from '../ui/Summary';
import { SelectedEntity } from './types';

const PhaserGame = dynamic(() => import('./phaser/PhaserGame'), { ssr: false });

export default function ForestBoard() {
  const { world, loading, error, connected } = useViewmodel();
  const [selectedEntity, setSelectedEntity] = useState<SelectedEntity | null>(null);
  const gameRef = useRef<PhaserGameHandle>(null);

  if (loading) return <div className="flex items-center justify-center h-screen">Loading...</div>;
  if (error) return <div className="text-red-500">Error: {error.message}</div>;

  return (
    <div className="flex h-screen bg-gray-900">
      {/* Main game area */}
      <div className="flex-1 relative">
        <PhaserGame 
          ref={gameRef}
          world={world}
          onEntitySelect={setSelectedEntity}
        />
        
        {/* Connection indicator */}
        <div className={`absolute top-4 right-4 px-2 py-1 rounded text-sm ${
          connected ? 'bg-green-500' : 'bg-red-500'
        }`}>
          {connected ? '● Live' : '○ Disconnected'}
        </div>
      </div>
      
      {/* Sidebar */}
      <Sidebar 
        world={world}
        selectedEntity={selectedEntity}
        onClose={() => setSelectedEntity(null)}
      />
    </div>
  );
}
```

---

### Phase 3: Visual Assets

#### 3.1 Asset Requirements

| Asset | Description | Size | Notes |
|-------|-------------|------|-------|
| `1x1land.png` | Regular land tile | 44x22 | Green/brown isometric diamond |
| `1x1manaland.png` | GPU-enabled land | 44x22 | Purple/blue glow effect |
| `tree.png` | Tree process | ~32x48 | Stylized tree icon |
| `treehouse.png` | Treehouse process | ~32x48 | Small cabin/house |
| `nim.png` | Nim process | ~32x48 | Factory/workshop |
| `nim_ai.png` | AI-enabled Nim | ~32x48 | Glowing factory |
| `highlight.png` | Selection highlight | 44x22 | Semi-transparent overlay |

#### 3.2 Asset Generation Options

1. **Use pogicity assets directly** - Fork and adapt existing sprites (MIT licensed)
2. **AI-generated** - Use DALL-E/Midjourney for custom isometric sprites
3. **Purchased assets** - itch.io has many isometric asset packs
4. **Programmatic** - Generate simple shapes via Phaser graphics

---

### Phase 4: Advanced Features

#### 4.1 Visual Animations

```typescript
// Land joining animation (fade in + scale)
land.setAlpha(0).setScale(0.5);
this.tweens.add({
  targets: land,
  alpha: 1,
  scale: 1,
  duration: 500,
  ease: 'Back.easeOut'
});

// Process spawning (pop in)
process.setScale(0);
this.tweens.add({
  targets: process,
  scale: 1,
  duration: 300,
  ease: 'Bounce.easeOut'
});

// Process removal (shrink + fade)
this.tweens.add({
  targets: process,
  scale: 0,
  alpha: 0,
  duration: 200,
  onComplete: () => process.destroy()
});
```

#### 4.2 Occupancy Visualization

```typescript
// Color-code land by occupancy
private getLandTint(occupancy: number): number {
  if (occupancy < 50) return 0x00ff00; // Green - healthy
  if (occupancy < 80) return 0xffff00; // Yellow - warning
  return 0xff0000; // Red - critical
}

// Progress bar on land tile
private renderOccupancyBar(land: Land, x: number, y: number): void {
  const bar = this.add.graphics();
  bar.fillStyle(0x333333);
  bar.fillRect(x - 20, y + 10, 40, 4);
  bar.fillStyle(this.getLandTint(land.occupancy));
  bar.fillRect(x - 20, y + 10, 40 * (land.occupancy / 100), 4);
}
```

#### 4.3 Tooltip System

```typescript
// Show info on hover
landSprite.on('pointerover', () => {
  this.showTooltip(landSprite.x, landSprite.y - 30, `
    ${land.hostname}
    RAM: ${formatBytes(land.ram_allocated)}/${formatBytes(land.ram_total)}
    CPU: ${land.cpu_cores} cores
    ${land.is_manaland ? `GPU: ${formatBytes(land.gpu_vram)}` : ''}
  `);
});
```

#### 4.4 Grid Layout Algorithm

```typescript
// Position lands in a grid automatically
function assignGridPositions(lands: Land[]): Land[] {
  const gridSize = Math.ceil(Math.sqrt(lands.length));
  return lands.map((land, i) => ({
    ...land,
    grid_x: i % gridSize,
    grid_y: Math.floor(i / gridSize)
  }));
}

// Alternative: Cluster by characteristics
function clusterLayout(lands: Land[]): Land[] {
  const manalands = lands.filter(l => l.is_manaland);
  const regulars = lands.filter(l => !l.is_manaland);
  
  // Manalands on the left, regulars on the right
  return [
    ...manalands.map((l, i) => ({ ...l, grid_x: 0, grid_y: i })),
    ...regulars.map((l, i) => ({ ...l, grid_x: 2, grid_y: i }))
  ];
}
```

---

### Phase 5: Build & Deployment

#### 5.1 Embedded Web Assets

Option A: **Embed in Go binary**

```go
//go:embed web/dist/*
var webAssets embed.FS

func (s *Server) serveStaticFiles() http.Handler {
    return http.FileServer(http.FS(webAssets))
}
```

Option B: **Separate web server**

```bash
# Development
cd web && npm run dev  # Next.js dev server

# Production
cd web && npm run build && npm run start
```

#### 5.2 Development Workflow

```bash
# Terminal 1: Backend
forest daemon  # or forest viewmodel webview --dev

# Terminal 2: Frontend
cd web && npm run dev

# Frontend proxies /api/* to backend
```

#### 5.3 Production Build

```bash
# Build frontend
cd web && npm run build

# Embed in Go binary
go build -o forest ./cmd/forest

# Single binary serves both API and web UI
./forest viewmodel webview
```

---

## Implementation Order

1. **Phase 1.1-1.2**: Backend API (`/api/viewmodel` endpoint)
2. **Phase 2.1-2.2**: Frontend scaffold (types, project structure)
3. **Phase 2.4**: Basic Phaser scene rendering static world
4. **Phase 3**: Visual assets (can use placeholders initially)
5. **Phase 2.3**: Live updates via SSE
6. **Phase 2.5**: React integration with sidebar
7. **Phase 4**: Animations, tooltips, advanced features
8. **Phase 5**: Build system, embedding

## Timeline Estimate

| Phase | Effort |
|-------|--------|
| Phase 1 (Backend) | 2-3 days |
| Phase 2 (Frontend) | 4-5 days |
| Phase 3 (Assets) | 1-2 days |
| Phase 4 (Polish) | 2-3 days |
| Phase 5 (Deploy) | 1 day |
| **Total** | **~2 weeks** |

## Dependencies

### Go
- No new dependencies (uses standard library `net/http`)

### Node.js (web/)
```json
{
  "dependencies": {
    "next": "^16.0.0",
    "react": "^19.0.0",
    "react-dom": "^19.0.0",
    "phaser": "^3.90.0"
  },
  "devDependencies": {
    "typescript": "^5.0.0",
    "tailwindcss": "^4.0.0",
    "@types/node": "^22.0.0",
    "@types/react": "^19.0.0"
  }
}
```

## Success Criteria

- [ ] `forest viewmodel webview` starts HTTP server
- [ ] Browser displays isometric grid with Land tiles
- [ ] Each Land shows processes as stacked sprites
- [ ] Manaland visually distinct from regular Land
- [ ] Clicking entity shows details in sidebar
- [ ] New Land/process appears automatically (live updates)
- [ ] Removed Land/process disappears with animation
- [ ] Summary stats display correctly
- [ ] Works with 1-20 Land nodes
- [ ] Responsive and smooth (60 FPS)

## Future Enhancements

- **Zoom levels**: City view (all lands) → District view (land details) → Building view (process internals)
- **Data flow visualization**: Animated particles flowing between processes
- **Historical view**: Playback of cluster state over time
- **Command palette**: Trigger actions like "spawn tree" from UI
- **Multi-cluster support**: View multiple clusters side-by-side
- **Dark/light themes**: Toggle based on system preference
