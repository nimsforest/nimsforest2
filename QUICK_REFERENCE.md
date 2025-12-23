# Quick Reference - NimsForest Task Management

## 📚 Documentation Index

| File | Use When |
|------|----------|
| `README.md` | Starting point, overview |
| `TASK_BREAKDOWN.md` | Need task details & dependencies |
| `AGENT_INSTRUCTIONS.md` | Agent needs how-to guide |
| `COORDINATOR_GUIDE.md` | Assigning tasks to agents |
| `PROGRESS.md` | Checking current status |
| `Cursorinstructions.md` | Need technical specifications |
| `QUICK_REFERENCE.md` | Need quick lookup (this file) |

---

## 🎯 Task Phases At-a-Glance

```
Phase 1: Foundation         [1 task]   ⚡ Start here
Phase 2: Core Components    [5 tasks]  🔄 Can parallelize
Phase 3: Base Interfaces    [3 tasks]  🔄 Can parallelize
Phase 4: Examples           [3 tasks]  🔄 Can parallelize
Phase 5: Main App           [1 task]   ⚙️ Wire it up
Phase 6: Test & Docs        [2 tasks]  ✅ Quality gate
Phase 7: Enhancements       [3 tasks]  🚀 Optional
```

---

## 📋 All Tasks Quick List

### Phase 1: Foundation
- [ ] **1.1** - Infrastructure Setup (go.mod, docker-compose)

### Phase 2: Core Components
- [ ] **2.1** - Leaf Types (basic struct)
- [ ] **2.2** - Wind (NATS pub/sub)
- [ ] **2.3** - River (JetStream input)
- [ ] **2.4** - Soil (KV store)
- [ ] **2.5** - Humus (state stream)

### Phase 3: Base Interfaces
- [ ] **3.1** - Base Tree Interface
- [ ] **3.2** - Base Nim Interface
- [ ] **3.3** - Decomposer Worker

### Phase 4: Examples
- [ ] **4.1** - Leaf Type Definitions
- [ ] **4.2** - Payment Tree (Stripe parser)
- [ ] **4.3** - AfterSales Nim

### Phase 5: Main Application
- [ ] **5.1** - Main Entry Point

### Phase 6: Testing & Documentation
- [ ] **6.1** - End-to-End Testing
- [ ] **6.2** - Documentation

### Phase 7: Optional
- [ ] **7.1** - Additional Examples
- [ ] **7.2** - Monitoring & Observability
- [ ] **7.3** - Performance Testing

---

## 🔗 Dependency Quick Lookup

```
1.1 → No dependencies (START HERE)

2.1 → 1.1
2.2 → 1.1, 2.1
2.3 → 1.1
2.4 → 1.1
2.5 → 1.1

3.1 → 2.1, 2.2, 2.3
3.2 → 2.1, 2.2, 2.4, 2.5
3.3 → 2.4, 2.5

4.1 → 2.1
4.2 → 3.1, 4.1
4.3 → 3.2, 4.1

5.1 → 3.3, 4.2, 4.3

6.1 → 5.1
6.2 → 5.1

7.1 → 5.1
7.2 → 5.1
7.3 → 6.1
```

---

## 🎨 Status Icons

| Icon | Status | Meaning |
|------|--------|---------|
| ⏳ | Not Started | Task waiting to begin |
| 🏃 | In Progress | Agent actively working |
| ✅ | Complete | Task done & tested |
| ❌ | Blocked | Can't proceed |
| ⚠️ | Issues | Has problems |
| ⏸️ | Paused | Temporarily stopped |

---

## 📊 Task Complexity Guide

| Level | Tasks | Time Estimate |
|-------|-------|---------------|
| **Low** | 1.1, 2.1, 4.1, 6.2 | 2-4 hours |
| **Medium** | 2.2, 2.3, 2.4, 2.5, 3.1, 3.2, 3.3, 4.2, 4.3, 5.1, 7.2 | 4-8 hours |
| **High** | 6.1, 7.3 | 8+ hours |

---

## 🚀 Parallel Execution Batches

### Batch 1 (Sequential)
```
Task 1.1 → Infrastructure
```
**1 agent, ~1 hour**

### Batch 2 (Parallel)
```
Task 2.1 → Leaf
Task 2.3 → River
Task 2.4 → Soil  
Task 2.5 → Humus
```
**4 agents, ~4-6 hours**

### Batch 3 (Parallel)
```
Task 2.2 → Wind (after 2.1)
Task 3.3 → Decomposer (after 2.4, 2.5)
Task 4.1 → Leaf Types (after 2.1)
```
**3 agents, ~4-6 hours**

### Batch 4 (Parallel)
```
Task 3.1 → Base Tree
Task 3.2 → Base Nim
```
**2 agents, ~4-6 hours**

### Batch 5 (Parallel)
```
Task 4.2 → Payment Tree
Task 4.3 → AfterSales Nim
```
**2 agents, ~4-6 hours**

### Batch 6 (Sequential)
```
Task 5.1 → Main Application
```
**1 agent, ~4 hours**

### Batch 7 (Parallel)
```
Task 6.1 → E2E Testing
Task 6.2 → Documentation
```
**2 agents, ~8 hours**

---

## 🎯 Critical Path

The fastest path through the project:

```
1.1 (1h) → 2.1 (3h) → 2.2 (5h) → 2.3 (5h)
                                    ↓
                              3.1 (5h) → 4.1 (2h)
                                            ↓
                                       4.2 (5h) → 5.1 (4h)
                                                    ↓
                                                 6.1 (8h)
```

**Critical Path Time**: ~38 hours (sequential)

**With 4 Agents**: ~25-30 hours (parallel)

---

## 📁 File Locations Quick Reference

| Component | File Path |
|-----------|-----------|
| Leaf | `internal/core/leaf.go` |
| Wind | `internal/core/wind.go` |
| River | `internal/core/river.go` |
| Soil | `internal/core/soil.go` |
| Humus | `internal/core/humus.go` |
| Base Tree | `internal/core/tree.go` |
| Base Nim | `internal/core/nim.go` |
| Decomposer | `internal/core/decomposer.go` |
| Leaf Types | `internal/leaves/types.go` |
| Payment Tree | `internal/trees/payment.go` |
| AfterSales Nim | `internal/nims/aftersales.go` |
| Main | `cmd/forest/main.go` |
| Docker | `docker-compose.yml` |
| Module | `go.mod` |

---

## ⚡ Common Commands

### Setup
```bash
# Start NATS
docker-compose up -d

# Check NATS status
docker-compose ps
docker-compose logs nats

# Initialize Go module
go mod init github.com/yourusername/nimsforest
go mod tidy
```

### Testing
```bash
# Run all tests
go test ./... -v

# With coverage
go test ./... -cover

# Integration tests
go test ./... -tags=integration -v

# Race detection
go test ./... -race

# Specific package
go test ./internal/core/... -v
```

### Code Quality
```bash
# Format
go fmt ./...

# Vet
go vet ./...

# Lint (requires golangci-lint)
golangci-lint run
```

### Running
```bash
# Build
go build -o forest ./cmd/forest

# Run
./forest

# Or directly
go run ./cmd/forest/main.go
```

---

## 🧪 Testing Requirements Summary

| Test Type | Required For | Minimum Coverage |
|-----------|--------------|------------------|
| Unit Tests | All tasks | 80% |
| Integration Tests | Tasks with NATS | Pass |
| E2E Tests | Task 6.1 | 1 complete flow |

---

## 📝 Update Checklist (For Agents)

When starting a task:
- [ ] Update `PROGRESS.md` status to 🏃
- [ ] Add your name as assigned agent
- [ ] Add start date

When completing a task:
- [ ] Run all tests
- [ ] Check code coverage
- [ ] Format code
- [ ] Add documentation
- [ ] Update `PROGRESS.md` status to ✅
- [ ] Add completion date
- [ ] Note any issues

---

## 🎓 Key Concepts

| Term | What It Is | Tech |
|------|------------|------|
| **River** | External data input | JetStream Stream |
| **Tree** | Data parser/structurer | Go Service |
| **Leaf** | Structured event | Go Struct |
| **Wind** | Event bus | NATS Core |
| **Nim** | Business logic | Go Service |
| **Humus** | State change log | JetStream Stream |
| **Soil** | Current state | JetStream KV |

---

## 🔄 Data Flow

```
External System (webhook)
        ↓
    River (stream: unstructured)
        ↓
    Tree (parse & structure)
        ↓
    Leaf (structured event)
        ↓
    Wind (publish)
        ↓
    Nim (business logic)
        ↓
   ┌────┴────┐
   ↓         ↓
Humus    New Leaf
(state    (wind)
change)
   ↓
Decomposer
   ↓
Soil
(current state)
```

---

## 🎯 Quality Gates

### Before Marking Task Complete
- ✅ All code compiles
- ✅ Unit tests pass
- ✅ Integration tests pass (if applicable)
- ✅ Coverage ≥ 80%
- ✅ Code formatted
- ✅ No lint errors
- ✅ Documentation added
- ✅ PROGRESS.md updated

### Before Marking Phase Complete
- ✅ All phase tasks complete
- ✅ All tests pass
- ✅ No blockers
- ✅ Code quality verified

### Before Final Delivery
- ✅ All Phase 1-5 complete
- ✅ E2E test demonstrates full flow
- ✅ Documentation complete
- ✅ New dev can setup and run

---

## 🆘 Quick Troubleshooting

| Problem | Solution |
|---------|----------|
| NATS won't connect | Check docker-compose is running |
| Tests timeout | Add context timeouts, check for leaks |
| Import cycle | Check dependencies, core shouldn't import examples |
| Optimistic lock fails | Implement retry logic |
| Task blocked | Check PROGRESS.md for dependency status |

---

## 📞 Getting Help

1. Check **AGENT_INSTRUCTIONS.md** FAQ
2. Review **Cursorinstructions.md** spec
3. Look at completed tasks for patterns
4. Document issue in **PROGRESS.md**
5. Escalate through coordinator

---

## 🎉 Success Indicators

- ✅ All tests green
- ✅ E2E flow works
- ✅ New dev can onboard
- ✅ Code is documented
- ✅ No critical issues

---

## 📈 Project Velocity Tracking

Update daily in **PROGRESS.md**:
- Tasks completed today: X
- Tasks in progress: Y
- Tasks blocked: Z
- Estimated completion: Date

---

## 🔗 Quick Links

- [NATS Docs](https://docs.nats.io/)
- [JetStream Guide](https://docs.nats.io/nats-concepts/jetstream)
- [Go Testing](https://golang.org/pkg/testing/)
- [NATS Go Client](https://github.com/nats-io/nats.go)

---

**Pro Tip**: Keep this file open in a side tab for quick reference while working!

**Remember**: Communication is key. Update PROGRESS.md often! 📢
