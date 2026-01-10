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
┌─────────────────────────────────────────────────────────────────────┐
│                         Morpheus                                    │
│                   (provisions NimsForest nodes)                     │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        ▼                       ▼                       ▼
┌───────────────┐       ┌───────────────┐       ┌───────────────┐
│ NimsForest    │       │ NimsForest    │       │ NimsForest    │
│ Node 1        │       │ Node 2        │       │ Node 3        │
│               │       │               │       │               │
│ - Docker ✓    │       │ - Docker ✓    │       │ - Docker ✓    │
│ - Wind (NATS) │◄─────►│ - Wind (NATS) │◄─────►│ - Wind (NATS) │
│               │       │               │       │               │
│ ┌───────────┐ │       │ ┌───────────┐ │       │               │
│ │ Container │ │       │ │ Container │ │       │   (idle)      │
│ │ claude    │ │       │ │ playwright│ │       │               │
│ └───────────┘ │       │ └───────────┘ │       │               │
└───────────────┘       └───────────────┘       └───────────────┘
```

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
│ - Docker      │ │ - Songbird  │ │ - Webhooks │ │ - Playwright│
│ - Claude      │ │ - Telegram  │ │ - CI/CD    │ │ - Puppeteer│
│ - Aider       │ │ - Slack     │ │ - APIs     │ │ - Selenium │
│ - Cursor      │ │ - Email     │ │ - Scripts  │ │ - Headless │
└───────────────┘ └─────────────┘ └────────────┘ └────────────┘
```

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

Calls external systems via webhooks, APIs, or scripts.

```go
// pkg/nim/robot_agent.go

type RobotAgent interface {
    Agent
    Endpoint() string
    Method() string
}

type RobotAgentConfig struct {
    Name           string
    Role           string   // "builder", "deployer", "notifier"
    Responsibility string
    Method         string   // "webhook", "api", "script"
    Endpoint       string   // URL or command
    Auth           string   // Auth method/key reference
}
```

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

## Part 3: Land Registry

Tracks available capacity across NimsForest nodes.

### Land Model

```go
// pkg/nim/land.go

type Land struct {
    ID           string
    NodeID       string    // NimsForest node ID
    Status       string    // "available", "busy", "offline"
    Capacity     int       // Concurrent tasks
    CurrentTasks int
    Docker       bool      // Has Docker
    Tools        []string  // Installed tools
}

type LandRegistry interface {
    // Find node with capacity for agent type
    FindAvailable(ctx context.Context, agentType AgentType) (*Land, error)
    
    // Reserve capacity
    Reserve(ctx context.Context, landID string) error
    
    // Release capacity
    Release(ctx context.Context, landID string) error
    
    // List all lands
    List(ctx context.Context) ([]Land, error)
}
```

### CoderNim Uses Land Registry

```go
func (c *CoderNim) Action(ctx context.Context, action string, params map[string]interface{}) (interface{}, error) {
    task := buildTask(action, params)
    
    // Find available land
    land, err := c.registry.FindAvailable(ctx, task.RequiredAgent)
    if err != nil {
        return nil, fmt.Errorf("no agent capacity available")
    }
    
    // Reserve
    if err := c.registry.Reserve(ctx, land.ID); err != nil {
        return nil, err
    }
    defer c.registry.Release(ctx, land.ID)
    
    // Launch agent on that land
    agent, err := c.launchAgent(ctx, land, task)
    if err != nil {
        return nil, err
    }
    
    // Execute
    return agent.Run(ctx, task)
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
├── land.go             # Land, LandRegistry
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
│       ├── robot_agent.go # Webhook/API robot agent
│       └── browser_agent.go # Playwright browser agent
├── land/
│   └── registry.go        # Land registry implementation
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

type WebhookRobotAgent struct {
    config nim.RobotAgentConfig
    client *http.Client
}

func NewWebhookRobotAgent(config nim.RobotAgentConfig) *WebhookRobotAgent {
    return &WebhookRobotAgent{
        config: config,
        client: &http.Client{},
    }
}

func (a *WebhookRobotAgent) Run(ctx context.Context, task nim.Task) (*nim.Result, error) {
    payload, _ := json.Marshal(task.Params)
    
    req, err := http.NewRequestWithContext(ctx, "POST", a.config.Endpoint, bytes.NewReader(payload))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := a.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode >= 400 {
        return &nim.Result{
            Success: false,
            Error:   fmt.Sprintf("HTTP %d", resp.StatusCode),
        }, nil
    }
    
    return &nim.Result{
        Success: true,
        Output:  fmt.Sprintf("Webhook triggered: %s", a.config.Endpoint),
    }, nil
}

func (a *WebhookRobotAgent) Type() nim.AgentType { return nim.AgentTypeRobot }
func (a *WebhookRobotAgent) Available(ctx context.Context) bool { return true }
func (a *WebhookRobotAgent) Endpoint() string { return a.config.Endpoint }
func (a *WebhookRobotAgent) Method() string { return a.config.Method }
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
    
    "github.com/yourusername/nimsforest/internal/core"
    "github.com/yourusername/nimsforest/internal/land"
    "github.com/yourusername/nimsforest/pkg/nim"
    "github.com/yourusername/nimsforest/pkg/runtime"
)

type CoderNim struct {
    *core.BaseNim
    asker    nim.AIAsker
    registry land.Registry
    forest   *runtime.Forest
}

func New(base *core.BaseNim, asker nim.AIAsker, registry land.Registry, forest *runtime.Forest) *CoderNim {
    return &CoderNim{
        BaseNim:  base,
        asker:    asker,
        registry: registry,
        forest:   forest,
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
    
    // Find available land
    landInfo, err := c.registry.FindAvailable(ctx, agentType)
    if err != nil {
        return nil, fmt.Errorf("no agent capacity: %w", err)
    }
    
    // Reserve land
    if err := c.registry.Reserve(ctx, landInfo.ID); err != nil {
        return nil, err
    }
    defer c.registry.Release(ctx, landInfo.ID)
    
    // Create and run agent
    agent, err := c.createAgent(ctx, agentType, landInfo, params)
    if err != nil {
        return nil, err
    }
    
    return agent.Run(ctx, task)
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
    case "deploy", "build", "notify":
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
    ci-runner:
      role: builder
      responsibility: "Run CI pipelines"
      method: webhook
      endpoint: "https://api.github.com/repos/org/repo/actions/workflows/ci.yml/dispatches"
      
    deploy-bot:
      role: deployer
      responsibility: "Deploy to production"
      method: api
      endpoint: "https://deploy.example.com/trigger"
      
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
| 1.11 | Create `pkg/nim/land.go` - Land and LandRegistry |

### Phase 2: Internal Implementations

| Task | Description |
|------|-------------|
| 2.1 | Create `internal/ai/asker.go` - Wrap existing aiservice |
| 2.2 | Create `internal/ai/agents/ai_agent.go` - Docker AI agent |
| 2.3 | Create `internal/ai/agents/human_agent.go` - Songbird human agent |
| 2.4 | Create `internal/ai/agents/robot_agent.go` - Webhook robot agent |
| 2.5 | Create `internal/ai/agents/browser_agent.go` - Playwright browser agent |
| 2.6 | Create `internal/land/registry.go` - Land registry |

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
| **Create** | `pkg/nim/` (interfaces), `internal/ai/` (agents), `internal/land/`, `internal/nims/coder/`, `examples/` |
| **Update** | `internal/core/` (AAA support), `internal/songbirds/` (Send), `pkg/runtime/` (config) |
| **Move** | `pkg/brain/` → `pkg/nim/`, example code → `examples/` |

### Agent Types

| Type | Runs On | Use Case |
|------|---------|----------|
| AIAgent | Docker | Code tasks (Claude, Aider) |
| HumanAgent | Songbird (async) | Approvals, reviews |
| RobotAgent | Webhook/API | CI/CD, deployments |
| BrowserAgent | Docker + Playwright | Web automation |

### AAA Methods

| Method | Implementation |
|--------|----------------|
| **Advice** | AI query via existing `pkg/integrations/aiservice/` |
| **Action** | Dispatch to Agent |
| **Automate** | Generate TreeHouse (Lua) or Nim (config) |
