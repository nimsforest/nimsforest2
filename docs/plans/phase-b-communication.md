# Phase B: Communication Infrastructure

**Status**: Ready to Start
**Goal**: Enable Lands to communicate capacity via Wind (event-driven)
**Estimated Time**: 1.5 hours
**Branch**: `claude/plan-agentic-coding-KVNq3`

---

## 🎯 Objective

Implement the event-driven communication layer for Land capacity queries and agent task dispatch.

**Success Criteria:**
- LandHouse responds to `land.query` with `land.info.*`
- AgentHouse handles `agent.task.*` and returns `agent.result.*`
- Both Houses wire into Forest lifecycle
- Unit tests validate request/response flow

---

## 📋 Tasks

### Task 5: LandHouse Implementation

**File**: `internal/treehouses/landhouse.go`

**Requirements:**
- Implement `GoTreeHouse` interface
- Subscribe to `land.query` subject
- Match capacity queries against `thisLand`
- Respond with `land.info.{id}` if match

**Implementation:**

```go
package treehouses

import (
    "encoding/json"
    "fmt"
    "github.com/nimsforest/nimsforest2/internal/core"
)

// LandHouse is a compile-time TreeHouse that handles Land capacity queries.
// Deterministic: same query + same Land capabilities = same response
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

// Process handles a land.query Leaf and returns land.info.{id} if we match
func (lh *LandHouse) Process(leaf core.Leaf) *core.Leaf {
    var query CapacityQuery
    if err := json.Unmarshal(leaf.GetData(), &query); err != nil {
        return nil
    }

    // Deterministic matching logic
    if query.NeedsDocker && !lh.land.HasDocker {
        return nil // Don't respond
    }
    if query.NeedsGPU && lh.land.GPUVram == 0 {
        return nil // Don't respond
    }
    if query.MinRAM > 0 && lh.land.RAMTotal < query.MinRAM {
        return nil // Not enough RAM
    }

    // Return our Land info
    data, _ := json.Marshal(lh.land)
    return core.NewLeaf(
        fmt.Sprintf("land.info.%s", lh.land.ID),
        data,
        "treehouse:landhouse",
    )
}

// CapacityQuery specifies required Land capabilities
type CapacityQuery struct {
    NeedsDocker bool   `json:"needs_docker"`
    NeedsGPU    bool   `json:"needs_gpu"`
    MinRAM      uint64 `json:"min_ram,omitempty"`
}
```

**Validation:**
- File compiles without errors
- Implements all `GoTreeHouse` methods
- Logic correctly filters based on capabilities

---

### Task 6: AgentHouse Skeleton Implementation

**File**: `internal/treehouses/agenthouse.go`

**Requirements:**
- Implement `GoTreeHouse` interface
- Subscribe to `agent.task.{land_id}` subject
- Execute tasks in Docker containers
- Return results via `agent.result.{task_id}`

**Implementation:**

```go
package treehouses

import (
    "encoding/json"
    "fmt"
    "os/exec"
    "github.com/nimsforest/nimsforest2/internal/core"
)

// AgentHouse is a compile-time TreeHouse that executes agent tasks in Docker.
// Deterministic dispatch: task → Docker → result
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

// Process handles an agent.task.{land_id} Leaf and returns agent.result.{task_id}
func (ah *AgentHouse) Process(leaf core.Leaf) *core.Leaf {
    var task AgentTask
    if err := json.Unmarshal(leaf.GetData(), &task); err != nil {
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

// runDocker executes a task in a Docker container
func (ah *AgentHouse) runDocker(task AgentTask) AgentResult {
    args := []string{"run", "--rm"}

    // Mount workspace if specified
    if task.Workdir != "" {
        args = append(args, "-v", fmt.Sprintf("%s:/workspace", task.Workdir))
    }

    // Add environment variables
    for k, v := range task.Env {
        args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
    }

    // Add image and command
    args = append(args, task.Image)
    if task.Command != "" {
        args = append(args, "sh", "-c", task.Command)
    }

    cmd := exec.Command("docker", args...)
    output, err := cmd.CombinedOutput()

    return AgentResult{
        TaskID:  task.ID,
        Success: err == nil,
        Output:  string(output),
        Error:   errorString(err),
    }
}

func errorString(err error) string {
    if err != nil {
        return err.Error()
    }
    return ""
}

// AgentTask represents a task to execute
type AgentTask struct {
    ID      string            `json:"id"`
    Image   string            `json:"image"`
    Command string            `json:"command,omitempty"`
    Workdir string            `json:"workdir,omitempty"`
    Env     map[string]string `json:"env,omitempty"`
}

// AgentResult represents the outcome of a task
type AgentResult struct {
    TaskID  string `json:"task_id"`
    Success bool   `json:"success"`
    Output  string `json:"output"`
    Error   string `json:"error,omitempty"`
}
```

**Validation:**
- File compiles without errors
- Docker command construction is correct
- Handles missing Docker gracefully (returns error in result)

---

### Task 7: Wire Houses to Forest

**File**: `pkg/runtime/forest.go`

**Requirements:**
- Add fields for LandHouse and AgentHouse
- Create helper method to start GoTreeHouses
- Start LandHouse on all nodes
- Start AgentHouse only on Nimland/Manaland

**Changes:**

```go
// Add to Forest struct
type Forest struct {
    // ... existing fields ...

    thisLand   *core.LandInfo
    landHouse  treehouses.GoTreeHouse
    agentHouse treehouses.GoTreeHouse
}

// Add to Start() method, after existing initialization
func (f *Forest) Start(ctx context.Context) error {
    // ... existing startup code ...

    // Start LandHouse (all nodes)
    f.landHouse = treehouses.NewLandHouse(f.thisLand)
    if err := f.startTreeHouse(ctx, f.landHouse); err != nil {
        return fmt.Errorf("failed to start LandHouse: %w", err)
    }

    // Start AgentHouse (only Nimland/Manaland)
    if f.thisLand.HasDocker {
        f.agentHouse = treehouses.NewAgentHouse(f.thisLand.ID)
        if err := f.startTreeHouse(ctx, f.agentHouse); err != nil {
            return fmt.Errorf("failed to start AgentHouse: %w", err)
        }
    }

    // ... rest of startup ...
}

// Add new helper method
// startTreeHouse subscribes a GoTreeHouse to its subjects
func (f *Forest) startTreeHouse(ctx context.Context, house treehouses.GoTreeHouse) error {
    for _, subject := range house.Subjects() {
        _, err := f.wind.Catch(subject, func(leaf core.Leaf) {
            if result := house.Process(leaf); result != nil {
                if err := f.wind.Whisper(ctx, result); err != nil {
                    f.logger.Printf("[Forest] Failed to whisper result from %s: %v",
                        house.Name(), err)
                }
            }
        })
        if err != nil {
            return fmt.Errorf("failed to subscribe %s to %s: %w",
                house.Name(), subject, err)
        }
    }
    f.logger.Printf("[Forest] Started %s on subjects: %v", house.Name(), house.Subjects())
    return nil
}
```

**Validation:**
- Forest starts without errors
- Log shows: `"Started landhouse on subjects: [land.query]"`
- Log shows: `"Started agenthouse on subjects: [agent.task.{id}]"` (if Docker available)

---

### Task 8: Unit Tests for LandHouse

**File**: `internal/treehouses/landhouse_test.go`

**Requirements:**
- Test capacity query matching logic
- Test response format
- Test filtering (Docker, GPU, RAM)

**Implementation:**

```go
package treehouses

import (
    "encoding/json"
    "testing"
    "github.com/nimsforest/nimsforest2/internal/core"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestLandHouse_ProcessMatchesCapacity(t *testing.T) {
    tests := []struct {
        name      string
        land      *core.LandInfo
        query     CapacityQuery
        shouldMatch bool
    }{
        {
            name: "matches docker requirement",
            land: &core.LandInfo{
                ID:        "test-1",
                HasDocker: true,
                RAMTotal:  8 * 1024 * 1024 * 1024,
            },
            query: CapacityQuery{
                NeedsDocker: true,
            },
            shouldMatch: true,
        },
        {
            name: "rejects docker requirement when unavailable",
            land: &core.LandInfo{
                ID:        "test-2",
                HasDocker: false,
            },
            query: CapacityQuery{
                NeedsDocker: true,
            },
            shouldMatch: false,
        },
        {
            name: "matches GPU requirement",
            land: &core.LandInfo{
                ID:        "test-3",
                HasDocker: true,
                GPUVram:   24 * 1024 * 1024 * 1024,
            },
            query: CapacityQuery{
                NeedsDocker: true,
                NeedsGPU:    true,
            },
            shouldMatch: true,
        },
        {
            name: "rejects GPU requirement when unavailable",
            land: &core.LandInfo{
                ID:        "test-4",
                HasDocker: true,
                GPUVram:   0,
            },
            query: CapacityQuery{
                NeedsGPU: true,
            },
            shouldMatch: false,
        },
        {
            name: "matches RAM requirement",
            land: &core.LandInfo{
                ID:       "test-5",
                RAMTotal: 16 * 1024 * 1024 * 1024,
            },
            query: CapacityQuery{
                MinRAM: 8 * 1024 * 1024 * 1024,
            },
            shouldMatch: true,
        },
        {
            name: "rejects insufficient RAM",
            land: &core.LandInfo{
                ID:       "test-6",
                RAMTotal: 4 * 1024 * 1024 * 1024,
            },
            query: CapacityQuery{
                MinRAM: 8 * 1024 * 1024 * 1024,
            },
            shouldMatch: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            house := NewLandHouse(tt.land)

            queryData, err := json.Marshal(tt.query)
            require.NoError(t, err)

            leaf := core.NewLeaf("land.query", queryData, "test")
            result := house.Process(*leaf)

            if tt.shouldMatch {
                require.NotNil(t, result, "expected response but got nil")
                assert.Equal(t, "land.info."+tt.land.ID, result.GetSubject())

                var landInfo core.LandInfo
                err := json.Unmarshal(result.GetData(), &landInfo)
                require.NoError(t, err)
                assert.Equal(t, tt.land.ID, landInfo.ID)
            } else {
                assert.Nil(t, result, "expected no response but got one")
            }
        })
    }
}

func TestLandHouse_Name(t *testing.T) {
    land := &core.LandInfo{ID: "test"}
    house := NewLandHouse(land)
    assert.Equal(t, "landhouse", house.Name())
}

func TestLandHouse_Subjects(t *testing.T) {
    land := &core.LandInfo{ID: "test"}
    house := NewLandHouse(land)
    assert.Equal(t, []string{"land.query"}, house.Subjects())
}
```

**Validation:**
- All tests pass: `go test ./internal/treehouses/`
- Coverage includes all matching logic paths

---

## 🚦 Milestones

- [ ] **M2**: LandHouse responds to queries
  - **Test**: Unit test passes for capacity matching
  - **Test**: Can publish `land.query` and receive `land.info.*`

- [ ] **M3**: AgentHouse executes Docker tasks
  - **Test**: Unit test publishes `agent.task.*`, receives `agent.result.*`
  - **Test**: Docker container runs and returns output

- [ ] **M4**: Houses wire into Forest lifecycle
  - **Test**: Forest starts without errors
  - **Test**: Logs show both Houses starting
  - **Test**: Subscriptions are active

---

## 🔍 Integration Testing

After completing all tasks, validate the complete flow:

```bash
# Start Forest
./forest

# In another terminal, use NATS CLI to test
nats pub land.query '{"needs_docker":true}'

# Should see response on land.info.>
nats sub "land.info.>"
```

---

## 📝 Implementation Notes

### Order of Execution
1. Task 5 (LandHouse) - Independent
2. Task 6 (AgentHouse) - Independent
3. Task 8 (Tests) - Depends on Task 5
4. Task 7 (Wire to Forest) - Depends on Tasks 5 & 6

### Dependencies
- Phase A must be complete (Land detection)
- `internal/core/land.go` must exist
- `internal/treehouses/interface.go` must exist
- Forest must have `thisLand` field

### Testing Strategy
- Unit tests validate individual Houses
- Integration test validates end-to-end communication
- Manual testing with NATS CLI validates pub/sub

---

## ⚠️ Edge Cases to Handle

1. **Docker not available**: AgentHouse returns error in result, doesn't crash
2. **Invalid JSON in query**: Process returns nil (no response)
3. **Multiple Lands match**: All respond (requester picks first)
4. **No Lands match**: Requester times out waiting for response

---

## 🎓 Key Concepts

**GoTreeHouse**: Compile-time TreeHouse implemented in Go (vs runtime Lua)
- Must be deterministic
- Has access to system resources
- Implements: Name(), Subjects(), Process()

**Event-driven**: Houses don't return values directly
- Receive Leaf via Process()
- Return new Leaf (or nil)
- Forest whispers result to Wind

**Capacity matching**: Declarative requirements
- Query specifies needs (Docker, GPU, RAM)
- Each Land checks if it matches
- Multiple Lands can respond

---

## ✅ Definition of Done

- [ ] All 4 tasks completed
- [ ] All unit tests pass
- [ ] Forest starts and logs House initialization
- [ ] Integration test with NATS CLI succeeds
- [ ] Code committed to branch
- [ ] Ready for Phase C (First AAA Flow)

---

**Next Phase**: [phase-c-first-flow.md](./phase-c-first-flow.md) - CoderNim with AAA methods
