# AAA Nim Implementation Roadmap

**Status**: In Progress
**Goal**: Implement the AAA (Advice/Action/Automate) pattern for Nims in nimsforest2
**Reference**: See [plan-aaa-nim.md](./plan-aaa-nim.md) for complete architecture

---

## 🎯 First Flow Objective

Get **CoderNim** working end-to-end with this flow:

```
User Request
    ↓
CoderNim.Action("fix bug X")
    ↓
Query Land via: "land.query" {needs_docker: true}
    ↓
LandHouse responds: "land.info.{id}"
    ↓
Dispatch task via: "agent.task.{land_id}"
    ↓
AgentHouse executes in Docker container
    ↓
Result arrives via: "agent.result.{task_id}"
    ↓
CoderNim.Handle(result)
```

**Why this flow?** It exercises the complete AAA infrastructure:
- Land detection and capacity queries
- Event-driven communication via Wind
- Agent execution in Docker
- Nim intelligence (AI + Action)

---

## 📋 Three-Phase Orchestration

### Phase A: Foundation (Get Land Standing)
**Goal**: Know what compute we're running on

| Task | File | Output |
|------|------|--------|
| 1. Land data structures | `internal/core/land.go` | LandInfo, LandType constants |
| 2. Land detection logic | `internal/land/detect.go` | Detect RAM, CPU, Docker, GPU |
| 3. Wire into Forest | `pkg/runtime/forest.go` | Log detected Land on startup |
| 4. TreeHouse interface | `internal/treehouses/interface.go` | GoTreeHouse interface |

**Validation**: Forest logs `"Running on nimland: node-1"` on startup

---

### Phase B: Communication Infrastructure
**Goal**: Lands can talk via Wind (event-driven)

| Task | File | Output |
|------|------|--------|
| 5. LandHouse | `internal/treehouses/landhouse.go` | Responds to `land.query` |
| 6. AgentHouse skeleton | `internal/treehouses/agenthouse.go` | Handles `agent.task.*` |
| 7. Wire Houses to Forest | `pkg/runtime/forest.go` | Houses subscribe and log |
| 8. Test Land queries | `internal/treehouses/landhouse_test.go` | Unit test passes |

**Validation**: Can publish `land.query` and receive `land.info.*` response

---

### Phase C: First AAA Flow
**Goal**: CoderNim dispatches work to AIAgent via events

| Task | File | Output |
|------|------|--------|
| 9. pkg/nim interfaces | `pkg/nim/*.go` | Nim, Agent, AIAsker, Leaf, Wind |
| 10. AIAgent implementation | `internal/ai/agents/ai_agent.go` | Docker-based AI agent |
| 11. Wire AgentHouse | `internal/treehouses/agenthouse.go` | Calls AIAgent.Run() |
| 12. CoderNim skeleton | `internal/nims/coder/coder.go` | BaseNim + AAA methods |
| 13. CoderNim.Advice() | Same | Uses existing aiservice |
| 14. CoderNim.Action() | Same | Queries Land, dispatches agent |
| 15. CoderNim.Handle() | Same | Processes agent results |
| 16. Integration test | `internal/nims/coder/coder_test.go` | **END-TO-END WORKS** |

**Validation**: Complete flow from user request → Docker execution → result

---

## 🔧 Detailed Implementation Order

### Step 1: Land Detection (30 min)

**Create `internal/core/land.go`:**
```go
type LandType string
const (
    LandTypeBase     LandType = "land"
    LandTypeNimland  LandType = "nimland"
    LandTypeManaland LandType = "manaland"
)

type LandInfo struct {
    ID         string   `json:"id"`
    Name       string   `json:"name"`
    Type       LandType `json:"type"`
    Hostname   string   `json:"hostname"`
    RAMTotal   uint64   `json:"ram_total"`
    CPUCores   int      `json:"cpu_cores"`
    CPUModel   string   `json:"cpu_model"`
    HasDocker  bool     `json:"has_docker"`
    GPUVram    uint64   `json:"gpu_vram,omitempty"`
}
```

**Create `internal/land/detect.go`:**
```go
func Detect(natsID, natsName string) *core.LandInfo {
    info := &core.LandInfo{ID: natsID, Name: natsName}

    // Use gopsutil for RAM/CPU (already in go.mod)
    vmStat, _ := mem.VirtualMemory()
    info.RAMTotal = vmStat.Total
    info.CPUCores = runtime.NumCPU()

    // Probe for Docker
    info.HasDocker = detectDocker()

    // Probe for GPU
    detectGPU(info)

    // Determine type
    info.Type = determineType(info)

    return info
}
```

**Wire to Forest:**
```go
// pkg/runtime/forest.go - in Start()
varz, _ := f.natsServer.InternalServer().Varz(&server.VarzOptions{})
f.thisLand = land.Detect(varz.ID, varz.Name)
log.Printf("[Forest] Running on %s: %s (RAM: %s, CPU: %d cores, Docker: %v)",
    f.thisLand.Type, f.thisLand.Name,
    formatBytes(f.thisLand.RAMTotal), f.thisLand.CPUCores, f.thisLand.HasDocker)
```

**Test**: Run Forest, verify log output

---

### Step 2: TreeHouse Interface (15 min)

**Create `internal/treehouses/interface.go`:**
```go
package treehouses

import "github.com/nimsforest/nimsforest2/internal/core"

// GoTreeHouse is a compile-time TreeHouse implemented in Go.
// Must be deterministic: same input Leaf = same output Leaf.
type GoTreeHouse interface {
    Name() string
    Subjects() []string
    Process(leaf core.Leaf) *core.Leaf  // nil = no output
}
```

---

### Step 3: LandHouse (30 min)

**Create `internal/treehouses/landhouse.go`:**
```go
type LandHouse struct {
    land *core.LandInfo
}

func NewLandHouse(land *core.LandInfo) *LandHouse {
    return &LandHouse{land: land}
}

func (lh *LandHouse) Name() string {
    return "landhouse"
}

func (lh *LandHouse) Subjects() []string {
    return []string{"land.query"}
}

func (lh *LandHouse) Process(leaf core.Leaf) *core.Leaf {
    var query CapacityQuery
    if err := json.Unmarshal(leaf.GetData(), &query); err != nil {
        return nil
    }

    // Check if we match
    if query.NeedsDocker && !lh.land.HasDocker {
        return nil
    }
    if query.NeedsGPU && lh.land.GPUVram == 0 {
        return nil
    }

    // Return our Land info
    data, _ := json.Marshal(lh.land)
    return core.NewLeaf(
        fmt.Sprintf("land.info.%s", lh.land.ID),
        data,
        "treehouse:landhouse",
    )
}

type CapacityQuery struct {
    NeedsDocker bool   `json:"needs_docker"`
    NeedsGPU    bool   `json:"needs_gpu"`
    MinRAM      uint64 `json:"min_ram,omitempty"`
}
```

**Wire to Forest:**
```go
// pkg/runtime/forest.go - in Start()
f.landHouse = treehouses.NewLandHouse(f.thisLand)
f.startTreeHouse(f.landHouse)

func (f *Forest) startTreeHouse(house treehouses.GoTreeHouse) error {
    for _, subject := range house.Subjects() {
        _, err := f.wind.Catch(subject, func(leaf core.Leaf) {
            if result := house.Process(leaf); result != nil {
                f.wind.Whisper(context.Background(), result)
            }
        })
        if err != nil {
            return err
        }
    }
    log.Printf("[Forest] Started %s on subjects: %v", house.Name(), house.Subjects())
    return nil
}
```

**Test**: Publish `land.query`, verify response

---

### Step 4: AgentHouse (45 min)

**Create `internal/treehouses/agenthouse.go`:**
```go
type AgentHouse struct {
    landID string
}

func NewAgentHouse(landID string) *AgentHouse {
    return &AgentHouse{landID: landID}
}

func (ah *AgentHouse) Name() string {
    return "agenthouse"
}

func (ah *AgentHouse) Subjects() []string {
    return []string{fmt.Sprintf("agent.task.%s", ah.landID)}
}

func (ah *AgentHouse) Process(leaf core.Leaf) *core.Leaf {
    var task AgentTask
    if err := json.Unmarshal(leaf.GetData(), &task); err != nil {
        return nil
    }

    // Run in Docker
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

**Wire to Forest:**
```go
// Only on Nimland/Manaland
if f.thisLand.HasDocker {
    f.agentHouse = treehouses.NewAgentHouse(f.thisLand.ID)
    f.startTreeHouse(f.agentHouse)
}
```

---

### Step 5: pkg/nim Interfaces (30 min)

**Create `pkg/nim/nim.go`:**
```go
package nim

type Nim interface {
    Name() string

    // AAA Model
    Advice(ctx context.Context, query string) (string, error)
    Action(ctx context.Context, action string, params map[string]interface{}) (interface{}, error)
    Automate(ctx context.Context, automation string, enabled bool) (*AutomateResult, error)

    // Event handling
    Handle(ctx context.Context, leaf Leaf) error

    // Lifecycle
    Start(ctx context.Context) error
    Stop() error
}

var ErrNotSupported = errors.New("operation not supported by this nim")
```

**Create other interfaces**: `agent.go`, `asker.go`, `leaf.go`, `wind.go`

---

### Step 6: CoderNim (60 min)

**Create `internal/nims/coder/coder.go`:**
```go
type CoderNim struct {
    *core.BaseNim
    asker  nim.AIAsker
    wind   nim.Whisperer
    forest *runtime.Forest
}

func (c *CoderNim) Advice(ctx context.Context, query string) (string, error) {
    return c.asker.Ask(ctx, query)
}

func (c *CoderNim) Action(ctx context.Context, action string, params map[string]interface{}) (interface{}, error) {
    // 1. Query for capacity
    // 2. Wait for responses
    // 3. Dispatch to agent
    // 4. Return immediately (result comes via Handle)
}

func (c *CoderNim) Automate(ctx context.Context, automation string, enabled bool) (*nim.AutomateResult, error) {
    return nil, nim.ErrNotSupported  // For now
}

func (c *CoderNim) Handle(ctx context.Context, leaf nim.Leaf) error {
    subject := leaf.GetSubject()

    if strings.HasPrefix(subject, "agent.result.") {
        return c.handleAgentResult(ctx, leaf)
    }

    return nil
}
```

---

### Step 7: Integration Test (30 min)

**Create `internal/nims/coder/coder_test.go`:**
```go
func TestCoderNimFirstFlow(t *testing.T) {
    // Start Forest with Land detection
    forest := setupTestForest(t)
    ctx := context.Background()
    forest.Start(ctx)
    defer forest.Stop()

    // Create CoderNim
    coder := setupCoderNim(t, forest)
    coder.Start(ctx)
    defer coder.Stop()

    // Test Advice
    answer, err := coder.Advice(ctx, "What is 2+2?")
    assert.NoError(t, err)
    assert.Contains(t, answer, "4")

    // Test Action (dispatch to Docker)
    result, err := coder.Action(ctx, "echo hello from docker", map[string]interface{}{
        "workdir": "/tmp",
        "image": "alpine:latest",
    })
    assert.NoError(t, err)
    // Wait for result via Handle
    time.Sleep(2 * time.Second)
    // Verify result was processed
}
```

---

## 🚦 Milestones & Validation

- [ ] **M1**: Forest detects and logs Land type
  - **Test**: Run `./forest` and see log like `"Running on nimland: node-1"`

- [ ] **M2**: LandHouse responds to queries
  - **Test**: Unit test publishes `land.query`, receives `land.info.*`

- [ ] **M3**: AgentHouse executes Docker tasks
  - **Test**: Unit test publishes `agent.task.*`, receives `agent.result.*`

- [ ] **M4**: CoderNim.Advice() works
  - **Test**: `coder.Advice("what is 2+2?")` returns answer

- [ ] **M5**: CoderNim.Action() dispatches and receives results
  - **Test**: `coder.Action("echo hello")` executes in Docker

- [ ] **M6**: End-to-end integration test passes
  - **Test**: `TestCoderNimFirstFlow` passes

---

## ⚠️ Critical Decisions

### What to Include in First Flow
✅ **Include:**
- Land detection (RAM, CPU, Docker, GPU)
- LandHouse and AgentHouse (GoTreeHouses)
- AIAgent (Docker-based)
- CoderNim with Advice() and Action()
- Existing aiservice package

### What to Defer
❌ **Defer:**
- Human, Robot, Browser agents → Phase 2
- Songbird.Send() extension → Phase 2
- CoderNim.Automate() → Phase 2
- Reorganizing examples to `examples/` → Cleanup phase
- Moving `pkg/brain/` → Later refactor
- Renaming AfterSales/General "Nims" → After first flow works

### Dependencies to Keep
- `pkg/integrations/aiservice/` - Keep as-is
- `gopsutil` - Already in use for detection
- Existing Wind, River, Leaf interfaces

---

## 📊 Progress Tracking

**Current Status**: [ ] Not Started

**Time Estimates:**
- Phase A: ~45 minutes
- Phase B: ~1.5 hours
- Phase C: ~3 hours
- **Total**: ~5 hours for first working flow

**Blockers**: None identified yet

**Next Immediate Action**: Start with Step 1 (Land Detection)

---

## 🔄 Resume Points

If interrupted, resume from:
1. **Last completed milestone** (see checkboxes above)
2. **Last file created** (check git status)
3. **Last passing test** (run test suite)

---

## 📝 Notes & Learnings

*Document discoveries and decisions here as we implement*

---

**Last Updated**: 2026-01-10
**Branch**: `claude/plan-core-implementation-w1Ei8`
