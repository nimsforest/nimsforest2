# Cloud Agent Instructions

## How to Use This Breakdown

This project has been broken down into discrete tasks that can be executed by multiple cloud agents working in parallel or sequence.

## Quick Start for Cloud Agents

### 1. Check Dependencies
Before starting your assigned task:
- Review `TASK_BREAKDOWN.md` to find your task
- Verify all dependency tasks are marked as complete
- If blocked, update the progress tracker with ⚠️ status

### 2. Read the Specification
- Open `Cursorinstructions.md` for full context
- Find the section relevant to your component
- Understand the interfaces and patterns

### 3. Execute Your Task
- Create the files specified in your task deliverables
- Implement all required functions/methods
- Add error handling and logging
- Write unit tests (minimum 80% coverage)
- Write integration tests if needed

### 4. Test Your Work
```bash
# Start NATS server if not running
make start

# Run tests for your component
go test ./internal/core/... -v
go test ./internal/trees/... -v
go test ./internal/nims/... -v

# Run with coverage
go test ./... -cover

# Integration tests (requires NATS running)
go test ./... -tags=integration
```

### 5. Update Progress
- Mark your task status in the progress tracker
- Document any issues or blockers
- Note completion with ✅

### 6. Handoff
- Ensure your code is committed
- Update any dependent agents
- Document any deviations from spec

---

## Sample Task Assignment

### For Agent Working on Task 2.2 (Wind)

**Task**: Implement NATS Core Pub/Sub wrapper (Wind)

**Dependencies**:
- Task 1.1 (Infrastructure) - Complete ✅
- Task 2.1 (Leaf Types) - Complete ✅

**Reference Specification**:
See `Cursorinstructions.md` section "4. Wind (NATS Core)"

**File to Create**: `internal/core/wind.go`

**Implementation Checklist**:
```go
// internal/core/wind.go

type Wind struct {
    nc *nats.Conn
}

func NewWind(nc *nats.Conn) *Wind {
    // TODO: Implement
}

func (w *Wind) Drop(leaf Leaf) error {
    // TODO: Serialize leaf to JSON
    // TODO: Publish to leaf.Subject
    // TODO: Handle errors
}

func (w *Wind) Catch(subject string, handler func(leaf Leaf)) (*nats.Subscription, error) {
    // TODO: Subscribe to subject pattern
    // TODO: Deserialize messages to Leaf
    // TODO: Call handler
    // TODO: Handle errors
}
```

**Unit Tests** (`internal/core/wind_test.go`):
- Test Drop publishes correctly
- Test Catch receives messages
- Test subject pattern matching
- Test error cases

**Integration Test**:
- Requires NATS running
- Test actual pub/sub with real connection

**Acceptance**:
- [ ] All functions implemented
- [ ] Unit tests pass
- [ ] Integration test passes
- [ ] Code documented
- [ ] Task marked complete in tracker

---

## Multi-Agent Coordination

### Parallel Work (Same Phase)
Multiple agents can work on tasks in the same batch simultaneously:

**Example - Batch 2**:
- Agent A: Task 2.3 (River)
- Agent B: Task 2.4 (Soil)  
- Agent C: Task 2.5 (Humus)
- Agent D: Task 2.1 (Leaf) → then Task 4.1

These tasks don't depend on each other, so work in parallel.

### Sequential Work (Different Phases)
Some tasks must wait for others:

**Example**:
- Agent A completes Task 3.1 (BaseTree)
- Agent A notifies Agent B
- Agent B starts Task 4.2 (PaymentTree)

### Communication Protocol
1. **Task Start**: Comment in progress tracker: "🏃 In Progress - Agent X"
2. **Blocked**: "⚠️ Blocked - Waiting for Task Y"
3. **Complete**: "✅ Complete - All tests passing"
4. **Issues**: "⚠️ Issues - [description]"

---

## Testing Requirements

### Unit Tests (Required)
Every task must include unit tests:
```go
// Example: wind_test.go
func TestWind_Drop(t *testing.T) {
    // Use mock NATS connection
    // Test happy path
    // Test error cases
}
```

### Integration Tests (Required for Core)
Core components need integration tests:
```go
// Example: wind_integration_test.go
// +build integration

func TestWind_RealNATS(t *testing.T) {
    // Connect to real NATS
    // Test actual pub/sub
}
```

Run with: `go test -tags=integration`

### Test Coverage
- Minimum: 80% coverage
- Check with: `go test ./... -cover`
- Generate report: `go test ./... -coverprofile=coverage.out`

---

## Code Quality Standards

### Formatting
```bash
go fmt ./...
gofmt -s -w .
```

### Linting
```bash
golangci-lint run
```

### Documentation
All public APIs must have godoc comments:
```go
// NewWind creates a new Wind instance wrapping a NATS connection.
// The Wind provides a higher-level abstraction for leaf pub/sub.
func NewWind(nc *nats.Conn) *Wind {
    // ...
}
```

### Error Handling
Always return and check errors:
```go
// Good
if err := wind.Drop(leaf); err != nil {
    return fmt.Errorf("failed to drop leaf: %w", err)
}

// Bad
wind.Drop(leaf) // ignoring error
```

### Logging
Use structured logging:
```go
log.Printf("[%s] dropped leaf: subject=%s", w.name, leaf.Subject)
```

---

## Common Patterns

### NATS Connection Setup
```go
nc, err := nats.Connect("nats://localhost:4222")
if err != nil {
    log.Fatal(err)
}
defer nc.Close()

js, err := nc.JetStream()
if err != nil {
    log.Fatal(err)
}
```

### JetStream Stream Creation
```go
stream, err := js.AddStream(&nats.StreamConfig{
    Name:     "RIVER",
    Subjects: []string{"river.>"},
    Storage:  nats.FileStorage,
})
```

### JetStream KV Bucket Creation
```go
kv, err := js.CreateKeyValue(&nats.KeyValueConfig{
    Bucket: "SOIL",
})
```

### Optimistic Locking Pattern
```go
// Read current state with revision
data, revision, err := soil.Dig(entity)

// Modify data
// ...

// Write back with expected revision
err = soil.Bury(entity, newData, revision)
if err == nats.ErrKeyExists {
    // Conflict - retry
}
```

---

## Troubleshooting

### NATS Connection Issues
```bash
# Check NATS is running
ps aux | grep nats-server

# Check NATS logs
tail -f /tmp/nats-server.log

# Test connection
nc -zv localhost 4222
curl http://localhost:8222/varz

# Restart if needed
make stop
make start
```

### JetStream Not Enabled
```bash
# NATS must be started with --jetstream flag
# Verify with: curl http://localhost:8222/jsz
# make start includes --jetstream flag
```

### Tests Hanging
- Check for goroutine leaks
- Ensure subscriptions are cleaned up
- Use context with timeout in tests

### Import Cycles
- Keep dependencies unidirectional
- Core components shouldn't import trees/nims
- Trees/nims can import core

---

## File Templates

### Basic Core Component
```go
package core

import (
    "github.com/nats-io/nats.go"
)

// ComponentName does X
type ComponentName struct {
    nc *nats.Conn
    // fields
}

// NewComponentName creates a new ComponentName
func NewComponentName(nc *nats.Conn) (*ComponentName, error) {
    return &ComponentName{
        nc: nc,
    }, nil
}

// Method does something
func (c *ComponentName) Method() error {
    return nil
}
```

### Test File
```go
package core

import (
    "testing"
)

func TestComponentName_Method(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:  "happy path",
            input: "test",
            want:  "expected",
        },
        {
            name:    "error case",
            input:   "bad",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test logic
        })
    }
}
```

---

## FAQ

**Q: Can I modify the interface?**
A: Only if absolutely necessary. Document any changes clearly and notify dependent agents.

**Q: What if I find a bug in a dependency?**
A: Fix it, add a test, update the task owner.

**Q: Should I add extra features?**
A: Stick to the spec for initial implementation. Note enhancements for Phase 7.

**Q: How do I handle errors?**
A: Return errors, don't panic. Use `fmt.Errorf` with `%w` for wrapping.

**Q: What Go version?**
A: 1.22+ as specified in `go.mod`

**Q: Can I use additional dependencies?**
A: Minimize dependencies. Core should only use NATS. Discuss first.

---

## Getting Help

1. Check `Cursorinstructions.md` for full context
2. Review `TASK_BREAKDOWN.md` for task details
3. Look at completed tasks for patterns
4. Check test files for examples
5. Review NATS documentation: https://docs.nats.io/

---

## Completion Checklist

Before marking your task complete:

- [ ] All deliverables implemented
- [ ] Unit tests written and passing
- [ ] Integration tests (if required) passing
- [ ] Code formatted (`go fmt`)
- [ ] No lint errors
- [ ] Public APIs documented
- [ ] Error handling implemented
- [ ] Logging added
- [ ] Progress tracker updated
- [ ] Tests run successfully: `go test ./...`
- [ ] No race conditions: `go test -race ./...`

---

## Example Workflow

### Agent receives Task 2.4 (Soil)

1. **Check dependencies**: ✅ Task 1.1 complete
2. **Read spec**: Review section 7 in `Cursorinstructions.md`
3. **Update tracker**: "🏃 In Progress - Agent Soil"
4. **Implement**:
   ```bash
   # Create file
   touch internal/core/soil.go
   touch internal/core/soil_test.go

   # Implement (following spec)
   # Write tests

   # Test (ensure NATS is running)
   make start
   go test ./internal/core/soil_test.go -v
   go test ./internal/core/soil_test.go -tags=integration -v
   ```
5. **Verify**:
   ```bash
   go test ./... -cover
   go fmt ./...
   ```
6. **Complete**: Update tracker "✅ Complete - All tests passing"
7. **Notify**: Agent working on Task 3.2 (needs soil)

---

Good luck! 🚀
