# Plan: AAA Nim Implementation

## Overview

Implement the AAA (Advice/Action/Automate) pattern for Nims in nimsforest2.

### Key Decisions

| Decision | Choice |
|----------|--------|
| TreeHouse vs Nim | TreeHouse = deterministic, Nim = intelligent (AAA required) |
| Agent execution | Docker containers on NimsForest nodes |
| AI service for Advice | Keep existing `pkg/integrations/aiservice/` |
| Human communication | Extend Songbird with Send (non-blocking) |
| Example code | Move compile-time examples to `examples/` |

---

## Conceptual Model

### TreeHouse vs Nim

| Component | Purpose | AAA |
|-----------|---------|-----|
| **TreeHouse** | Deterministic event processing | No |
| **Nim** | Intelligent agent | **Required** |

Both can be **compile-time (Go)** or **runtime (Lua/config)**.

If a component doesn't need AAA, it's a TreeHouse—regardless of language.

### Files to Move to `examples/`

| File | Reason |
|------|--------|
| `internal/nims/aftersales.go` | No AAA = TreeHouse |
| `internal/nims/general.go` | No AAA = TreeHouse |
| `internal/trees/payment.go` | Domain-specific |
| `internal/trees/general.go` | Domain-specific |
| `internal/leaves/chat.go` | Domain-specific |

Runtime examples (`scripts/`) stay in repo.

---

## Architecture

### System Overview

```
┌───────────────────────────────────────────────────────────────────────────────────┐
│                                   Morpheus                                        │
│                         (provisions NimsForest nodes)                             │
└─────────────────────────────────────┬─────────────────────────────────────────────┘
                                      │
        ┌─────────────────────────────┼─────────────────────────────┐
        ▼                             ▼                             ▼
┌─────────────────┐           ┌─────────────────┐           ┌─────────────────┐
│      LAND       │           │    NIMLAND      │           │    MANALAND     │
│   (backbone)    │           │    (docker)     │           │   (docker+gpu)  │
│                 │           │                 │           │                 │
│ - Wind (NATS)   │◄─────────►│ - Wind (NATS)   │◄─────────►│ - Wind (NATS)   │
│ - River         │           │ - Docker ✓      │           │ - Docker ✓      │
│ - Trees         │           │                 │           │ - GPU/VRAM ✓    │
│ - Treehouses    │           │ ┌─────────────┐ │           │                 │
│ - Nims          │           │ │  AIAgent    │ │           │ ┌─────────────┐ │
│                 │           │ │  Browser    │ │           │ │ GPU Agent   │ │
│ NO Docker       │           │ └─────────────┘ │           │ │ LLM local   │ │
│ Low latency     │           │                 │           │ └─────────────┘ │
└─────────────────┘           └─────────────────┘           └─────────────────┘
```

| Type | Docker | GPU | Purpose |
|------|--------|-----|---------|
| **Land** | No | No | Event processing backbone (low latency) |
| **Nimland** | Yes | No | Docker agents (AIAgent, BrowserAgent) |
| **Manaland** | Yes | Yes | GPU-accelerated workloads |

### AAA Pattern (Mandatory for Nims)

| Method | Purpose | Implementation |
|--------|---------|----------------|
| **Advice** | Ask question, get answer | aiservicefactory API call |
| **Action** | Execute task via agent | Dispatch to Agent (AI/Human/Robot/Browser) |
| **Automate** | Create persistent automation | Generate TreeHouse (Lua) or Nim (config) |

**Note:** At least one AAA method must be meaningful. If none are needed, use a TreeHouse instead.

---

## Part 1: Agent Types

### Agent Hierarchy

```
                            ┌─────────────┐
                            │    Agent    │
                            │ (interface) │
                            └──────┬──────┘
                                   │
        ┌────────────┬─────────────┼─────────────┬────────────┐
        │            │             │             │            │
┌───────▼───────┐ ┌──▼──────────┐ ┌▼───────────┐ ┌▼───────────┐
│   AIAgent     │ │ HumanAgent  │ │ RobotAgent │ │BrowserAgent│
│               │ │             │ │            │ │            │
│ - Docker      │ │ - Songbird  │ │ - Temi     │ │ - Playwright│
│ - Claude      │ │ - Telegram  │ │ - SO-ARM100│ │ - Puppeteer│
│ - Aider       │ │ - Slack     │ │ - Humanoid │ │ - Selenium │
│ - Cursor      │ │ - Email     │ │            │ │ - Headless │
└───────────────┘ └─────────────┘ └────────────┘ └────────────┘
```

**Note:** CI/CD, webhooks, and external APIs flow through **River**, not agents.

### 1.1 Agent Interface

Create `pkg/nim/agent.go`:

```go
package nim

import "context"

// Agent executes tasks (AI, Human, Robot, or Browser)
type Agent interface {
    Run(ctx context.Context, task Task) (*Result, error)
    Type() AgentType
    Available(ctx context.Context) bool
}

type AgentType string

const (
    AgentTypeAI      AgentType = "ai"
    AgentTypeHuman   AgentType = "human"
    AgentTypeRobot   AgentType = "robot"
    AgentTypeBrowser AgentType = "browser"
)

type Task struct {
    ID          string
    Description string
    Params      map[string]interface{}
    RequiredAgent AgentType  // Optional: specify agent type
}

type Result struct {
    Success bool
    Output  string
    Files   []FileDiff
    Error   string
}

type FileDiff struct {
    Path   string
    Action string // created, modified, deleted
    Diff   string
}
```

### 1.2 AIAgent

Runs in Docker containers on NimsForest nodes.

```go
// pkg/nim/ai_agent.go

type AIAgent interface {
    Agent
    Image() string   // Docker image
    Tools() []string // Available tools (claude, aider, etc.)
}

type AIAgentConfig struct {
    Name   string
    Image  string   // nimsforest/claude-agent:latest
    Tools  []string // ["claude"]
    Memory string   // "4g"
    CPU    int      // 2
}
```

**Docker Images:**
```
nimsforest/claude-agent:latest    - Claude Code CLI
nimsforest/aider-agent:latest     - Aider CLI
nimsforest/cursor-agent:latest    - Cursor CLI
```

### 1.3 HumanAgent

Routes tasks to humans via Songbird. Has role, responsibility, and list of members.

```go
// pkg/nim/human_agent.go

type HumanAgent interface {
    Agent
    Role() string
    Responsibility() string
    Members() []Human
}

type HumanAgentConfig struct {
    Name           string
    Role           string   // "approver", "reviewer", "decision-maker"
    Responsibility string   // "Approve PRs before merge"
    Members        []Human
}

type Human struct {
    ID       string
    Name     string
    Platform string // "telegram", "slack", "email"
    Contact  string // Platform-specific contact
    Status   string // "available", "busy", "offline"
}
```

### 1.4 RobotAgent

Controls physical robots (humanoids, mobile robots, robotic arms).

```go
// pkg/nim/robot_agent.go

type RobotAgent interface {
    Agent
    Model() string      // Robot model (temi, so-arm100, humanoid)
    Location() string   // Physical location
    Capabilities() []string
}

type RobotAgentConfig struct {
    Name           string
    Model          string   // "temi", "so-arm100", "humanoid"
    Location       string   // "office-a", "warehouse-1"
    Endpoint       string   // Robot's API endpoint
    Capabilities   []string // ["navigate", "speak", "pick", "place"]
}
```

**Robot Types:**
```
temi           - Mobile telepresence robot
so-arm100      - Robotic arm for manipulation
humanoid       - Humanoid robot (various models)
```

**Note:** CI/CD, webhooks, and external APIs are handled by the **River** system, not agents.

### 1.5 BrowserAgent

Automates web browser interactions.

```go
// pkg/nim/browser_agent.go

type BrowserAgent interface {
    Agent
    Browser() string
    Headless() bool
}

type BrowserAgentConfig struct {
    Name           string
    Role           string   // "scraper", "tester", "automator"
    Responsibility string
    Image          string   // Docker image with browser
    Browser        string   // "chromium", "firefox"
    Headless       bool
}
```

**Docker Images:**
```
nimsforest/browser-agent:playwright  - Playwright + Chromium
nimsforest/browser-agent:puppeteer   - Puppeteer + Chrome
nimsforest/browser-agent:selenium    - Selenium + Firefox
```

---

## Part 2: Songbird (Human Communication)

Songbird wraps all human communication channels.

### Songbird Hierarchy

```
                    ┌─────────────┐
                    │  Songbird   │
                    │ (interface) │
                    └──────┬──────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
┌───────▼───────┐  ┌───────▼───────┐  ┌───────▼───────┐
│   Telegram    │  │    Slack      │  │    Email      │
│   Songbird    │  │   Songbird    │  │   Songbird    │
└───────────────┘  └───────────────┘  └───────────────┘
```

### Songbird Interface

```go
// internal/songbirds/songbird.go (already exists, extend)

type Songbird interface {
    Name() string
    Type() string    // "telegram", "slack", "email" (existing method)
    Pattern() string // Wind subject pattern (existing method)
    Start(ctx context.Context) error
    Stop() error
    IsRunning() bool // existing method
    
    // Add for human agent support (non-blocking)
    Send(ctx context.Context, msg Message) error
}

type Message struct {
    ID      string   // Correlation ID - links request to response
    To      string   // Recipient
    Text    string   // Message content
    Options []string // For choices like "Approve / Reject"
    ReplyTo string   // Wind subject for response (e.g., "human.response.{ID}")
}
```

### Response Flow (Event-Driven)

Responses arrive as Leaves on the Wind, not as return values:

```
Send(msg) ──► Telegram ──► Human
                              │
                              ▼ (human replies)
                          Telegram webhook
                              │
                              ▼
                    Songbird receives reply
                              │
                              ▼
              Whisper(Leaf{Subject: "human.response.{ID}"})
                              │
                              ▼
                    Nim.Handle(leaf) ◄── response arrives here
```

The correlation ID in the original message links request to response.

### Implementations Needed

| Songbird | Status |
|----------|--------|
| TelegramSongbird | Exists |
| SlackSongbird | To create |
| EmailSongbird | To create |

---

## Part 3: Land Capacity (via Houses)

Land info flows as Leaves via **Go TreeHouses** (compile-time, deterministic).

### TreeHouse Types

| Type | Definition | Requirement |
|------|------------|-------------|
| **TreeHouse** | Lua (runtime) OR Go (compile-time) | **Deterministic** processing |
| **Nim** | Go with AI | **Non-deterministic** (uses intelligence) |

### Land Types

| Type | Docker | GPU | Can Run Agents |
|------|--------|-----|----------------|
| **Land** | No | No | No |
| **Nimland** | Yes | No | Yes (AI, Browser) |
| **Manaland** | Yes | Yes | Yes (GPU workloads) |

### Houses for Land

Two compile-time Go TreeHouses handle Land communication:

| House | Subscribes | Publishes | Purpose |
|-------|------------|-----------|---------|
| **LandHouse** | `land.query` | `land.info.{id}` | Respond to capacity queries |
| **AgentHouse** | `agent.task.{id}` | `agent.result.{task_id}` | Execute agents in Docker |

### Architecture

```
Forest
├── thisLand (LandInfo)          ← Data, detected at startup
│
├── LandHouse (Go TreeHouse)     ← Deterministic Leaf handler
│   ├── Subscribes: land.query
│   ├── Publishes: land.info.{id}
│   └── Has reference to thisLand
│
└── AgentHouse (Go TreeHouse)    ← Deterministic (Nimland/Manaland only)
    ├── Subscribes: agent.task.{land_id}
    ├── Publishes: agent.result.{task_id}
    └── Runs Docker containers
```

### LandHouse Implementation

```go
// internal/treehouses/landhouse.go

// LandHouse is a compile-time TreeHouse that handles Land queries.
// Deterministic: same query + same Land capabilities = same response
type LandHouse struct {
    land *core.LandInfo
    wind *core.Wind
}

func NewLandHouse(land *core.LandInfo, wind *core.Wind) *LandHouse {
    return &LandHouse{land: land, wind: wind}
}

func (lh *LandHouse) Name() string { return "landhouse" }

func (lh *LandHouse) Subjects() []string {
    return []string{"land.query"}
}

// Process handles a land.query Leaf and returns land.info.{id} if we match
func (lh *LandHouse) Process(leaf core.Leaf) *core.Leaf {
    var query CapacityQuery
    if err := json.Unmarshal(leaf.Data, &query); err != nil {
        return nil
    }
    
    // Deterministic matching logic
    if query.NeedsDocker && !lh.land.HasDocker {
        return nil // Don't respond
    }
    if query.NeedsGPU && lh.land.GPUVram == 0 {
        return nil // Don't respond
    }
    
    // Return our Land info
    data, _ := json.Marshal(lh.land)
    return core.NewLeaf(
        fmt.Sprintf("land.info.%s", lh.land.ID),
        data,
        "treehouse:landhouse",
    )
}
```

### AgentHouse Implementation

```go
// internal/treehouses/agenthouse.go

// AgentHouse is a compile-time TreeHouse that executes agent tasks in Docker.
// Deterministic: dispatch task to Docker, capture result
type AgentHouse struct {
    landID string
    wind   *core.Wind
}

func NewAgentHouse(landID string, wind *core.Wind) *AgentHouse {
    return &AgentHouse{landID: landID, wind: wind}
}

func (ah *AgentHouse) Name() string { return "agenthouse" }

func (ah *AgentHouse) Subjects() []string {
    return []string{fmt.Sprintf("agent.task.%s", ah.landID)}
}

// Process handles an agent.task.{land_id} Leaf and returns agent.result.{task_id}
func (ah *AgentHouse) Process(leaf core.Leaf) *core.Leaf {
    var task AgentTask
    if err := json.Unmarshal(leaf.Data, &task); err != nil {
        return nil
    }
    
    // Run in Docker (deterministic dispatch)
    result := ah.runDocker(task)
    
    data, _ := json.Marshal(result)
    return core.NewLeaf(
        fmt.Sprintf("agent.result.%s", task.ID),
        data,
        "treehouse:agenthouse",
    )
}

func (ah *AgentHouse) runDocker(task AgentTask) AgentResult {
    cmd := exec.Command("docker", "run", "--rm",
        "-v", fmt.Sprintf("%s:/workspace", task.Workdir),
        task.Image,
        task.Command,
    )
    
    output, err := cmd.CombinedOutput()
    
    return AgentResult{
        TaskID:  task.ID,
        Success: err == nil,
        Output:  string(output),
        Error:   errorString(err),
    }
}
```

### Flow

```
┌──────────┐                              ┌───────────────────────────┐
│ CoderNim │                              │ Forest B (Nimland)        │
└────┬─────┘                              │  ├── LandHouse            │
     │                                    │  └── AgentHouse           │
     │                                    └────────────┬──────────────┘
     │                                                 │
     │── Leaf("land.query", {needs_docker}) ──────────►│
     │                                                 │ LandHouse.Process()
     │◄── Leaf("land.info.B", landInfo) ──────────────│
     │                                                 │
     │── Leaf("agent.task.B", task) ──────────────────►│
     │                                                 │ AgentHouse.Process()
     │                                                 │ (runs Docker)
     │◄── Leaf("agent.result.{id}", result) ──────────│
```

### Forest Wires Up Houses

```go
func (f *Forest) Start(ctx context.Context) error {
    // ... existing startup ...
    
    // Create and start LandHouse
    f.landHouse = treehouses.NewLandHouse(f.thisLand, f.wind)
    f.startTreeHouse(f.landHouse)
    
    // Create AgentHouse only if we have Docker
    if f.thisLand.HasDocker {
        f.agentHouse = treehouses.NewAgentHouse(f.thisLand.ID, f.wind)
        f.startTreeHouse(f.agentHouse)
    }
    
    // ... rest of startup ...
}
```

---

## Part 4: pkg/nim/ Package

### Directory Structure

```
pkg/nim/
├── nim.go              # Nim interface with AAA
├── brain.go            # Brain interface (from pkg/brain)
├── leaf.go             # Leaf interface
├── wind.go             # Whisperer interface
├── agent.go            # Agent interface + types
├── ai_agent.go         # AIAgent interface
├── human_agent.go      # HumanAgent interface
├── robot_agent.go      # RobotAgent interface
├── browser_agent.go    # BrowserAgent interface
├── land.go             # Land types and capacity events
└── asker.go            # AIAsker interface
```

### 4.1 Nim Interface

```go
// pkg/nim/nim.go

package nim

import "context"

// Nim is an intelligent agent with AAA capabilities.
// AAA is MANDATORY - if you don't need AAA, use a TreeHouse instead.
type Nim interface {
    Name() string
    
    // AAA Model - at least one must be meaningful
    Advice(ctx context.Context, query string) (string, error)
    Action(ctx context.Context, action string, params map[string]interface{}) (interface{}, error)
    Automate(ctx context.Context, automation string, enabled bool) (*AutomateResult, error)
    
    // Event handling (can trigger AAA logic)
    Handle(ctx context.Context, leaf Leaf) error
    
    // Lifecycle
    Start(ctx context.Context) error
    Stop() error
}

// AutomateResult describes what Automate created
type AutomateResult struct {
    Created     string // "treehouse" or "nim"
    Name        string
    Reason      string // Why this type was chosen
    ScriptPath  string // Path to generated script/config
    NeedsReview bool   // Requires human review before activation
}

// ErrNotSupported is returned when a Nim doesn't support an AAA method.
// At least one method must NOT return this error.
var ErrNotSupported = errors.New("operation not supported by this nim")
```

### 4.2 Brain Interface

Move from `pkg/brain/`:

```go
// pkg/nim/brain.go

package nim

import (
    "context"
    "errors"
    "time"
)

type Brain interface {
    Store(ctx context.Context, content string, tags []string) (*Knowledge, error)
    Retrieve(ctx context.Context, id string) (*Knowledge, error)
    Search(ctx context.Context, query string) ([]*Knowledge, error)
    Update(ctx context.Context, id string, content string) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context) ([]*Knowledge, error)
    Ask(ctx context.Context, question string) (string, error)
    Initialize(ctx context.Context) error
    Close(ctx context.Context) error
}

type Knowledge struct {
    ID        string
    Content   string
    Tags      []string
    CreatedAt time.Time
    UpdatedAt time.Time
}

var ErrKnowledgeNotFound = errors.New("knowledge not found")
```

### 4.3 Leaf and Wind Interfaces

```go
// pkg/nim/leaf.go
package nim

type Leaf interface {
    GetSubject() string
    GetData() []byte
    GetSource() string
}

// pkg/nim/wind.go
package nim

import "context"

type Whisperer interface {
    Whisper(ctx context.Context, leaf Leaf) error
}
```

### 4.4 Asker Interface

```go
// pkg/nim/asker.go
package nim

import "context"

// AIAsker provides prompt → response (for Advice)
type AIAsker interface {
    Ask(ctx context.Context, prompt string) (string, error)
}
```

---

## Part 5: Implementation in internal/

### Directory Structure

```
internal/
├── ai/
│   ├── asker.go           # Wraps existing aiservice
│   └── agents/
│       ├── ai_agent.go    # Docker-based AI agent
│       ├── human_agent.go # Songbird-based human agent
│       ├── robot_agent.go # Physical robot agent
│       └── browser_agent.go # Playwright browser agent
├── land/
│   └── handler.go         # Land capacity handler (event-driven)
├── core/
│   ├── nim.go             # Update BaseNim with AAA
│   ├── leaf.go            # Implement nim.Leaf
│   └── wind.go            # Implement nim.Whisperer
├── songbirds/
│   ├── songbird.go        # Existing interface (extend)
│   ├── telegram.go        # Existing (extend)
│   ├── slack.go           # To create
│   └── email.go           # To create
├── nims/
│   ├── .gitkeep           # Example nims moved to examples/
│   └── coder/             # CoderNim is CORE infrastructure, not an example
│       ├── coder.go
│       └── coder_test.go
├── trees/
│   └── .gitkeep           # Example trees moved to examples/
└── leaves/
    ├── types.go           # Core leaf type definitions (keep)
    └── .gitkeep           # Example leaves moved to examples/
```

**Note:** CoderNim is core AAA infrastructure (not an example). Runtime examples stay in `scripts/`.

### 5.1 Asker Implementation

```go
// internal/ai/asker.go

package ai

import (
    "context"
    
    // Use existing internal AI service (keep pkg/integrations/aiservice)
    "github.com/yourusername/nimsforest/pkg/integrations/aiservice"
    "github.com/yourusername/nimsforest/pkg/nim"
)

type Asker struct {
    service aiservice.AIService
}

func NewAsker(serviceType, apiKey, model string) (nim.AIAsker, error) {
    service, err := aiservice.NewAIService(serviceType, aiservice.Config{
        APIKey: apiKey,
        Model:  model,
    })
    if err != nil {
        return nil, err
    }
    return &Asker{service: service}, nil
}

func (a *Asker) Ask(ctx context.Context, prompt string) (string, error) {
    return a.service.Ask(ctx, prompt)
}
```

**Note:** Keep existing `pkg/integrations/aiservice/` rather than replacing with external aiservicefactory. The external dependency can be evaluated later.

### 5.2 AI Agent Implementation

```go
// internal/ai/agents/ai_agent.go

package agents

import (
    "context"
    "fmt"
    "os/exec"
    
    "github.com/yourusername/nimsforest/pkg/nim"
)

type DockerAIAgent struct {
    config nim.AIAgentConfig
    landID string
}

func NewDockerAIAgent(config nim.AIAgentConfig, landID string) *DockerAIAgent {
    return &DockerAIAgent{config: config, landID: landID}
}

func (a *DockerAIAgent) Run(ctx context.Context, task nim.Task) (*nim.Result, error) {
    // Build docker run command
    args := []string{
        "run", "--rm",
        "-v", fmt.Sprintf("%s:/workspace", task.Params["workdir"]),
        "-e", fmt.Sprintf("ANTHROPIC_API_KEY=%s", task.Params["api_key"]),
        a.config.Image,
        task.Description,
    }
    
    cmd := exec.CommandContext(ctx, "docker", args...)
    output, err := cmd.CombinedOutput()
    
    if err != nil {
        return &nim.Result{
            Success: false,
            Error:   err.Error(),
            Output:  string(output),
        }, nil
    }
    
    return &nim.Result{
        Success: true,
        Output:  string(output),
    }, nil
}

func (a *DockerAIAgent) Type() nim.AgentType { return nim.AgentTypeAI }
func (a *DockerAIAgent) Available(ctx context.Context) bool { return true }
func (a *DockerAIAgent) Image() string { return a.config.Image }
func (a *DockerAIAgent) Tools() []string { return a.config.Tools }
```

### 5.3 Human Agent Implementation

```go
// internal/ai/agents/human_agent.go

package agents

import (
    "context"
    "fmt"
    
    "github.com/yourusername/nimsforest/internal/songbirds"
    "github.com/yourusername/nimsforest/pkg/nim"
)

type SongbirdHumanAgent struct {
    config   nim.HumanAgentConfig
    songbird songbirds.Songbird
}

func NewSongbirdHumanAgent(config nim.HumanAgentConfig, sb songbirds.Songbird) *SongbirdHumanAgent {
    return &SongbirdHumanAgent{config: config, songbird: sb}
}

// Run sends the task to a human - does NOT block waiting for response.
// Response arrives later as a Leaf on the Wind.
func (a *SongbirdHumanAgent) Run(ctx context.Context, task nim.Task) (*nim.Result, error) {
    // Find available member
    var member *nim.Human
    for _, m := range a.config.Members {
        if m.Status == "available" {
            member = &m
            break
        }
    }
    if member == nil {
        return nil, fmt.Errorf("no available members for role %s", a.config.Role)
    }
    
    // Send message via songbird (non-blocking)
    msg := songbirds.Message{
        ID:      task.ID,  // Correlation ID
        To:      member.Contact,
        Text:    fmt.Sprintf("[%s] %s", a.config.Role, task.Description),
        ReplyTo: fmt.Sprintf("human.response.%s", task.ID),
    }
    
    if err := a.songbird.Send(ctx, msg); err != nil {
        return nil, err
    }
    
    // Return immediately - response comes later as event
    return &nim.Result{
        Success: true,
        Output:  fmt.Sprintf("Request sent to %s, awaiting response on human.response.%s", member.Name, task.ID),
    }, nil
}

func (a *SongbirdHumanAgent) Type() nim.AgentType { return nim.AgentTypeHuman }
func (a *SongbirdHumanAgent) Available(ctx context.Context) bool {
    for _, m := range a.config.Members {
        if m.Status == "available" {
            return true
        }
    }
    return false
}
func (a *SongbirdHumanAgent) Role() string { return a.config.Role }
func (a *SongbirdHumanAgent) Responsibility() string { return a.config.Responsibility }
func (a *SongbirdHumanAgent) Members() []nim.Human { return a.config.Members }
```

**Handling the Response:**

The Nim that dispatched the human task must subscribe to `human.response.>` and handle responses:

```go
func (c *CoderNim) Handle(ctx context.Context, leaf nim.Leaf) error {
    subject := leaf.GetSubject()
    
    // Handle human responses
    if strings.HasPrefix(subject, "human.response.") {
        taskID := strings.TrimPrefix(subject, "human.response.")
        return c.handleHumanResponse(ctx, taskID, leaf)
    }
    
    // Handle other events...
    return nil
}
```

### 5.4 Robot Agent Implementation

```go
// internal/ai/agents/robot_agent.go

package agents

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    
    "github.com/yourusername/nimsforest/pkg/nim"
)

type PhysicalRobotAgent struct {
    config nim.RobotAgentConfig
    client *http.Client
}

func NewPhysicalRobotAgent(config nim.RobotAgentConfig) *PhysicalRobotAgent {
    return &PhysicalRobotAgent{
        config: config,
        client: &http.Client{},
    }
}

func (a *PhysicalRobotAgent) Run(ctx context.Context, task nim.Task) (*nim.Result, error) {
    // Build command for the robot
    command := map[string]interface{}{
        "task_id":     task.ID,
        "description": task.Description,
        "params":      task.Params,
    }
    payload, _ := json.Marshal(command)
    
    // Send to robot's API endpoint
    req, err := http.NewRequestWithContext(ctx, "POST", a.config.Endpoint+"/execute", bytes.NewReader(payload))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := a.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("robot %s unreachable: %w", a.config.Name, err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode >= 400 {
        return &nim.Result{
            Success: false,
            Error:   fmt.Sprintf("robot returned HTTP %d", resp.StatusCode),
        }, nil
    }
    
    return &nim.Result{
        Success: true,
        Output:  fmt.Sprintf("Task dispatched to robot %s (%s)", a.config.Name, a.config.Model),
    }, nil
}

func (a *PhysicalRobotAgent) Type() nim.AgentType { return nim.AgentTypeRobot }
func (a *PhysicalRobotAgent) Available(ctx context.Context) bool {
    // Ping robot to check availability
    resp, err := a.client.Get(a.config.Endpoint + "/status")
    if err != nil {
        return false
    }
    defer resp.Body.Close()
    return resp.StatusCode == 200
}
func (a *PhysicalRobotAgent) Model() string { return a.config.Model }
func (a *PhysicalRobotAgent) Location() string { return a.config.Location }
func (a *PhysicalRobotAgent) Capabilities() []string { return a.config.Capabilities }
```

### 5.5 Browser Agent Implementation

```go
// internal/ai/agents/browser_agent.go

package agents

import (
    "context"
    "fmt"
    "os/exec"
    
    "github.com/yourusername/nimsforest/pkg/nim"
)

type PlaywrightBrowserAgent struct {
    config nim.BrowserAgentConfig
}

func NewPlaywrightBrowserAgent(config nim.BrowserAgentConfig) *PlaywrightBrowserAgent {
    return &PlaywrightBrowserAgent{config: config}
}

func (a *PlaywrightBrowserAgent) Run(ctx context.Context, task nim.Task) (*nim.Result, error) {
    headlessFlag := ""
    if a.config.Headless {
        headlessFlag = "--headless"
    }
    
    args := []string{
        "run", "--rm",
        "-v", fmt.Sprintf("%s:/workspace", task.Params["workdir"]),
        a.config.Image,
        a.config.Browser,
        headlessFlag,
        task.Description,
    }
    
    cmd := exec.CommandContext(ctx, "docker", args...)
    output, err := cmd.CombinedOutput()
    
    if err != nil {
        return &nim.Result{
            Success: false,
            Error:   err.Error(),
            Output:  string(output),
        }, nil
    }
    
    return &nim.Result{
        Success: true,
        Output:  string(output),
    }, nil
}

func (a *PlaywrightBrowserAgent) Type() nim.AgentType { return nim.AgentTypeBrowser }
func (a *PlaywrightBrowserAgent) Available(ctx context.Context) bool { return true }
func (a *PlaywrightBrowserAgent) Browser() string { return a.config.Browser }
func (a *PlaywrightBrowserAgent) Headless() bool { return a.config.Headless }
```

---

## Part 6: CoderNim

### Implementation

```go
// internal/nims/coder/coder.go

package coder

import (
    "context"
    "fmt"
    "time"
    
    "github.com/yourusername/nimsforest/internal/core"
    "github.com/yourusername/nimsforest/pkg/nim"
    "github.com/yourusername/nimsforest/pkg/runtime"
)

type CoderNim struct {
    *core.BaseNim
    asker  nim.AIAsker
    wind   nim.Whisperer
    forest *runtime.Forest
}

func New(base *core.BaseNim, asker nim.AIAsker, wind nim.Whisperer, forest *runtime.Forest) *CoderNim {
    return &CoderNim{
        BaseNim: base,
        asker:   asker,
        wind:    wind,
        forest:  forest,
    }
}

func (c *CoderNim) Subjects() []string {
    return []string{"code.request", "code.>"}
}

// Advice - simple Q&A via aiservicefactory
func (c *CoderNim) Advice(ctx context.Context, query string) (string, error) {
    return c.asker.Ask(ctx, query)
}

// Action - execute via Agent (AI/Human/Robot/Browser)
func (c *CoderNim) Action(ctx context.Context, action string, params map[string]interface{}) (interface{}, error) {
    task := nim.Task{
        ID:          fmt.Sprintf("task-%d", time.Now().UnixNano()),
        Description: buildTaskDescription(action, params),
        Params:      params,
    }
    
    // Determine required agent type
    agentType := determineAgentType(action)
    needsGPU := params["gpu"] == true
    
    // Request capacity via event
    c.wind.Whisper(ctx, &Leaf{
        Subject: "land.capacity.request",
        Data: CapacityRequest{TaskID: task.ID, NeedsGPU: needsGPU, AgentType: agentType},
    })
    
    // Collect responses
    responses := c.collectResponses(ctx, "land.capacity.response", task.ID, 2*time.Second)
    if len(responses) == 0 {
        return nil, fmt.Errorf("no land capacity available")
    }
    
    // Reserve first available
    land := responses[0]
    c.wind.Whisper(ctx, &Leaf{
        Subject: "land.reserve",
        Data: ReserveRequest{TaskID: task.ID, LandID: land.LandID},
    })
    
    // Dispatch to agent (result comes back as event)
    c.wind.Whisper(ctx, &Leaf{
        Subject: fmt.Sprintf("agent.execute.%s", land.LandID),
        Data: task,
    })
    
    return &nim.Result{
        Success: true,
        Output:  fmt.Sprintf("Task %s dispatched to %s", task.ID, land.LandID),
    }, nil
}

// automationAnalysis holds the AI's analysis of what type of automation is needed
type automationAnalysis struct {
    Type       string `json:"type"`
    Reason     string `json:"reason"`
    Subscribes string `json:"subscribes"`
    Publishes  string `json:"publishes"`
}

// Automate - create TreeHouse or Nim based on complexity
func (c *CoderNim) Automate(ctx context.Context, automation string, enabled bool) (*nim.AutomateResult, error) {
    if !enabled {
        // Disable existing automation
        return c.disableAutomation(ctx, automation)
    }
    
    // Analyze what type of automation is needed
    analysisPrompt := fmt.Sprintf(`Analyze this automation request: "%s"

Determine if this requires:
1. TreeHouse (Lua) - for simple rule-based event processing
2. Nim (config) - for complex logic requiring AI reasoning

Respond with JSON only:
{"type": "treehouse" or "nim", "reason": "why", "subscribes": "pattern", "publishes": "pattern"}`, automation)
    
    analysisJSON, err := c.asker.Ask(ctx, analysisPrompt)
    if err != nil {
        return nil, err
    }
    
    var analysis automationAnalysis
    if err := json.Unmarshal([]byte(analysisJSON), &analysis); err != nil {
        return nil, fmt.Errorf("failed to parse analysis: %w", err)
    }
    
    if analysis.Type == "treehouse" {
        return c.createTreeHouseAutomation(ctx, automation, analysis)
    }
    return c.createNimAutomation(ctx, automation, analysis)
}

func (c *CoderNim) createTreeHouseAutomation(ctx context.Context, name string, analysis automationAnalysis) (*nim.AutomateResult, error) {
    // Generate Lua script
    luaPrompt := fmt.Sprintf(`Generate a Lua TreeHouse script for: %s

The script should:
- Subscribe to: %s
- Publish to: %s
- Reason: %s

Return only valid Lua code with a process(leaf) function.`, name, analysis.Subscribes, analysis.Publishes, analysis.Reason)
    
    luaCode, err := c.asker.Ask(ctx, luaPrompt)
    if err != nil {
        return nil, err
    }
    
    // Save script
    scriptPath := fmt.Sprintf("scripts/treehouses/%s.lua", name)
    if err := os.WriteFile(scriptPath, []byte(luaCode), 0644); err != nil {
        return nil, err
    }
    
    // Add TreeHouse to forest
    config := runtime.TreeHouseConfig{
        Name:       name,
        Subscribes: analysis.Subscribes,
        Publishes:  analysis.Publishes,
        Script:     scriptPath,
    }
    
    if err := c.forest.AddTreeHouse(name, config); err != nil {
        return nil, err
    }
    
    return &nim.AutomateResult{
        Created:    "treehouse",
        Name:       name,
        Reason:     analysis.Reason,
        ScriptPath: scriptPath,
    }, nil
}

func (c *CoderNim) createNimAutomation(ctx context.Context, name string, analysis automationAnalysis) (*nim.AutomateResult, error) {
    // Generate Nim config/prompt
    promptContent := fmt.Sprintf(`# %s

## Purpose
%s

## Subscribes
%s

## Publishes
%s
`, name, analysis.Reason, analysis.Subscribes, analysis.Publishes)
    
    // Save prompt
    promptPath := fmt.Sprintf("scripts/nims/%s.md", name)
    if err := os.WriteFile(promptPath, []byte(promptContent), 0644); err != nil {
        return nil, err
    }
    
    // Add Nim to forest via config
    config := runtime.NimConfig{
        Name:       name,
        Subscribes: analysis.Subscribes,
        Publishes:  analysis.Publishes,
        Prompt:     promptPath,
    }
    
    if err := c.forest.AddNim(name, config); err != nil {
        return nil, err
    }
    
    return &nim.AutomateResult{
        Created:     "nim",
        Name:        name,
        Reason:      analysis.Reason,
        ScriptPath:  promptPath,
        NeedsReview: true, // AI-generated Nims should be reviewed
    }, nil
}

func determineAgentType(action string) nim.AgentType {
    switch action {
    case "approve", "review", "decide":
        return nim.AgentTypeHuman
    case "navigate", "pick", "place", "speak":
        return nim.AgentTypeRobot
    case "scrape", "fill_form", "test_ui":
        return nim.AgentTypeBrowser
    default:
        return nim.AgentTypeAI
    }
}
```

---

## Part 7: Configuration

### forest.yaml

```yaml
# Agent configuration
# NOTE: AI and Browser agents run on Nimland or Manaland (Docker required)
#       GPU workloads require Manaland
#       Human and Robot agents don't require Docker
agents:
  ai:
    claude-coder:
      image: nimsforest/claude-agent:latest
      tools: [claude]
      memory: "4g"
      cpu: 2
      
    aider-coder:
      image: nimsforest/aider-agent:latest
      tools: [aider]
      
  human:
    pr-approvers:
      role: approver
      responsibility: "Approve pull requests before merge"
      members:
        - name: Alice
          platform: telegram
          contact: "@alice_dev"
        - name: Bob
          platform: slack
          contact: "U12345"
        - name: Charlie
          platform: email
          contact: "charlie@company.com"
          
    security-reviewers:
      role: reviewer
      responsibility: "Review security-sensitive changes"
      members:
        - name: Dave
          platform: telegram
          contact: "@dave_security"
          
  robot:
    office-temi:
      model: temi
      location: "office-floor-1"
      endpoint: "http://192.168.1.50:8080"
      capabilities: [navigate, speak, video_call]
      
    warehouse-arm:
      model: so-arm100
      location: "warehouse-a"
      endpoint: "http://192.168.1.60:8080"
      capabilities: [pick, place, inspect]
      
  browser:
    web-scraper:
      role: scraper
      responsibility: "Extract data from web pages"
      image: nimsforest/browser-agent:playwright
      browser: chromium
      headless: true
      
    ui-tester:
      role: tester
      responsibility: "Run UI tests"
      image: nimsforest/browser-agent:playwright
      browser: chromium
      headless: true

# Nims
nims:
  coder:
    subscribes: code.request
    publishes: code.result
    prompt: scripts/nims/coder.md

# Songbirds
songbirds:
  telegram:
    type: telegram
    listens: song.telegram.>
    bot_token: ${TELEGRAM_BOT_TOKEN}
    
  slack:
    type: slack
    listens: song.slack.>
    bot_token: ${SLACK_BOT_TOKEN}
    
  email:
    type: email
    listens: song.email.>
    smtp_host: smtp.example.com
    smtp_user: ${SMTP_USER}
    smtp_pass: ${SMTP_PASS}
```

---

## Part 8: Task Breakdown

### Phase 1: Land & Houses (Foundation - Do First)

Land is data detected at startup. LandHouse and AgentHouse are Go TreeHouses.

| Task | Description |
|------|-------------|
| 1.1 | Create `internal/core/land.go` - LandInfo struct, LandType constants |
| 1.2 | Create `internal/land/detect.go` - Detect RAM, CPU, Docker, GPU |
| 1.3 | Create `internal/treehouses/landhouse.go` - Responds to `land.query` |
| 1.4 | Create `internal/treehouses/agenthouse.go` - Handles `agent.task.*`, runs Docker |
| 1.5 | Update `pkg/runtime/forest.go` - Detect Land, wire up Houses |
| 1.6 | Update `internal/viewmodel/` - Subscribe to `land.info.>` Leaves |
| 1.7 | Tests for Land detection and Houses |

### Phase 2: pkg/nim/ Interfaces

| Task | Description |
|------|-------------|
| 2.1 | Create `pkg/nim/nim.go` - Nim interface with AAA |
| 2.2 | Create `pkg/nim/brain.go` - Move from pkg/brain |
| 2.3 | Create `pkg/nim/leaf.go` - Leaf interface |
| 2.4 | Create `pkg/nim/wind.go` - Whisperer interface |
| 2.5 | Create `pkg/nim/asker.go` - AIAsker interface |
| 2.6 | Create `pkg/nim/agent.go` - Agent interface |
| 2.7 | Create `pkg/nim/ai_agent.go` - AIAgent interface |
| 2.8 | Create `pkg/nim/human_agent.go` - HumanAgent interface |
| 2.9 | Create `pkg/nim/robot_agent.go` - RobotAgent interface |
| 2.10 | Create `pkg/nim/browser_agent.go` - BrowserAgent interface |

### Phase 3: Agent Implementations

| Task | Description |
|------|-------------|
| 3.1 | Create `internal/ai/asker.go` - Wrap existing aiservice |
| 3.2 | Create `internal/ai/agents/ai_agent.go` - Docker AI agent |
| 3.3 | Create `internal/ai/agents/human_agent.go` - Songbird human agent |
| 3.4 | Create `internal/ai/agents/robot_agent.go` - Physical robot agent |
| 3.5 | Create `internal/ai/agents/browser_agent.go` - Playwright browser agent |

### Phase 4: Songbirds

| Task | Description |
|------|-------------|
| 4.1 | Update `internal/songbirds/songbird.go` - Add Send + Message type |
| 4.2 | Update `internal/songbirds/telegram.go` - Implement Send, emit response Leaves |
| 4.3 | Create `internal/songbirds/slack.go` - Slack songbird |
| 4.4 | Create `internal/songbirds/email.go` - Email songbird |

### Phase 5: Core Updates

| Task | Description |
|------|-------------|
| 5.1 | Update `internal/core/nim.go` - BaseNim with AAA |
| 5.2 | Update `internal/core/leaf.go` - Implement nim.Leaf |
| 5.3 | Update `internal/core/wind.go` - Implement nim.Whisperer |

### Phase 6: CoderNim

| Task | Description |
|------|-------------|
| 6.1 | Create `internal/nims/coder/coder.go` - Full implementation |
| 6.2 | Create `internal/nims/coder/coder_test.go` - Tests |

### Phase 7: Configuration

| Task | Description |
|------|-------------|
| 7.1 | Update `pkg/runtime/config.go` - Add agent configs |
| 7.2 | Update `pkg/runtime/forest.go` - Load agents, integrate Land |
| 7.3 | Update `cmd/forest/main.go` - Wire up CoderNim |

### Phase 8: Cleanup & Examples

| Task | Description |
|------|-------------|
| 8.1 | Move `pkg/brain/` → `pkg/nim/`, update imports |
| 8.2 | Create `examples/` and move example code from `internal/` |
| 8.3 | Update `cmd/forest/main.go` to not auto-load examples |

### Phase 9: Testing

| Task | Description |
|------|-------------|
| 9.1 | Test pkg/nim interfaces |
| 9.2 | Test agent implementations |
| 9.3 | Test CoderNim AAA methods |
| 9.4 | Test Land auto-discovery |
| 9.5 | Integration tests |

---

## Summary

### Core Systems

NimsForest has five core systems that work together:

```
┌─────────────────────────────────────────────────────────────────────┐
│                            FOREST                                   │
│                                                                     │
│   ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐ │
│   │  WIND   │  │  RIVER  │  │   NIM   │  │TREEHOUSE│  │  LAND   │ │
│   │(pub/sub)│  │ (data)  │  │  (AAA)  │  │ (Lua)   │  │(compute)│ │
│   └─────────┘  └─────────┘  └─────────┘  └─────────┘  └─────────┘ │
│                                                                     │
│   Wind: Message passing (NATS pub/sub)                             │
│   River: External data ingestion (NATS JetStream)                  │
│   Nim: Intelligent agents with AAA pattern                         │
│   TreeHouse: Deterministic Lua processors                          │
│   Land: Compute substrate (auto-discovered, event-driven)          │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Land: Data + Houses

Land is **data** detected at startup. Communication via **Go TreeHouses** (deterministic).

| Component | Type | Purpose |
|-----------|------|---------|
| `thisLand` | LandInfo struct | Detected capabilities |
| `LandHouse` | Go TreeHouse | Responds to `land.query` |
| `AgentHouse` | Go TreeHouse | Runs agents in Docker |

```
NimsForest starts
      │
      ├──► natsembed.New() ──► Get server.ID, server.Name
      │
      ├──► land.Detect() ──► RAM, CPU, Docker?, GPU? → LandInfo struct
      │
      ├──► forest.thisLand = landInfo
      │
      └──► forest.Start()
            ├──► LandHouse subscribes to land.query
            └──► AgentHouse subscribes to agent.task.{id} (if Docker)
```

### Land Types

| Type | Docker | GPU | Houses |
|------|--------|-----|--------|
| **Land** | ❌ | ❌ | LandHouse only |
| **Nimland** | ✅ | ❌ | LandHouse + AgentHouse |
| **Manaland** | ✅ | ✅ | LandHouse + AgentHouse |

### Agent Types

| Type | Runs On | Use Case |
|------|---------|----------|
| AIAgent | Nimland/Manaland | Code tasks (Claude, Aider) |
| BrowserAgent | Nimland/Manaland | Web automation (Playwright) |
| HumanAgent | Songbird (async) | Approvals, reviews |
| RobotAgent | Physical robot | Temi, SO-ARM100, humanoids |

### AAA Methods (Nim Intelligence)

| Method | Implementation |
|--------|----------------|
| **Advice** | AI query via existing `pkg/integrations/aiservice/` |
| **Action** | Dispatch to Agent on appropriate Land |
| **Automate** | Generate TreeHouse (Lua) or Nim (config) |

### Changes Overview

| Action | Items |
|--------|-------|
| **Create** | `internal/core/land.go`, `internal/land/detect.go`, `internal/treehouses/landhouse.go`, `internal/treehouses/agenthouse.go`, `pkg/nim/`, `internal/nims/coder/`, `examples/` |
| **Update** | `pkg/runtime/forest.go` (detect Land, wire Houses), `internal/viewmodel/` (subscribe to `land.info.>`), `internal/songbirds/` (Send) |
| **Move** | `pkg/brain/` → `pkg/nim/`, example code → `examples/` |

### Directory Structure

```
internal/
├── core/
│   ├── land.go           # LandInfo struct, LandType constants
│   ├── nim.go            # Nim with AAA
│   ├── wind.go, river.go, leaf.go, ...
│
├── land/
│   └── detect.go         # Auto-detect RAM, CPU, Docker, GPU
│
├── treehouses/
│   ├── landhouse.go     # Go TreeHouse: land.query → land.info.*
│   └── agenthouse.go    # Go TreeHouse: agent.task.* → agent.result.*
│
├── viewmodel/            # Subscribes to land.info.> Leaves
│   └── ...
│
└── nims/
    └── coder/            # CoderNim (core AAA infrastructure)

pkg/
├── nim/
│   └── ...               # Public Nim interfaces
└── runtime/
    └── forest.go         # Detects thisLand, starts Houses
```

---

## Part 9: Land as Data in Forest

Land is the compute substrate - it **exists implicitly** when NimsForest starts. Rather than being a separate component with its own lifecycle, Land is **data that Forest knows about itself**.

### Design Principles

1. **Land is data** - Detected at startup, stored as `LandInfo` struct in Forest
2. **All events are Leaves** - No special "land events", just Leaves on the Wind
3. **Houses handle communication** - Go TreeHouses (deterministic) handle Land queries and agent execution

| Component | What It Is | Responsibility |
|-----------|------------|----------------|
| `Forest.thisLand` | LandInfo struct | Hold detected capabilities |
| `LandHouse` | Go TreeHouse | Respond to `land.query` Leaves |
| `AgentHouse` | Go TreeHouse | Execute `agent.task.*` Leaves in Docker |

### Land Info Struct

```go
// internal/core/land.go

package core

// LandType identifies the capabilities of this node.
type LandType string

const (
    LandTypeBase     LandType = "land"     // No Docker, backbone only
    LandTypeNimland  LandType = "nimland"  // Docker-capable
    LandTypeManaland LandType = "manaland" // Docker + GPU
)

// LandInfo holds information about a compute node.
// This is detected at startup and stored in Forest.
type LandInfo struct {
    ID        string   `json:"id"`         // From NATS server ID
    Name      string   `json:"name"`       // From NATS server name (config)
    Type      LandType `json:"type"`       // Detected: land/nimland/manaland
    Hostname  string   `json:"hostname"`   // OS hostname
    
    // Capacity (detected from system)
    RAMTotal   uint64  `json:"ram_total"`    // Bytes
    CPUCores   int     `json:"cpu_cores"`
    CPUModel   string  `json:"cpu_model"`
    CPUFreqMHz float64 `json:"cpu_freq_mhz"`
    
    // Capabilities (probed)
    HasDocker bool `json:"has_docker"`
    
    // GPU (if available)
    GPUVendor string  `json:"gpu_vendor,omitempty"` // "nvidia", "amd"
    GPUModel  string  `json:"gpu_model,omitempty"`
    GPUVram   uint64  `json:"gpu_vram,omitempty"`   // Bytes
    GPUTflops float64 `json:"gpu_tflops,omitempty"`
}

// Leaf subject for Land announcements
const SubjectLandInfo = "land.info"
```

### Detection at Startup

```go
// internal/land/detect.go

package land

import (
    "os"
    "os/exec"
    "runtime"
    
    "github.com/shirou/gopsutil/v3/cpu"
    "github.com/shirou/gopsutil/v3/mem"
    "github.com/yourusername/nimsforest/internal/core"
)

// Detect probes the local system and returns LandInfo.
// Called once during Forest startup.
func Detect(natsID, natsName string) *core.LandInfo {
    info := &core.LandInfo{
        ID:   natsID,
        Name: natsName,
    }
    
    // Hostname
    info.Hostname, _ = os.Hostname()
    
    // RAM
    if vmStat, err := mem.VirtualMemory(); err == nil {
        info.RAMTotal = vmStat.Total
    }
    
    // CPU
    info.CPUCores = runtime.NumCPU()
    if cpuInfo, err := cpu.Info(); err == nil && len(cpuInfo) > 0 {
        info.CPUModel = cpuInfo[0].ModelName
        info.CPUFreqMHz = cpuInfo[0].Mhz
    }
    
    // Docker
    info.HasDocker = detectDocker()
    
    // GPU
    detectGPU(info)
    
    // Determine type
    info.Type = determineType(info)
    
    return info
}

func detectDocker() bool {
    if _, err := exec.LookPath("docker"); err != nil {
        return false
    }
    cmd := exec.Command("docker", "info")
    return cmd.Run() == nil
}

func detectGPU(info *core.LandInfo) {
    // Try nvidia-smi
    cmd := exec.Command("nvidia-smi", 
        "--query-gpu=name,memory.total",
        "--format=csv,noheader,nounits")
    if output, err := cmd.Output(); err == nil {
        // Parse output: "NVIDIA GeForce RTX 4090, 24564"
        info.GPUVendor = "nvidia"
        // ... parse info.GPUModel, info.GPUVram
    }
}

func determineType(info *core.LandInfo) core.LandType {
    if info.GPUVram > 0 && info.HasDocker {
        return core.LandTypeManaland
    }
    if info.HasDocker {
        return core.LandTypeNimland
    }
    return core.LandTypeBase
}
```

### Integration in Forest Startup

```go
// pkg/runtime/forest.go

type Forest struct {
    // ... existing fields ...
    
    thisLand   *core.LandInfo            // This node's Land info
    landHouse  *treehouses.LandHouse     // Handles land.query
    agentHouse *treehouses.AgentHouse    // Handles agent.task.* (if Docker)
}

// NewForest creates a Forest - Land is detected automatically.
func NewForest(configPath string, natsServer *natsembed.Server, wind *core.Wind, b brain.Brain) (*Forest, error) {
    cfg, err := LoadConfig(configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to load config: %w", err)
    }
    
    // Detect this Land (using NATS server identity)
    varz, _ := natsServer.InternalServer().Varz(&server.VarzOptions{})
    thisLand := land.Detect(varz.ID, varz.Name)
    
    log.Printf("[Forest] This Land: %s (%s) - RAM: %s, CPU: %d cores, Docker: %v",
        thisLand.Name, thisLand.Type, 
        formatBytes(thisLand.RAMTotal), thisLand.CPUCores, thisLand.HasDocker)
    
    f := &Forest{
        config:   cfg,
        wind:     wind,
        brain:    b,
        thisLand: thisLand,
        // ... rest of init
    }
    
    return f, nil
}

// Start wires up Houses and starts all components.
func (f *Forest) Start(ctx context.Context) error {
    // ... existing startup ...
    
    // Create and start LandHouse (all nodes)
    f.landHouse = treehouses.NewLandHouse(f.thisLand, f.wind)
    if err := f.startHouse(ctx, f.landHouse); err != nil {
        return fmt.Errorf("failed to start LandHouse: %w", err)
    }
    
    // Create and start AgentHouse (only Nimland/Manaland)
    if f.thisLand.HasDocker {
        f.agentHouse = treehouses.NewAgentHouse(f.thisLand.ID, f.wind)
        if err := f.startHouse(ctx, f.agentHouse); err != nil {
            return fmt.Errorf("failed to start AgentHouse: %w", err)
        }
    }
    
    // ... rest of startup ...
}

// startHouse subscribes a Go TreeHouse to its subjects.
func (f *Forest) startHouse(ctx context.Context, house GoTreeHouse) error {
    for _, subject := range house.Subjects() {
        _, err := f.wind.Catch(subject, func(leaf core.Leaf) {
            if result := house.Process(leaf); result != nil {
                f.wind.Drop(*result)
            }
        })
        if err != nil {
            return err
        }
    }
    log.Printf("[Forest] Started %s on subjects: %v", house.Name(), house.Subjects())
    return nil
}

// ThisLand returns this node's Land info.
func (f *Forest) ThisLand() *core.LandInfo {
    return f.thisLand
}
```

### GoTreeHouse Interface

```go
// internal/treehouses/interface.go

// GoTreeHouse is a compile-time TreeHouse implemented in Go.
// Unlike Lua TreeHouses, these have access to system resources.
// Must be deterministic: same input Leaf = same output Leaf.
type GoTreeHouse interface {
    Name() string
    Subjects() []string
    Process(leaf core.Leaf) *core.Leaf  // nil = no output
}
```

### Persistence with JetStream (Optional)

For new nodes to discover existing Lands, we can use a JetStream stream:

```go
// Create a LAND stream that retains the latest info per Land
streamConfig := &nats.StreamConfig{
    Name:              "LAND",
    Subjects:          []string{"land.info.>"},
    MaxMsgsPerSubject: 1,  // Keep only latest per Land
    Storage:           nats.FileStorage,
}
```

New nodes read existing Lands from the stream on startup.

### ViewWorld Subscribes to Land Leaves

```go
// internal/viewmodel/viewmodel.go

func New(ns *server.Server, wind *core.Wind) *ViewModel {
    vm := &ViewModel{
        territory: NewWorld(),
    }
    
    // Subscribe to Land info Leaves
    wind.Catch("land.info.>", func(leaf core.Leaf) {
        var info core.LandInfo
        json.Unmarshal(leaf.Data, &info)
        
        land := vm.landInfoToViewModel(&info)
        vm.territory.AddLand(land)
    })
    
    return vm
}
```

### Capacity Queries via Request/Reply

When a Nim needs to find capacity, it uses NATS request/reply:

```go
// Nim wants to find a Nimland with capacity
func (n *CoderNim) findCapacity(ctx context.Context, needsGPU bool) (*core.LandInfo, error) {
    query := CapacityQuery{
        NeedsDocker: true,
        NeedsGPU:    needsGPU,
        MinRAM:      4 * 1024 * 1024 * 1024, // 4GB
    }
    
    queryData, _ := json.Marshal(query)
    
    // Request/reply - all Lands that can help will respond
    msg, err := n.natsConn.Request("land.query", queryData, 2*time.Second)
    if err != nil {
        return nil, err
    }
    
    var info core.LandInfo
    json.Unmarshal(msg.Data, &info)
    return &info, nil
}
```

Each Forest listens on `land.query` and responds if it can help:

```go
// In Forest.Start()
f.wind.Catch("land.query", func(leaf core.Leaf) {
    var query CapacityQuery
    json.Unmarshal(leaf.Data, &query)
    
    if f.canFulfill(query) {
        // Reply with this Land's info
        f.replyWithLandInfo(leaf.ReplyTo)
    }
})
```

### Summary: Land is Simple Data

```
┌─────────────────────────────────────────────────────────────────┐
│                     Forest Startup                              │
├─────────────────────────────────────────────────────────────────┤
│  1. natsembed.New()                                             │
│     └── NATS server starts                                      │
│     └── Get: server.ID, server.Name                            │
│                                                                 │
│  2. land.Detect(id, name)                                       │
│     └── gopsutil: RAM, CPU                                      │
│     └── exec: docker info, nvidia-smi                          │
│     └── Returns: LandInfo struct                               │
│                                                                 │
│  3. forest.thisLand = landInfo                                  │
│     └── Stored in Forest struct                                │
│                                                                 │
│  4. forest.Start()                                              │
│     └── Drop Leaf("land.info.{id}", thisLand)                  │
│     └── Subscribe to "land.query" for capacity requests        │
│                                                                 │
│  5. ViewWorld catches "land.info.>" Leaves                      │
│     └── Builds World from Land announcements                   │
└─────────────────────────────────────────────────────────────────┘
```

**Key Points:**
- Land is a **struct**, not a component with lifecycle
- Detection happens **once at startup**, next to NATS init
- Announcement is a **normal Leaf** on `land.info.{id}`
- Capacity queries use **NATS request/reply** 
- ViewWorld **subscribes to Leaves** instead of polling NATS APIs
- No special "land events" - just Leaves with `land.*` subjects

---

## Part 10: Land Detection Details

Land **already exists** the moment NimsForest starts - we're running on it. The question is: how do we discover what this Land is capable of?

### What NATS Server Already Knows

The embedded NATS server provides rich information via its monitoring APIs:

```go
// From server.Varz()
varz.ID         // Unique server ID (e.g., "NCXXX...")
varz.Name       // Server name (from config.NodeName)
varz.Host       // Host address
varz.Port       // Client port
varz.Start      // When server started
varz.Cluster    // Cluster info (name, port, routes)

// From server.Routez()
routez.Routes   // Connected peer nodes (ID, IP, RTT)

// From server.Subsz()
subsz.Subs      // All subscriptions (subject, queue, msgs)

// From server.Connz()
connz.Conns     // All connections with their subscriptions

// From server.Jsz()
jsz.AccountDetails.Streams    // JetStream streams
jsz.AccountDetails.Consumers  // JetStream consumers
```

### What We Detect from System

Already using `gopsutil` in `viewmodel/reader.go`:

```go
// RAM
vmStat, _ := mem.VirtualMemory()
vmStat.Total    // Total RAM in bytes

// CPU
runtime.NumCPU() // Number of CPU cores
```

### What We Could Add: Docker & GPU Detection

```go
// internal/land/detect.go

package land

import (
    "os/exec"
    "runtime"
    "strings"
    
    "github.com/shirou/gopsutil/v3/cpu"
    "github.com/shirou/gopsutil/v3/mem"
)

// DetectLocalLand auto-detects the current machine's capabilities.
func DetectLocalLand() (*LocalLandInfo, error) {
    info := &LocalLandInfo{}
    
    // RAM
    if vmStat, err := mem.VirtualMemory(); err == nil {
        info.RAMTotal = vmStat.Total
    }
    
    // CPU
    info.CPUCores = runtime.NumCPU()
    if cpuInfo, err := cpu.Info(); err == nil && len(cpuInfo) > 0 {
        info.CPUModel = cpuInfo[0].ModelName
        info.CPUFreqMHz = cpuInfo[0].Mhz
    }
    
    // Docker detection
    info.HasDocker = detectDocker()
    
    // GPU detection
    info.GPU = detectGPU()
    
    return info, nil
}

// detectDocker checks if Docker is available and running.
func detectDocker() bool {
    // Check if docker command exists
    if _, err := exec.LookPath("docker"); err != nil {
        return false
    }
    
    // Check if docker daemon is running
    cmd := exec.Command("docker", "info")
    if err := cmd.Run(); err != nil {
        return false
    }
    
    return true
}

// GPUInfo holds GPU detection results.
type GPUInfo struct {
    Available bool
    Vendor    string  // "nvidia", "amd", "intel"
    Model     string  // "RTX 4090", "RX 7900 XTX"
    VRAM      uint64  // Bytes
    Tflops    float64 // Compute power
}

// detectGPU attempts to detect GPU information.
func detectGPU() *GPUInfo {
    // Try nvidia-smi first (most common for compute)
    if gpu := detectNvidiaGPU(); gpu != nil {
        return gpu
    }
    
    // Try AMD ROCm
    if gpu := detectAMDGPU(); gpu != nil {
        return gpu
    }
    
    return nil
}

// detectNvidiaGPU uses nvidia-smi to detect NVIDIA GPUs.
func detectNvidiaGPU() *GPUInfo {
    cmd := exec.Command("nvidia-smi", 
        "--query-gpu=name,memory.total,compute_cap",
        "--format=csv,noheader,nounits")
    
    output, err := cmd.Output()
    if err != nil {
        return nil
    }
    
    // Parse: "NVIDIA GeForce RTX 4090, 24564, 8.9"
    parts := strings.Split(strings.TrimSpace(string(output)), ", ")
    if len(parts) < 2 {
        return nil
    }
    
    gpu := &GPUInfo{
        Available: true,
        Vendor:    "nvidia",
        Model:     parts[0],
    }
    
    // Parse VRAM (nvidia-smi reports in MiB)
    if vramMiB, err := strconv.ParseUint(parts[1], 10, 64); err == nil {
        gpu.VRAM = vramMiB * 1024 * 1024
    }
    
    return gpu
}

// LocalLandInfo holds auto-detected land information.
type LocalLandInfo struct {
    RAMTotal   uint64
    CPUCores   int
    CPUModel   string
    CPUFreqMHz float64
    HasDocker  bool
    GPU        *GPUInfo
}

// LandType returns the detected land type.
func (l *LocalLandInfo) LandType() LandType {
    if l.GPU != nil && l.GPU.Available && l.HasDocker {
        return LandTypeManaland
    }
    if l.HasDocker {
        return LandTypeNimland
    }
    return LandTypeBase
}
```

### Bootstrap Flow

When NimsForest starts, Land bootstraps automatically:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         NimsForest Startup                              │
└─────────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  1. Embedded NATS Server Starts                                         │
│     └── server.Varz() → ID, Name, Host, Port                           │
│     └── server.Routez() → Peer nodes (if clustered)                    │
└─────────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  2. System Detection (gopsutil + probing)                               │
│     └── mem.VirtualMemory() → RAM                                       │
│     └── runtime.NumCPU() → CPU cores                                    │
│     └── exec.LookPath("docker") → Docker available?                     │
│     └── nvidia-smi / rocm-smi → GPU available?                          │
└─────────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  3. Create "This Land" (core.Land)                                      │
│     └── ID from NATS server ID                                          │
│     └── Type from detection (Land/Nimland/Manaland)                     │
│     └── Capacity from system info                                       │
└─────────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  4. Land Starts & Announces via Wind                                    │
│     └── Whisper("land.announce", {id, type, capacity})                  │
│     └── Subscribe to "land.capacity.query"                              │
│     └── Subscribe to "land.reserve"                                     │
│     └── Start heartbeat loop                                            │
└─────────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  5. World/ViewModel Listens to Land Events                              │
│     └── Catch("land.announce") → AddLand to World                       │
│     └── Catch("land.heartbeat") → UpdateLand                            │
│     └── When NATS route disconnects → RemoveLand                        │
└─────────────────────────────────────────────────────────────────────────┘
```

### Integration Code

```go
// pkg/runtime/forest.go (updated)

func NewForest(configPath string, wind *core.Wind, b brain.Brain) (*Forest, error) {
    // ... existing code ...
    
    // Auto-detect and create "this land"
    localLandInfo, err := land.DetectLocalLand()
    if err != nil {
        log.Printf("[Forest] Warning: land detection failed: %v", err)
    }
    
    // Create the Land for this node
    thisLand := land.NewFromDetection(
        f.natsServer.ID(),       // Use NATS server ID
        f.natsServer.Name(),     // Use configured node name
        localLandInfo,
        wind,
    )
    
    f.thisLand = thisLand
    
    return f, nil
}

func (f *Forest) Start(ctx context.Context) error {
    // ... existing startup code ...
    
    // Start this Land (announces to cluster, listens for queries)
    if f.thisLand != nil {
        if err := f.thisLand.Start(ctx); err != nil {
            return fmt.Errorf("failed to start land: %w", err)
        }
    }
    
    // ... rest of startup ...
}
```

### What NATS Peers Know About Each Other

When NATS nodes cluster, they know:
- **Route info**: Remote ID, IP address, RTT (latency)
- **But NOT**: RAM, CPU, GPU of the remote node

This is why Lands need to **announce themselves via Wind**:

```
Node A starts:
  └── Detects: 16GB RAM, 4 CPU, no Docker, no GPU
  └── Creates: BaseLand
  └── Whispers: land.announce {id: "A", type: "land", ram: 16GB, ...}

Node B starts:
  └── Detects: 32GB RAM, 8 CPU, Docker ✓, GPU ✓
  └── Creates: Manaland
  └── Whispers: land.announce {id: "B", type: "manaland", ram: 32GB, gpu_vram: 24GB, ...}
  └── Catches: land.announce from A
  └── Now knows: "Node A is a BaseLand with 16GB"

Node A catches: land.announce from B
  └── Now knows: "Node B is a Manaland with GPU"
```

### ViewWorld Becomes Event-Driven

The existing `viewmodel.World` becomes a consumer of Land events instead of polling:

```go
// internal/viewmodel/viewmodel.go (updated)

func New(ns *server.Server, wind *core.Wind) *ViewModel {
    vm := &ViewModel{
        server:    ns,
        territory: NewWorld(),
        // ...
    }
    
    // Subscribe to Land events instead of polling
    wind.Catch("land.announce", func(leaf core.Leaf) {
        var ann core.LandAnnounce
        json.Unmarshal(leaf.Data, &ann)
        
        land := vm.announcementToLand(ann)
        vm.territory.AddLand(land)
    })
    
    wind.Catch("land.heartbeat", func(leaf core.Leaf) {
        var hb core.LandHeartbeat
        json.Unmarshal(leaf.Data, &hb)
        
        if land := vm.territory.GetLand(hb.LandID); land != nil {
            land.LastSeen = time.Now()
            // Update available capacity
        }
    })
    
    // Detect when NATS routes drop (node left cluster)
    // This would come from NATS server events
    
    return vm
}
```

### Summary: Land Bootstrap

| Source | Information |
|--------|-------------|
| **NATS Server** | ID, Name, Cluster peers, Subscriptions |
| **gopsutil** | RAM total, CPU cores, CPU frequency |
| **Docker probe** | Docker available? |
| **nvidia-smi** | GPU vendor, model, VRAM |
| **Wind events** | Other Lands' capabilities |

The flow is:
1. **Implicit**: Land exists because we're running on it
2. **Detection**: Probe system for capabilities  
3. **NATS identity**: Use server ID as Land ID
4. **Announce**: Tell the cluster what we are
5. **Listen**: Learn about other Lands via their announcements
6. **Respond**: Answer capacity queries from Nims
