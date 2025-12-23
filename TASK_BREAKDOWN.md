# NimsForest Project - Task Breakdown

## Overview
This document breaks down the NimsForest prototype into discrete tasks that can be executed by multiple cloud agents in parallel or sequence. Each task is designed to be self-contained with clear dependencies.

---

## Phase 1: Foundation Setup (No Dependencies)

### Task 1.1: Project Infrastructure
**Agent**: Infrastructure Agent
**Estimated Complexity**: Low
**Dependencies**: None

**Deliverables**:
- [ ] Create `go.mod` file with Go 1.22+ and NATS dependencies
- [ ] Create `docker-compose.yml` with NATS server configuration
- [ ] Create basic project directory structure:
  ```
  nimsforest/
  ├── cmd/forest/
  ├── internal/core/
  ├── internal/trees/
  ├── internal/nims/
  └── internal/leaves/
  ```
- [ ] Create `.gitignore` for Go projects
- [ ] Create `README.md` with setup instructions

**Acceptance Criteria**:
- `go mod init` runs successfully
- `docker-compose up` starts NATS with JetStream enabled
- NATS accessible on ports 4222 (client) and 8222 (monitoring)

---

## Phase 2: Core Components (Parallel Execution Possible)

### Task 2.1: Leaf Type Definition
**Agent**: Core Types Agent
**Estimated Complexity**: Low
**Dependencies**: Task 1.1 (go.mod)

**Deliverables**:
- [ ] Implement `internal/core/leaf.go`
  - `Leaf` struct with Subject, Data, Source, Timestamp
  - JSON marshaling/unmarshaling support
- [ ] Add basic validation methods
- [ ] Add unit tests for leaf serialization

**Acceptance Criteria**:
- Leaf can be created, marshaled to JSON, and unmarshaled
- Tests pass with >80% coverage

---

### Task 2.2: Wind (NATS Core Pub/Sub)
**Agent**: Wind Agent
**Estimated Complexity**: Medium
**Dependencies**: Task 1.1, Task 2.1

**Deliverables**:
- [ ] Implement `internal/core/wind.go`
  - `Wind` struct with NATS connection
  - `NewWind(nc *nats.Conn) *Wind`
  - `Drop(leaf Leaf) error` - publishes leaf
  - `Catch(subject string, handler func(leaf Leaf)) (*nats.Subscription, error)` - subscribes
- [ ] Handle JSON encoding/decoding of leaves
- [ ] Add error handling and logging
- [ ] Unit tests with mock NATS connections
- [ ] Integration tests with real NATS

**Acceptance Criteria**:
- Can publish and subscribe to leaves
- Subject patterns work correctly (wildcards)
- Tests pass including integration test with NATS

---

### Task 2.3: River (JetStream Input Stream)
**Agent**: River Agent
**Estimated Complexity**: Medium
**Dependencies**: Task 1.1

**Deliverables**:
- [ ] Implement `internal/core/river.go`
  - `RiverData` struct
  - `River` struct with JetStream context
  - `NewRiver(js nats.JetStreamContext) (*River, error)` - creates stream "RIVER"
  - `Flow(subject string, data []byte) error` - adds data to stream
  - `Observe(pattern string, handler func(data RiverData)) error` - consumes from stream
- [ ] Configure JetStream stream with retention policy
- [ ] Add consumer setup for observers
- [ ] Unit and integration tests

**Acceptance Criteria**:
- Stream "RIVER" is created with proper configuration
- Data can be added and consumed
- Pattern matching works for observers
- Tests pass with real JetStream

---

### Task 2.4: Soil (JetStream KV Store)
**Agent**: Soil Agent
**Estimated Complexity**: Medium
**Dependencies**: Task 1.1

**Deliverables**:
- [ ] Implement `internal/core/soil.go`
  - `Soil` struct with KV store
  - `NewSoil(js nats.JetStreamContext) (*Soil, error)` - creates KV bucket "SOIL"
  - `Dig(entity string) ([]byte, uint64, error)` - reads with revision
  - `Bury(entity string, data []byte, expectedRevision uint64) error` - writes with optimistic locking
  - `Delete(entity string) error` - removes entity
  - `Watch(pattern string, handler func(...)) error` - watches changes
- [ ] Handle optimistic locking conflicts
- [ ] Unit and integration tests

**Acceptance Criteria**:
- KV bucket "SOIL" is created
- CRUD operations work correctly
- Optimistic locking prevents conflicts
- Watch functionality triggers on changes
- Tests pass with real JetStream KV

---

### Task 2.5: Humus (JetStream State Stream)
**Agent**: Humus Agent
**Estimated Complexity**: Medium
**Dependencies**: Task 1.1

**Deliverables**:
- [ ] Implement `internal/core/humus.go`
  - `Compost` struct
  - `Humus` struct with JetStream context
  - `NewHumus(js nats.JetStreamContext) (*Humus, error)` - creates stream "HUMUS"
  - `Add(nimName, entity, action string, data []byte) (uint64, error)` - adds compost
  - `Decompose(handler func(compost Compost)) error` - consumes compost entries
- [ ] Configure stream with proper retention
- [ ] Add sequence number tracking (slot)
- [ ] Unit and integration tests

**Acceptance Criteria**:
- Stream "HUMUS" is created
- Compost entries are persisted with sequence numbers
- Decompose can process entries in order
- Tests pass with real JetStream

---

## Phase 3: Base Interfaces (Depends on Core Components)

### Task 3.1: Base Tree Interface & Implementation
**Agent**: Tree Interface Agent
**Estimated Complexity**: Medium
**Dependencies**: Task 2.1, Task 2.2, Task 2.3

**Deliverables**:
- [ ] Implement `internal/core/tree.go`
  - `Tree` interface (Name, Patterns, Parse, Start, Stop)
  - `BaseTree` struct
  - `NewBaseTree(name string, wind *Wind) *BaseTree`
  - `Drop(leaf Leaf) error` - sends leaf to wind
- [ ] Add lifecycle management (Start/Stop)
- [ ] Add error handling and logging
- [ ] Unit tests for BaseTree

**Acceptance Criteria**:
- BaseTree can be instantiated
- Drop sends leaves to wind correctly
- Interface defines clear contract for concrete trees
- Tests pass

---

### Task 3.2: Base Nim Interface & Implementation
**Agent**: Nim Interface Agent
**Estimated Complexity**: Medium
**Dependencies**: Task 2.1, Task 2.2, Task 2.4, Task 2.5

**Deliverables**:
- [ ] Implement `internal/core/nim.go`
  - `Nim` interface (Name, Subjects, Handle, Start, Stop)
  - `BaseNim` struct with wind, humus, soil
  - `NewBaseNim(name string, wind *Wind, humus *Humus, soil *Soil) *BaseNim`
  - `Leaf(subject string, data []byte) error` - drops leaf on wind
  - `Compost(entity, action string, data []byte) (uint64, error)` - sends to humus
  - `Dig(entity string) ([]byte, uint64, error)` - reads from soil
  - `Bury(entity string, data []byte, expectedRevision uint64) error` - writes to soil
- [ ] Add lifecycle management
- [ ] Unit tests for BaseNim

**Acceptance Criteria**:
- BaseNim provides helper methods for all operations
- Interface defines clear contract for concrete nims
- Tests pass

---

### Task 3.3: Decomposer Worker
**Agent**: Decomposer Agent
**Estimated Complexity**: Medium
**Dependencies**: Task 2.4, Task 2.5

**Deliverables**:
- [ ] Implement `internal/core/decomposer.go`
  - `RunDecomposer(humus *Humus, soil *Soil)` - processes compost to soil
  - Handle create/update/delete actions
  - Add error handling and retry logic
  - Add graceful shutdown
- [ ] Unit tests with mocks
- [ ] Integration tests

**Acceptance Criteria**:
- Decomposer consumes from humus
- State changes are applied to soil correctly
- Handles optimistic locking conflicts
- Runs in background goroutine
- Tests pass

---

## Phase 4: Example Implementations (Can be parallel)

### Task 4.1: Leaf Type Definitions
**Agent**: Types Agent
**Estimated Complexity**: Low
**Dependencies**: Task 2.1

**Deliverables**:
- [ ] Implement `internal/leaves/types.go`
  - `PaymentCompleted` struct
  - `PaymentFailed` struct (optional)
  - `FollowupRequired` struct
  - Additional types as needed for examples
- [ ] Add JSON tags
- [ ] Add validation methods
- [ ] Unit tests

**Acceptance Criteria**:
- All example leaf types are defined
- Can be marshaled/unmarshaled
- Tests pass

---

### Task 4.2: Payment Tree Example
**Agent**: Payment Tree Agent
**Estimated Complexity**: Medium
**Dependencies**: Task 3.1, Task 4.1

**Deliverables**:
- [ ] Implement `internal/trees/payment.go`
  - `PaymentTree` struct
  - `NewPaymentTree(base *core.BaseTree, river *core.River) *PaymentTree`
  - Implement Tree interface methods
  - `parseStripe(data core.RiverData) *core.Leaf` - parse Stripe webhooks
  - Handle "charge.succeeded" and "charge.failed" events
- [ ] Add helper functions for extracting webhook data
- [ ] Unit tests with sample Stripe webhook payloads
- [ ] Integration test with river

**Acceptance Criteria**:
- Parses Stripe webhooks correctly
- Emits structured leaves
- Tests pass with sample webhook data

---

### Task 4.3: AfterSales Nim Example
**Agent**: AfterSales Agent
**Estimated Complexity**: Medium
**Dependencies**: Task 3.2, Task 4.1

**Deliverables**:
- [ ] Implement `internal/nims/aftersales.go`
  - `AfterSalesNim` struct
  - `NewAfterSalesNim(base *core.BaseNim) *AfterSalesNim`
  - Implement Nim interface methods
  - Handle "payment.completed" - create followup task
  - Handle "payment.refunded" - different logic
- [ ] Business logic for creating tasks
- [ ] Unit tests
- [ ] Integration test with wind, humus, soil

**Acceptance Criteria**:
- Catches payment leaves
- Creates followup tasks in soil via humus
- Emits communication leaves
- Tests pass

---

## Phase 5: Main Application (Depends on everything)

### Task 5.1: Main Entry Point
**Agent**: Main Application Agent
**Estimated Complexity**: Medium
**Dependencies**: All previous tasks

**Deliverables**:
- [ ] Implement `cmd/forest/main.go`
  - NATS connection setup
  - JetStream initialization
  - Create all core components (wind, river, humus, soil)
  - Start decomposer goroutine
  - Initialize and start example tree (PaymentTree)
  - Initialize and start example nim (AfterSalesNim)
  - Graceful shutdown handling
  - Configuration management (env vars or flags)
- [ ] Add structured logging
- [ ] Add health check endpoint (optional)
- [ ] Integration test

**Acceptance Criteria**:
- Application starts all components
- Handles SIGINT/SIGTERM gracefully
- All components shut down cleanly
- Can process end-to-end flow

---

## Phase 6: Testing & Documentation

### Task 6.1: End-to-End Testing
**Agent**: E2E Testing Agent
**Estimated Complexity**: High
**Dependencies**: Task 5.1

**Deliverables**:
- [ ] Create `test/e2e/` directory
- [ ] Implement end-to-end test:
  - Start NATS with docker-compose
  - Start forest application
  - Send Stripe webhook to river
  - Verify leaf appears on wind
  - Verify nim processes leaf
  - Verify task created in soil
  - Verify compost in humus
- [ ] Add test utilities and helpers
- [ ] Create sample webhook payloads

**Acceptance Criteria**:
- Complete flow from river to soil works
- Tests are repeatable and isolated
- Tests pass consistently

---

### Task 6.2: Documentation & README
**Agent**: Documentation Agent
**Estimated Complexity**: Low
**Dependencies**: Task 5.1

**Deliverables**:
- [ ] Update `README.md` with:
  - Architecture overview
  - Quick start guide
  - Development setup
  - Running tests
  - Example flows
  - API documentation
- [ ] Add code comments to all public APIs
- [ ] Create architecture diagram (optional)
- [ ] Add troubleshooting guide

**Acceptance Criteria**:
- A new developer can set up and run the project
- All public APIs are documented
- Examples are clear and working

---

## Phase 7: Optional Enhancements

### Task 7.1: Additional Examples
**Agent**: Examples Agent
**Estimated Complexity**: Medium
**Dependencies**: Task 5.1

**Deliverables**:
- [ ] Implement `internal/trees/crm.go` - CRM event parser
- [ ] Implement `internal/nims/inventory.go` - Stock management
- [ ] Implement `internal/nims/comms.go` - Communication handler
- [ ] Add corresponding leaf types
- [ ] Tests for each

**Acceptance Criteria**:
- Multiple trees and nims demonstrate flexibility
- Different patterns of usage are shown
- Tests pass

---

### Task 7.2: Monitoring & Observability
**Agent**: Observability Agent
**Estimated Complexity**: Medium
**Dependencies**: Task 5.1

**Deliverables**:
- [ ] Add structured logging throughout
- [ ] Add metrics collection (Prometheus format)
- [ ] Add tracing support (OpenTelemetry)
- [ ] Create Grafana dashboard (optional)
- [ ] Health check endpoints

**Acceptance Criteria**:
- Key metrics are exposed
- Logs are structured and searchable
- Can trace requests end-to-end

---

### Task 7.3: Performance & Scalability
**Agent**: Performance Agent
**Estimated Complexity**: High
**Dependencies**: Task 6.1

**Deliverables**:
- [ ] Load testing suite
- [ ] Benchmarking tests
- [ ] Concurrency stress tests
- [ ] Performance profiling
- [ ] Optimization recommendations

**Acceptance Criteria**:
- System handles high throughput
- Resource usage is reasonable
- Bottlenecks are identified

---

## Dependency Graph

```
Phase 1: Task 1.1 (Infrastructure)
           ↓
Phase 2: ┌─────────────────────────────────┐
         │ Task 2.1 (Leaf)                 │
         │ Task 2.2 (Wind) ← 2.1          │
         │ Task 2.3 (River)                │
         │ Task 2.4 (Soil)                 │
         │ Task 2.5 (Humus)                │
         └─────────────────────────────────┘
           ↓
Phase 3: ┌─────────────────────────────────┐
         │ Task 3.1 (BaseTree) ← 2.1,2.2,2.3 │
         │ Task 3.2 (BaseNim) ← 2.1,2.2,2.4,2.5 │
         │ Task 3.3 (Decomposer) ← 2.4,2.5 │
         └─────────────────────────────────┘
           ↓
Phase 4: ┌─────────────────────────────────┐
         │ Task 4.1 (Leaf Types) ← 2.1    │
         │ Task 4.2 (PaymentTree) ← 3.1,4.1 │
         │ Task 4.3 (AfterSalesNim) ← 3.2,4.1 │
         └─────────────────────────────────┘
           ↓
Phase 5: Task 5.1 (Main) ← All Phase 4
           ↓
Phase 6: ┌─────────────────────────────────┐
         │ Task 6.1 (E2E Tests) ← 5.1     │
         │ Task 6.2 (Documentation) ← 5.1 │
         └─────────────────────────────────┘
           ↓
Phase 7: Optional enhancements
```

---

## Parallel Execution Strategy

### Batch 1 (No dependencies)
- Task 1.1: Infrastructure Setup

### Batch 2 (After infrastructure)
- Task 2.1: Leaf Types (low complexity)
- Task 2.3: River (independent)
- Task 2.4: Soil (independent)
- Task 2.5: Humus (independent)

### Batch 3 (After Batch 2)
- Task 2.2: Wind (needs 2.1)
- Task 3.3: Decomposer (needs 2.4, 2.5)
- Task 4.1: Leaf Type Definitions (needs 2.1)

### Batch 4 (After Batch 3)
- Task 3.1: Base Tree (needs 2.1, 2.2, 2.3)
- Task 3.2: Base Nim (needs 2.1, 2.2, 2.4, 2.5)

### Batch 5 (After Batch 4)
- Task 4.2: Payment Tree (needs 3.1, 4.1)
- Task 4.3: AfterSales Nim (needs 3.2, 4.1)

### Batch 6 (After Batch 5)
- Task 5.1: Main Application (needs all)

### Batch 7 (After working application)
- Task 6.1: E2E Testing
- Task 6.2: Documentation
(Can run in parallel)

### Batch 8 (Optional)
- Task 7.1: Additional Examples
- Task 7.2: Monitoring
- Task 7.3: Performance
(Can run in parallel)

---

## Task Assignment Format

Each cloud agent should receive:

1. **Task ID** (e.g., Task 2.2)
2. **Task Title** (e.g., Wind - NATS Core Pub/Sub)
3. **Full specification** from original document for their component
4. **Dependencies** (what must be complete first)
5. **Acceptance criteria** (how to verify completion)
6. **Context**: Link to original `Cursorinstructions.md`

---

## Progress Tracking

Create a simple progress tracker:

```markdown
## Progress Dashboard

| Phase | Task | Status | Agent | Notes |
|-------|------|--------|-------|-------|
| 1     | 1.1  | ⏳ Not Started | - | - |
| 2     | 2.1  | ⏳ Not Started | - | - |
| 2     | 2.2  | ⏳ Not Started | - | - |
| 2     | 2.3  | ⏳ Not Started | - | - |
| 2     | 2.4  | ⏳ Not Started | - | - |
| 2     | 2.5  | ⏳ Not Started | - | - |
| ...   | ...  | ...            | ... | ... |

Legend:
⏳ Not Started | 🏃 In Progress | ✅ Complete | ❌ Blocked | ⚠️ Issues
```

---

## Notes for Cloud Agents

1. **Read the original spec**: Always reference `Cursorinstructions.md` for detailed implementation
2. **Run tests**: All tasks must include unit tests minimum
3. **Integration test**: Tasks with external dependencies need integration tests
4. **Error handling**: Add proper error handling and logging
5. **Documentation**: Add godoc comments to all public APIs
6. **Dependencies**: Check your dependencies are complete before starting
7. **Report blockers**: If blocked, document clearly in progress tracker

---

## Estimated Timeline

- **Phase 1**: 1 hour
- **Phase 2**: 1-2 days (parallel execution)
- **Phase 3**: 1 day (sequential after Phase 2)
- **Phase 4**: 1 day (parallel after Phase 3)
- **Phase 5**: 4-6 hours
- **Phase 6**: 1 day
- **Phase 7**: Optional, 2-3 days

**Total**: 4-6 days with 3-4 agents working in parallel

---

## Success Criteria

The project is complete when:

1. ✅ All Phase 1-5 tasks are complete
2. ✅ E2E test passes (Task 6.1)
3. ✅ Documentation is complete (Task 6.2)
4. ✅ A Stripe webhook can flow from river → tree → leaf → wind → nim → humus → soil
5. ✅ State is correctly stored and retrievable
6. ✅ All tests pass
7. ✅ Code is properly documented
