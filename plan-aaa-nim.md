# Plan: AAA Nim Implementation

## Overview

Implement the AAA (Advice/Action/Automate) pattern for Nims across three repositories:

| Repo | Purpose | Pattern |
|------|---------|---------|
| `aiservicefactory` | Stateless API calls to AI models | Advice (Ask) |
| `aiagentfactory` | Stateful agent sessions with tool use | Action |
| `nimsforest2` | Nim framework, uses both factories | AAA interface |

## Architecture

```
┌─────────────────────────────────────┐
│           nimsforest2               │
│  ┌─────────────────────────────┐    │
│  │        pkg/nim/             │    │
│  │  (interfaces only)          │    │
│  └─────────────────────────────┘    │
│              ↑                      │
│  ┌─────────────────────────────┐    │
│  │       internal/ai/          │    │
│  │  asker.go    agent.go       │    │
│  └──────┬────────────┬─────────┘    │
└─────────┼────────────┼──────────────┘
          │            │
          ▼            ▼
┌─────────────────┐  ┌─────────────────┐
│aiservicefactory │  │ aiagentfactory  │
│   (Ask)         │  │ (Action/Agent)  │
│                 │  │                 │
│ - Claude API    │  │ - Claude Code   │
│ - OpenAI API    │  │ - Aider         │
│ - Gemini API    │  │ - Cursor        │
│ - xAI API       │  │ - Local LLM     │
└─────────────────┘  └─────────────────┘
```

---

## Part 1: aiagentfactory Repository

### 1.1 Create Repository

Create `github.com/nimsforest/aiagentfactory` with the following structure:

```
aiagentfactory/
├── go.mod
├── README.md
├── agent.go                    # Core interfaces
├── session.go                  # Session management
├── result.go                   # Result types
├── providers/
│   ├── claudecode/             # Claude Code CLI in VM
│   │   ├── provider.go
│   │   ├── session.go
│   │   ├── vm.go
│   │   └── install.go
│   ├── aider/                  # Future
│   └── cursor/                 # Future
└── vm/
    ├── interface.go            # VM abstraction
    ├── docker.go               # Docker implementation
    └── cloud.go                # Future: cloud VMs
```

### 1.2 Core Interfaces

Create `agent.go`:

```go
package aiagentfactory

import "context"

// AgentProvider creates agent sessions
type AgentProvider interface {
    // Launch starts the agent environment, returns when ready
    Launch(ctx context.Context, opts LaunchOptions) (Session, error)
    
    // Name returns the provider name (e.g., "claudecode", "aider")
    Name() string
}

// LaunchOptions configures agent launch
type LaunchOptions struct {
    WorkDir    string            // Working directory to mount
    APIKey     string            // API key for the AI service
    Model      string            // Model to use (optional)
    Env        map[string]string // Additional environment variables
}

// Session represents an active agent session
type Session interface {
    // Run executes a task, blocks until complete
    Run(ctx context.Context, task string) (*Result, error)
    
    // Stream executes with streaming output
    Stream(ctx context.Context, task string) (<-chan Event, error)
    
    // Close terminates the session and cleans up resources
    Close(ctx context.Context) error
    
    // Status returns current session state
    Status() SessionStatus
    
    // ID returns unique session identifier
    ID() string
}

// SessionStatus represents session state
type SessionStatus string

const (
    StatusStarting SessionStatus = "starting"
    StatusReady    SessionStatus = "ready"
    StatusRunning  SessionStatus = "running"
    StatusDone     SessionStatus = "done"
    StatusError    SessionStatus = "error"
)
```

Create `result.go`:

```go
package aiagentfactory

// Result is the outcome of an agent task
type Result struct {
    Success   bool       `json:"success"`
    Output    string     `json:"output"`
    Files     []FileDiff `json:"files,omitempty"`
    Error     string     `json:"error,omitempty"`
    SessionID string     `json:"session_id"`
}

// FileDiff represents a file change
type FileDiff struct {
    Path   string `json:"path"`
    Action string `json:"action"` // created, modified, deleted
    Diff   string `json:"diff,omitempty"`
}

// Event is a streaming event from the agent
type Event struct {
    Type    EventType `json:"type"`
    Content string    `json:"content"`
    Tool    *ToolUse  `json:"tool,omitempty"`
}

// EventType categorizes streaming events
type EventType string

const (
    EventThinking EventType = "thinking"
    EventToolUse  EventType = "tool_use"
    EventOutput   EventType = "output"
    EventError    EventType = "error"
    EventDone     EventType = "done"
)

// ToolUse records an agent tool invocation
type ToolUse struct {
    Name   string                 `json:"name"`
    Input  map[string]interface{} `json:"input"`
    Output interface{}            `json:"output,omitempty"`
}
```

### 1.3 VM Interface

Create `vm/interface.go`:

```go
package vm

import "context"

// Provider manages VM instances
type Provider interface {
    Create(ctx context.Context, config Config) (Instance, error)
    List(ctx context.Context) ([]Instance, error)
}

// Config for VM creation
type Config struct {
    Image    string            // Docker image or VM image
    Memory   string            // Memory limit (e.g., "4g")
    CPU      int               // CPU cores
    Mounts   []Mount           // Volume mounts
    Env      map[string]string // Environment variables
    WorkDir  string            // Working directory
}

// Mount represents a volume mount
type Mount struct {
    Source string // Host path
    Target string // Container path
    Mode   string // ro, rw
}

// Instance represents a running VM
type Instance interface {
    // Exec runs a command and returns output
    Exec(ctx context.Context, cmd string) (string, error)
    
    // ExecStream runs a command with streaming output
    ExecStream(ctx context.Context, cmd string) (<-chan string, error)
    
    // Stop stops the instance
    Stop(ctx context.Context) error
    
    // ID returns instance identifier
    ID() string
    
    // Status returns instance status
    Status() string
}
```

### 1.4 Docker VM Implementation

Create `vm/docker.go`:

```go
package vm

import (
    "context"
    "fmt"
    "os/exec"
    "strings"
)

type DockerProvider struct{}

func NewDockerProvider() *DockerProvider {
    return &DockerProvider{}
}

func (p *DockerProvider) Create(ctx context.Context, config Config) (Instance, error) {
    // Build docker run command
    args := []string{"run", "-d", "--rm"}
    
    // Add mounts
    for _, m := range config.Mounts {
        args = append(args, "-v", fmt.Sprintf("%s:%s:%s", m.Source, m.Target, m.Mode))
    }
    
    // Add env vars
    for k, v := range config.Env {
        args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
    }
    
    // Add resource limits
    if config.Memory != "" {
        args = append(args, "--memory", config.Memory)
    }
    if config.CPU > 0 {
        args = append(args, "--cpus", fmt.Sprintf("%d", config.CPU))
    }
    
    // Add working directory
    if config.WorkDir != "" {
        args = append(args, "-w", config.WorkDir)
    }
    
    // Add image and default command (keep container running)
    args = append(args, config.Image, "tail", "-f", "/dev/null")
    
    // Run docker
    cmd := exec.CommandContext(ctx, "docker", args...)
    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("docker run failed: %w", err)
    }
    
    containerID := strings.TrimSpace(string(output))
    return &DockerInstance{id: containerID}, nil
}

type DockerInstance struct {
    id string
}

func (i *DockerInstance) Exec(ctx context.Context, cmd string) (string, error) {
    execCmd := exec.CommandContext(ctx, "docker", "exec", i.id, "sh", "-c", cmd)
    output, err := execCmd.CombinedOutput()
    return string(output), err
}

func (i *DockerInstance) Stop(ctx context.Context) error {
    return exec.CommandContext(ctx, "docker", "stop", i.id).Run()
}

func (i *DockerInstance) ID() string {
    return i.id
}

func (i *DockerInstance) Status() string {
    cmd := exec.Command("docker", "inspect", "-f", "{{.State.Status}}", i.id)
    output, err := cmd.Output()
    if err != nil {
        return "unknown"
    }
    return strings.TrimSpace(string(output))
}
```

### 1.5 Claude Code Provider

Create `providers/claudecode/provider.go`:

```go
package claudecode

import (
    "context"
    "fmt"
    
    "github.com/nimsforest/aiagentfactory"
    "github.com/nimsforest/aiagentfactory/vm"
)

type Provider struct {
    vmProvider vm.Provider
}

func New(vmProvider vm.Provider) *Provider {
    return &Provider{vmProvider: vmProvider}
}

func (p *Provider) Name() string {
    return "claudecode"
}

func (p *Provider) Launch(ctx context.Context, opts aiagentfactory.LaunchOptions) (aiagentfactory.Session, error) {
    // 1. Create VM instance
    instance, err := p.vmProvider.Create(ctx, vm.Config{
        Image:   "node:20-bookworm", // Node.js for npm
        Memory:  "4g",
        CPU:     2,
        WorkDir: "/workspace",
        Mounts: []vm.Mount{
            {Source: opts.WorkDir, Target: "/workspace", Mode: "rw"},
        },
        Env: map[string]string{
            "ANTHROPIC_API_KEY": opts.APIKey,
        },
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create VM: %w", err)
    }
    
    // 2. Install Claude Code CLI
    _, err = instance.Exec(ctx, "npm install -g @anthropic-ai/claude-code")
    if err != nil {
        instance.Stop(ctx)
        return nil, fmt.Errorf("failed to install claude-code: %w", err)
    }
    
    // 3. Verify installation
    _, err = instance.Exec(ctx, "claude --version")
    if err != nil {
        instance.Stop(ctx)
        return nil, fmt.Errorf("claude-code not working: %w", err)
    }
    
    return &Session{
        instance: instance,
        workDir:  "/workspace",
        model:    opts.Model,
    }, nil
}
```

Create `providers/claudecode/session.go`:

```go
package claudecode

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
    
    "github.com/nimsforest/aiagentfactory"
    "github.com/nimsforest/aiagentfactory/vm"
)

type Session struct {
    instance vm.Instance
    workDir  string
    model    string
    status   aiagentfactory.SessionStatus
    mu       sync.RWMutex
}

func (s *Session) ID() string {
    return s.instance.ID()
}

func (s *Session) Status() aiagentfactory.SessionStatus {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.status
}

func (s *Session) Run(ctx context.Context, task string) (*aiagentfactory.Result, error) {
    s.mu.Lock()
    s.status = aiagentfactory.StatusRunning
    s.mu.Unlock()
    
    defer func() {
        s.mu.Lock()
        s.status = aiagentfactory.StatusReady
        s.mu.Unlock()
    }()
    
    // Build claude command
    cmd := fmt.Sprintf("cd %s && claude --print --output-format json %q", s.workDir, task)
    if s.model != "" {
        cmd = fmt.Sprintf("cd %s && claude --print --output-format json --model %s %q", s.workDir, s.model, task)
    }
    
    // Execute
    output, err := s.instance.Exec(ctx, cmd)
    if err != nil {
        return &aiagentfactory.Result{
            Success:   false,
            Error:     err.Error(),
            Output:    output,
            SessionID: s.ID(),
        }, nil
    }
    
    // Parse JSON output
    var result aiagentfactory.Result
    if err := json.Unmarshal([]byte(output), &result); err != nil {
        // If not JSON, return raw output
        return &aiagentfactory.Result{
            Success:   true,
            Output:    output,
            SessionID: s.ID(),
        }, nil
    }
    
    result.SessionID = s.ID()
    return &result, nil
}

func (s *Session) Stream(ctx context.Context, task string) (<-chan aiagentfactory.Event, error) {
    // TODO: Implement streaming with claude --stream
    return nil, fmt.Errorf("streaming not yet implemented")
}

func (s *Session) Close(ctx context.Context) error {
    s.mu.Lock()
    s.status = aiagentfactory.StatusDone
    s.mu.Unlock()
    
    return s.instance.Stop(ctx)
}
```

### 1.6 Factory Function

Create `factory.go`:

```go
package aiagentfactory

import (
    "fmt"
)

// NewProvider creates an agent provider by name
func NewProvider(name string, vmProvider interface{}) (AgentProvider, error) {
    // Import cycle prevention: providers register themselves
    factory, ok := providers[name]
    if !ok {
        return nil, fmt.Errorf("unknown provider: %s", name)
    }
    return factory(vmProvider)
}

var providers = make(map[string]func(interface{}) (AgentProvider, error))

// RegisterProvider registers a provider factory
func RegisterProvider(name string, factory func(interface{}) (AgentProvider, error)) {
    providers[name] = factory
}
```

---

## Part 2: nimsforest2 Updates

### 2.1 Create pkg/nim/ Package

Create the following files in `pkg/nim/`:

#### `pkg/nim/nim.go`

```go
package nim

import "context"

// Nim is an intelligent agent that provides advice, executes actions, and runs automations.
type Nim interface {
    // Name returns the nim's identifier
    Name() string
    
    // AAA Model
    Advice(ctx context.Context, query string) (string, error)
    Action(ctx context.Context, action string, params map[string]interface{}) (interface{}, error)
    Automate(ctx context.Context, automation string, enabled bool) error
    
    // Event handling
    Handle(ctx context.Context, leaf Leaf) error
    
    // Lifecycle
    Start(ctx context.Context) error
    Stop() error
}

// ActionSpec describes an available action
type ActionSpec struct {
    Name        string            `json:"name"`
    Description string            `json:"description"`
    Params      map[string]string `json:"params"`
}

// AutomationSpec describes an available automation
type AutomationSpec struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    Running     bool   `json:"running"`
}

// Capable extends Nim with introspection
type Capable interface {
    Nim
    ListActions() []ActionSpec
    ListAutomations() []AutomationSpec
}
```

#### `pkg/nim/brain.go`

Move and consolidate from `pkg/brain/`:

```go
package nim

import (
    "context"
    "errors"
    "time"
)

// Brain provides knowledge storage and retrieval
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

// Knowledge represents stored information
type Knowledge struct {
    ID        string    `json:"id"`
    Content   string    `json:"content"`
    Tags      []string  `json:"tags"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// ErrKnowledgeNotFound is returned when knowledge doesn't exist
var ErrKnowledgeNotFound = errors.New("knowledge not found")
```

#### `pkg/nim/leaf.go`

```go
package nim

// Leaf is a message that flows through the wind system
type Leaf interface {
    GetSubject() string
    GetData() []byte
    GetSource() string
}
```

#### `pkg/nim/wind.go`

```go
package nim

import "context"

// Whisperer can send leaves through the wind
type Whisperer interface {
    Whisper(ctx context.Context, leaf Leaf) error
}

// Listener can receive leaves from the wind
type Listener interface {
    Listen(ctx context.Context, subject string) (<-chan Leaf, error)
}
```

#### `pkg/nim/asker.go`

```go
package nim

import "context"

// AIAsker provides simple prompt → response interactions
// Used for the Advice pattern
type AIAsker interface {
    Ask(ctx context.Context, prompt string) (string, error)
}

// AskerFactory creates AIAsker instances
type AskerFactory interface {
    NewAsker(serviceType string, apiKey, model string) (AIAsker, error)
}
```

#### `pkg/nim/agent.go`

```go
package nim

import "context"

// AIAgent provides agentic interactions with tool use
// Used for the Action pattern
type AIAgent interface {
    // Run executes a task, blocks until complete
    Run(ctx context.Context, task string) (*AgentResult, error)
    
    // Close terminates the agent session
    Close(ctx context.Context) error
}

// AgentResult is the outcome of an agent task
type AgentResult struct {
    Success bool        `json:"success"`
    Output  string      `json:"output"`
    Files   []FileDiff  `json:"files,omitempty"`
    Error   string      `json:"error,omitempty"`
}

// FileDiff represents a file change made by the agent
type FileDiff struct {
    Path   string `json:"path"`
    Action string `json:"action"` // created, modified, deleted
    Diff   string `json:"diff,omitempty"`
}

// AgentFactory creates AIAgent instances
type AgentFactory interface {
    NewAgent(providerType string, apiKey, model string, workDir string) (AIAgent, error)
}
```

### 2.2 Create internal/ai/ Adapters

#### `internal/ai/asker.go`

```go
package ai

import (
    "context"
    
    "github.com/nimsforest/aiservicefactory/internal/ai"
    "github.com/yourusername/nimsforest/pkg/nim"
)

// AskerAdapter wraps aiservicefactory for nim.AIAsker
type AskerAdapter struct {
    service ai.AIService
}

// NewAsker creates an AIAsker using aiservicefactory
func NewAsker(serviceType, apiKey, model string) (nim.AIAsker, error) {
    service, err := ai.NewAIService(serviceType, ai.Config{
        APIKey: apiKey,
        Model:  model,
    })
    if err != nil {
        return nil, err
    }
    return &AskerAdapter{service: service}, nil
}

func (a *AskerAdapter) Ask(ctx context.Context, prompt string) (string, error) {
    return a.service.MakeRequest(ctx, prompt)
}

// AskerFactory implements nim.AskerFactory
type AskerFactory struct{}

func (f *AskerFactory) NewAsker(serviceType, apiKey, model string) (nim.AIAsker, error) {
    return NewAsker(serviceType, apiKey, model)
}
```

#### `internal/ai/agent.go`

```go
package ai

import (
    "context"
    
    "github.com/nimsforest/aiagentfactory"
    "github.com/nimsforest/aiagentfactory/providers/claudecode"
    "github.com/nimsforest/aiagentfactory/vm"
    "github.com/yourusername/nimsforest/pkg/nim"
)

// AgentAdapter wraps aiagentfactory for nim.AIAgent
type AgentAdapter struct {
    session aiagentfactory.Session
}

// NewAgent creates an AIAgent using aiagentfactory
func NewAgent(providerType, apiKey, model, workDir string) (nim.AIAgent, error) {
    // Create VM provider
    vmProvider := vm.NewDockerProvider()
    
    // Create agent provider
    var provider aiagentfactory.AgentProvider
    switch providerType {
    case "claudecode":
        provider = claudecode.New(vmProvider)
    default:
        return nil, fmt.Errorf("unknown provider: %s", providerType)
    }
    
    // Launch session
    session, err := provider.Launch(context.Background(), aiagentfactory.LaunchOptions{
        WorkDir: workDir,
        APIKey:  apiKey,
        Model:   model,
    })
    if err != nil {
        return nil, err
    }
    
    return &AgentAdapter{session: session}, nil
}

func (a *AgentAdapter) Run(ctx context.Context, task string) (*nim.AgentResult, error) {
    result, err := a.session.Run(ctx, task)
    if err != nil {
        return nil, err
    }
    
    // Convert to nim.AgentResult
    files := make([]nim.FileDiff, len(result.Files))
    for i, f := range result.Files {
        files[i] = nim.FileDiff{
            Path:   f.Path,
            Action: f.Action,
            Diff:   f.Diff,
        }
    }
    
    return &nim.AgentResult{
        Success: result.Success,
        Output:  result.Output,
        Files:   files,
        Error:   result.Error,
    }, nil
}

func (a *AgentAdapter) Close(ctx context.Context) error {
    return a.session.Close(ctx)
}

// AgentFactory implements nim.AgentFactory
type AgentFactory struct{}

func (f *AgentFactory) NewAgent(providerType, apiKey, model, workDir string) (nim.AIAgent, error) {
    return NewAgent(providerType, apiKey, model, workDir)
}
```

### 2.3 Update internal/core/

#### Update `internal/core/leaf.go`

Add interface methods:

```go
// Add to existing Leaf struct

func (l *Leaf) GetSubject() string { return l.Subject }
func (l *Leaf) GetData() []byte    { return l.Data }
func (l *Leaf) GetSource() string  { return l.Source }

// Compile-time interface check
var _ nim.Leaf = (*Leaf)(nil)
```

#### Update `internal/core/wind.go`

Add Whisperer implementation:

```go
// Add to existing Wind struct

func (w *Wind) Whisper(ctx context.Context, leaf nim.Leaf) error {
    return w.Drop(Leaf{
        Subject:   leaf.GetSubject(),
        Data:      json.RawMessage(leaf.GetData()),
        Source:    leaf.GetSource(),
        Timestamp: time.Now(),
    })
}

// Compile-time interface check
var _ nim.Whisperer = (*Wind)(nil)
```

#### Update `internal/core/nim.go`

Add AAA methods to BaseNim:

```go
// Update BaseNim struct
type BaseNim struct {
    name   string
    wind   *Wind
    humus  *Humus
    soil   *Soil
    asker  nim.AIAsker  // Add
    agent  nim.AIAgent  // Add
}

// Add AAA method implementations
func (n *BaseNim) Advice(ctx context.Context, query string) (string, error) {
    if n.asker == nil {
        return "", fmt.Errorf("no asker configured")
    }
    return n.asker.Ask(ctx, query)
}

func (n *BaseNim) Action(ctx context.Context, action string, params map[string]interface{}) (interface{}, error) {
    // Default: not implemented, concrete nims override
    return nil, fmt.Errorf("action %q not implemented", action)
}

func (n *BaseNim) Automate(ctx context.Context, automation string, enabled bool) error {
    // Default: not implemented, concrete nims override
    return fmt.Errorf("automation %q not implemented", automation)
}

// Add setters
func (n *BaseNim) SetAsker(asker nim.AIAsker) {
    n.asker = asker
}

func (n *BaseNim) SetAgent(agent nim.AIAgent) {
    n.agent = agent
}

// Compile-time interface check
var _ nim.Nim = (*BaseNim)(nil)
```

### 2.4 Create Coding Agent

Create `internal/nims/coder/coder.go`:

```go
package coder

import (
    "context"
    "fmt"
    
    "github.com/yourusername/nimsforest/internal/core"
    "github.com/yourusername/nimsforest/pkg/nim"
)

// CoderNim is a coding agent that uses AI for advice and actions
type CoderNim struct {
    *core.BaseNim
    asker   nim.AIAsker
    agent   nim.AIAgent
    workDir string
}

// New creates a new CoderNim
func New(base *core.BaseNim, asker nim.AIAsker, agent nim.AIAgent, workDir string) *CoderNim {
    return &CoderNim{
        BaseNim: base,
        asker:   asker,
        agent:   agent,
        workDir: workDir,
    }
}

func (c *CoderNim) Subjects() []string {
    return []string{"code.request", "code.>"}
}

// Advice uses the asker for simple Q&A
func (c *CoderNim) Advice(ctx context.Context, query string) (string, error) {
    prompt := fmt.Sprintf(`You are a coding assistant. Working directory: %s

Question: %s

Provide a helpful, concise answer.`, c.workDir, query)
    
    return c.asker.Ask(ctx, prompt)
}

// Action uses the agent for tool-based tasks
func (c *CoderNim) Action(ctx context.Context, action string, params map[string]interface{}) (interface{}, error) {
    if c.agent == nil {
        return nil, fmt.Errorf("no agent configured")
    }
    
    // Build task description from action and params
    task := buildTask(action, params)
    
    // Run agent
    result, err := c.agent.Run(ctx, task)
    if err != nil {
        return nil, err
    }
    
    return result, nil
}

// Automate is not yet implemented
func (c *CoderNim) Automate(ctx context.Context, automation string, enabled bool) error {
    return fmt.Errorf("automation not yet implemented")
}

func (c *CoderNim) ListActions() []nim.ActionSpec {
    return []nim.ActionSpec{
        {Name: "implement", Description: "Implement a feature", Params: map[string]string{"description": "what to implement"}},
        {Name: "fix", Description: "Fix a bug", Params: map[string]string{"description": "bug description"}},
        {Name: "refactor", Description: "Refactor code", Params: map[string]string{"description": "refactoring goals"}},
        {Name: "test", Description: "Write tests", Params: map[string]string{"target": "what to test"}},
        {Name: "review", Description: "Review code", Params: map[string]string{"path": "file or directory"}},
    }
}

func (c *CoderNim) ListAutomations() []nim.AutomationSpec {
    return []nim.AutomationSpec{
        {Name: "watch", Description: "Watch for changes and suggest improvements", Running: false},
    }
}

func buildTask(action string, params map[string]interface{}) string {
    switch action {
    case "implement":
        return fmt.Sprintf("Implement the following: %v", params["description"])
    case "fix":
        return fmt.Sprintf("Fix this bug: %v", params["description"])
    case "refactor":
        return fmt.Sprintf("Refactor with these goals: %v", params["description"])
    case "test":
        return fmt.Sprintf("Write tests for: %v", params["target"])
    case "review":
        return fmt.Sprintf("Review the code in: %v", params["path"])
    default:
        return fmt.Sprintf("Action: %s, Params: %v", action, params)
    }
}
```

### 2.5 Update go.mod

Add dependencies:

```go
require (
    github.com/nimsforest/aiservicefactory v0.1.0
    github.com/nimsforest/aiagentfactory v0.1.0
    // ... existing deps
)
```

### 2.6 Wire Up in main.go

Add to `cmd/forest/main.go`:

```go
// In runForest() or runStandalone()

// Create coder nim if enabled
if os.Getenv("CODER_ENABLED") == "true" {
    workDir := os.Getenv("CODER_WORKDIR")
    if workDir == "" {
        workDir = "."
    }
    
    apiKey := os.Getenv("ANTHROPIC_API_KEY")
    model := os.Getenv("ANTHROPIC_MODEL")
    if model == "" {
        model = "claude-sonnet-4-20250514"
    }
    
    // Create asker (for Advice)
    asker, err := ai.NewAsker("claude", apiKey, model)
    if err != nil {
        log.Printf("Failed to create asker: %v", err)
    }
    
    // Create agent (for Action) - optional, requires Docker
    var agent nim.AIAgent
    if os.Getenv("CODER_AGENT_ENABLED") == "true" {
        agent, err = ai.NewAgent("claudecode", apiKey, model, workDir)
        if err != nil {
            log.Printf("Failed to create agent: %v", err)
        }
    }
    
    // Create coder nim
    coderBase := core.NewBaseNim("coder", wind, humus, soil)
    coderNim := coder.New(coderBase, asker, agent, workDir)
    
    if err := coderNim.Start(ctx); err != nil {
        log.Printf("Failed to start coder nim: %v", err)
    } else {
        defer coderNim.Stop()
        log.Println("🤖 CoderNim started")
    }
}
```

### 2.7 Cleanup

After migration is complete, delete:

- `pkg/brain/` (moved to `pkg/nim/brain.go`)
- `pkg/infrastructure/aiservice/` (replaced by aiservicefactory import)
- `pkg/integrations/aiservice/` (replaced by aiservicefactory import)

Update all imports throughout the codebase to use new locations.

---

## Part 3: Testing

### 3.1 aiagentfactory Tests

- Test VM provider (Docker)
- Test Claude Code provider launch
- Test session Run()
- Integration test: full flow

### 3.2 nimsforest2 Tests

- Test pkg/nim interfaces are satisfied
- Test internal/ai adapters
- Test CoderNim Advice/Action
- Integration test: full flow with mock agent

---

## Summary

### Repositories

1. **aiservicefactory** (existing) - Stateless API calls
2. **aiagentfactory** (new) - Stateful agent sessions
3. **nimsforest2** - Nim framework using both

### Key Interfaces

| Package | Interface | Purpose |
|---------|-----------|---------|
| `pkg/nim` | `Nim` | Core nim with AAA |
| `pkg/nim` | `AIAsker` | Simple prompt/response |
| `pkg/nim` | `AIAgent` | Agent with tool use |
| `pkg/nim` | `Leaf` | Message interface |
| `pkg/nim` | `Whisperer` | Send messages |
| `pkg/nim` | `Brain` | Knowledge storage |

### AAA Pattern

| Method | Factory | Use Case |
|--------|---------|----------|
| `Advice()` | aiservicefactory | Q&A, explanations |
| `Action()` | aiagentfactory | Tool-based tasks |
| `Automate()` | (future) | Long-running loops |
