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

## Part 3: Land Capacity (Event-Driven)

No centralized registry. Capacity discovery via events.

### Land Types

| Type | Docker | GPU | Can Run Agents |
|------|--------|-----|----------------|
| **Land** | No | No | No |
| **Nimland** | Yes | No | Yes (AI, Browser) |
| **Manaland** | Yes | Yes | Yes (GPU workloads) |

### Capacity Discovery Pattern

```
┌──────────┐                              ┌──────────┐
│ CoderNim │                              │ Nimland  │
└────┬─────┘                              └────┬─────┘
     │                                         │
     │── Whisper("land.capacity.request") ────►│
     │   {task_id, needs_gpu: false}           │
     │                                         │
     │◄── Leaf("land.capacity.response") ──────│
     │    {task_id, land_id, available: true}  │
     │                                         │
     │── Whisper("land.reserve") ─────────────►│
     │   {task_id, land_id}                    │
     │                                         │
     │◄── Leaf("land.reserved") ───────────────│
     │    {task_id, land_id, success: true}    │
```

### Event Subjects

| Subject | Publisher | Purpose |
|---------|-----------|---------|
| `land.capacity.request` | CoderNim | "Who has capacity?" |
| `land.capacity.response` | Nimland/Manaland | "I have capacity" |
| `land.reserve` | CoderNim | "Reserve this land" |
| `land.reserved` | Nimland/Manaland | "Reserved for you" |
| `land.release` | CoderNim | "Done, release" |

### Nimland/Manaland Handler

Each Nimland/Manaland runs a handler that responds to capacity requests:

```go
// Runs on each Nimland/Manaland node
func (h *LandHandler) Handle(ctx context.Context, leaf nim.Leaf) error {
    switch leaf.GetSubject() {
    case "land.capacity.request":
        return h.handleCapacityRequest(ctx, leaf)
    case "land.reserve":
        return h.handleReserve(ctx, leaf)
    case "land.release":
        return h.handleRelease(ctx, leaf)
    }
    return nil
}

func (h *LandHandler) handleCapacityRequest(ctx context.Context, leaf nim.Leaf) error {
    var req CapacityRequest
    json.Unmarshal(leaf.GetData(), &req)
    
    // Check if we match requirements
    if req.NeedsGPU && !h.hasGPU {
        return nil // Don't respond, we don't have GPU
    }
    
    // Check if we have capacity
    if h.runningAgents >= h.maxAgents {
        return nil // Don't respond, at capacity
    }
    
    // Respond with availability
    return h.wind.Whisper(ctx, &Leaf{
        Subject: "land.capacity.response",
        Data: CapacityResponse{
            TaskID:    req.TaskID,
            LandID:    h.landID,
            LandType:  h.landType,
            Available: true,
        },
    })
}
```

### CoderNim Uses Events

```go
func (c *CoderNim) Action(ctx context.Context, action string, params map[string]interface{}) (interface{}, error) {
    task := buildTask(action, params)
    taskID := task.ID
    needsGPU := params["gpu"] == true
    
    // Request capacity (broadcast)
    c.wind.Whisper(ctx, &Leaf{
        Subject: "land.capacity.request",
        Data: CapacityRequest{TaskID: taskID, NeedsGPU: needsGPU},
    })
    
    // Collect responses (with timeout)
    responses := c.collectResponses(ctx, "land.capacity.response", taskID, 2*time.Second)
    if len(responses) == 0 {
        return nil, fmt.Errorf("no land capacity available")
    }
    
    // Pick first available
    land := responses[0]
    
    // Reserve it
    c.wind.Whisper(ctx, &Leaf{
        Subject: "land.reserve",
        Data: ReserveRequest{TaskID: taskID, LandID: land.LandID},
    })
    
    // Wait for confirmation
    reserved := c.waitForResponse(ctx, "land.reserved", taskID, 5*time.Second)
    if !reserved.Success {
        return nil, fmt.Errorf("failed to reserve land")
    }
    
    // Execute agent task (result comes back as event)
    c.wind.Whisper(ctx, &Leaf{
        Subject: fmt.Sprintf("agent.execute.%s", land.LandID),
        Data: task,
    })
    
    // Return - result will come back as event
    return &nim.Result{
        Success: true,
        Output:  fmt.Sprintf("Task %s dispatched to %s", taskID, land.LandID),
    }, nil
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

### Phase 1: pkg/nim/ Interfaces

| Task | Description |
|------|-------------|
| 1.1 | Create `pkg/nim/nim.go` - Nim interface with AAA |
| 1.2 | Create `pkg/nim/brain.go` - Move from pkg/brain |
| 1.3 | Create `pkg/nim/leaf.go` - Leaf interface |
| 1.4 | Create `pkg/nim/wind.go` - Whisperer interface |
| 1.5 | Create `pkg/nim/asker.go` - AIAsker interface |
| 1.6 | Create `pkg/nim/agent.go` - Agent interface |
| 1.7 | Create `pkg/nim/ai_agent.go` - AIAgent interface |
| 1.8 | Create `pkg/nim/human_agent.go` - HumanAgent interface |
| 1.9 | Create `pkg/nim/robot_agent.go` - RobotAgent interface |
| 1.10 | Create `pkg/nim/browser_agent.go` - BrowserAgent interface |
| 1.11 | Create `pkg/nim/land.go` - Land types and capacity events |

### Phase 2: Internal Implementations

| Task | Description |
|------|-------------|
| 2.1 | Create `internal/ai/asker.go` - Wrap existing aiservice |
| 2.2 | Create `internal/ai/agents/ai_agent.go` - Docker AI agent |
| 2.3 | Create `internal/ai/agents/human_agent.go` - Songbird human agent |
| 2.4 | Create `internal/ai/agents/robot_agent.go` - Physical robot agent |
| 2.5 | Create `internal/ai/agents/browser_agent.go` - Playwright browser agent |
| 2.6 | Create `internal/land/handler.go` - Land capacity handler |

### Phase 3: Songbirds

| Task | Description |
|------|-------------|
| 3.1 | Update `internal/songbirds/songbird.go` - Add Send + Message type |
| 3.2 | Update `internal/songbirds/telegram.go` - Implement Send, emit response Leaves |
| 3.3 | Create `internal/songbirds/slack.go` - Slack songbird |
| 3.4 | Create `internal/songbirds/email.go` - Email songbird |

### Phase 4: Core Updates

| Task | Description |
|------|-------------|
| 4.1 | Update `internal/core/nim.go` - BaseNim with AAA |
| 4.2 | Update `internal/core/leaf.go` - Implement nim.Leaf |
| 4.3 | Update `internal/core/wind.go` - Implement nim.Whisperer |

### Phase 5: CoderNim

| Task | Description |
|------|-------------|
| 5.1 | Create `internal/nims/coder/coder.go` - Full implementation |
| 5.2 | Create `internal/nims/coder/coder_test.go` - Tests |

### Phase 6: Configuration

| Task | Description |
|------|-------------|
| 6.1 | Update `pkg/runtime/config.go` - Add agent configs |
| 6.2 | Update `pkg/runtime/forest.go` - Load agents |
| 6.3 | Update `cmd/forest/main.go` - Wire up CoderNim |

### Phase 7: Cleanup & Examples

| Task | Description |
|------|-------------|
| 7.1 | Move `pkg/brain/` → `pkg/nim/`, update imports |
| 7.2 | Create `examples/` and move example code from `internal/` |
| 7.3 | Update `cmd/forest/main.go` to not auto-load examples |

### Phase 8: Testing

| Task | Description |
|------|-------------|
| 8.1 | Test pkg/nim interfaces |
| 8.2 | Test agent implementations |
| 8.3 | Test CoderNim AAA methods |
| 8.4 | Integration tests |

---

## Summary

### Changes Overview

| Action | Items |
|--------|-------|
| **Create** | `pkg/nim/` (interfaces), `internal/ai/` (agents), `internal/land/` (handler), `internal/nims/coder/`, `examples/` |
| **Update** | `internal/core/` (AAA support), `internal/songbirds/` (Send), `pkg/runtime/` (config) |
| **Move** | `pkg/brain/` → `pkg/nim/`, example code → `examples/` |

### Agent Types

| Type | Runs On | Use Case |
|------|---------|----------|
| AIAgent | Nimland/Manaland | Code tasks (Claude, Aider) |
| BrowserAgent | Nimland/Manaland | Web automation (Playwright) |
| HumanAgent | Songbird (async) | Approvals, reviews |
| RobotAgent | Physical robot | Temi, SO-ARM100, humanoids |

### Land Types

| Type | Docker | GPU | Purpose |
|------|--------|-----|---------|
| Land | No | No | Event backbone |
| Nimland | Yes | No | Docker agents |
| Manaland | Yes | Yes | GPU workloads |

### AAA Methods

| Method | Implementation |
|--------|----------------|
| **Advice** | AI query via existing `pkg/integrations/aiservice/` |
| **Action** | Dispatch to Agent |
| **Automate** | Generate TreeHouse (Lua) or Nim (config) |

---

## Part 9: Land as Core Concept

Land is becoming central to the system as the representation of compute capacity. Currently, Land exists only as a ViewModel concept for display purposes. This section proposes elevating Land to a first-class core concept like Wind, River, Nim, and TreeHouse.

### Why Land Should Be Core

| Current Status | Proposed Status |
|----------------|-----------------|
| `viewmodel.LandViewModel` (display only) | `core.Land` interface + implementations |
| Passive (read-only view) | Active (participates in capacity negotiation) |
| No events | Publishes/subscribes to Wind events |
| No lifecycle | Has Start/Stop lifecycle |

### Land Hierarchy

Like other core concepts, Land will have an interface and multiple implementations:

```
                        ┌─────────────┐
                        │    Land     │
                        │ (interface) │
                        └──────┬──────┘
                               │
        ┌──────────────────────┼──────────────────────┐
        │                      │                      │
┌───────▼───────┐      ┌───────▼───────┐      ┌───────▼───────┐
│   BaseLand    │      │   Nimland     │      │   Manaland    │
│ (backbone)    │      │ (Docker)      │      │ (Docker+GPU)  │
│               │      │               │      │               │
│ - Wind ✓      │      │ - Wind ✓      │      │ - Wind ✓      │
│ - River ✓     │      │ - River ✓     │      │ - River ✓     │
│ - No Docker   │      │ - Docker ✓    │      │ - Docker ✓    │
│ - No GPU      │      │ - No GPU      │      │ - GPU ✓       │
└───────────────┘      └───────────────┘      └───────────────┘
```

### Core Land Interface

```go
// internal/core/land.go

package core

import (
    "context"
    "time"
)

// LandType identifies the capabilities of a Land node.
type LandType string

const (
    LandTypeBase     LandType = "land"     // No Docker, backbone only
    LandTypeNimland  LandType = "nimland"  // Docker-capable
    LandTypeManaland LandType = "manaland" // Docker + GPU
)

// Land represents a compute node in the NimsForest cluster.
// It actively participates in capacity negotiation via Wind events.
type Land interface {
    // Identity
    ID() string
    Type() LandType
    Hostname() string
    
    // Capacity
    Capacity() Capacity
    Available() Capacity
    Occupancy() float64
    
    // Capabilities
    HasDocker() bool
    HasGPU() bool
    CanRun(requirements Requirements) bool
    
    // Process management
    Processes() []Process
    Reserve(ctx context.Context, req ReserveRequest) (*Reservation, error)
    Release(ctx context.Context, reservationID string) error
    
    // Lifecycle
    Start(ctx context.Context) error
    Stop() error
    IsRunning() bool
    
    // Events - Land listens on Wind and responds to capacity queries
    HandleCapacityRequest(ctx context.Context, req CapacityRequest) error
}

// Capacity represents compute resources.
type Capacity struct {
    RAM       uint64  // Bytes
    CPUCores  int
    CPUFreqHz uint64  // Hz
    GPUVram   uint64  // Bytes (0 if no GPU)
    GPUTflops float64 // TFLOPS
}

// Requirements specifies what a task needs.
type Requirements struct {
    MinRAM      uint64
    MinCPUCores int
    NeedsDocker bool
    NeedsGPU    bool
    MinGPUVram  uint64
}

// ReserveRequest is a request to reserve capacity on a Land.
type ReserveRequest struct {
    TaskID       string
    Requirements Requirements
    Duration     time.Duration // Expected duration (for scheduling hints)
}

// Reservation represents reserved capacity on a Land.
type Reservation struct {
    ID         string
    LandID     string
    TaskID     string
    Reserved   Capacity
    ExpiresAt  time.Time
}

// Process represents something running on a Land.
type Process struct {
    ID        string
    Type      string // "tree", "treehouse", "nim", "agent"
    Name      string
    Allocated Capacity
    StartedAt time.Time
}
```

### Land Wind Events

Land uses Wind for event-driven capacity discovery (no central registry):

```go
// internal/core/land_events.go

package core

// Wind subjects for Land operations
const (
    // Discovery
    SubjectLandAnnounce       = "land.announce"          // Land broadcasts presence
    SubjectLandCapacityQuery  = "land.capacity.query"    // "Who has capacity?"
    SubjectLandCapacityReply  = "land.capacity.reply"    // "I have capacity"
    
    // Reservation
    SubjectLandReserve        = "land.reserve"           // "Reserve this for me"
    SubjectLandReserved       = "land.reserved"          // "Reserved"
    SubjectLandRelease        = "land.release"           // "Done, release"
    
    // Agent execution (on Nimland/Manaland)
    SubjectAgentExecute       = "agent.execute.>"        // Dispatch agent task
    SubjectAgentResult        = "agent.result.>"         // Agent task result
    
    // Heartbeat
    SubjectLandHeartbeat      = "land.heartbeat"         // Periodic health check
)

// CapacityQuery is broadcast when someone needs compute capacity.
type CapacityQuery struct {
    QueryID      string       `json:"query_id"`
    Requirements Requirements `json:"requirements"`
    ReplyTo      string       `json:"reply_to"` // Subject for responses
}

// CapacityReply is sent by Lands that can fulfill a capacity query.
type CapacityReply struct {
    QueryID    string   `json:"query_id"`
    LandID     string   `json:"land_id"`
    LandType   LandType `json:"land_type"`
    Available  Capacity `json:"available"`
    Latency    int64    `json:"latency_ms"` // Network latency hint
}

// LandAnnounce is broadcast when a Land joins or updates.
type LandAnnounce struct {
    LandID    string   `json:"land_id"`
    LandType  LandType `json:"land_type"`
    Hostname  string   `json:"hostname"`
    Capacity  Capacity `json:"capacity"`
    Available Capacity `json:"available"`
}

// LandHeartbeat is sent periodically by each Land.
type LandHeartbeat struct {
    LandID    string   `json:"land_id"`
    Available Capacity `json:"available"`
    Processes int      `json:"processes"`
    Timestamp int64    `json:"timestamp"`
}
```

### BaseLand Implementation

```go
// internal/core/land_base.go

package core

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "sync"
    "time"
)

// BaseLand is the default Land implementation for backbone nodes.
// It does NOT have Docker or GPU. Embed this for Nimland/Manaland.
type BaseLand struct {
    id       string
    hostname string
    landType LandType
    capacity Capacity
    
    wind      *Wind
    processes map[string]*Process
    reservations map[string]*Reservation
    
    mu      sync.RWMutex
    running bool
    stopCh  chan struct{}
}

// NewBaseLand creates a new backbone Land node.
func NewBaseLand(id string, wind *Wind, capacity Capacity) *BaseLand {
    return &BaseLand{
        id:           id,
        landType:     LandTypeBase,
        capacity:     capacity,
        wind:         wind,
        processes:    make(map[string]*Process),
        reservations: make(map[string]*Reservation),
        stopCh:       make(chan struct{}),
    }
}

func (l *BaseLand) ID() string       { return l.id }
func (l *BaseLand) Type() LandType   { return l.landType }
func (l *BaseLand) Hostname() string { return l.hostname }
func (l *BaseLand) Capacity() Capacity { return l.capacity }
func (l *BaseLand) HasDocker() bool  { return false }
func (l *BaseLand) HasGPU() bool     { return false }

// Available returns current available capacity.
func (l *BaseLand) Available() Capacity {
    l.mu.RLock()
    defer l.mu.RUnlock()
    
    used := Capacity{}
    for _, p := range l.processes {
        used.RAM += p.Allocated.RAM
        used.CPUCores += p.Allocated.CPUCores
    }
    
    return Capacity{
        RAM:       l.capacity.RAM - used.RAM,
        CPUCores:  l.capacity.CPUCores - used.CPUCores,
        GPUVram:   l.capacity.GPUVram, // GPU not used on BaseLand
        GPUTflops: l.capacity.GPUTflops,
    }
}

// Occupancy returns RAM usage as percentage.
func (l *BaseLand) Occupancy() float64 {
    avail := l.Available()
    if l.capacity.RAM == 0 {
        return 0
    }
    used := l.capacity.RAM - avail.RAM
    return float64(used) / float64(l.capacity.RAM) * 100
}

// CanRun checks if this Land can run a task with given requirements.
func (l *BaseLand) CanRun(req Requirements) bool {
    if req.NeedsDocker && !l.HasDocker() {
        return false
    }
    if req.NeedsGPU && !l.HasGPU() {
        return false
    }
    
    avail := l.Available()
    return avail.RAM >= req.MinRAM && avail.CPUCores >= req.MinCPUCores
}

// Start begins the Land's event loop.
func (l *BaseLand) Start(ctx context.Context) error {
    l.mu.Lock()
    if l.running {
        l.mu.Unlock()
        return fmt.Errorf("land %s already running", l.id)
    }
    l.running = true
    l.mu.Unlock()
    
    // Subscribe to capacity queries
    _, err := l.wind.Catch(SubjectLandCapacityQuery, func(leaf Leaf) {
        l.handleCapacityQuery(ctx, leaf)
    })
    if err != nil {
        return fmt.Errorf("failed to subscribe to capacity queries: %w", err)
    }
    
    // Subscribe to reserve requests
    _, err = l.wind.Catch(SubjectLandReserve, func(leaf Leaf) {
        l.handleReserveRequest(ctx, leaf)
    })
    if err != nil {
        return fmt.Errorf("failed to subscribe to reserve requests: %w", err)
    }
    
    // Announce presence
    l.announce(ctx)
    
    // Start heartbeat goroutine
    go l.heartbeatLoop(ctx)
    
    log.Printf("[Land:%s] Started (%s)", l.id, l.landType)
    return nil
}

// handleCapacityQuery responds to capacity queries if we can fulfill them.
func (l *BaseLand) handleCapacityQuery(ctx context.Context, leaf Leaf) {
    var query CapacityQuery
    if err := json.Unmarshal(leaf.Data, &query); err != nil {
        log.Printf("[Land:%s] Invalid capacity query: %v", l.id, err)
        return
    }
    
    // Check if we can fulfill the requirements
    if !l.CanRun(query.Requirements) {
        return // Don't respond if we can't help
    }
    
    // Send reply
    reply := CapacityReply{
        QueryID:   query.QueryID,
        LandID:    l.id,
        LandType:  l.landType,
        Available: l.Available(),
    }
    
    replyData, _ := json.Marshal(reply)
    replyLeaf := NewLeaf(query.ReplyTo, replyData, "land:"+l.id)
    
    if err := l.wind.Drop(*replyLeaf); err != nil {
        log.Printf("[Land:%s] Failed to reply to capacity query: %v", l.id, err)
    }
}

// announce broadcasts this Land's presence.
func (l *BaseLand) announce(ctx context.Context) {
    ann := LandAnnounce{
        LandID:    l.id,
        LandType:  l.landType,
        Hostname:  l.hostname,
        Capacity:  l.capacity,
        Available: l.Available(),
    }
    
    data, _ := json.Marshal(ann)
    leaf := NewLeaf(SubjectLandAnnounce, data, "land:"+l.id)
    l.wind.Drop(*leaf)
}

// heartbeatLoop sends periodic heartbeats.
func (l *BaseLand) heartbeatLoop(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            hb := LandHeartbeat{
                LandID:    l.id,
                Available: l.Available(),
                Processes: len(l.processes),
                Timestamp: time.Now().Unix(),
            }
            data, _ := json.Marshal(hb)
            leaf := NewLeaf(SubjectLandHeartbeat, data, "land:"+l.id)
            l.wind.Drop(*leaf)
            
        case <-l.stopCh:
            return
        case <-ctx.Done():
            return
        }
    }
}

func (l *BaseLand) Stop() error {
    l.mu.Lock()
    defer l.mu.Unlock()
    
    if !l.running {
        return nil
    }
    
    close(l.stopCh)
    l.running = false
    log.Printf("[Land:%s] Stopped", l.id)
    return nil
}

func (l *BaseLand) IsRunning() bool {
    l.mu.RLock()
    defer l.mu.RUnlock()
    return l.running
}
```

### Nimland Implementation

```go
// internal/land/nimland.go

package land

import (
    "context"
    "fmt"
    "os/exec"
    
    "github.com/yourusername/nimsforest/internal/core"
)

// Nimland is a Land with Docker capability for running agents.
type Nimland struct {
    *core.BaseLand
    dockerAvailable bool
}

// NewNimland creates a Docker-capable Land node.
func NewNimland(id string, wind *core.Wind, capacity core.Capacity) (*Nimland, error) {
    // Check Docker availability
    _, err := exec.LookPath("docker")
    if err != nil {
        return nil, fmt.Errorf("docker not available: %w", err)
    }
    
    base := core.NewBaseLand(id, wind, capacity)
    return &Nimland{
        BaseLand:        base,
        dockerAvailable: true,
    }, nil
}

func (n *Nimland) Type() core.LandType { return core.LandTypeNimland }
func (n *Nimland) HasDocker() bool     { return n.dockerAvailable }

// Start extends BaseLand.Start to also listen for agent execution requests.
func (n *Nimland) Start(ctx context.Context) error {
    if err := n.BaseLand.Start(ctx); err != nil {
        return err
    }
    
    // Subscribe to agent execution for this land
    _, err := n.GetWind().Catch(
        fmt.Sprintf("agent.execute.%s", n.ID()),
        func(leaf core.Leaf) {
            n.handleAgentExecute(ctx, leaf)
        },
    )
    if err != nil {
        return fmt.Errorf("failed to subscribe to agent execute: %w", err)
    }
    
    return nil
}

func (n *Nimland) handleAgentExecute(ctx context.Context, leaf core.Leaf) {
    // Execute agent in Docker container
    // Result published to agent.result.{task_id}
}
```

### Manaland Implementation

```go
// internal/land/manaland.go

package land

import (
    "github.com/yourusername/nimsforest/internal/core"
)

// Manaland is a Land with Docker and GPU capability.
type Manaland struct {
    *Nimland
    gpuInfo GPUInfo
}

type GPUInfo struct {
    Vendor  string  // "nvidia", "amd"
    Model   string  // "RTX 4090"
    Vram    uint64  // Bytes
    Tflops  float64
}

// NewManaland creates a GPU-capable Land node.
func NewManaland(id string, wind *core.Wind, capacity core.Capacity, gpu GPUInfo) (*Manaland, error) {
    nimland, err := NewNimland(id, wind, capacity)
    if err != nil {
        return nil, err
    }
    
    return &Manaland{
        Nimland: nimland,
        gpuInfo: gpu,
    }, nil
}

func (m *Manaland) Type() core.LandType { return core.LandTypeManaland }
func (m *Manaland) HasGPU() bool        { return true }
func (m *Manaland) GPUInfo() GPUInfo    { return m.gpuInfo }
```

### pkg/land/ Public Package

```go
// pkg/land/land.go

package land

import (
    "context"
)

// Land is the public interface for a compute node.
type Land interface {
    ID() string
    Type() Type
    Capacity() Capacity
    Available() Capacity
    
    HasDocker() bool
    HasGPU() bool
    CanRun(Requirements) bool
    
    Start(ctx context.Context) error
    Stop() error
}

type Type string

const (
    TypeBase     Type = "land"
    TypeNimland  Type = "nimland"
    TypeManaland Type = "manaland"
)

type Capacity struct {
    RAM       uint64
    CPUCores  int
    GPUVram   uint64
    GPUTflops float64
}

type Requirements struct {
    MinRAM      uint64
    MinCPUCores int
    NeedsDocker bool
    NeedsGPU    bool
}
```

### Integration with Forest Runtime

```go
// pkg/runtime/forest.go (additions)

type Forest struct {
    // ... existing fields ...
    
    // Land nodes managed by this Forest
    lands map[string]land.Land
}

// AddLand registers a Land node with the forest.
func (f *Forest) AddLand(l land.Land) error {
    f.mu.Lock()
    defer f.mu.Unlock()
    
    if _, exists := f.lands[l.ID()]; exists {
        return fmt.Errorf("land '%s' already exists", l.ID())
    }
    
    if f.running {
        if err := l.Start(context.Background()); err != nil {
            return fmt.Errorf("failed to start land: %w", err)
        }
    }
    
    f.lands[l.ID()] = l
    log.Printf("[Forest] Added land '%s' (%s)", l.ID(), l.Type())
    return nil
}

// FindCapacity queries the cluster for available capacity.
func (f *Forest) FindCapacity(ctx context.Context, req land.Requirements) ([]land.CapacityReply, error) {
    // Broadcast capacity query via Wind
    // Collect responses with timeout
    // Return sorted by latency/availability
}
```

### Configuration Update

```yaml
# forest.yaml

# Land configuration for this node
land:
  id: ${HOSTNAME}
  type: nimland  # land, nimland, or manaland
  capacity:
    ram: 16GB
    cpu_cores: 8
  
  # Only for manaland
  gpu:
    vendor: nvidia
    model: "RTX 4090"
    vram: 24GB
```

### Directory Structure Update

```
internal/
├── core/
│   ├── land.go           # Land interface + types
│   ├── land_base.go      # BaseLand implementation
│   ├── land_events.go    # Wind events for Land
│   └── ... (existing)
├── land/
│   ├── nimland.go        # Nimland (Docker) implementation
│   ├── manaland.go       # Manaland (GPU) implementation
│   └── detector.go       # Auto-detect node capabilities
└── ... (existing)

pkg/
├── land/
│   ├── land.go           # Public Land interface
│   └── types.go          # Public types
└── ... (existing)
```

### Task Breakdown for Land Core Concept

| Phase | Task | Description |
|-------|------|-------------|
| **9.1** | Create `internal/core/land.go` | Land interface, types, events |
| **9.2** | Create `internal/core/land_base.go` | BaseLand implementation |
| **9.3** | Create `internal/land/nimland.go` | Nimland (Docker) |
| **9.4** | Create `internal/land/manaland.go` | Manaland (GPU) |
| **9.5** | Create `internal/land/detector.go` | Auto-detect capabilities |
| **9.6** | Create `pkg/land/` | Public interfaces |
| **9.7** | Update `pkg/runtime/forest.go` | Land management |
| **9.8** | Update `pkg/runtime/config.go` | Land configuration |
| **9.9** | Migrate `viewmodel.LandViewModel` | Use core.Land |
| **9.10** | Tests | Land unit and integration tests |

### How Land Relates to Other Systems

```
┌─────────────────────────────────────────────────────────────────────┐
│                            FOREST                                   │
│                                                                     │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐         │
│  │  WIND   │◄──►│  RIVER  │    │   NIM   │    │TREEHOUSE│         │
│  │(pub/sub)│    │ (data)  │    │  (AAA)  │    │ (Lua)   │         │
│  └────┬────┘    └────┬────┘    └────┬────┘    └────┬────┘         │
│       │              │              │              │               │
│       └──────────────┴──────────────┴──────────────┘               │
│                           │                                        │
│                           ▼                                        │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │                         LAND                                 │  │
│  │  (compute substrate - everything runs ON Land)               │  │
│  │                                                              │  │
│  │   ┌──────────┐   ┌──────────┐   ┌──────────┐               │  │
│  │   │ BaseLand │   │ Nimland  │   │ Manaland │               │  │
│  │   │(backbone)│   │(+Docker) │   │(+GPU)    │               │  │
│  │   └──────────┘   └──────────┘   └──────────┘               │  │
│  │                                                              │  │
│  └─────────────────────────────────────────────────────────────┘  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

**Key Insight:** Land is the physical substrate. Everything else (Wind, River, Nim, TreeHouse) runs **on** Land. Making Land a core concept allows:

1. **Capacity-aware scheduling** - Nims can query available Land before dispatching agents
2. **Event-driven discovery** - No central registry, Lands announce themselves via Wind
3. **Type-based routing** - Tasks routed to appropriate Land type (GPU tasks → Manaland)
4. **Self-managing cluster** - Lands join/leave dynamically, capacity adapts automatically
