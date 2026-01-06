# NimsForest Viewmodel Webview - MVP Plan

## Overview

Leverage [pogicity-demo](https://github.com/twofactor/pogicity-demo) to create an isometric webview for the viewmodel. The cluster state (Land, Trees, Treehouses, Nims) renders as an interactive isometric world.

## Usage

### Starting the Webview

```bash
# Start the forest daemon (if not already running)
forest daemon &

# Launch the webview
forest viewmodel webview
# → Webview available at http://localhost:8080

# Or specify a port
forest viewmodel webview --port 3000
```

### In the Browser

1. **Open** `http://localhost:8080`
2. **View** the isometric grid showing all Land (nodes) in your cluster
3. **Pan** by clicking and dragging the canvas
4. **Zoom** with mouse wheel
5. **Click** on a Land tile or process sprite to see details in the sidebar
6. **Refresh** button fetches latest cluster state

### What You See

```
┌─────────────────────────────────────────────────────────┐
│  [Refresh]                              World Summary   │
│                                         ─────────────   │
│         🌲                              Land: 3 (1 mana)│
│        ╱  ╲      🏠                     Trees: 2        │
│   ○───┼────┼───○      🌲               Treehouses: 1   │
│       │Land│   ╱  ╲                     Nims: 1         │
│       │ A  │──┼────┼──○                 Occupancy: 34%  │
│        ╲  ╱   │Land│  ╱  ╲              ─────────────   │
│         ○     │ B  │─┼────┼             Selected: Land A│
│                ╲  ╱  │Mana│             Hostname: node-1│
│                 ○    │land│             RAM: 4GB/16GB   │
│                       ╲  ╱              CPU: 4 cores    │
│                        ○                                │
└─────────────────────────────────────────────────────────┘

Legend:
  ○ = Land tile (isometric diamond)
  🌲 = Tree process
  🏠 = Treehouse process  
  ⚙️ = Nim process
  Purple tile = Manaland (GPU-enabled)
```

### Typical Workflow

1. **Monitor cluster** - See at a glance which nodes exist and what's running
2. **Inspect node** - Click a Land to see RAM/CPU usage, hostname
3. **Inspect process** - Click a Tree/Treehouse/Nim to see its details
4. **Check changes** - Hit Refresh after deploying new processes

## Concept Mapping

| Viewmodel | Isometric Visual |
|-----------|------------------|
| `LandViewModel` (regular) | Green terrain tile |
| `LandViewModel` (Manaland/GPU) | Purple/blue terrain tile |
| `TreeViewModel` | Tree sprite on land |
| `TreehouseViewModel` | Cabin sprite on land |
| `NimViewModel` | Factory sprite on land |
| `World` | Isometric grid |

## MVP Architecture

```
┌─────────────────────────────────────────────────┐
│  Browser                                         │
│  ┌─────────────────────────────────────────────┐│
│  │  Next.js + Phaser 3                         ││
│  │  - Isometric grid renderer                  ││
│  │  - Click to select entity                   ││
│  │  - Sidebar with details                     ││
│  └──────────────────┬──────────────────────────┘│
└─────────────────────┼───────────────────────────┘
                      │ fetch /api/viewmodel
┌─────────────────────▼───────────────────────────┐
│  Go Backend (forest)                             │
│  ┌─────────────────────────────────────────────┐│
│  │  GET /api/viewmodel → World JSON            ││
│  │  GET /              → Static web assets     ││
│  └─────────────────────────────────────────────┘│
│  ┌─────────────────────────────────────────────┐│
│  │  Existing viewmodel package                 ││
│  └─────────────────────────────────────────────┘│
└─────────────────────────────────────────────────┘
```

## MVP Scope

**Include:**
- HTTP endpoint returning World as JSON
- Static Next.js + Phaser 3 app rendering isometric grid
- Land tiles with processes stacked on top
- Click entity → show details in sidebar
- Manual refresh button

**Exclude (post-MVP):**
- Live updates via SSE/WebSocket
- Animations
- Tooltips on hover
- Occupancy color coding
- Embedded assets in Go binary
- Auto-open browser flag

---

## Implementation

### 1. Backend: JSON API

**File:** `internal/webview/server.go`

```go
package webview

import (
    "encoding/json"
    "net/http"
    "github.com/yourusername/nimsforest/internal/viewmodel"
)

type Server struct {
    vm *viewmodel.ViewModel
}

func New(vm *viewmodel.ViewModel) *Server {
    return &Server{vm: vm}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    mux := http.NewServeMux()
    mux.HandleFunc("/api/viewmodel", s.handleViewmodel)
    mux.Handle("/", http.FileServer(http.Dir("web/out")))
    mux.ServeHTTP(w, r)
}

func (s *Server) handleViewmodel(w http.ResponseWriter, r *http.Request) {
    s.vm.Refresh()
    world := s.vm.GetWorld()
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(worldToJSON(world))
}
```

**File:** `internal/webview/json.go`

```go
package webview

// WorldJSON - JSON representation with grid positions assigned
type WorldJSON struct {
    Lands   []LandJSON `json:"lands"`
    Summary Summary    `json:"summary"`
}

type LandJSON struct {
    ID          string        `json:"id"`
    Hostname    string        `json:"hostname"`
    RAMTotal    uint64        `json:"ram_total"`
    RAMAllocated uint64       `json:"ram_allocated"`
    CPUCores    int           `json:"cpu_cores"`
    GPUVram     uint64        `json:"gpu_vram,omitempty"`
    Occupancy   float64       `json:"occupancy"`
    IsManaland  bool          `json:"is_manaland"`
    GridX       int           `json:"grid_x"`
    GridY       int           `json:"grid_y"`
    Trees       []ProcessJSON `json:"trees"`
    Treehouses  []ProcessJSON `json:"treehouses"`
    Nims        []ProcessJSON `json:"nims"`
}

type ProcessJSON struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    RAMAllocated uint64 `json:"ram_allocated"`
}

// worldToJSON converts World to WorldJSON, assigning grid positions
func worldToJSON(w *viewmodel.World) WorldJSON {
    lands := w.Lands()
    gridSize := int(math.Ceil(math.Sqrt(float64(len(lands)))))
    
    result := WorldJSON{Lands: make([]LandJSON, len(lands))}
    for i, land := range lands {
        result.Lands[i] = LandJSON{
            ID:          land.ID,
            GridX:       i % gridSize,
            GridY:       i / gridSize,
            // ... map other fields
        }
    }
    return result
}
```

**File:** `cmd/forest/webview.go`

```go
func handleWebview(args []string) {
    port := "8080"
    // parse --port flag
    
    ns, cleanup := getOrStartNATSServer()
    defer cleanup()
    
    vm := viewmodel.New(ns)
    server := webview.New(vm)
    
    fmt.Printf("Starting webview at http://localhost:%s\n", port)
    http.ListenAndServe(":"+port, server)
}
```

---

### 2. Frontend: Project Structure

```
web/
├── app/
│   ├── components/
│   │   ├── game/
│   │   │   ├── ForestScene.ts      # Phaser scene
│   │   │   ├── PhaserGame.tsx      # React wrapper
│   │   │   ├── ForestBoard.tsx     # Main component
│   │   │   └── types.ts            # TypeScript types
│   │   └── ui/
│   │       └── Sidebar.tsx         # Details panel
│   ├── page.tsx
│   └── globals.css
├── public/
│   ├── tiles/
│   │   ├── land.png                # 44x22 isometric tile
│   │   └── manaland.png
│   └── sprites/
│       ├── tree.png
│       ├── treehouse.png
│       └── nim.png
├── package.json
└── next.config.ts
```

---

### 3. Frontend: Types

**File:** `web/app/components/game/types.ts`

```typescript
export interface Process {
  id: string;
  name: string;
  ram_allocated: number;
}

export interface Land {
  id: string;
  hostname: string;
  ram_total: number;
  ram_allocated: number;
  cpu_cores: number;
  gpu_vram?: number;
  occupancy: number;
  is_manaland: boolean;
  grid_x: number;
  grid_y: number;
  trees: Process[];
  treehouses: Process[];
  nims: Process[];
}

export interface World {
  lands: Land[];
  summary: {
    land_count: number;
    manaland_count: number;
    tree_count: number;
    treehouse_count: number;
    nim_count: number;
    occupancy: number;
  };
}

export const TILE_WIDTH = 44;
export const TILE_HEIGHT = 22;

export function gridToIso(gridX: number, gridY: number) {
  return {
    x: (gridX - gridY) * (TILE_WIDTH / 2),
    y: (gridX + gridY) * (TILE_HEIGHT / 2),
  };
}
```

---

### 4. Frontend: Phaser Scene

**File:** `web/app/components/game/ForestScene.ts`

```typescript
import Phaser from 'phaser';
import { World, Land, TILE_WIDTH, TILE_HEIGHT, gridToIso } from './types';

export class ForestScene extends Phaser.Scene {
  private world: World | null = null;
  private onSelect: ((type: string, id: string, landId?: string) => void) | null = null;
  
  constructor() {
    super({ key: 'ForestScene' });
  }
  
  preload() {
    this.load.image('land', '/tiles/land.png');
    this.load.image('manaland', '/tiles/manaland.png');
    this.load.image('tree', '/sprites/tree.png');
    this.load.image('treehouse', '/sprites/treehouse.png');
    this.load.image('nim', '/sprites/nim.png');
  }
  
  create() {
    // Camera drag to pan
    this.input.on('pointermove', (p: Phaser.Input.Pointer) => {
      if (p.isDown) {
        this.cameras.main.scrollX -= p.velocity.x / 10;
        this.cameras.main.scrollY -= p.velocity.y / 10;
      }
    });
    
    // Mouse wheel zoom
    this.input.on('wheel', (_: any, __: any, ___: any, dy: number) => {
      const cam = this.cameras.main;
      cam.zoom = Phaser.Math.Clamp(cam.zoom - dy * 0.001, 0.5, 2);
    });
    
    if (this.world) this.renderWorld();
  }
  
  setWorld(world: World) {
    this.world = world;
    if (this.scene.isActive()) this.renderWorld();
  }
  
  setOnSelect(fn: (type: string, id: string, landId?: string) => void) {
    this.onSelect = fn;
  }
  
  private renderWorld() {
    this.children.removeAll();
    if (!this.world) return;
    
    for (const land of this.world.lands) {
      this.renderLand(land);
    }
    
    // Center camera on grid
    const centerX = this.world.lands.length > 0 ? 0 : 0;
    this.cameras.main.centerOn(centerX, 100);
  }
  
  private renderLand(land: Land) {
    const { x, y } = gridToIso(land.grid_x, land.grid_y);
    const depth = land.grid_x + land.grid_y;
    
    // Land tile
    const tile = this.add.image(x, y, land.is_manaland ? 'manaland' : 'land');
    tile.setDepth(depth);
    tile.setInteractive();
    tile.on('pointerdown', () => this.onSelect?.('land', land.id));
    
    // Processes stacked on land
    let offsetY = -15;
    const allProcesses = [
      ...land.trees.map(p => ({ ...p, type: 'tree' })),
      ...land.treehouses.map(p => ({ ...p, type: 'treehouse' })),
      ...land.nims.map(p => ({ ...p, type: 'nim' })),
    ];
    
    for (const proc of allProcesses) {
      const sprite = this.add.image(x, y + offsetY, proc.type);
      sprite.setDepth(depth + 0.1);
      sprite.setScale(0.5);
      sprite.setInteractive();
      sprite.on('pointerdown', () => this.onSelect?.(proc.type, proc.id, land.id));
      offsetY -= 20;
    }
  }
}
```

---

### 5. Frontend: React Components

**File:** `web/app/components/game/ForestBoard.tsx`

```tsx
'use client';

import { useState, useEffect, useRef } from 'react';
import dynamic from 'next/dynamic';
import { World, Land, Process } from './types';
import Sidebar from '../ui/Sidebar';

const PhaserGame = dynamic(() => import('./PhaserGame'), { ssr: false });

export default function ForestBoard() {
  const [world, setWorld] = useState<World | null>(null);
  const [selected, setSelected] = useState<{type: string; id: string; landId?: string} | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchWorld = async () => {
    const res = await fetch('/api/viewmodel');
    const data = await res.json();
    setWorld(data);
    setLoading(false);
  };

  useEffect(() => { fetchWorld(); }, []);

  if (loading) return <div className="flex items-center justify-center h-screen">Loading...</div>;

  return (
    <div className="flex h-screen bg-gray-900">
      <div className="flex-1 relative">
        <PhaserGame world={world} onSelect={(type, id, landId) => setSelected({type, id, landId})} />
        <button 
          onClick={fetchWorld}
          className="absolute top-4 left-4 px-3 py-1 bg-blue-600 text-white rounded"
        >
          Refresh
        </button>
      </div>
      <Sidebar world={world} selected={selected} onClose={() => setSelected(null)} />
    </div>
  );
}
```

**File:** `web/app/components/ui/Sidebar.tsx`

```tsx
import { World, Land, Process } from '../game/types';

interface Props {
  world: World | null;
  selected: {type: string; id: string; landId?: string} | null;
  onClose: () => void;
}

export default function Sidebar({ world, selected, onClose }: Props) {
  if (!world) return null;

  const selectedLand = selected?.type === 'land' 
    ? world.lands.find(l => l.id === selected.id)
    : selected?.landId 
      ? world.lands.find(l => l.id === selected.landId)
      : null;

  return (
    <div className="w-80 bg-gray-800 text-white p-4 overflow-y-auto">
      <h2 className="text-xl font-bold mb-4">World Summary</h2>
      <div className="mb-4 text-sm">
        <p>Land: {world.summary.land_count} ({world.summary.manaland_count} mana)</p>
        <p>Trees: {world.summary.tree_count}</p>
        <p>Treehouses: {world.summary.treehouse_count}</p>
        <p>Nims: {world.summary.nim_count}</p>
        <p>Occupancy: {world.summary.occupancy.toFixed(0)}%</p>
      </div>
      
      {selected && selectedLand && (
        <div className="border-t border-gray-600 pt-4">
          <div className="flex justify-between items-center mb-2">
            <h3 className="font-bold">{selected.type}: {selected.id}</h3>
            <button onClick={onClose} className="text-gray-400">✕</button>
          </div>
          {selected.type === 'land' && (
            <div className="text-sm">
              <p>Hostname: {selectedLand.hostname}</p>
              <p>RAM: {formatBytes(selectedLand.ram_allocated)}/{formatBytes(selectedLand.ram_total)}</p>
              <p>CPU: {selectedLand.cpu_cores} cores</p>
              {selectedLand.gpu_vram && <p>GPU: {formatBytes(selectedLand.gpu_vram)}</p>}
              <p>Occupancy: {selectedLand.occupancy.toFixed(0)}%</p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function formatBytes(bytes: number): string {
  if (bytes >= 1e9) return (bytes / 1e9).toFixed(0) + 'GB';
  if (bytes >= 1e6) return (bytes / 1e6).toFixed(0) + 'MB';
  return bytes + 'B';
}
```

---

### 6. Assets

MVP can use simple placeholder assets:

| Asset | Description |
|-------|-------------|
| `land.png` | 44x22 green isometric diamond |
| `manaland.png` | 44x22 purple isometric diamond |
| `tree.png` | Simple tree icon (~32x32) |
| `treehouse.png` | Simple house icon (~32x32) |
| `nim.png` | Simple gear/factory icon (~32x32) |

Can source from:
- pogicity-demo assets (MIT license)
- Free isometric asset packs
- Simple colored shapes as placeholder

---

## CLI Usage

```bash
forest viewmodel webview              # Start on :8080
forest viewmodel webview --port 3000  # Custom port
```

---

## Success Criteria

- [ ] `forest viewmodel webview` starts HTTP server
- [ ] `/api/viewmodel` returns World JSON
- [ ] Browser shows isometric grid with Land tiles
- [ ] Processes render as sprites stacked on Land
- [ ] Manaland visually distinct from regular Land
- [ ] Click entity → details in sidebar
- [ ] Refresh button fetches latest state

---

## Post-MVP Enhancements

- SSE/WebSocket for live updates
- Animations (spawn, remove, pulse)
- Tooltips on hover
- Occupancy color coding (green → yellow → red)
- Embedded static assets in Go binary
- `--open` flag to auto-launch browser
- Zoom to selected entity
- Data flow visualization between processes
