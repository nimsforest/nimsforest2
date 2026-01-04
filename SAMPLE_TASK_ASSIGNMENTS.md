# Sample Task Assignments

Copy-paste ready task assignments for cloud agents. Coordinators can use these templates when assigning tasks.

---

## Task 1.1: Infrastructure Setup

### Task Assignment

**Task ID**: 1.1  
**Component**: Project Infrastructure  
**Complexity**: Low  
**Estimated Time**: 1 hour  
**Dependencies**: None - **START HERE**

### Objective
Set up the foundational project structure, dependency management, and NATS infrastructure.

### References
- **Task Details**: `TASK_BREAKDOWN.md` - Phase 1, Task 1.1
- **Agent Guide**: `AGENT_INSTRUCTIONS.md`
- **Spec**: `Cursorinstructions.md` - Tech Stack & Project Structure sections

### Deliverables
1. Create `go.mod` with Go 1.22+ and dependencies:
   - github.com/nats-io/nats.go
2. Setup NATS server via Makefile:
   - Use `make setup` to install NATS server binary
   - Configure automatic installation for different platforms
3. Create project directory structure:
   ```
   nimsforest/
   ├── cmd/forest/
   ├── internal/core/
   ├── internal/trees/
   ├── internal/nims/
   └── internal/leaves/
   ```
5. Create `.gitignore` for Go projects
6. Create comprehensive `README.md` with setup instructions
7. Create test program to verify NATS connectivity

### Acceptance Criteria
- [ ] `go mod init` runs successfully
- [ ] NATS server binary installed and accessible
- [ ] `make start` starts NATS with JetStream enabled
- [ ] NATS accessible on localhost:4222
- [ ] Monitoring UI accessible on localhost:8222
- [ ] All directories created
- [ ] Test program verifies full connectivity (pub/sub, JetStream, KV)

### Commands to Verify
```bash
# Setup environment
make setup

# Start NATS server
make start

# Verify NATS is running
make status
curl http://localhost:8222/varz
curl http://localhost:8222/jsz

# Run integration test
go run test_nats_connection.go

# Stop NATS
make stop
```

### Update Progress
Mark task as 🏃 when starting and ✅ when complete in `PROGRESS.md`

---

## Task 2.1: Leaf Types

### Task Assignment

**Task ID**: 2.1  
**Component**: Leaf (Core Event Type)  
**Complexity**: Low  
**Estimated Time**: 2-3 hours  
**Dependencies**: Task 1.1 ✅

### Objective
Implement the core `Leaf` data structure representing structured events in the system.

### References
- **Task Details**: `TASK_BREAKDOWN.md` - Phase 2, Task 2.1
- **Spec**: `Cursorinstructions.md` - Section "1. Leaf Types"

### Deliverables
1. Create `internal/core/leaf.go`
2. Implement `Leaf` struct:
   - Subject string (event name)
   - Data json.RawMessage (payload)
   - Source string (creator)
   - Timestamp time.Time
3. Add JSON marshaling/unmarshaling support
4. Add validation methods
5. Create `internal/core/leaf_test.go` with unit tests

### Implementation Template
```go
// internal/core/leaf.go
package core

import (
    "encoding/json"
    "time"
)

type Leaf struct {
    Subject   string          `json:"subject"`
    Data      json.RawMessage `json:"data"`
    Source    string          `json:"source"`
    Timestamp time.Time       `json:"ts"`
}

// Add methods for validation, serialization, etc.
```

### Acceptance Criteria
- [ ] Leaf struct defined with all fields
- [ ] Can marshal to JSON
- [ ] Can unmarshal from JSON
- [ ] Unit tests pass
- [ ] Coverage ≥ 80%

### Testing Requirements
```bash
go test ./internal/core/leaf_test.go -v
go test ./internal/core/leaf_test.go -cover
```

### Update Progress
Mark in `PROGRESS.md` as 🏃 → ✅

---

## Task 2.2: Wind (NATS Pub/Sub)

### Task Assignment

**Task ID**: 2.2  
**Component**: Wind (NATS Core Pub/Sub)  
**Complexity**: Medium  
**Estimated Time**: 4-6 hours  
**Dependencies**: Task 1.1 ✅, Task 2.1 ✅

### Objective
Implement the Wind component that wraps NATS Core pub/sub for leaf distribution.

### References
- **Task Details**: `TASK_BREAKDOWN.md` - Phase 2, Task 2.2
- **Spec**: `Cursorinstructions.md` - Section "4. Wind (NATS Core)"

### Deliverables
1. Create `internal/core/wind.go`
2. Implement `Wind` struct with NATS connection
3. Implement `NewWind(nc *nats.Conn) *Wind`
4. Implement `Drop(leaf Leaf) error` - publishes leaf
5. Implement `Catch(subject string, handler func(leaf Leaf)) (*nats.Subscription, error)` - subscribes
6. Handle JSON encoding/decoding
7. Add error handling and logging
8. Create unit tests with mocks
9. Create integration tests with real NATS

### Implementation Template
```go
// internal/core/wind.go
package core

import (
    "encoding/json"
    "github.com/nats-io/nats.go"
)

type Wind struct {
    nc *nats.Conn
}

func NewWind(nc *nats.Conn) *Wind {
    return &Wind{nc: nc}
}

func (w *Wind) Drop(leaf Leaf) error {
    // Marshal leaf to JSON
    // Publish to leaf.Subject
    // Handle errors
}

func (w *Wind) Catch(subject string, handler func(leaf Leaf)) (*nats.Subscription, error) {
    // Subscribe to subject
    // Unmarshal to Leaf
    // Call handler
    // Handle errors
}
```

### Acceptance Criteria
- [ ] Wind struct implemented
- [ ] Drop publishes correctly
- [ ] Catch receives and deserializes
- [ ] Subject patterns work (wildcards)
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Coverage ≥ 80%

### Testing Requirements
```bash
# Unit tests
go test ./internal/core/wind_test.go -v

# Integration tests (requires NATS)
make start
go test ./internal/core/wind_test.go -tags=integration -v
```

### Update Progress
Mark in `PROGRESS.md` as 🏃 → ✅

---

## Task 2.3: River (JetStream Input Stream)

### Task Assignment

**Task ID**: 2.3  
**Component**: River (External Data Stream)  
**Complexity**: Medium  
**Estimated Time**: 4-6 hours  
**Dependencies**: Task 1.1 ✅

### Objective
Implement the River component - a JetStream stream for receiving unstructured external data.

### References
- **Task Details**: `TASK_BREAKDOWN.md` - Phase 2, Task 2.3
- **Spec**: `Cursorinstructions.md` - Section "5. River (JetStream Stream)"

### Deliverables
1. Create `internal/core/river.go`
2. Implement `RiverData` struct
3. Implement `River` struct with JetStream context
4. Implement `NewRiver(js nats.JetStreamContext) (*River, error)` - creates "RIVER" stream
5. Implement `Flow(subject string, data []byte) error` - adds data to stream
6. Implement `Observe(pattern string, handler func(data RiverData)) error` - consumes from stream
7. Configure stream with retention policy
8. Add consumer setup for observers
9. Unit and integration tests

### Stream Configuration
```go
streamConfig := &nats.StreamConfig{
    Name:     "RIVER",
    Subjects: []string{"river.>"},
    Storage:  nats.FileStorage,
    // Add retention policy
}
```

### Acceptance Criteria
- [ ] Stream "RIVER" is created automatically
- [ ] Data can be added via Flow
- [ ] Observers receive data via Observe
- [ ] Pattern matching works
- [ ] Unit tests pass
- [ ] Integration tests pass with real JetStream
- [ ] Coverage ≥ 80%

### Testing Requirements
```bash
make start
go test ./internal/core/river_test.go -v
go test ./internal/core/river_test.go -tags=integration -v
```

### Update Progress
Mark in `PROGRESS.md` as 🏃 → ✅

---

## Task 4.2: Payment Tree Example

### Task Assignment

**Task ID**: 4.2  
**Component**: Payment Tree (Stripe Webhook Parser)  
**Complexity**: Medium  
**Estimated Time**: 4-6 hours  
**Dependencies**: Task 3.1 ✅, Task 4.1 ✅

### Objective
Implement an example Tree that parses Stripe webhooks into structured leaves.

### References
- **Task Details**: `TASK_BREAKDOWN.md` - Phase 4, Task 4.2
- **Spec**: `Cursorinstructions.md` - Section "8. Example Tree"

### Deliverables
1. Create `internal/trees/payment.go`
2. Implement `PaymentTree` struct embedding `*core.BaseTree`
3. Implement `NewPaymentTree(base *core.BaseTree, river *core.River) *PaymentTree`
4. Implement Tree interface methods
5. Implement `parseStripe(data core.RiverData) *core.Leaf`
6. Handle Stripe events:
   - `charge.succeeded` → `payment.completed` leaf
   - `charge.failed` → `payment.failed` leaf (optional)
7. Add helper functions for extracting webhook data
8. Unit tests with sample Stripe payloads
9. Integration test with river

### Implementation Pattern
```go
// internal/trees/payment.go
package trees

import (
    "nimsforest/internal/core"
    "nimsforest/internal/leaves"
)

type PaymentTree struct {
    *core.BaseTree
    river *core.River
}

func NewPaymentTree(base *core.BaseTree, river *core.River) *PaymentTree {
    return &PaymentTree{
        BaseTree: base,
        river:    river,
    }
}

func (t *PaymentTree) Name() string { return "payment" }

func (t *PaymentTree) Patterns() []string {
    return []string{"stripe.>", "paypal.>"}
}

func (t *PaymentTree) Start(ctx context.Context) error {
    // Watch river for Stripe webhooks
    // Parse and drop structured leaves
}

func (t *PaymentTree) parseStripe(data core.RiverData) *core.Leaf {
    // Parse Stripe webhook JSON
    // Extract relevant fields
    // Create and return structured leaf
}
```

### Test Data Needed
Create sample Stripe webhook payloads in test file:
```json
{
  "type": "charge.succeeded",
  "data": {
    "object": {
      "customer": "cus_123",
      "amount": 9900,
      "currency": "usd"
    }
  }
}
```

### Acceptance Criteria
- [ ] PaymentTree implements Tree interface
- [ ] Parses Stripe webhooks correctly
- [ ] Emits structured leaves
- [ ] Unit tests with sample data pass
- [ ] Integration test with river works
- [ ] Coverage ≥ 80%

### Testing Requirements
```bash
go test ./internal/trees/payment_test.go -v
go test ./internal/trees/payment_test.go -tags=integration -v
```

### Update Progress
Mark in `PROGRESS.md` as 🏃 → ✅

---

## Task 5.1: Main Application Entry Point

### Task Assignment

**Task ID**: 5.1  
**Component**: Main Application  
**Complexity**: Medium  
**Estimated Time**: 4 hours  
**Dependencies**: Task 3.3 ✅, Task 4.2 ✅, Task 4.3 ✅

### Objective
Create the main application entry point that wires up all components and starts the forest.

### References
- **Task Details**: `TASK_BREAKDOWN.md` - Phase 5, Task 5.1
- **Spec**: `Cursorinstructions.md` - Section "12. Main Entry Point"

### Deliverables
1. Create `cmd/forest/main.go`
2. NATS connection setup
3. JetStream initialization
4. Create all core components (wind, river, humus, soil)
5. Start decomposer goroutine
6. Initialize example tree (PaymentTree)
7. Initialize example nim (AfterSalesNim)
8. Start all components
9. Graceful shutdown handling (SIGINT/SIGTERM)
10. Configuration management (env vars or flags)
11. Structured logging
12. Integration test

### Implementation Structure
```go
// cmd/forest/main.go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"

    "github.com/nats-io/nats.go"
    "nimsforest/internal/core"
    "nimsforest/internal/trees"
    "nimsforest/internal/nims"
)

func main() {
    // Connect to NATS
    nc, err := nats.Connect("nats://localhost:4222")
    if err != nil {
        log.Fatal(err)
    }
    defer nc.Close()

    js, err := nc.JetStream()
    if err != nil {
        log.Fatal(err)
    }

    // Initialize core components
    wind := core.NewWind(nc)
    river, _ := core.NewRiver(js)
    humus, _ := core.NewHumus(js)
    soil, _ := core.NewSoil(js)

    // Start decomposer
    go core.RunDecomposer(humus, soil)

    // Initialize trees
    paymentTreeBase := core.NewBaseTree("payment", wind)
    paymentTree := trees.NewPaymentTree(paymentTreeBase, river)

    // Initialize nims
    afterSalesBase := core.NewBaseNim("aftersales", wind, humus, soil)
    afterSalesNim := nims.NewAfterSalesNim(afterSalesBase)

    // Start everything
    ctx := context.Background()
    paymentTree.Start(ctx)
    afterSalesNim.Start(ctx)

    log.Println("Forest started...")

    // Wait for shutdown signal
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)
    <-sigCh

    log.Println("Forest shutting down...")

    // Cleanup
    paymentTree.Stop()
    afterSalesNim.Stop()
}
```

### Configuration
Allow configuration via environment variables:
- `NATS_URL` - NATS server URL (default: nats://localhost:4222)
- `LOG_LEVEL` - Logging level (default: info)

### Acceptance Criteria
- [ ] Application compiles and runs
- [ ] All components initialize successfully
- [ ] Decomposer runs in background
- [ ] Graceful shutdown works (Ctrl+C)
- [ ] All components clean up properly
- [ ] Can process test event end-to-end
- [ ] Structured logging throughout
- [ ] Integration test passes

### Testing Requirements
```bash
# Build
go build -o forest ./cmd/forest

# Run
make start
./forest

# Test end-to-end flow
# (Send test webhook, verify processing)
```

### Update Progress
Mark in `PROGRESS.md` as 🏃 → ✅

---

## Task 6.1: End-to-End Testing

### Task Assignment

**Task ID**: 6.1  
**Component**: E2E Test Suite  
**Complexity**: High  
**Estimated Time**: 8 hours  
**Dependencies**: Task 5.1 ✅

### Objective
Create comprehensive end-to-end tests that verify the complete flow from webhook to state storage.

### References
- **Task Details**: `TASK_BREAKDOWN.md` - Phase 6, Task 6.1
- **Spec**: `Cursorinstructions.md` - Section "Example Flow"

### Deliverables
1. Create `test/e2e/` directory
2. Implement E2E test that verifies:
   - NATS starts successfully
   - Forest application starts
   - Stripe webhook sent to river
   - Tree parses and structures webhook
   - Leaf appears on wind
   - Nim catches and processes leaf
   - State change written to humus
   - Decomposer applies change to soil
   - Final state is correct
3. Add test utilities and helpers
4. Create sample webhook payloads
5. Add assertions at each step
6. Ensure test is repeatable and isolated

### Test Flow
```go
// test/e2e/forest_test.go
// +build integration,e2e

func TestCompletePaymentFlow(t *testing.T) {
    // 1. Setup: Start NATS, Forest
    // 2. Send Stripe webhook to river
    // 3. Wait and verify leaf on wind
    // 4. Verify nim processes leaf
    // 5. Verify compost in humus
    // 6. Verify state in soil
    // 7. Verify thank-you email leaf on wind
    // 8. Cleanup
}
```

### Acceptance Criteria
- [ ] E2E test demonstrates complete flow
- [ ] Test is repeatable
- [ ] Test is isolated (cleanup between runs)
- [ ] Covers happy path
- [ ] Covers error cases (optional)
- [ ] Test passes consistently
- [ ] Documentation for running test

### Testing Requirements
```bash
# Start dependencies
make start

# Run E2E test
go test ./test/e2e/... -tags=integration,e2e -v

# Cleanup
make stop
```

### Success Indicator
Test output shows:
```
✅ Webhook received in river
✅ Tree parsed webhook
✅ Leaf published to wind
✅ Nim caught leaf
✅ Compost written to humus
✅ Decomposer applied to soil
✅ Final state matches expected
```

### Update Progress
Mark in `PROGRESS.md` as 🏃 → ✅

---

## Notes for Coordinators

### When to Assign Tasks

1. **Sequential**: Wait for dependencies before assigning
2. **Parallel**: Assign multiple tasks from same batch simultaneously
3. **Priority**: Critical path tasks take priority

### Communication

- Notify agent when dependencies complete
- Check on progress if no update for 24 hours
- Document blockers immediately
- Celebrate completions! 🎉

### Quality Control

Before accepting task completion:
- [ ] All tests pass
- [ ] Code formatted
- [ ] Coverage meets requirements
- [ ] Documentation added
- [ ] PROGRESS.md updated

---

**Last Updated**: 2025-12-23
