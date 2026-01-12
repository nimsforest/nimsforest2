# Plan: 0 A.D. Game Engine Interface for NimsForest

**Status**: Planning
**Goal**: Integrate 0 A.D. RTS game engine as a controllable agent in nimsforest2
**Reference**: Similar to AIAgent, BrowserAgent, RobotAgent pattern

---

## Executive Summary

Integrate 0 A.D. (https://github.com/0ad/0ad) as a **GameEngineAgent** in nimsforest2, enabling:
- Automated gameplay and testing
- AI-driven strategy execution
- Game state monitoring and event emission
- Multi-instance game orchestration
- Integration with other nimsforest agents (AI, Human, Robot)

---

## 1. 0 A.D. Engine Overview

### Architecture
- **Core Engine**: C++ (63.7%) and C (24.0%)
- **Scripting**: JavaScript (6.1%), Lua (2.5%)
- **Type**: Real-time strategy (RTS) game engine
- **Features**: Historical warfare simulation, multiplayer, AI opponents, modding support

### Key Components
```
0 A.D. Engine
├── Pyrogenesis (Game executable)
│   ├── Graphics renderer
│   ├── Physics engine
│   ├── Audio system
│   └── Network layer (multiplayer)
├── SpiderMonkey (JavaScript engine)
│   ├── Game logic scripts
│   ├── AI behavior
│   └── Mod scripts
├── Atlas (Map editor)
└── Simulation core
    ├── Entity system
    ├── Path finding
    └── Combat resolution
```

### Integration Points
1. **JavaScript API**: Game exposes extensive JS API for gameplay control
2. **AI Scripts**: Replaceable AI behaviors written in JavaScript
3. **Replay System**: Games can be saved/loaded for analysis
4. **Headless Mode**: Can run without graphics for simulation
5. **Mod System**: Extensible via mods (XML configs + JS scripts)

---

## 2. Architectural Design

### 2.1 GameEngineAgent Interface

Following the established agent pattern:

```go
// pkg/nim/agent.go (extend)

const (
    AgentTypeAI      AgentType = "ai"
    AgentTypeHuman   AgentType = "human"
    AgentTypeRobot   AgentType = "robot"
    AgentTypeBrowser AgentType = "browser"
    AgentTypeGame    AgentType = "game"      // NEW
)

// pkg/nim/game_agent.go (new)

package nim

import "context"

// GameEngineAgent controls game engines (0 A.D., Unity, Godot, etc.)
type GameEngineAgent interface {
    Agent

    // Engine identification
    Engine() string              // "0ad", "unity", "godot"
    Version() string             // Engine version
    SupportsHeadless() bool      // Can run without graphics

    // Game lifecycle
    LaunchGame(ctx context.Context, config GameConfig) error
    StopGame(ctx context.Context) error
    GetGameState(ctx context.Context) (*GameState, error)

    // Game control
    ExecuteCommand(ctx context.Context, cmd GameCommand) (*GameResult, error)
    LoadReplay(ctx context.Context, replayPath string) error
    SaveReplay(ctx context.Context) (string, error)

    // Event streaming
    SubscribeEvents(ctx context.Context, eventTypes []string) (<-chan GameEvent, error)
}

// GameConfig defines how to launch a game instance
type GameConfig struct {
    Scenario    string                 `json:"scenario"`     // Map/scenario to load
    Players     []PlayerConfig         `json:"players"`      // Player configurations
    GameMode    string                 `json:"game_mode"`    // "singleplayer", "multiplayer"
    Headless    bool                   `json:"headless"`     // Run without graphics
    SpeedFactor float64                `json:"speed_factor"` // Game speed multiplier
    Options     map[string]interface{} `json:"options"`      // Engine-specific options
}

type PlayerConfig struct {
    ID           int    `json:"id"`
    Name         string `json:"name"`
    Team         int    `json:"team"`
    Civilization string `json:"civilization"` // e.g., "romans", "carthaginians"
    ControlType  string `json:"control_type"` // "ai", "human", "script"
    AIScript     string `json:"ai_script"`    // AI behavior script
}

// GameState represents current game state
type GameState struct {
    GameTime     float64                `json:"game_time"`      // In-game time (seconds)
    Paused       bool                   `json:"paused"`
    Players      []PlayerState          `json:"players"`
    Resources    map[string]interface{} `json:"resources"`      // Engine-specific
    Entities     []EntityState          `json:"entities"`       // Units, buildings
    TechTree     map[string][]string    `json:"tech_tree"`      // Technologies researched
    Victory      *VictoryCondition      `json:"victory,omitempty"`
}

type PlayerState struct {
    ID           int                    `json:"id"`
    Name         string                 `json:"name"`
    Alive        bool                   `json:"alive"`
    Resources    map[string]float64     `json:"resources"`      // food, wood, stone, metal
    Population   int                    `json:"population"`
    MaxPop       int                    `json:"max_population"`
    Technologies []string               `json:"technologies"`
    Score        int                    `json:"score"`
}

type EntityState struct {
    ID         int     `json:"id"`
    Type       string  `json:"type"`        // "unit", "building", "resource"
    Template   string  `json:"template"`    // Entity template name
    Owner      int     `json:"owner"`       // Player ID
    Position   Vector3 `json:"position"`
    Health     float64 `json:"health"`
    MaxHealth  float64 `json:"max_health"`
    State      string  `json:"state"`       // "idle", "moving", "attacking"
}

type Vector3 struct {
    X float64 `json:"x"`
    Y float64 `json:"y"`
    Z float64 `json:"z"`
}

// GameCommand represents an action to execute
type GameCommand struct {
    Type       string                 `json:"type"`        // "train", "build", "attack", "move"
    PlayerID   int                    `json:"player_id"`
    EntityIDs  []int                  `json:"entity_ids"`  // Units/buildings to command
    Target     *CommandTarget         `json:"target,omitempty"`
    Parameters map[string]interface{} `json:"parameters"`
}

type CommandTarget struct {
    EntityID int      `json:"entity_id,omitempty"` // Target entity
    Position *Vector3 `json:"position,omitempty"`  // Target position
}

// GameResult represents command execution result
type GameResult struct {
    Success    bool     `json:"success"`
    Message    string   `json:"message"`
    NewState   *GameState `json:"new_state,omitempty"`
    AffectedIDs []int    `json:"affected_ids"` // Entities affected
}

// GameEvent represents game occurrences
type GameEvent struct {
    Type      string                 `json:"type"`       // "unit_created", "battle", "victory"
    Timestamp float64                `json:"timestamp"`  // Game time
    PlayerID  int                    `json:"player_id"`
    Data      map[string]interface{} `json:"data"`
}

type VictoryCondition struct {
    Type   string `json:"type"`   // "conquest", "wonder", "relic"
    Winner int    `json:"winner"` // Player ID
}
```

---

### 2.2 0 A.D. Agent Implementation

```go
// internal/ai/agents/game/0ad_agent.go

package game

import (
    "context"
    "encoding/json"
    "fmt"
    "os/exec"
    "sync"

    "github.com/nimsforest/nimsforest2/pkg/nim"
)

// ZeroADAgent implements GameEngineAgent for 0 A.D.
type ZeroADAgent struct {
    config      nim.GameAgentConfig
    landID      string

    // Runtime state
    mu          sync.RWMutex
    gameProcess *exec.Cmd
    gameState   *nim.GameState
    running     bool

    // Communication channels
    commandPipe   *GamePipe  // Send commands to game
    eventPipe     *GamePipe  // Receive events from game
    eventSub      chan nim.GameEvent
}

type GameAgentConfig struct {
    Name         string `json:"name"`
    Engine       string `json:"engine"`        // "0ad"
    Version      string `json:"version"`
    BinaryPath   string `json:"binary_path"`   // Path to pyrogenesis
    DataDir      string `json:"data_dir"`      // Game data directory
    ModsDir      string `json:"mods_dir"`      // Custom mods
    Headless     bool   `json:"headless"`
    DockerImage  string `json:"docker_image"`  // Optional: run in container
}

func NewZeroADAgent(config GameAgentConfig, landID string) *ZeroADAgent {
    return &ZeroADAgent{
        config:    config,
        landID:    landID,
        eventSub:  make(chan nim.GameEvent, 100),
    }
}

// Agent interface implementation

func (a *ZeroADAgent) Run(ctx context.Context, task nim.Task) (*nim.Result, error) {
    // Parse task as game command
    var gameCmd nim.GameCommand
    if err := json.Unmarshal(task.Params["command"].([]byte), &gameCmd); err != nil {
        return nil, fmt.Errorf("invalid game command: %w", err)
    }

    // Ensure game is running
    if !a.running {
        return &nim.Result{
            Success: false,
            Error:   "game not running - call LaunchGame first",
        }, nil
    }

    // Execute command
    result, err := a.ExecuteCommand(ctx, gameCmd)
    if err != nil {
        return &nim.Result{
            Success: false,
            Error:   err.Error(),
        }, nil
    }

    return &nim.Result{
        Success: result.Success,
        Output:  result.Message,
    }, nil
}

func (a *ZeroADAgent) Type() nim.AgentType {
    return nim.AgentTypeGame
}

func (a *ZeroADAgent) Available(ctx context.Context) bool {
    // Check if 0 A.D. binary exists
    if _, err := exec.LookPath(a.config.BinaryPath); err != nil {
        return false
    }
    return true
}

// GameEngineAgent interface implementation

func (a *ZeroADAgent) Engine() string {
    return "0ad"
}

func (a *ZeroADAgent) Version() string {
    return a.config.Version
}

func (a *ZeroADAgent) SupportsHeadless() bool {
    return true // 0 A.D. supports headless mode
}

func (a *ZeroADAgent) LaunchGame(ctx context.Context, config nim.GameConfig) error {
    a.mu.Lock()
    defer a.mu.Unlock()

    if a.running {
        return fmt.Errorf("game already running")
    }

    // Build command line arguments
    args := a.buildLaunchArgs(config)

    // Start game process
    if a.config.DockerImage != "" {
        // Run in Docker
        a.gameProcess = a.launchInDocker(ctx, args)
    } else {
        // Run natively
        a.gameProcess = exec.CommandContext(ctx, a.config.BinaryPath, args...)
    }

    // Set up stdio pipes for communication
    if err := a.setupGamePipes(); err != nil {
        return fmt.Errorf("failed to setup game pipes: %w", err)
    }

    // Start the game
    if err := a.gameProcess.Start(); err != nil {
        return fmt.Errorf("failed to start game: %w", err)
    }

    a.running = true

    // Start event reader goroutine
    go a.readGameEvents(ctx)

    return nil
}

func (a *ZeroADAgent) buildLaunchArgs(config nim.GameConfig) []string {
    args := []string{}

    // Headless mode
    if a.config.Headless || config.Headless {
        args = append(args, "-autostart-nonvisual")
    }

    // Load scenario
    if config.Scenario != "" {
        args = append(args, "-autostart="+config.Scenario)
    }

    // Game speed
    if config.SpeedFactor > 0 {
        args = append(args, fmt.Sprintf("-autostart-speed=%.2f", config.SpeedFactor))
    }

    // Enable mod for nimsforest control
    args = append(args, "-mod=nimsforest_control")

    return args
}

func (a *ZeroADAgent) StopGame(ctx context.Context) error {
    a.mu.Lock()
    defer a.mu.Unlock()

    if !a.running {
        return nil
    }

    // Send quit command
    if err := a.sendCommand("quit"); err != nil {
        // Force kill if graceful quit fails
        if a.gameProcess != nil {
            a.gameProcess.Process.Kill()
        }
    }

    a.running = false
    close(a.eventSub)

    return nil
}

func (a *ZeroADAgent) GetGameState(ctx context.Context) (*nim.GameState, error) {
    a.mu.RLock()
    defer a.mu.RUnlock()

    if !a.running {
        return nil, fmt.Errorf("game not running")
    }

    // Query game state via JavaScript API
    stateJSON, err := a.queryGameState()
    if err != nil {
        return nil, err
    }

    var state nim.GameState
    if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
        return nil, fmt.Errorf("failed to parse game state: %w", err)
    }

    a.gameState = &state
    return &state, nil
}

func (a *ZeroADAgent) ExecuteCommand(ctx context.Context, cmd nim.GameCommand) (*nim.GameResult, error) {
    // Convert nim.GameCommand to 0 A.D. JS command
    jsCommand := a.buildJSCommand(cmd)

    // Send to game via pipe
    response, err := a.sendCommandAndWait(jsCommand)
    if err != nil {
        return nil, err
    }

    // Parse response
    var result nim.GameResult
    if err := json.Unmarshal([]byte(response), &result); err != nil {
        return nil, fmt.Errorf("failed to parse result: %w", err)
    }

    return &result, nil
}

func (a *ZeroADAgent) buildJSCommand(cmd nim.GameCommand) string {
    // Convert to 0 A.D. JavaScript API call
    switch cmd.Type {
    case "train":
        // Example: Train unit
        return fmt.Sprintf(`
            var entities = %v;
            var template = "%s";
            for (var ent of entities) {
                Engine.PostCommand(ent, {
                    "type": "train",
                    "template": template,
                    "count": 1
                });
            }
        `, cmd.EntityIDs, cmd.Parameters["template"])

    case "move":
        // Example: Move units
        return fmt.Sprintf(`
            var entities = %v;
            var target = {x: %.2f, z: %.2f};
            Engine.PostCommand(entities, {
                "type": "walk",
                "x": target.x,
                "z": target.z,
                "queued": false
            });
        `, cmd.EntityIDs, cmd.Target.Position.X, cmd.Target.Position.Z)

    case "attack":
        // Example: Attack target
        return fmt.Sprintf(`
            var entities = %v;
            var target = %d;
            Engine.PostCommand(entities, {
                "type": "attack",
                "target": target,
                "queued": false
            });
        `, cmd.EntityIDs, cmd.Target.EntityID)

    // Add more command types...
    default:
        return fmt.Sprintf(`console.error("Unknown command type: %s");`, cmd.Type)
    }
}

func (a *ZeroADAgent) LoadReplay(ctx context.Context, replayPath string) error {
    args := []string{
        "-replay=" + replayPath,
        "-autostart-nonvisual",
    }

    // Similar to LaunchGame but with replay
    // Implementation details...
    return nil
}

func (a *ZeroADAgent) SaveReplay(ctx context.Context) (string, error) {
    // 0 A.D. automatically saves replays
    // Query the replay path from game
    replayPath, err := a.queryReplayPath()
    if err != nil {
        return "", err
    }

    return replayPath, nil
}

func (a *ZeroADAgent) SubscribeEvents(ctx context.Context, eventTypes []string) (<-chan nim.GameEvent, error) {
    // Register event types with game
    for _, eventType := range eventTypes {
        if err := a.registerEventType(eventType); err != nil {
            return nil, err
        }
    }

    return a.eventSub, nil
}

// Internal communication methods

func (a *ZeroADAgent) setupGamePipes() error {
    // Set up named pipes or sockets for bi-directional communication
    // 0 A.D. mod will read commands and write events
    // Implementation uses Unix domain sockets or named pipes
    return nil
}

func (a *ZeroADAgent) sendCommand(cmd string) error {
    if a.commandPipe == nil {
        return fmt.Errorf("command pipe not initialized")
    }

    _, err := a.commandPipe.Write([]byte(cmd + "\n"))
    return err
}

func (a *ZeroADAgent) sendCommandAndWait(cmd string) (string, error) {
    // Send command and wait for response
    // Uses request/response correlation IDs
    return "", nil
}

func (a *ZeroADAgent) queryGameState() (string, error) {
    // Query full game state via JS API
    jsQuery := `
        JSON.stringify({
            game_time: Engine.QueryInterface(SYSTEM_ENTITY, IID_Timer).GetTime(),
            players: getPlayerStates(),
            entities: getEntityStates(),
            // ... more state queries
        });
    `

    return a.sendCommandAndWait(jsQuery)
}

func (a *ZeroADAgent) readGameEvents(ctx context.Context) {
    // Continuously read events from game
    for {
        select {
        case <-ctx.Done():
            return
        default:
            // Read from event pipe
            eventJSON, err := a.eventPipe.ReadLine()
            if err != nil {
                continue
            }

            var event nim.GameEvent
            if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
                continue
            }

            // Send to subscribers
            select {
            case a.eventSub <- event:
            default:
                // Channel full, drop event
            }
        }
    }
}

func (a *ZeroADAgent) launchInDocker(ctx context.Context, gameArgs []string) *exec.Cmd {
    dockerArgs := []string{
        "run", "--rm",
        "-v", fmt.Sprintf("%s:/data", a.config.DataDir),
        a.config.DockerImage,
    }
    dockerArgs = append(dockerArgs, gameArgs...)

    return exec.CommandContext(ctx, "docker", dockerArgs...)
}

// GamePipe provides bi-directional communication with the game
type GamePipe struct {
    // Implementation using Unix domain sockets or named pipes
}

func (gp *GamePipe) Write(data []byte) (int, error) {
    return 0, nil
}

func (gp *GamePipe) ReadLine() (string, error) {
    return "", nil
}
```

---

### 2.3 NimsForest Control Mod for 0 A.D.

To enable control from nimsforest, we need a custom 0 A.D. mod:

```
mods/
└── nimsforest_control/
    ├── mod.json
    ├── simulation/
    │   └── components/
    │       └── NimsforestBridge.js
    └── gui/
        └── nimsforest_hud.js
```

**mod.json:**
```json
{
    "name": "nimsforest_control",
    "version": "1.0.0",
    "label": "NimsForest Control Bridge",
    "description": "Enables external control of 0 A.D. via NimsForest",
    "dependencies": [],
    "url": "https://github.com/nimsforest/0ad-control-mod"
}
```

**simulation/components/NimsforestBridge.js:**
```javascript
/**
 * Component that bridges 0 A.D. with NimsForest orchestration system
 */
function NimsforestBridge() {}

NimsforestBridge.prototype.Schema = "<empty/>";

NimsforestBridge.prototype.Init = function() {
    this.commandSocket = this.OpenSocket("/tmp/nimsforest-0ad-commands.sock");
    this.eventSocket = this.OpenSocket("/tmp/nimsforest-0ad-events.sock");

    // Start listening for commands
    this.SetupCommandListener();

    // Register event emitters
    this.RegisterEventHandlers();
};

NimsforestBridge.prototype.OpenSocket = function(path) {
    // Open Unix domain socket for IPC
    // Note: Requires custom C++ binding in 0 A.D. engine
    return Engine.OpenUnixSocket(path);
};

NimsforestBridge.prototype.SetupCommandListener = function() {
    var self = this;

    // Poll for commands every 100ms
    this.commandTimer = Engine.SetInterval(function() {
        var command = self.ReadCommand();
        if (command) {
            self.ExecuteCommand(command);
        }
    }, 100);
};

NimsforestBridge.prototype.ReadCommand = function() {
    var data = this.commandSocket.Read();
    if (!data) return null;

    try {
        return JSON.parse(data);
    } catch (e) {
        error("Invalid command JSON: " + e);
        return null;
    }
};

NimsforestBridge.prototype.ExecuteCommand = function(cmd) {
    switch (cmd.type) {
        case "train":
            this.CommandTrain(cmd);
            break;
        case "build":
            this.CommandBuild(cmd);
            break;
        case "move":
            this.CommandMove(cmd);
            break;
        case "attack":
            this.CommandAttack(cmd);
            break;
        case "query_state":
            this.SendGameState();
            break;
        default:
            error("Unknown command type: " + cmd.type);
    }
};

NimsforestBridge.prototype.CommandTrain = function(cmd) {
    for (var entId of cmd.entity_ids) {
        Engine.PostCommand(cmd.player_id, {
            "type": "train",
            "entities": [entId],
            "template": cmd.parameters.template,
            "count": cmd.parameters.count || 1
        });
    }

    this.SendResponse({
        success: true,
        message: "Train command issued",
        affected_ids: cmd.entity_ids
    });
};

NimsforestBridge.prototype.CommandMove = function(cmd) {
    Engine.PostCommand(cmd.player_id, {
        "type": "walk",
        "entities": cmd.entity_ids,
        "x": cmd.target.position.x,
        "z": cmd.target.position.z,
        "queued": false
    });

    this.SendResponse({
        success: true,
        message: "Move command issued"
    });
};

NimsforestBridge.prototype.SendGameState = function() {
    var state = {
        game_time: Engine.QueryInterface(SYSTEM_ENTITY, IID_Timer).GetTime(),
        paused: Engine.IsGamePaused(),
        players: this.GetPlayerStates(),
        entities: this.GetEntityStates()
    };

    this.SendResponse(state);
};

NimsforestBridge.prototype.GetPlayerStates = function() {
    var players = [];
    var numPlayers = Engine.QueryInterface(SYSTEM_ENTITY, IID_PlayerManager).GetNumPlayers();

    for (var i = 1; i < numPlayers; ++i) {
        var playerEnt = Engine.QueryInterface(SYSTEM_ENTITY, IID_PlayerManager).GetPlayerByID(i);
        var playerState = Engine.QueryInterface(playerEnt, IID_Player);

        players.push({
            id: i,
            name: playerState.GetName(),
            civilization: playerState.GetCiv(),
            alive: playerState.GetState() === "active",
            resources: playerState.GetResourceCounts(),
            population: playerState.GetPopulationCount(),
            max_population: playerState.GetPopulationLimit(),
            score: playerState.GetScore()
        });
    }

    return players;
};

NimsforestBridge.prototype.GetEntityStates = function() {
    // Query all entities and return their state
    var entities = [];
    var cmpRangeManager = Engine.QueryInterface(SYSTEM_ENTITY, IID_RangeManager);
    var allEntities = cmpRangeManager.GetEntities();

    for (var entId of allEntities) {
        var cmpPosition = Engine.QueryInterface(entId, IID_Position);
        var cmpHealth = Engine.QueryInterface(entId, IID_Health);
        var cmpOwnership = Engine.QueryInterface(entId, IID_Ownership);

        if (!cmpPosition || !cmpPosition.IsInWorld())
            continue;

        var pos = cmpPosition.GetPosition();

        entities.push({
            id: entId,
            template: Engine.GetEntityTemplateName(entId),
            owner: cmpOwnership ? cmpOwnership.GetOwner() : 0,
            position: {x: pos.x, y: pos.y, z: pos.z},
            health: cmpHealth ? cmpHealth.GetHitpoints() : 0,
            max_health: cmpHealth ? cmpHealth.GetMaxHitpoints() : 0
        });
    }

    return entities;
};

NimsforestBridge.prototype.RegisterEventHandlers = function() {
    // Hook into game events
    Engine.RegisterGlobal("OnPlayerDefeated", this.OnPlayerDefeated.bind(this));
    Engine.RegisterGlobal("OnPlayerWon", this.OnPlayerWon.bind(this));
    Engine.RegisterGlobal("OnEntityKilled", this.OnEntityKilled.bind(this));
    Engine.RegisterGlobal("OnResearchFinished", this.OnResearchFinished.bind(this));
};

NimsforestBridge.prototype.EmitEvent = function(type, data) {
    var event = {
        type: type,
        timestamp: Engine.QueryInterface(SYSTEM_ENTITY, IID_Timer).GetTime(),
        data: data
    };

    this.eventSocket.Write(JSON.stringify(event) + "\n");
};

NimsforestBridge.prototype.OnPlayerDefeated = function(playerID) {
    this.EmitEvent("player_defeated", {
        player_id: playerID
    });
};

NimsforestBridge.prototype.OnPlayerWon = function(playerID) {
    this.EmitEvent("player_won", {
        player_id: playerID
    });
};

NimsforestBridge.prototype.OnEntityKilled = function(entityID, killerID) {
    this.EmitEvent("entity_killed", {
        entity_id: entityID,
        killer_id: killerID
    });
};

Engine.RegisterSystemComponentType(IID_NimsforestBridge, "NimsforestBridge", NimsforestBridge);
```

---

## 3. Integration with NimsForest AAA Pattern

### 3.1 GameNim - AI-Driven Game Controller

```go
// internal/nims/game/game_nim.go

package game

import (
    "context"
    "fmt"

    "github.com/nimsforest/nimsforest2/internal/core"
    "github.com/nimsforest/nimsforest2/pkg/nim"
)

// GameNim is an intelligent Nim that controls game engine agents
type GameNim struct {
    *core.BaseNim
    asker      nim.AIAsker
    wind       nim.Whisperer
    gameAgent  nim.GameEngineAgent
}

func NewGameNim(base *core.BaseNim, asker nim.AIAsker, wind nim.Whisperer, agent nim.GameEngineAgent) *GameNim {
    return &GameNim{
        BaseNim:   base,
        asker:     asker,
        wind:      wind,
        gameAgent: agent,
    }
}

// Subjects returns the event subjects this Nim subscribes to
func (g *GameNim) Subjects() []string {
    return []string{
        "game.command",        // Direct game commands
        "game.test.>",         // Automated testing
        "game.strategy.>",     // Strategy execution
        "game.event.>",        // Game events from agent
    }
}

// Advice - Ask AI about game strategy
func (g *GameNim) Advice(ctx context.Context, query string) (string, error) {
    // Get current game state
    state, err := g.gameAgent.GetGameState(ctx)
    if err != nil {
        return "", err
    }

    // Ask AI for advice with context
    prompt := fmt.Sprintf(`
Game: 0 A.D.
Current State:
- Game Time: %.2f
- Players: %d
- My Population: %d/%d
- My Resources: Food=%d, Wood=%d, Stone=%d, Metal=%d

Question: %s

Provide strategic advice considering the current game state.
`,
        state.GameTime,
        len(state.Players),
        state.Players[0].Population,
        state.Players[0].MaxPop,
        int(state.Players[0].Resources["food"]),
        int(state.Players[0].Resources["wood"]),
        int(state.Players[0].Resources["stone"]),
        int(state.Players[0].Resources["metal"]),
        query,
    )

    return g.asker.Ask(ctx, prompt)
}

// Action - Execute game actions via agent
func (g *GameNim) Action(ctx context.Context, action string, params map[string]interface{}) (interface{}, error) {
    // Parse action into game command
    cmd := g.parseAction(action, params)

    // Execute via game agent
    result, err := g.gameAgent.ExecuteCommand(ctx, cmd)
    if err != nil {
        return nil, err
    }

    // Emit result as leaf
    g.emitGameResult(ctx, result)

    return result, nil
}

// Automate - Create automated game behaviors
func (g *GameNim) Automate(ctx context.Context, automation string, enabled bool) (*nim.AutomateResult, error) {
    if !enabled {
        return g.disableAutomation(ctx, automation)
    }

    // Analyze what type of automation is needed
    analysisPrompt := fmt.Sprintf(`
Analyze this game automation request for 0 A.D.: "%s"

Determine if this requires:
1. Simple script - deterministic rules (e.g., "train villagers when idle")
2. AI-driven strategy - adaptive decision making (e.g., "counter opponent's army composition")

Respond with JSON:
{"type": "script" or "ai", "reason": "why", "logic": "pseudocode/strategy"}
`, automation)

    analysisJSON, err := g.asker.Ask(ctx, analysisPrompt)
    if err != nil {
        return nil, err
    }

    // Parse and create automation
    // This would generate either a Lua script or a TreeHouse
    return g.createGameAutomation(ctx, automation, analysisJSON)
}

// Handle - Process game events
func (g *GameNim) Handle(ctx context.Context, leaf nim.Leaf) error {
    subject := leaf.GetSubject()

    switch {
    case strings.HasPrefix(subject, "game.command"):
        return g.handleGameCommand(ctx, leaf)
    case strings.HasPrefix(subject, "game.test"):
        return g.handleGameTest(ctx, leaf)
    case strings.HasPrefix(subject, "game.strategy"):
        return g.handleStrategyRequest(ctx, leaf)
    case strings.HasPrefix(subject, "game.event"):
        return g.handleGameEvent(ctx, leaf)
    }

    return nil
}

func (g *GameNim) handleGameCommand(ctx context.Context, leaf nim.Leaf) error {
    var cmd nim.GameCommand
    if err := json.Unmarshal(leaf.GetData(), &cmd); err != nil {
        return err
    }

    result, err := g.gameAgent.ExecuteCommand(ctx, cmd)
    if err != nil {
        return err
    }

    g.emitGameResult(ctx, result)
    return nil
}

func (g *GameNim) handleGameTest(ctx context.Context, leaf nim.Leaf) error {
    // Automated game testing
    // Example: Test if unit training works
    // Example: Test if AI makes valid moves
    return nil
}

func (g *GameNim) handleStrategyRequest(ctx context.Context, leaf nim.Leaf) error {
    // AI-driven strategy execution
    // Example: "Execute aggressive rush strategy"
    // Example: "Play defensively and boom economy"

    var strategyReq struct {
        Strategy string                 `json:"strategy"`
        Duration float64                `json:"duration"` // Game time
        Params   map[string]interface{} `json:"params"`
    }

    if err := json.Unmarshal(leaf.GetData(), &strategyReq); err != nil {
        return err
    }

    // Use AI to plan and execute strategy
    return g.executeStrategy(ctx, strategyReq.Strategy, strategyReq.Params)
}

func (g *GameNim) executeStrategy(ctx context.Context, strategy string, params map[string]interface{}) error {
    // Get AI to break down strategy into steps
    prompt := fmt.Sprintf(`
Strategy: %s
Parameters: %v

Break down this 0 A.D. strategy into concrete steps with timing:
1. [0:00-2:00] Build economy - train 10 villagers
2. [2:00-5:00] Build barracks, train soldiers
...

Respond with JSON array of steps.
`, strategy, params)

    stepsJSON, err := g.asker.Ask(ctx, prompt)
    if err != nil {
        return err
    }

    // Parse steps and execute sequentially
    var steps []StrategyStep
    if err := json.Unmarshal([]byte(stepsJSON), &steps); err != nil {
        return err
    }

    // Execute steps with timing
    for _, step := range steps {
        if err := g.executeStep(ctx, step); err != nil {
            return err
        }
    }

    return nil
}

type StrategyStep struct {
    StartTime float64            `json:"start_time"`
    EndTime   float64            `json:"end_time"`
    Action    string             `json:"action"`
    Params    map[string]interface{} `json:"params"`
}

func (g *GameNim) executeStep(ctx context.Context, step StrategyStep) error {
    // Wait until game time reaches step.StartTime
    // Execute the action
    // Continue until step.EndTime
    return nil
}

func (g *GameNim) parseAction(action string, params map[string]interface{}) nim.GameCommand {
    // Parse natural language action into GameCommand
    // Example: "train 5 spearmen" -> GameCommand{Type: "train", ...}
    return nim.GameCommand{}
}

func (g *GameNim) emitGameResult(ctx context.Context, result *nim.GameResult) {
    // Emit result as leaf for other Nims to catch
    data, _ := json.Unmarshal(result)
    leaf := core.NewLeaf("game.result", data, "nim:game")
    g.wind.Whisper(ctx, leaf)
}
```

---

## 4. Configuration

### 4.1 forest.yaml Extension

```yaml
# Game Engine Agents
agents:
  game:
    0ad-player-1:
      engine: "0ad"
      version: "0.0.26"
      binary_path: "/usr/games/pyrogenesis"
      data_dir: "/usr/share/0ad"
      mods_dir: "/home/user/.local/share/0ad/mods"
      headless: true
      docker_image: "nimsforest/0ad:latest"  # Optional

    0ad-opponent-1:
      engine: "0ad"
      version: "0.0.26"
      binary_path: "/usr/games/pyrogenesis"
      headless: true
      # Can run multiple instances for multi-agent games

# Game Nims
nims:
  game-controller:
    subscribes: game.>
    publishes: game.result
    prompt: scripts/nims/game_controller.md
    agent: 0ad-player-1

  game-tester:
    subscribes: game.test.>
    publishes: game.test.result
    prompt: scripts/nims/game_tester.md
    agent: 0ad-opponent-1
```

---

## 5. Use Cases

### 5.1 Automated Game Testing

```go
// Test if game mechanics work correctly
test := nim.Task{
    Description: "Test unit training speed",
    Params: map[string]interface{}{
        "scenario": "test_maps/unit_training.xml",
        "tests": []string{
            "train_spearman_time_check",
            "train_cavalry_time_check",
        },
    },
}

result, err := gameAgent.Run(ctx, test)
// Result includes pass/fail for each test
```

### 5.2 AI vs AI Matches

```go
// Set up two game instances with different AI strategies
gameConfig1 := nim.GameConfig{
    Scenario: "scenarios/skirmish/alpine_valley.xml",
    Players: []nim.PlayerConfig{
        {ID: 1, Name: "NimsForest AI 1", ControlType: "script", AIScript: "aggressive_rush.js"},
        {ID: 2, Name: "0AD AI", ControlType: "ai", AIScript: "petra"},
    },
}

// Launch and observe
gameAgent.LaunchGame(ctx, gameConfig1)

// Subscribe to events
events, _ := gameAgent.SubscribeEvents(ctx, []string{"battle", "victory", "entity_killed"})

// Wind emits events for analysis
for event := range events {
    wind.Whisper(ctx, core.NewLeaf("game.event."+event.Type, eventData, "agent:game"))
}
```

### 5.3 AI-Assisted Gameplay

```go
// Human asks for strategic advice
advice, err := gameNim.Advice(ctx, "Should I attack now or wait?")
// AI analyzes game state and provides recommendation

// Human triggers action through Nim
gameNim.Action(ctx, "attack with cavalry", map[string]interface{}{
    "target": "enemy_town_center",
})
```

### 5.4 Replay Analysis

```go
// Load and analyze replay
gameAgent.LoadReplay(ctx, "replays/tournament_finals.0ad")

// AI watches and provides commentary
events, _ := gameAgent.SubscribeEvents(ctx, []string{"all"})

for event := range events {
    // Ask AI to analyze
    analysis, _ := gameNim.Advice(ctx, fmt.Sprintf("Analyze this event: %v", event))
    // Emit analysis as leaf
}
```

### 5.5 Tournament Orchestration

```go
// Run automated tournament
tournament := []Match{
    {Player1: "AI_Aggressive", Player2: "AI_Defensive"},
    {Player1: "AI_Rush", Player2: "AI_Boom"},
    // ...
}

for _, match := range tournament {
    // Launch game
    config := buildMatchConfig(match)
    gameAgent.LaunchGame(ctx, config)

    // Wait for victory
    result := <-waitForVictory(gameAgent)

    // Record result
    wind.Whisper(ctx, core.NewLeaf("tournament.result", result, "nim:game"))

    // Stop game
    gameAgent.StopGame(ctx)
}
```

---

## 6. Implementation Phases

### Phase 1: Core Game Agent (Week 1-2)
- [ ] Create GameEngineAgent interface in `pkg/nim/game_agent.go`
- [ ] Implement ZeroADAgent skeleton in `internal/ai/agents/game/0ad_agent.go`
- [ ] Basic game launch/stop functionality
- [ ] Headless mode support
- [ ] Unit tests for agent interface

**Validation**: Can launch 0 A.D. headless and stop it

### Phase 2: Communication Bridge (Week 2-3)
- [ ] Create nimsforest_control mod for 0 A.D.
- [ ] Implement IPC layer (Unix sockets or named pipes)
- [ ] Command sending (train, move, attack)
- [ ] Event receiving (victory, defeat, entity_killed)
- [ ] Integration tests

**Validation**: Can send commands and receive events from running game

### Phase 3: Game State Management (Week 3-4)
- [ ] Implement GetGameState() with full state query
- [ ] Parse player states, resources, entities
- [ ] Add replay save/load functionality
- [ ] State caching and updates
- [ ] Performance optimization

**Validation**: Can query game state in real-time with <100ms latency

### Phase 4: GameNim Integration (Week 4-5)
- [ ] Create GameNim in `internal/nims/game/game_nim.go`
- [ ] Implement AAA methods (Advice, Action, Automate)
- [ ] Event handling for game events
- [ ] AI-driven strategy execution
- [ ] Wire into Forest

**Validation**: Complete AAA flow works end-to-end

### Phase 5: Advanced Features (Week 5-6)
- [ ] Multi-instance game management
- [ ] Docker support for isolated game instances
- [ ] Automated testing framework
- [ ] AI vs AI match orchestration
- [ ] Replay analysis and commentary

**Validation**: Can run automated AI tournaments

### Phase 6: Production Hardening (Week 6-7)
- [ ] Error handling and recovery
- [ ] Resource cleanup (game processes, sockets)
- [ ] Monitoring and metrics
- [ ] Documentation
- [ ] Example scenarios and strategies

**Validation**: Production-ready with 80%+ test coverage

---

## 7. Technical Challenges & Solutions

### Challenge 1: Game Engine Communication
**Problem**: 0 A.D. doesn't have built-in external control API

**Solution**: Create custom mod that:
- Opens Unix domain socket for IPC
- Exposes JavaScript API bridge
- Handles command parsing and execution
- Emits events to external listener

### Challenge 2: Headless Mode Performance
**Problem**: Even in headless mode, 0 A.D. may be resource-intensive

**Solution**:
- Run in Docker containers with resource limits
- Use Land detection to only use Nimland/Manaland
- Implement game instance pooling
- Add game speed multiplier for faster simulation

### Challenge 3: Game State Synchronization
**Problem**: Querying full game state every frame is expensive

**Solution**:
- Cache game state and update incrementally via events
- Only query full state on-demand or periodically
- Use event-driven updates for entity changes
- Implement smart state diffing

### Challenge 4: AI Strategy Execution
**Problem**: AI needs to make real-time decisions in fast-paced game

**Solution**:
- Pre-plan strategies with AI before game starts
- Use deterministic scripts for micro-management
- AI provides high-level strategy, scripts execute tactics
- Hybrid approach: AI + rule-based systems

### Challenge 5: Multi-Instance Management
**Problem**: Running multiple game instances simultaneously

**Solution**:
- Each game instance runs in separate Docker container
- Use unique IPC socket paths per instance
- AgentHouse manages container lifecycle
- Load balancing across Nimlands

---

## 8. Testing Strategy

### Unit Tests
```go
// Test game agent interface
func TestZeroADAgent_LaunchGame(t *testing.T)
func TestZeroADAgent_ExecuteCommand(t *testing.T)
func TestZeroADAgent_GetGameState(t *testing.T)
```

### Integration Tests
```go
// Test full game flow
func TestGameNim_FullGameFlow(t *testing.T) {
    // 1. Launch game
    // 2. Query state
    // 3. Execute commands
    // 4. Verify state changes
    // 5. Stop game
}
```

### E2E Tests
```go
// Test real game scenarios
func TestE2E_AIvsAIMatch(t *testing.T)
func TestE2E_StrategyExecution(t *testing.T)
func TestE2E_ReplayAnalysis(t *testing.T)
```

---

## 9. Docker Setup

### Dockerfile for 0 A.D.

```dockerfile
FROM ubuntu:22.04

# Install 0 A.D.
RUN apt-get update && apt-get install -y \
    0ad \
    0ad-data \
    && rm -rf /var/lib/apt/lists/*

# Copy nimsforest control mod
COPY mods/nimsforest_control /root/.local/share/0ad/mods/nimsforest_control

# Set up IPC directory
RUN mkdir -p /var/run/nimsforest

# Entry point
CMD ["/usr/games/pyrogenesis", "-autostart-nonvisual", "-mod=nimsforest_control"]
```

### Docker Compose for Multi-Instance

```yaml
version: '3.8'

services:
  game-1:
    build: .
    image: nimsforest/0ad:latest
    volumes:
      - ./replays:/root/.local/share/0ad/replays
      - ./ipc:/var/run/nimsforest
    environment:
      - NIMSFOREST_INSTANCE_ID=game-1
    mem_limit: 2g
    cpus: 2

  game-2:
    image: nimsforest/0ad:latest
    volumes:
      - ./replays:/root/.local/share/0ad/replays
      - ./ipc:/var/run/nimsforest
    environment:
      - NIMSFOREST_INSTANCE_ID=game-2
    mem_limit: 2g
    cpus: 2
```

---

## 10. Deployment Considerations

### Resource Requirements
- **CPU**: 2 cores per game instance minimum
- **RAM**: 2GB per game instance
- **Storage**: 5GB for game data + mods
- **Network**: Minimal (only IPC, no network gameplay)

### Land Requirements
- **Land Type**: Nimland minimum (Docker required)
- **Manaland**: Optional, for GPU-accelerated rendering (if not headless)

### Scaling Strategy
- Horizontal: Add more Nimlands to run more game instances
- Vertical: Increase resources per Land for faster games
- Load Balancing: AgentHouse distributes games across available Lands

---

## 11. Future Enhancements

### Phase 2 Additions
1. **Multi-Engine Support**: Add Unity, Godot, Unreal Engine agents
2. **Visual Observation**: Computer vision to observe game screen
3. **Human-in-the-Loop**: Allow human to override AI decisions
4. **Reinforcement Learning**: Train AI by playing thousands of games
5. **Live Streaming**: Stream games to Twitch/YouTube via StreamNim

### Advanced Features
1. **Tournament System**: Automated bracket tournaments with rankings
2. **Meta-Game Analysis**: Analyze winning strategies across many games
3. **Mod Development**: AI generates and tests game mods
4. **Bug Detection**: Automated testing to find game bugs
5. **Balance Tuning**: Simulate thousands of games to balance gameplay

---

## 12. Success Metrics

### Phase 1 Success
- ✅ Can launch 0 A.D. headless via agent
- ✅ Can send basic commands (move, attack, train)
- ✅ Can receive game events (victory, defeat)
- ✅ Unit tests pass with 80%+ coverage

### Phase 4 Success
- ✅ GameNim can control game via AAA pattern
- ✅ AI can provide strategic advice based on game state
- ✅ Can execute AI-driven strategies
- ✅ Integration tests pass

### Phase 6 Success
- ✅ Can run automated AI vs AI tournaments
- ✅ Can analyze replays with AI commentary
- ✅ Production-ready with monitoring
- ✅ Documentation complete
- ✅ Demo video showing full capabilities

---

## 13. Example Usage Scenarios

### Scenario 1: AI Learning to Play

```go
// GameNim learns 0 A.D. by playing against built-in AI
for i := 0; i < 1000; i++ {
    // Launch game
    config := nim.GameConfig{
        Scenario: "random",
        Players: []nim.PlayerConfig{
            {ID: 1, Name: "NimsForest Learner", ControlType: "script"},
            {ID: 2, Name: "Petra AI", ControlType: "ai", AIScript: "petra"},
        },
        SpeedFactor: 4.0, // 4x speed
    }

    gameAgent.LaunchGame(ctx, config)

    // Play game with AI making decisions
    for {
        state, _ := gameAgent.GetGameState(ctx)

        if state.Victory != nil {
            break
        }

        // Ask AI what to do
        advice, _ := gameNim.Advice(ctx, "What should I do next?")

        // Execute AI decision
        gameNim.Action(ctx, advice, nil)

        time.Sleep(1 * time.Second)
    }

    // Record result for learning
    replay, _ := gameAgent.SaveReplay(ctx)
    gameAgent.StopGame(ctx)
}
```

### Scenario 2: Automated QA Testing

```go
// Test all civilizations for balance
civilizations := []string{"romans", "carthaginians", "gauls", "iberians", ...}

for _, civ1 := range civilizations {
    for _, civ2 := range civilizations {
        if civ1 == civ2 {
            continue
        }

        // Run 10 matches between each pair
        wins1, wins2 := 0, 0

        for game := 0; game < 10; game++ {
            config := nim.GameConfig{
                Scenario: "balanced_map",
                Players: []nim.PlayerConfig{
                    {ID: 1, Civilization: civ1, ControlType: "ai"},
                    {ID: 2, Civilization: civ2, ControlType: "ai"},
                },
            }

            gameAgent.LaunchGame(ctx, config)
            result := <-waitForVictory(gameAgent)
            gameAgent.StopGame(ctx)

            if result.Winner == 1 {
                wins1++
            } else {
                wins2++
            }
        }

        // Check for balance issues
        if wins1 > 7 || wins2 > 7 {
            // Emit imbalance warning
            wind.Whisper(ctx, core.NewLeaf("game.balance.warning", balanceData, "nim:game"))
        }
    }
}
```

---

## 14. References

- **0 A.D. Engine**: https://github.com/0ad/0ad (migrated to https://gitea.wildfiregames.com)
- **0 A.D. Modding Guide**: https://trac.wildfiregames.com/wiki/Modding_Guide
- **JavaScript API**: https://trac.wildfiregames.com/wiki/JavaScriptAPI
- **NimsForest AAA Pattern**: `/home/user/nimsforest2/plan-aaa-nim.md`
- **Agent Patterns**: `/home/user/nimsforest2/plan-aaa-nim.md` (Part 1: Agent Types)

---

## 15. Timeline & Milestones

| Phase | Duration | Deliverable | Validation |
|-------|----------|-------------|------------|
| Phase 1 | Week 1-2 | Core Game Agent | Can launch/stop 0 A.D. |
| Phase 2 | Week 2-3 | Communication Bridge | Can send commands, receive events |
| Phase 3 | Week 3-4 | State Management | Can query game state real-time |
| Phase 4 | Week 4-5 | GameNim Integration | AAA pattern works |
| Phase 5 | Week 5-6 | Advanced Features | Automated tournaments work |
| Phase 6 | Week 6-7 | Production Ready | 80%+ coverage, documented |

**Total Estimated Time**: 6-7 weeks for complete implementation

---

## 16. Conclusion

This plan provides a comprehensive path to integrate 0 A.D. as a game engine agent in nimsforest2, following the established AAA pattern and agent architecture. The implementation enables:

1. **Automated Gameplay**: AI-controlled players in 0 A.D.
2. **Strategic Advice**: AI provides human players with tactical recommendations
3. **Game Testing**: Automated QA and balance testing
4. **AI Training**: Reinforcement learning through thousands of simulated games
5. **Tournament Orchestration**: Fully automated competitive matches

The modular design allows future extension to other game engines (Unity, Godot, Unreal) using the same GameEngineAgent interface.

---

**Next Steps**: Begin Phase 1 implementation by creating the GameEngineAgent interface and ZeroADAgent skeleton.

**Branch**: `claude/plan-ad0-engine-interface-2p5BU`
**Last Updated**: 2026-01-12
