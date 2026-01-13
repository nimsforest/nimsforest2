# Plan: 0 A.D. as NimsForest Viewer Interface

**Status**: Planning
**Goal**: Use 0 A.D. RTS game engine as an interactive visualization and control interface for nimsforest2
**Pattern**: External viewer subscribing to `forest.viewmodel.*` NATS subjects

---

## Executive Summary

Create a **0 A.D.-based viewer** that visualizes and controls the nimsforest cluster using an RTS game interface. The viewer subscribes to the existing viewmodel state published on NATS and maps forest components to game entities:

- **Lands** → territories/bases on the map
- **Nims** → units/workers
- **Trees/Treehouses** → buildings/structures
- **Resources (RAM/CPU/GPU)** → game resources (food/wood/stone/metal)
- **Events** → visual effects and notifications
- **Commands** → user interactions that emit Wind events

---

## 1. Architecture Overview

### 1.1 Existing NimsForest Viewmodel

Current state publishing:
```go
// internal/viewmodel/viewmodel.go publishes to NATS:
forest.viewmodel.state   // Full World snapshot (JSON)
forest.viewmodel.events  // Real-time change events (JSON)
```

Viewmodel structure:
```go
type World struct {
    Lands []LandViewModel
}

type LandViewModel struct {
    ID         string   // Node identifier
    Hostname   string
    RAMTotal   uint64   // Total RAM in bytes
    CPUCores   int
    GPUVram    uint64   // 0 if no GPU
    Trees      []TreeViewModel
    Treehouses []TreehouseViewModel
    Nims       []NimViewModel
}

type Process struct {
    ID           string
    Name         string
    Type         ProcessType  // tree, treehouse, nim
    RAMAllocated uint64
    Subjects     []string     // NATS subjects subscribed
}
```

### 1.2 0 A.D. Viewer Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    NimsForest Core                          │
│  (runs independently, publishes to NATS)                    │
└──────────────────────┬──────────────────────────────────────┘
                       │ NATS pub/sub
                       │ forest.viewmodel.state
                       │ forest.viewmodel.events
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              0 A.D. Viewer (External Process)               │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  NATS Subscriber (Go Bridge)                         │  │
│  │  - Connects to NATS                                  │  │
│  │  - Subscribes to viewmodel.state & events            │  │
│  │  - Parses JSON → internal data structures            │  │
│  └──────────────┬───────────────────────────────────────┘  │
│                 │ IPC (Unix socket)                         │
│                 ▼                                            │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  0 A.D. Mod: nimsforest_viewer                       │  │
│  │  - Reads state updates from bridge                   │  │
│  │  - Maps forest → game entities                       │  │
│  │  - Renders visualization                             │  │
│  │  - Handles user input                                │  │
│  └──────────────┬───────────────────────────────────────┘  │
│                 │ Game Engine                               │
│                 ▼                                            │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  0 A.D. Engine (Pyrogenesis)                         │  │
│  │  - Renders 3D world                                  │  │
│  │  - Displays entities and resources                   │  │
│  │  - User interaction via mouse/keyboard               │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                       │ User actions
                       ▼
                   Wind events
              (published back to NATS)
```

---

## 2. Entity Mapping

### 2.1 Lands → Territories

Each `LandViewModel` becomes a territory on the map:

**Visual Representation:**
- **Base/Town Center**: Represents the Land itself
- **Territory Color**: Different color per Land
- **Resource Display**: RAM/CPU/GPU shown as harvestable resources
- **Position**: Distributed across the map strategically

**Entity Template:**
```javascript
// 0 A.D. entity: structures/nimsforest_land.xml
{
    "Identity": {
        "GenericName": "Land Node",
        "Icon": "structures/land.png"
    },
    "Position": {
        "Anchor": "upright"
    },
    "ResourceSupply": {
        "Amount": {RAM_TOTAL},  // Scaled representation
        "Type": "ram"
    },
    "ProductionQueue": {
        // Can deploy new processes
    }
}
```

**Manaland (GPU-enabled Land):**
- Special visual effect (glowing/magical)
- Additional resource: "Mana" (represents GPU/VRAM)
- Different building appearance

### 2.2 Nims → Units

Each `NimViewModel` becomes a unit:

**Visual Representation:**
- **Unit Type**: Worker/specialist unit
- **Name**: Nim's name displayed
- **Animation**: Idle/working based on activity
- **Selection**: Click to see details panel

**Properties:**
- Health bar → RAM usage
- Attack/Defense → Processing capacity
- Movement → Task execution
- Garrison → Paused state

**Entity Template:**
```javascript
// 0 A.D. entity: units/nimsforest_nim.xml
{
    "Identity": {
        "GenericName": "Nim",
        "Classes": ["Nim"]
    },
    "Health": {
        "Max": {RAM_ALLOCATED}
    },
    "UnitAI": {
        "DefaultStance": "passive"
    },
    "Selectable": {
        "Overlay": {
            "Texture": "selection/nimsforest_nim.png"
        }
    }
}
```

**AI-Powered Nims:**
- Glowing effect
- Special icon indicator
- Different unit appearance

### 2.3 Trees → Resource Gathering Buildings

`TreeViewModel` instances become structures that "gather" from the River:

**Visual Representation:**
- **Building**: Mill/storehouse appearance
- **Animation**: Active processing indicator
- **Connection**: Visual link to River stream

**Entity Template:**
```javascript
// 0 A.D. entity: structures/nimsforest_tree.xml
{
    "Identity": {
        "GenericName": "Tree Parser",
        "Classes": ["Tree", "Structure"]
    },
    "Cost": {
        "Resources": {
            "ram": {RAM_ALLOCATED}
        }
    },
    "ProductionQueue": {
        "Entities": ["leaf"]  // Emits leaves
    }
}
```

### 2.4 Treehouses → Processing Buildings

`TreehouseViewModel` instances become worker buildings:

**Visual Representation:**
- **Building**: Workshop appearance
- **Script Path**: Displayed in building info
- **Activity**: Smoke/light when processing

**Entity Template:**
```javascript
// 0 A.D. entity: structures/nimsforest_treehouse.xml
{
    "Identity": {
        "GenericName": "TreeHouse",
        "SpecificName": {SCRIPT_NAME}
    },
    "ProductionQueue": {
        // Can process subscribed events
    }
}
```

### 2.5 Resources → Game Resources

Map nimsforest resources to 0 A.D.'s resource system:

| NimsForest Resource | 0 A.D. Resource | Display |
|---------------------|-----------------|---------|
| RAM (bytes) | Food | Converts to GB for readability |
| CPU (cores) | Wood | Direct mapping |
| GPU VRAM (bytes) | Stone | Converts to GB, only on Manaland |
| GPU Compute (TFLOPS) | Metal | Direct mapping |

**Resource Panel:**
```
Resources Available:
  RAM:  24GB / 32GB  (75%)
  CPU:  6 / 8 cores  (75%)
  VRAM: 18GB / 24GB  (75%)  [Manaland only]
```

---

## 3. Visualization Features

### 3.1 Map Layout

**Strategic Placement:**
- Each Land gets a territory quadrant
- Lands arranged geographically (can map to actual server locations)
- Manalands placed in special "magical" biomes

**Map Themes:**
```
Standard Layout (3 Lands):
┌────────────┬────────────┐
│  Land 1    │  Land 2    │
│  (Base)    │  (Base)    │
│            │            │
│ • Trees    │ • Nims     │
│ • Houses   │            │
└────────────┴────────────┘
        Manaland
        (Center)
    • GPU indicator
    • Special effects
```

### 3.2 Visual Effects

**Events → Visual Feedback:**

| Event Type | Visual Effect |
|------------|---------------|
| Land joined | Base construction animation |
| Land disconnected | Base destruction/fade |
| Process started | Unit/building spawned |
| Process stopped | Unit/building removed |
| High CPU usage | Red warning indicator |
| Low memory | Yellow caution indicator |
| Task completed | Green checkmark particle effect |
| Error occurred | Red X particle effect |

### 3.3 Information Panels

**Land Selection Panel:**
```
╔════════════════════════════════════╗
║  Land: node-abc                    ║
║  Hostname: server1.example.com     ║
╟────────────────────────────────────╢
║  Resources:                        ║
║    RAM:  16GB / 32GB (50%)        ║
║    CPU:  4 / 8 cores (50%)        ║
║    GPU:  None                      ║
╟────────────────────────────────────╢
║  Processes (3):                    ║
║    • payment-tree      [4GB]      ║
║    • scoring-house     [2GB]      ║
║    • qualify-nim       [2GB]      ║
╟────────────────────────────────────╢
║  [Deploy Process] [View Details]  ║
╚════════════════════════════════════╝
```

**Nim Selection Panel:**
```
╔════════════════════════════════════╗
║  Nim: qualify-nim                  ║
║  Type: Business Logic              ║
║  AI: Enabled (Claude)              ║
╟────────────────────────────────────╢
║  Status: Active                    ║
║  RAM: 2GB / 2GB (100%)            ║
║  Subjects:                         ║
║    • lead.new                      ║
║    • lead.updated                  ║
╟────────────────────────────────────╢
║  Last Activity: 2s ago             ║
║  Events Processed: 1,245           ║
╟────────────────────────────────────╢
║  [Pause] [Restart] [View Logs]    ║
╚════════════════════════════════════╝
```

---

## 4. User Interactions

### 4.1 View Operations (Read-Only)

**Selection:**
- Click Land → View capacity and processes
- Click Nim → View status, subjects, RAM usage
- Click Tree → View subjects, throughput
- Click Treehouse → View script path, processing stats

**Camera Controls:**
- Pan: Arrow keys or mouse drag
- Zoom: Mouse wheel
- Focus: Double-click entity to center camera

**Overlays:**
- Toggle RAM usage heatmap
- Toggle CPU usage heatmap
- Toggle network traffic visualization
- Toggle event flow animation

### 4.2 Control Operations (Emit Wind Events)

**Process Management:**
- Pause Nim → Emit `nim.pause.{nim_id}`
- Resume Nim → Emit `nim.resume.{nim_id}`
- Stop Process → Emit `process.stop.{process_id}`
- View Logs → Emit `logs.query.{process_id}`

**Deployment:**
- Deploy Nim → Open deployment dialog → Emit `nim.deploy` with config
- Deploy Tree → Emit `tree.deploy` with config
- Deploy Treehouse → Emit `treehouse.deploy` with script

**Resource Allocation:**
- Increase RAM → Emit `process.scale.{id}` with new allocation
- Migrate Process → Emit `process.migrate` with source/target Land

---

## 5. Implementation Architecture

### 5.1 Component Breakdown

```
nimsforest-0ad-viewer/
├── bridge/               # NATS ↔ 0 A.D. bridge (Go)
│   ├── main.go           # Entry point
│   ├── subscriber.go     # Subscribe to viewmodel.state & events
│   ├── mapper.go         # Map viewmodel → game commands
│   ├── ipc.go            # IPC with 0 A.D. (Unix socket)
│   └── publisher.go      # Publish user commands to Wind
│
├── mod/                  # 0 A.D. mod
│   ├── mod.json          # Mod definition
│   ├── simulation/
│   │   └── components/
│   │       ├── NimsforestBridge.js       # IPC handler
│   │       └── NimsforestWorldSync.js    # State synchronizer
│   ├── art/
│   │   ├── textures/
│   │   │   └── ui/                       # Custom UI panels
│   │   └── meshes/
│   │       └── structures/               # Custom buildings
│   └── gui/
│       ├── session/
│       │   └── nimsforest_hud.js         # Custom HUD
│       └── nimsforest/
│           ├── land_panel.xml            # Land info panel
│           └── nim_panel.xml             # Nim info panel
│
└── README.md
```

### 5.2 Go Bridge Implementation

```go
// bridge/main.go

package main

import (
    "context"
    "encoding/json"
    "log"

    "github.com/nats-io/nats.go"
)

type ViewerBridge struct {
    nc          *nats.Conn
    ipc         *GameIPC
    mapper      *StateMapper
    currentWorld *World
}

func main() {
    // Connect to NATS
    nc, err := nats.Connect("nats://localhost:4222")
    if err != nil {
        log.Fatal(err)
    }
    defer nc.Close()

    // Connect to 0 A.D. via IPC
    ipc, err := NewGameIPC("/tmp/nimsforest-0ad.sock")
    if err != nil {
        log.Fatal(err)
    }
    defer ipc.Close()

    bridge := &ViewerBridge{
        nc:     nc,
        ipc:    ipc,
        mapper: NewStateMapper(),
    }

    // Subscribe to viewmodel state
    nc.Subscribe("forest.viewmodel.state", bridge.handleStateUpdate)

    // Subscribe to viewmodel events
    nc.Subscribe("forest.viewmodel.events", bridge.handleEvent)

    // Listen for user commands from 0 A.D.
    go bridge.handleUserCommands()

    log.Println("🎮 0 A.D. Viewer Bridge running...")
    select {}
}

func (b *ViewerBridge) handleStateUpdate(msg *nats.Msg) {
    var world World
    if err := json.Unmarshal(msg.Data, &world); err != nil {
        log.Printf("Error parsing state: %v", err)
        return
    }

    // Map to game commands
    commands := b.mapper.WorldToGameCommands(b.currentWorld, &world)

    // Send to 0 A.D.
    for _, cmd := range commands {
        if err := b.ipc.SendCommand(cmd); err != nil {
            log.Printf("Error sending command: %v", err)
        }
    }

    b.currentWorld = &world
}

func (b *ViewerBridge) handleEvent(msg *nats.Msg) {
    var event Event
    if err := json.Unmarshal(msg.Data, &event); err != nil {
        log.Printf("Error parsing event: %v", err)
        return
    }

    // Convert event to visual effect
    effect := b.mapper.EventToEffect(event)

    // Send to 0 A.D.
    b.ipc.SendEffect(effect)
}

func (b *ViewerBridge) handleUserCommands() {
    for {
        // Read command from IPC
        cmd, err := b.ipc.ReadCommand()
        if err != nil {
            continue
        }

        // Parse user action
        switch cmd.Type {
        case "pause_nim":
            b.nc.Publish("nim.pause." + cmd.TargetID, []byte{})
        case "deploy_nim":
            b.nc.Publish("nim.deploy", cmd.Data)
        // ... more actions
        }
    }
}
```

### 5.3 State Mapper

```go
// bridge/mapper.go

package main

type StateMapper struct {}

func NewStateMapper() *StateMapper {
    return &StateMapper{}
}

// WorldToGameCommands generates game commands for state changes
func (m *StateMapper) WorldToGameCommands(oldWorld, newWorld *World) []GameCommand {
    var commands []GameCommand

    // Detect new Lands
    for _, land := range newWorld.Lands {
        if !m.landExists(oldWorld, land.ID) {
            commands = append(commands, GameCommand{
                Type: "create_base",
                Data: map[string]interface{}{
                    "id":       land.ID,
                    "name":     land.Hostname,
                    "ram":      land.RAMTotal,
                    "cpu":      land.CPUCores,
                    "gpu_vram": land.GPUVram,
                    "position": m.calculateLandPosition(land),
                },
            })
        }
    }

    // Detect new processes
    for _, land := range newWorld.Lands {
        for _, nim := range land.Nims {
            if !m.processExists(oldWorld, nim.ID) {
                commands = append(commands, GameCommand{
                    Type: "spawn_unit",
                    Data: map[string]interface{}{
                        "id":       nim.ID,
                        "name":     nim.Name,
                        "type":     "nim",
                        "land_id":  land.ID,
                        "ram":      nim.RAMAllocated,
                        "position": m.calculateProcessPosition(land, nim),
                    },
                })
            }
        }

        // Same for Trees and Treehouses...
    }

    // Detect removed entities
    commands = append(commands, m.detectRemovals(oldWorld, newWorld)...)

    // Detect resource changes
    commands = append(commands, m.detectResourceChanges(oldWorld, newWorld)...)

    return commands
}

func (m *StateMapper) EventToEffect(event Event) GameEffect {
    switch event.Type {
    case "process_added":
        return GameEffect{
            Type:     "particle",
            Particle: "spawn",
            Position: m.getEntityPosition(event.EntityID),
            Color:    "green",
        }
    case "process_removed":
        return GameEffect{
            Type:     "particle",
            Particle: "death",
            Position: m.getEntityPosition(event.EntityID),
            Color:    "red",
        }
    case "high_cpu":
        return GameEffect{
            Type:     "overlay",
            EntityID: event.EntityID,
            Overlay:  "warning_red",
        }
    }
    return GameEffect{}
}

func (m *StateMapper) calculateLandPosition(land LandViewModel) Position {
    // Distribute Lands across map
    // Could use actual geographic data if available
    return Position{X: 100, Z: 100}
}
```

### 5.4 0 A.D. Mod Bridge Component

```javascript
// mod/simulation/components/NimsforestWorldSync.js

function NimsforestWorldSync() {}

NimsforestWorldSync.prototype.Schema = "<empty/>";

NimsforestWorldSync.prototype.Init = function() {
    // Connect to bridge via Unix socket
    this.socket = Engine.OpenUnixSocket("/tmp/nimsforest-0ad.sock");

    // Entity tracking
    this.landEntities = new Map();      // land_id → entity
    this.processEntities = new Map();   // process_id → entity

    // Start listening for commands
    this.StartCommandListener();
};

NimsforestWorldSync.prototype.StartCommandListener = function() {
    var self = this;

    this.commandTimer = Engine.SetInterval(function() {
        var command = self.ReadCommand();
        if (command) {
            self.ExecuteCommand(command);
        }
    }, 50);  // Poll every 50ms for responsiveness
};

NimsforestWorldSync.prototype.ReadCommand = function() {
    var data = this.socket.Read();
    if (!data) return null;

    try {
        return JSON.parse(data);
    } catch (e) {
        error("Invalid command JSON: " + e);
        return null;
    }
};

NimsforestWorldSync.prototype.ExecuteCommand = function(cmd) {
    switch (cmd.type) {
        case "create_base":
            this.CreateLand(cmd.data);
            break;
        case "spawn_unit":
            this.SpawnProcess(cmd.data);
            break;
        case "destroy_entity":
            this.DestroyEntity(cmd.data.id);
            break;
        case "update_resources":
            this.UpdateResources(cmd.data);
            break;
        case "play_effect":
            this.PlayEffect(cmd.data);
            break;
        default:
            warn("Unknown command: " + cmd.type);
    }
};

NimsforestWorldSync.prototype.CreateLand = function(data) {
    // Spawn base/town center entity
    var template = data.gpu_vram > 0
        ? "structures/nimsforest_manaland"
        : "structures/nimsforest_land";

    var entity = Engine.AddEntity(template);

    // Set position
    var cmpPosition = Engine.QueryInterface(entity, IID_Position);
    cmpPosition.JumpTo(data.position.x, data.position.z);

    // Set name
    var cmpIdentity = Engine.QueryInterface(entity, IID_Identity);
    cmpIdentity.SetCustomName(data.name);

    // Store mapping
    this.landEntities.set(data.id, entity);

    log("Created Land: " + data.name + " at " + data.position.x + "," + data.position.z);
};

NimsforestWorldSync.prototype.SpawnProcess = function(data) {
    // Determine template based on type
    var template;
    switch (data.type) {
        case "nim":
            template = "units/nimsforest_nim";
            break;
        case "tree":
            template = "structures/nimsforest_tree";
            break;
        case "treehouse":
            template = "structures/nimsforest_treehouse";
            break;
    }

    var entity = Engine.AddEntity(template);

    // Position relative to Land
    var landEntity = this.landEntities.get(data.land_id);
    var landPos = Engine.QueryInterface(landEntity, IID_Position).GetPosition();

    var cmpPosition = Engine.QueryInterface(entity, IID_Position);
    cmpPosition.JumpTo(
        landPos.x + data.position.x,
        landPos.z + data.position.z
    );

    // Set custom properties
    var cmpIdentity = Engine.QueryInterface(entity, IID_Identity);
    cmpIdentity.SetCustomName(data.name);

    // Store mapping
    this.processEntities.set(data.id, entity);

    log("Spawned " + data.type + ": " + data.name);
};

NimsforestWorldSync.prototype.DestroyEntity = function(id) {
    var entity = this.processEntities.get(id) || this.landEntities.get(id);
    if (!entity) {
        warn("Entity not found: " + id);
        return;
    }

    // Play destruction effect
    Engine.PlayEffectAtPosition("particle/destruction",
        Engine.QueryInterface(entity, IID_Position).GetPosition());

    // Remove entity
    Engine.DestroyEntity(entity);

    // Remove from maps
    this.processEntities.delete(id);
    this.landEntities.delete(id);
};

NimsforestWorldSync.prototype.UpdateResources = function(data) {
    // Update resource panel display
    var cmpPlayer = Engine.QueryInterface(SYSTEM_ENTITY, IID_Player);

    cmpPlayer.SetResourceCounts({
        "ram": data.ram_available,
        "cpu": data.cpu_available,
        "vram": data.vram_available
    });
};

NimsforestWorldSync.prototype.PlayEffect = function(data) {
    var position;

    if (data.entity_id) {
        var entity = this.processEntities.get(data.entity_id);
        var cmpPosition = Engine.QueryInterface(entity, IID_Position);
        position = cmpPosition.GetPosition();
    } else {
        position = data.position;
    }

    Engine.PlayEffectAtPosition(data.effect, position);
};

// Handle user input from UI
NimsforestWorldSync.prototype.OnUserCommand = function(cmd) {
    // Send command to bridge via socket
    this.socket.Write(JSON.stringify(cmd) + "\n");
};

Engine.RegisterSystemComponentType(IID_NimsforestWorldSync, "NimsforestWorldSync", NimsforestWorldSync);
```

---

## 6. User Workflow Examples

### 6.1 Monitoring Workflow

```
1. User launches 0 A.D. viewer
2. Viewer connects to NATS, subscribes to viewmodel.state
3. Initial state loads → Map populated with current cluster
4. User sees:
   - 3 Lands (bases) with different colors
   - Various Nims (units) moving around
   - Trees (buildings) processing data
   - Resource counters showing RAM/CPU/GPU usage
5. User clicks on a Nim
   → Info panel appears
   → Shows status, RAM usage, subjects
   → "Last processed: 0.5s ago"
6. User enables "RAM Usage Heatmap" overlay
   → Lands colored red (high) to green (low)
   → Immediately sees which Lands are overloaded
```

### 6.2 Deployment Workflow

```
1. User right-clicks on a Land
   → Context menu: [Deploy Process]
2. Deployment dialog appears:
   - Type: Nim / Tree / Treehouse
   - Name: "customer-analyzer"
   - RAM: 2GB (slider)
   - AI Enabled: ✓
3. User configures and clicks [Deploy]
4. Viewer emits: nim.deploy {config}
5. Forest processes deployment
6. Event published: process_added
7. Viewer receives event
   → Spawns new Nim unit on the Land
   → Green particle effect
   → Info toast: "customer-analyzer deployed"
```

### 6.3 Troubleshooting Workflow

```
1. User notices a Land with red warning indicator
2. Clicks on Land
   → Info panel shows: "RAM: 31GB / 32GB (97%)"
3. User clicks [View Processes]
   → List shows all processes with RAM usage
   → Finds: "image-processor" using 20GB
4. User clicks on "image-processor" Nim
5. Info panel shows high RAM usage
6. User clicks [Migrate to Another Land]
   → Dialog shows available Lands with capacity
   → Selects "Land 2" (lots of free RAM)
7. Viewer emits: process.migrate {src, dst, process_id}
8. Forest processes migration
9. Viewer shows:
   → Nim disappears from Land 1 (fade out)
   → Nim appears on Land 2 (fade in)
   → Land 1 indicator turns green
```

---

## 7. Implementation Phases

### Phase 1: Bridge + Basic Visualization (Week 1-2)
- [ ] Create Go bridge that subscribes to viewmodel.state
- [ ] Implement IPC layer (Unix socket)
- [ ] Create basic 0 A.D. mod structure
- [ ] Map Lands → bases (spawn at fixed positions)
- [ ] Display Land info panel on click

**Validation**: Can see cluster Lands as bases in 0 A.D.

### Phase 2: Process Visualization (Week 2-3)
- [ ] Map Nims → units
- [ ] Map Trees → buildings
- [ ] Map Treehouses → buildings
- [ ] Position processes relative to their Land
- [ ] Display process info panel on click

**Validation**: Can see all processes as entities in 0 A.D.

### Phase 3: Real-Time Updates (Week 3-4)
- [ ] Subscribe to viewmodel.events
- [ ] Handle process_added events
- [ ] Handle process_removed events
- [ ] Handle resource update events
- [ ] Visual effects for events

**Validation**: Cluster changes reflected in real-time

### Phase 4: Resource Visualization (Week 4-5)
- [ ] Map RAM/CPU/GPU to 0 A.D. resources
- [ ] Display resource panel
- [ ] Resource usage indicators on Lands
- [ ] RAM/CPU usage heatmap overlay
- [ ] Manaland special appearance (GPU)

**Validation**: Resource usage clearly visible

### Phase 5: User Controls (Week 5-6)
- [ ] Handle user selection events
- [ ] Emit pause/resume commands
- [ ] Emit deployment commands
- [ ] Context menus for entities
- [ ] Deployment dialog UI

**Validation**: Can control forest via viewer

### Phase 6: Polish & Production (Week 6-7)
- [ ] Custom 3D models for entities
- [ ] Improved textures and UI
- [ ] Smooth animations
- [ ] Error handling and recovery
- [ ] Documentation and examples

**Validation**: Production-ready viewer

---

## 8. Technical Considerations

### 8.1 Performance

**Scalability:**
- Support up to 10 Lands (typical cluster size)
- Support up to 100 processes total
- 60 FPS target framerate
- < 100ms latency for state updates

**Optimization:**
- Only update changed entities (delta updates)
- Batch visual effects
- LOD (level of detail) for distant entities
- Cache entity lookups

### 8.2 IPC Communication

**Protocol:**
```
Bridge → 0 A.D.:  JSON commands (line-delimited)
0 A.D. → Bridge:  JSON events (line-delimited)

Command format:
{"type": "create_base", "data": {...}}

Event format:
{"type": "user_click", "entity_id": "...", "action": "..."}
```

**Socket:**
- Unix domain socket: `/tmp/nimsforest-0ad.sock`
- Bi-directional communication
- Non-blocking I/O

### 8.3 State Synchronization

**Initial Sync:**
1. Bridge requests current state: `nats req forest.viewmodel.state ""`
2. Receives full World
3. Builds initial game entities
4. Subscribes to events for incremental updates

**Incremental Updates:**
- Subscribe to `forest.viewmodel.events`
- Apply events as they arrive
- Periodically resync (every 60s) to prevent drift

---

## 9. Alternative: Simpler Implementation

If 0 A.D. integration proves too complex, consider these alternatives:

### 9.1 Web-Based RTS Viewer (Phaser)

Same concept, but using Phaser game engine in the browser:
- HTML5 canvas rendering
- JavaScript client subscribes via NATS WebSocket
- Simpler deployment (just open browser)
- Same visualization metaphor

### 9.2 Terminal UI (Bubble Tea)

Text-based interactive viewer:
- Uses `bubbletea` Go TUI library
- ASCII art representation
- Keyboard navigation
- Much simpler implementation

```
┌─ NimsForest Cluster ──────────────────────────────┐
│ Land: node-1 [■■■■■■■■░░] 75% RAM                │
│   ↳ payment-tree [4GB]                            │
│   ↳ qualify-nim  [2GB] ◉ active                   │
│                                                    │
│ Land: node-2 [■■■░░░░░░░] 30% RAM                │
│   ↳ scoring-house [2GB]                           │
│                                                    │
│ Manaland: gpu-1 [■■■■■■░░░░] 60% RAM, 24GB GPU   │
│   (no processes)                                   │
└────────────────────────────────────────────────────┘

[d] Deploy  [p] Pause/Resume  [r] Refresh  [q] Quit
```

---

## 10. Conclusion

This plan creates a **visual and interactive interface** for managing nimsforest using 0 A.D.'s RTS engine. The key insight is:

**NimsForest is the "game" - we're visualizing the orchestration, not playing 0 A.D.**

The viewer provides:
1. **Intuitive Visualization**: See cluster topology at a glance
2. **Resource Monitoring**: Spot overloaded Lands immediately
3. **Interactive Control**: Deploy and manage processes via familiar RTS interface
4. **Real-Time Updates**: See cluster changes as they happen
5. **Scalable Architecture**: Decoupled external viewer via NATS

This is much simpler than the original plan because:
- No game AI or gameplay logic needed
- Viewer is read-mostly (control commands are optional)
- Leverages existing viewmodel infrastructure
- Can be implemented incrementally

**Next Steps**: Start with Phase 1 - create the bridge and basic Land visualization.

---

**Branch**: `claude/plan-ad0-engine-interface-2p5BU`
**Last Updated**: 2026-01-12
