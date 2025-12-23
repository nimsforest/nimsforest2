# NimsForest Prototype - Task Management

This repository contains a breakdown of the NimsForest prototype project into discrete, actionable tasks suitable for execution by multiple cloud agents working in parallel or sequence.

## 📁 Documentation Structure

| Document | Purpose | Audience |
|----------|---------|----------|
| **Cursorinstructions.md** | Original detailed specification | All (Reference) |
| **TASK_BREAKDOWN.md** | Comprehensive task breakdown with dependencies | Coordinators & Agents |
| **AGENT_INSTRUCTIONS.md** | How-to guide for executing tasks | Agents |
| **PROGRESS.md** | Real-time progress tracking | All |
| **COORDINATOR_GUIDE.md** | Guide for assigning and coordinating tasks | Coordinators |
| **README.md** | This file - Quick overview | All |

## 🚀 Quick Start

### For Coordinators
1. Read `COORDINATOR_GUIDE.md`
2. Start with **Task 1.1** (Infrastructure Setup)
3. Follow the batch assignment strategy
4. Monitor `PROGRESS.md` daily

### For Agents
1. Receive your task assignment
2. Read `AGENT_INSTRUCTIONS.md`
3. Check dependencies in `TASK_BREAKDOWN.md`
4. Reference `Cursorinstructions.md` for detailed specs
5. Update `PROGRESS.md` when starting and completing

## 📊 Project Overview

**Goal**: Build an event-driven organizational orchestration system in Go using NATS and JetStream.

**Components**:
- **River**: Unstructured external data stream (JetStream)
- **Tree**: Pattern matchers that parse and structure data
- **Leaf**: Structured events with schemas
- **Wind**: Event distribution (NATS Core pub/sub)
- **Nim**: Business logic processors
- **Humus**: Persistent state changes (JetStream)
- **Soil**: Current state (JetStream KV)

## 📋 Task Summary

| Phase | Tasks | Description |
|-------|-------|-------------|
| **1** | 1 task | Foundation setup (go.mod, docker-compose, directories) |
| **2** | 5 tasks | Core components (Leaf, Wind, River, Soil, Humus) |
| **3** | 3 tasks | Base interfaces (Tree, Nim, Decomposer) |
| **4** | 3 tasks | Example implementations (PaymentTree, AfterSalesNim) |
| **5** | 1 task | Main application entry point |
| **6** | 2 tasks | Testing and documentation |
| **7** | 3 tasks | Optional enhancements |
| **Total** | **18 tasks** | Complete prototype |

## 🔄 Execution Flow

```
Phase 1: Infrastructure (1 task)
           ↓
Phase 2: Core Components (5 tasks - parallel possible)
           ↓
Phase 3: Base Interfaces (3 tasks - parallel possible)
           ↓
Phase 4: Examples (3 tasks - parallel possible)
           ↓
Phase 5: Main Application (1 task)
           ↓
Phase 6: Testing & Docs (2 tasks - parallel)
           ↓
Phase 7: Optional (3 tasks - parallel)
```

## 🎯 Parallel Execution Strategy

### Maximum Parallelization (4 Agents)
- **Batch 1**: 1 agent on Task 1.1
- **Batch 2**: 4 agents on Tasks 2.1, 2.3, 2.4, 2.5
- **Batch 3**: 3 agents on Tasks 2.2, 3.3, 4.1
- **Batch 4**: 2 agents on Tasks 3.1, 3.2
- **Batch 5**: 2 agents on Tasks 4.2, 4.3
- **Batch 6**: 1 agent on Task 5.1
- **Batch 7**: 2 agents on Tasks 6.1, 6.2

**Estimated Timeline**: 4-5 days with 4 agents

## 📦 Deliverables

### Phase 1-5 (Core)
- Working Go application
- NATS/JetStream integration
- Example tree (Stripe payment parser)
- Example nim (AfterSales logic)
- End-to-end flow: webhook → processing → state storage

### Phase 6 (Quality)
- E2E tests demonstrating full flow
- Comprehensive documentation
- Setup guide for new developers

### Phase 7 (Optional)
- Additional examples (CRM, Inventory, Comms)
- Monitoring and observability
- Performance testing and optimization

## ✅ Success Criteria

Project is complete when:
1. All Phase 1-5 tasks are marked complete
2. E2E test passes showing: `river → tree → leaf → wind → nim → humus → soil`
3. A Stripe webhook can be processed end-to-end
4. State is correctly stored and retrievable
5. All tests pass (unit + integration)
6. Documentation is complete
7. A new developer can clone and run the project

## 🛠 Technology Stack

- **Language**: Go 1.22+
- **Messaging**: NATS Server with JetStream
- **Dependencies**: github.com/nats-io/nats.go
- **Testing**: Go testing + table-driven tests
- **Infrastructure**: Docker Compose

## 📈 Progress Tracking

Current progress is tracked in `PROGRESS.md`:
- Task status (Not Started / In Progress / Complete / Blocked)
- Agent assignments
- Completion dates
- Issues and blockers
- Test results

## 🔗 Dependencies

### Phase 1 → Phase 2
Phase 2 requires completed infrastructure (go.mod, docker-compose, NATS running)

### Phase 2 → Phase 3
Phase 3 requires core components (some can start before all complete)

### Phase 3 → Phase 4
Phase 4 requires base interfaces

### Phase 4 → Phase 5
Phase 5 requires all examples complete

### Phase 5 → Phase 6
Phase 6 requires working application

See `TASK_BREAKDOWN.md` for detailed dependency graph.

## 🧪 Testing Requirements

Each task must include:
- **Unit tests**: Test individual functions/methods
- **Integration tests**: Test with real NATS (where applicable)
- **Minimum 80% code coverage**
- **All tests passing** before marking complete

Run tests:
```bash
# Unit tests
go test ./... -v

# With coverage
go test ./... -cover

# Integration tests
docker-compose up -d
go test ./... -tags=integration

# Race detection
go test ./... -race
```

## 📝 Code Quality Standards

- **Formatting**: `go fmt ./...`
- **Linting**: `golangci-lint run`
- **Documentation**: Godoc comments on all public APIs
- **Error handling**: Always return and check errors
- **Logging**: Structured logging throughout

## 🎓 Learning Resources

- [NATS Documentation](https://docs.nats.io/)
- [JetStream Guide](https://docs.nats.io/nats-concepts/jetstream)
- [NATS Go Client](https://github.com/nats-io/nats.go)
- [Go Testing](https://golang.org/pkg/testing/)

## 🤝 Contributing Guidelines

### For Agents
1. Pick up assigned task from coordinator
2. Check dependencies are complete
3. Reference detailed spec in `Cursorinstructions.md`
4. Implement with tests
5. Update `PROGRESS.md`
6. Notify coordinator when complete

### For Coordinators
1. Verify dependencies before assigning
2. Use batch assignment strategy
3. Monitor progress daily
4. Unblock agents when needed
5. Ensure quality gates are met

## 🔍 File Structure (After Completion)

```
nimsforest/
├── cmd/
│   └── forest/
│       └── main.go              # Application entry point
├── internal/
│   ├── core/
│   │   ├── leaf.go              # Event type
│   │   ├── wind.go              # Pub/sub (NATS Core)
│   │   ├── river.go             # Input stream (JetStream)
│   │   ├── soil.go              # State store (KV)
│   │   ├── humus.go             # State changes (JetStream)
│   │   ├── tree.go              # Base tree interface
│   │   ├── nim.go               # Base nim interface
│   │   └── decomposer.go        # State processor
│   ├── trees/
│   │   └── payment.go           # Stripe webhook parser
│   ├── nims/
│   │   └── aftersales.go        # Business logic example
│   └── leaves/
│       └── types.go             # Event schemas
├── docker-compose.yml           # NATS infrastructure
├── go.mod                       # Dependencies
├── README.md                    # This file (updated)
└── test/
    └── e2e/                     # End-to-end tests
```

## 📞 Support

For issues or questions:
1. Check `AGENT_INSTRUCTIONS.md` FAQ section
2. Review `Cursorinstructions.md` for specification details
3. Document in `PROGRESS.md` issues section
4. Escalate through coordinator

## 🎉 Getting Started

**Coordinators**: Start here → `COORDINATOR_GUIDE.md`

**Agents**: Start here → `AGENT_INSTRUCTIONS.md`

**Everyone**: Check progress → `PROGRESS.md`

---

**Status**: Ready for task assignment
**Last Updated**: 2025-12-23
**Version**: 1.0

Let's build! 🚀
