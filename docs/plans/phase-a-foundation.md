# Phase A: Foundation (Land Detection)

**Status**: ✅ Complete (2026-01-10)
**Goal**: Know what compute substrate we're running on
**Actual Time**: ~1 hour (estimated 45 min)
**Branch**: `claude/plan-core-implementation-w1Ei8`

---

## 🎯 Objective

Implement Land detection so Forest knows its capabilities at startup.

**Success Criteria:**
- ✅ Forest detects RAM, CPU, Docker, GPU
- ✅ Determines Land type (Land/Nimland/Manaland)
- ✅ Logs detected capabilities on startup

---

## 📋 Tasks Completed

### ✅ Task 1: Land Data Structures

**File**: `internal/core/land.go`

Created data structures for Land information:

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
    CPUFreqMHz float64  `json:"cpu_freq_mhz"`
    HasDocker  bool     `json:"has_docker"`
    GPUVendor  string   `json:"gpu_vendor,omitempty"`
    GPUModel   string   `json:"gpu_model,omitempty"`
    GPUVram    uint64   `json:"gpu_vram,omitempty"`
    GPUTflops  float64  `json:"gpu_tflops,omitempty"`
}
```

**Outcome**: Clean type system for Land capabilities

---

### ✅ Task 2: Land Detection Logic

**File**: `internal/land/detect.go`

Implemented system probing:

```go
func Detect(natsID, natsName string) *core.LandInfo {
    // Uses gopsutil for RAM, CPU
    // Probes Docker with exec.Command
    // Detects GPU via nvidia-smi
    // Determines Land type based on capabilities
}
```

**Detection methods**:
- RAM: `gopsutil/v3/mem.VirtualMemory()`
- CPU: `runtime.NumCPU()`, `gopsutil/v3/cpu.Info()`
- Docker: `exec.Command("docker", "info")`
- GPU: `exec.Command("nvidia-smi", "--query-gpu=...")`

**Outcome**: Reliable detection on Linux systems

---

### ✅ Task 3: Wire into Forest

**Files**:
- `pkg/runtime/forest.go`
- `cmd/forest/main.go`

Added Land detection to Forest startup:

```go
// In Forest.Start()
varz, _ := f.natsServer.InternalServer().Varz(&server.VarzOptions{})
f.thisLand = land.Detect(varz.ID, varz.Name)

f.logger.Printf("[Forest] This Land: %s (%s) - RAM: %s, CPU: %d cores, Docker: %v, GPU: %s",
    f.thisLand.Name,
    f.thisLand.Type,
    formatBytes(f.thisLand.RAMTotal),
    f.thisLand.CPUCores,
    f.thisLand.HasDocker,
    f.thisLand.GPUModel)
```

**Outcome**: Forest logs Land info on every startup

---

### ✅ Task 4: TreeHouse Interface

**File**: `internal/treehouses/interface.go`

Created interface for compile-time Go TreeHouses:

```go
// GoTreeHouse is a compile-time TreeHouse implemented in Go.
// Must be deterministic: same input Leaf = same output Leaf.
type GoTreeHouse interface {
    Name() string
    Subjects() []string
    Process(leaf core.Leaf) *core.Leaf  // nil = no output
}
```

**Outcome**: Clear contract for deterministic event processors

---

## 🚦 Milestone Achieved

### ✅ M1: Forest detects and logs Land type

**Validation**:
```bash
$ ./forest
[Forest] This Land: node-1 (nimland) - RAM: 16.0 GB, CPU: 8 cores, Docker: true, GPU: none
```

**Result**: SUCCESS ✅

---

## 📊 What Was Built

### Files Created
1. `internal/core/land.go` - 99 lines
2. `internal/land/detect.go` - 156 lines
3. `internal/treehouses/interface.go` - 17 lines

### Files Modified
1. `pkg/runtime/forest.go` - Added Land detection
2. `cmd/forest/main.go` - No changes needed

### Tests Created
1. `internal/land/detect_test.go` - Unit tests for detection

**Total New Code**: ~350 lines

---

## 🎓 Lessons Learned

### What Went Well
- gopsutil integration was straightforward
- NATS server ID/Name perfect for Land identity
- Detection is fast (<100ms)

### Challenges
- GPU detection needs more vendors (only NVIDIA implemented)
- Docker probe can be slow if daemon unresponsive
- Need better error handling for detection failures

### Future Improvements
- Add AMD ROCm detection for GPUs
- Add timeout to Docker probe
- Consider caching detection results
- Add detection refresh mechanism

---

## 📝 Implementation Notes

### Dependencies Added
```go
require (
    github.com/shirou/gopsutil/v3 v3.23.12
)
```

### Key Design Decisions

1. **Use NATS server identity for Land ID**
   - Ensures unique ID across cluster
   - Consistent with NATS routing

2. **Detection on every startup**
   - Handles hardware changes
   - Simple implementation
   - Fast enough (<100ms)

3. **Deterministic Land type**
   - GPU + Docker = Manaland
   - Docker only = Nimland
   - Neither = Land (base)

4. **No persistent storage**
   - Land info is ephemeral
   - Detected fresh each time
   - Simplifies implementation

---

## 🔗 Related Code

### Uses
- `github.com/nats-io/nats-server/v2/server` - For NATS identity
- `github.com/shirou/gopsutil/v3` - System information
- `os/exec` - Docker and GPU probing

### Used By
- `pkg/runtime/forest.go` - Main integration point
- `internal/treehouses/landhouse.go` - Phase B (next)

---

## ⏭️ Next Steps

Phase A provides the foundation for Phase B:

1. **LandHouse** needs `thisLand` to respond to queries
2. **AgentHouse** needs `thisLand.HasDocker` to determine availability
3. **ViewWorld** will subscribe to Land announcements

**Proceed to**: [phase-b-communication.md](./phase-b-communication.md)

---

## ✅ Definition of Done

- [x] Land data structures defined
- [x] Detection logic implemented
- [x] Wired into Forest lifecycle
- [x] TreeHouse interface created
- [x] Milestone M1 validated
- [x] Unit tests pass
- [x] Code committed to branch
- [x] Documentation updated

---

**Completed**: 2026-01-10
**Branch**: `claude/plan-core-implementation-w1Ei8`
**Commits**: Land detection foundation
**Next Phase**: [phase-b-communication.md](./phase-b-communication.md)
