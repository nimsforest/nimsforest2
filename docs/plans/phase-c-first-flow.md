# Phase C: First AAA Flow

**Status**: Blocked (requires Phase B complete)
**Goal**: CoderNim dispatches work to AIAgent via events
**Estimated Time**: 3 hours
**Branch**: `claude/plan-agentic-coding-KVNq3`

---

## 🎯 Objective

Implement the complete AAA flow from user request through agent execution to result handling.

**Success Criteria:**
- CoderNim implements Advice(), Action(), Handle()
- Can query for Land capacity
- Can dispatch tasks to AIAgent
- Receives and processes agent results
- End-to-end integration test passes

---

## 🔄 Flow Diagram

```
User Request
    ↓
CoderNim.Action("fix bug X")
    ↓
Whisper("land.query", {needs_docker: true})
    ↓
LandHouse.Process() → responds with "land.info.{id}"
    ↓
CoderNim collects responses (2s timeout)
    ↓
Whisper("agent.task.{land_id}", task)
    ↓
AgentHouse.Process() → runs Docker
    ↓
Returns "agent.result.{task_id}"
    ↓
CoderNim.Handle(result) → processes outcome
```

---

## 📋 Tasks

### Task 9: pkg/nim Interfaces

**Files**: `pkg/nim/*.go`

Create the public interfaces that Nims implement.

#### 9.1: pkg/nim/nim.go

```go
package nim

import (
    "context"
    "errors"
)

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

#### 9.2: pkg/nim/leaf.go

```go
package nim

// Leaf represents an event on the Wind
type Leaf interface {
    GetSubject() string
    GetData() []byte
    GetSource() string
}
```

#### 9.3: pkg/nim/wind.go

```go
package nim

import "context"

// Whisperer publishes Leaves to the Wind
type Whisperer interface {
    Whisper(ctx context.Context, leaf Leaf) error
}
```

#### 9.4: pkg/nim/asker.go

```go
package nim

import "context"

// AIAsker provides prompt → response (for Advice)
type AIAsker interface {
    Ask(ctx context.Context, prompt string) (string, error)
}
```

#### 9.5: pkg/nim/agent.go

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

// Task represents work to be done by an agent
type Task struct {
    ID            string
    Description   string
    Params        map[string]interface{}
    RequiredAgent AgentType // Optional: specify agent type
}

// Result represents the outcome of agent execution
type Result struct {
    Success bool
    Output  string
    Files   []FileDiff
    Error   string
}

// FileDiff represents a file change made by an agent
type FileDiff struct {
    Path   string
    Action string // created, modified, deleted
    Diff   string
}
```

**Validation:**
- All files compile: `go build ./pkg/nim/...`
- Interfaces are minimal and focused
- No circular dependencies

---

### Task 10: AIAgent Implementation

**File**: `internal/ai/agents/ai_agent.go`

Docker-based AI agent that runs containerized tools.

```go
package agents

import (
    "context"
    "fmt"
    "os/exec"

    "github.com/nimsforest/nimsforest2/pkg/nim"
)

// DockerAIAgent runs AI tools in Docker containers
type DockerAIAgent struct {
    config AIAgentConfig
    landID string
}

// AIAgentConfig specifies Docker image and resources
type AIAgentConfig struct {
    Name   string
    Image  string   // nimsforest/claude-agent:latest
    Tools  []string // ["claude"]
    Memory string   // "4g"
    CPU    int      // 2
}

func NewDockerAIAgent(config AIAgentConfig, landID string) *DockerAIAgent {
    return &DockerAIAgent{
        config: config,
        landID: landID,
    }
}

func (a *DockerAIAgent) Run(ctx context.Context, task nim.Task) (*nim.Result, error) {
    // Build docker run command
    args := []string{
        "run", "--rm",
        "-m", a.config.Memory,
        "--cpus", fmt.Sprintf("%d", a.config.CPU),
    }

    // Mount workspace if provided
    if workdir, ok := task.Params["workdir"].(string); ok {
        args = append(args, "-v", fmt.Sprintf("%s:/workspace", workdir))
    }

    // Add API key if provided
    if apiKey, ok := task.Params["api_key"].(string); ok {
        args = append(args, "-e", fmt.Sprintf("ANTHROPIC_API_KEY=%s", apiKey))
    }

    // Add image and task description as command
    args = append(args, a.config.Image, task.Description)

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

func (a *DockerAIAgent) Type() nim.AgentType {
    return nim.AgentTypeAI
}

func (a *DockerAIAgent) Available(ctx context.Context) bool {
    // Check if Docker is available
    cmd := exec.CommandContext(ctx, "docker", "info")
    return cmd.Run() == nil
}

func (a *DockerAIAgent) Image() string {
    return a.config.Image
}

func (a *DockerAIAgent) Tools() []string {
    return a.config.Tools
}
```

**Validation:**
- Compiles without errors
- Docker command construction is correct
- Handles missing parameters gracefully

---

### Task 11: Wire AgentHouse to Use AIAgent

**File**: `internal/treehouses/agenthouse.go` (update)

Modify AgentHouse to use the AIAgent implementation.

```go
// Add to AgentHouse struct
type AgentHouse struct {
    landID string
    agent  nim.Agent  // Add this
}

// Update NewAgentHouse
func NewAgentHouse(landID string, agent nim.Agent) *AgentHouse {
    return &AgentHouse{
        landID: landID,
        agent:  agent,
    }
}

// Update runDocker to use agent
func (ah *AgentHouse) runDocker(task AgentTask) AgentResult {
    ctx := context.Background()

    // Convert AgentTask to nim.Task
    nimTask := nim.Task{
        ID:          task.ID,
        Description: task.Command,
        Params: map[string]interface{}{
            "workdir": task.Workdir,
        },
    }

    // Add env vars to params
    for k, v := range task.Env {
        nimTask.Params[k] = v
    }

    // Run via agent
    result, err := ah.agent.Run(ctx, nimTask)
    if err != nil {
        return AgentResult{
            TaskID:  task.ID,
            Success: false,
            Error:   err.Error(),
        }
    }

    return AgentResult{
        TaskID:  task.ID,
        Success: result.Success,
        Output:  result.Output,
        Error:   result.Error,
    }
}
```

**Update Forest wiring** (`pkg/runtime/forest.go`):

```go
// In Start() method, when creating AgentHouse
if f.thisLand.HasDocker {
    // Create AIAgent
    aiAgent := agents.NewDockerAIAgent(
        agents.AIAgentConfig{
            Name:   "default-ai-agent",
            Image:  "alpine:latest", // Start with simple image for testing
            Memory: "512m",
            CPU:    1,
        },
        f.thisLand.ID,
    )

    f.agentHouse = treehouses.NewAgentHouse(f.thisLand.ID, aiAgent)
    if err := f.startTreeHouse(ctx, f.agentHouse); err != nil {
        return fmt.Errorf("failed to start AgentHouse: %w", err)
    }
}
```

---

### Task 12: CoderNim Skeleton

**File**: `internal/nims/coder/coder.go`

Create the CoderNim with basic structure.

```go
package coder

import (
    "context"
    "fmt"
    "strings"
    "time"

    "github.com/nimsforest/nimsforest2/internal/core"
    "github.com/nimsforest/nimsforest2/pkg/nim"
)

// CoderNim is an intelligent agent that writes and fixes code
type CoderNim struct {
    *core.BaseNim
    asker    nim.AIAsker
    wind     nim.Whisperer
    landID   string
    pendingTasks map[string]chan *nim.Result
}

func New(base *core.BaseNim, asker nim.AIAsker, wind nim.Whisperer, landID string) *CoderNim {
    return &CoderNim{
        BaseNim:      base,
        asker:        asker,
        wind:         wind,
        landID:       landID,
        pendingTasks: make(map[string]chan *nim.Result),
    }
}

func (c *CoderNim) Name() string {
    return "coder"
}

func (c *CoderNim) Subjects() []string {
    return []string{"code.request", "agent.result.>"}
}

// Start subscribes to relevant subjects
func (c *CoderNim) Start(ctx context.Context) error {
    // BaseNim.Start() would handle subscriptions
    return c.BaseNim.Start(ctx)
}

func (c *CoderNim) Stop() error {
    return c.BaseNim.Stop()
}
```

**Validation:**
- Compiles without errors
- Struct fields are appropriate
- Lifecycle methods defined

---

### Task 13: CoderNim.Advice() Implementation

**File**: `internal/nims/coder/coder.go` (continue)

Implement simple Q&A via existing aiservice.

```go
// Advice - simple Q&A via AI service
func (c *CoderNim) Advice(ctx context.Context, query string) (string, error) {
    return c.asker.Ask(ctx, query)
}
```

**Create Asker wrapper** (`internal/ai/asker.go`):

```go
package ai

import (
    "context"

    "github.com/nimsforest/nimsforest2/pkg/integrations/aiservice"
    "github.com/nimsforest/nimsforest2/pkg/nim"
)

// Asker wraps existing aiservice to implement nim.AIAsker
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

**Validation:**
- Can call `coder.Advice("what is 2+2?")` and get response
- Uses existing aiservice package

---

### Task 14: CoderNim.Action() Implementation

**File**: `internal/nims/coder/coder.go` (continue)

Implement the event-driven action dispatch.

```go
// Action - execute via Agent (event-driven)
func (c *CoderNim) Action(ctx context.Context, action string, params map[string]interface{}) (interface{}, error) {
    taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())

    // Create result channel for this task
    resultChan := make(chan *nim.Result, 1)
    c.pendingTasks[taskID] = resultChan

    // Build agent task
    task := AgentTaskData{
        ID:      taskID,
        Image:   getImage(params),
        Command: action,
        Workdir: getString(params, "workdir"),
        Env:     getEnv(params),
    }

    // Dispatch to agent via Wind
    taskData, _ := json.Marshal(task)
    taskLeaf := core.NewLeaf(
        fmt.Sprintf("agent.task.%s", c.landID),
        taskData,
        fmt.Sprintf("nim:coder"),
    )

    if err := c.wind.Whisper(ctx, taskLeaf); err != nil {
        delete(c.pendingTasks, taskID)
        return nil, fmt.Errorf("failed to dispatch task: %w", err)
    }

    // Wait for result (with timeout)
    select {
    case result := <-resultChan:
        return result, nil
    case <-time.After(30 * time.Second):
        delete(c.pendingTasks, taskID)
        return nil, fmt.Errorf("task timeout waiting for result")
    case <-ctx.Done():
        delete(c.pendingTasks, taskID)
        return nil, ctx.Err()
    }
}

// Helper functions
func getImage(params map[string]interface{}) string {
    if img, ok := params["image"].(string); ok {
        return img
    }
    return "alpine:latest" // Default
}

func getString(params map[string]interface{}, key string) string {
    if val, ok := params[key].(string); ok {
        return val
    }
    return ""
}

func getEnv(params map[string]interface{}) map[string]string {
    if env, ok := params["env"].(map[string]string); ok {
        return env
    }
    return nil
}

type AgentTaskData struct {
    ID      string            `json:"id"`
    Image   string            `json:"image"`
    Command string            `json:"command"`
    Workdir string            `json:"workdir,omitempty"`
    Env     map[string]string `json:"env,omitempty"`
}
```

**Validation:**
- Action dispatches task and waits for result
- Timeout prevents hanging forever
- Task ID correlates request with response

---

### Task 15: CoderNim.Handle() Implementation

**File**: `internal/nims/coder/coder.go` (continue)

Handle incoming events, especially agent results.

```go
// Handle processes incoming Leaves
func (c *CoderNim) Handle(ctx context.Context, leaf nim.Leaf) error {
    subject := leaf.GetSubject()

    // Handle agent results
    if strings.HasPrefix(subject, "agent.result.") {
        return c.handleAgentResult(ctx, leaf)
    }

    // Handle code requests
    if subject == "code.request" {
        return c.handleCodeRequest(ctx, leaf)
    }

    return nil
}

func (c *CoderNim) handleAgentResult(ctx context.Context, leaf nim.Leaf) error {
    var result AgentResultData
    if err := json.Unmarshal(leaf.GetData(), &result); err != nil {
        return fmt.Errorf("failed to unmarshal agent result: %w", err)
    }

    // Find pending task
    if resultChan, ok := c.pendingTasks[result.TaskID]; ok {
        // Convert to nim.Result
        nimResult := &nim.Result{
            Success: result.Success,
            Output:  result.Output,
            Error:   result.Error,
        }

        // Send to waiting Action()
        select {
        case resultChan <- nimResult:
            // Sent successfully
        default:
            // Channel full or closed, ignore
        }

        // Cleanup
        delete(c.pendingTasks, result.TaskID)
    }

    return nil
}

func (c *CoderNim) handleCodeRequest(ctx context.Context, leaf nim.Leaf) error {
    // Future: handle external code requests
    // For now, just log
    return nil
}

type AgentResultData struct {
    TaskID  string `json:"task_id"`
    Success bool   `json:"success"`
    Output  string `json:"output"`
    Error   string `json:"error,omitempty"`
}
```

**Implement Automate stub**:

```go
// Automate - not implemented in Phase C
func (c *CoderNim) Automate(ctx context.Context, automation string, enabled bool) (*nim.AutomateResult, error) {
    return nil, nim.ErrNotSupported
}
```

**Validation:**
- Handle correctly routes to handleAgentResult
- Result channels work correctly
- No race conditions on pendingTasks map

---

### Task 16: Integration Test

**File**: `internal/nims/coder/coder_test.go`

End-to-end test of the complete flow.

```go
package coder

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    // ... other imports
)

func TestCoderNim_FirstFlow(t *testing.T) {
    // Skip if Docker not available
    if !hasDocker(t) {
        t.Skip("Docker not available")
    }

    ctx := context.Background()

    // Setup test infrastructure
    wind := setupTestWind(t)
    asker := setupTestAsker(t)
    landID := "test-land-1"

    // Create CoderNim
    baseNim := setupTestBaseNim(t)
    coder := New(baseNim, asker, wind, landID)

    // Start CoderNim
    err := coder.Start(ctx)
    require.NoError(t, err)
    defer coder.Stop()

    t.Run("Advice works", func(t *testing.T) {
        answer, err := coder.Advice(ctx, "What is 2+2?")
        require.NoError(t, err)
        assert.NotEmpty(t, answer)
        t.Logf("AI response: %s", answer)
    })

    t.Run("Action dispatches and receives result", func(t *testing.T) {
        result, err := coder.Action(ctx, "echo 'hello from docker'", map[string]interface{}{
            "image": "alpine:latest",
        })
        require.NoError(t, err)
        require.NotNil(t, result)

        nimResult := result.(*nim.Result)
        assert.True(t, nimResult.Success)
        assert.Contains(t, nimResult.Output, "hello from docker")
        t.Logf("Docker output: %s", nimResult.Output)
    })

    t.Run("Action with workspace mount", func(t *testing.T) {
        tmpDir := t.TempDir()

        result, err := coder.Action(ctx, "ls -la /workspace", map[string]interface{}{
            "image":   "alpine:latest",
            "workdir": tmpDir,
        })
        require.NoError(t, err)

        nimResult := result.(*nim.Result)
        assert.True(t, nimResult.Success)
        t.Logf("Workspace listing: %s", nimResult.Output)
    })
}

// Test helpers
func hasDocker(t *testing.T) bool {
    cmd := exec.Command("docker", "info")
    return cmd.Run() == nil
}

func setupTestWind(t *testing.T) nim.Whisperer {
    // Create test NATS or mock Wind
    // ...
}

func setupTestAsker(t *testing.T) nim.AIAsker {
    // Create mock or real AI asker
    // ...
}

func setupTestBaseNim(t *testing.T) *core.BaseNim {
    // Create BaseNim for testing
    // ...
}
```

**Validation:**
- All tests pass
- Docker executes successfully
- Results arrive via Handle()
- No race conditions or deadlocks

---

## 🚦 Milestones

- [ ] **M4**: CoderNim.Advice() works
  - **Test**: `coder.Advice("what is 2+2?")` returns answer

- [ ] **M5**: CoderNim.Action() dispatches and receives results
  - **Test**: `coder.Action("echo hello")` executes in Docker

- [ ] **M6**: End-to-end integration test passes
  - **Test**: `TestCoderNimFirstFlow` passes all subtests

---

## 📝 Implementation Order

**Recommended sequence:**

1. Task 9: Create pkg/nim interfaces (30 min)
   - All .go files in pkg/nim/
   - Compile check

2. Task 10: AIAgent implementation (30 min)
   - internal/ai/agents/ai_agent.go
   - Unit test with mock task

3. Task 13: Asker wrapper (15 min)
   - internal/ai/asker.go
   - Test with real AI service

4. Task 12: CoderNim skeleton (20 min)
   - internal/nims/coder/coder.go
   - Basic structure only

5. Task 13: Advice() implementation (10 min)
   - Add to coder.go
   - Test standalone

6. Task 14: Action() implementation (45 min)
   - Add to coder.go
   - Complex event handling

7. Task 15: Handle() implementation (30 min)
   - Add to coder.go
   - Result correlation logic

8. Task 11: Wire AgentHouse (20 min)
   - Update agenthouse.go
   - Update forest.go

9. Task 16: Integration test (40 min)
   - coder_test.go
   - Debug end-to-end flow

**Total: ~3 hours**

---

## ⚠️ Critical Points

### Thread Safety
- `pendingTasks` map needs mutex if Handle() and Action() run concurrently
- Consider using `sync.Map` or add `sync.RWMutex`

### Timeout Handling
- Action() waits up to 30s for result
- AgentHouse has no timeout (Docker command has its own)
- Consider adding task-level timeout in AgentTask

### Error Propagation
- Agent errors return in Result.Error, not as Go error
- Distinguish between "task failed" vs "system error"

---

## ✅ Definition of Done

- [ ] All tasks (9-16) completed
- [ ] pkg/nim interfaces compile
- [ ] AIAgent executes Docker tasks
- [ ] CoderNim implements Advice() and Action()
- [ ] CoderNim.Handle() processes results
- [ ] Integration test passes
- [ ] Code committed to branch
- [ ] Ready for Phase D (cleanup and expansion)

---

## 🎓 What You'll Learn

1. **Event-driven programming**: Request/response via pub/sub
2. **Channel-based coordination**: Correlating async responses
3. **Interface design**: Minimal, focused contracts
4. **Integration testing**: End-to-end validation

---

## 🔗 Dependencies

**Must be complete:**
- Phase A: Land detection
- Phase B: Houses and communication

**Required files:**
- `internal/core/land.go`
- `internal/treehouses/landhouse.go`
- `internal/treehouses/agenthouse.go`
- `pkg/integrations/aiservice/` (existing)

---

**Previous Phase**: [phase-b-communication.md](./phase-b-communication.md)
**Next Phase**: Phase D (future work - expansion and cleanup)
