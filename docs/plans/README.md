# Implementation Plans - AAA Nim Architecture

This directory contains **phase-specific implementation plans** optimized for agentic coding.

---

## 📚 Plan Structure

### Reference Documents
- **[plan-aaa-nim.md](../../plan-aaa-nim.md)**: Complete architecture reference (2,400+ lines)
  - Part 1-7: System architecture and design
  - Part 8-10: Implementation details and rationale
  - **Use as**: Reference for "why" and "how" decisions

### Executable Plans (This Directory)
Each phase is a **self-contained, actionable plan** that an agent can execute independently:

| Phase | Status | File | Goal | Time |
|-------|--------|------|------|------|
| **A** | ✅ Complete | [phase-a-foundation.md](./phase-a-foundation.md) | Land detection foundation | ~45 min |
| **B** | 🎯 Next | [phase-b-communication.md](./phase-b-communication.md) | Event-driven communication | ~1.5 hrs |
| **C** | ⏳ Blocked | [phase-c-first-flow.md](./phase-c-first-flow.md) | CoderNim AAA implementation | ~3 hrs |
| **D** | 📋 Future | [future-phases.md](./future-phases.md) | Additional agents, cleanup | TBD |

---

## 🎯 Design Principles

### Why Split Plans?

1. **Bounded Scope**: Each plan has 4-8 related tasks
2. **Clear Dependencies**: Phase N requires Phase N-1 complete
3. **Incremental Validation**: Test after each phase
4. **Reduced Context**: Agent loads 300-500 lines, not 2,400
5. **Better Tracking**: One phase = one PR = clear progress

### What Makes a Good Phase?

✅ **Good Phase:**
- Single clear objective
- 4-8 cohesive tasks
- Testable milestone at end
- Can complete in one session
- Dependencies explicit

❌ **Avoid:**
- >10 tasks (too complex)
- Mixed concerns (land + songbirds)
- Unclear validation
- Hidden dependencies

---

## 🚀 Using These Plans

### For Agents

1. **Read current phase plan** (not the entire reference)
2. **Execute tasks in order** (unless marked independent)
3. **Validate milestones** before proceeding
4. **Update status** in plan file as you go
5. **Move to next phase** only when current complete

### For Humans

1. **Review completed work** against phase milestones
2. **Identify blockers** early
3. **Adjust estimates** based on actuals
4. **Provide feedback** on plan clarity

---

## 📋 Current Progress

### Completed
- ✅ **Phase A**: Land detection wired into Forest startup
  - Files: `internal/core/land.go`, `internal/land/detect.go`
  - Milestone: Forest logs detected Land type on startup

### In Progress
- 🎯 **Phase B**: Communication infrastructure (next)
  - Tasks: LandHouse, AgentHouse, unit tests
  - Milestone: Can query Land capacity via Wind

### Upcoming
- ⏳ **Phase C**: CoderNim with AAA methods
  - Blocked by: Phase B completion
  - Estimated: 3 hours

---

## 🔄 Flow Overview

```
┌─────────────────────────────────────────────────────────────┐
│ Phase A: Foundation                                         │
│ - Detect Land (RAM, CPU, Docker, GPU)                      │
│ - Create LandInfo struct                                   │
│ - Wire into Forest startup                                 │
│ ✅ Complete                                                 │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│ Phase B: Communication                                      │
│ - LandHouse responds to capacity queries                   │
│ - AgentHouse executes tasks in Docker                      │
│ - Houses wire into Forest lifecycle                        │
│ 🎯 Next                                                     │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│ Phase C: First AAA Flow                                     │
│ - CoderNim implements Advice/Action/Handle                 │
│ - AIAgent runs in Docker containers                        │
│ - End-to-end integration test                              │
│ ⏳ Blocked by Phase B                                       │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│ Phase D+: Expansion                                         │
│ - Human/Robot/Browser agents                               │
│ - Songbird.Send() extension                                │
│ - Cleanup and reorganization                               │
│ 📋 Future work                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 🚦 Milestones

Track overall progress:

- [x] **M1**: Forest detects and logs Land type ✅
- [ ] **M2**: LandHouse responds to capacity queries
- [ ] **M3**: AgentHouse executes Docker tasks
- [ ] **M4**: CoderNim.Advice() works
- [ ] **M5**: CoderNim.Action() dispatches and receives results
- [ ] **M6**: End-to-end integration test passes

---

## 📊 Time Tracking

| Phase | Estimated | Actual | Notes |
|-------|-----------|--------|-------|
| A     | 45 min    | ~1 hr  | Included setup and debugging |
| B     | 1.5 hrs   | -      | Not started |
| C     | 3 hrs     | -      | Not started |
| **Total** | **5 hrs** | **1 hr** | **20% complete** |

---

## 📝 Conventions

### Status Markers
- ✅ Complete: All milestones met
- 🎯 Next: Ready to start
- ⏳ Blocked: Waiting on dependencies
- 📋 Future: Not yet planned in detail

### File Naming
- `phase-X-description.md`: Executable phase plan
- `reference-topic.md`: Architecture reference
- `future-*.md`: Deferred work

### Task Numbering
- Tasks numbered globally across phases
- Phase A: Tasks 1-4
- Phase B: Tasks 5-8
- Phase C: Tasks 9-16

---

## 🔗 Related Documents

- **[IMPLEMENTATION_ROADMAP.md](../../IMPLEMENTATION_ROADMAP.md)**: High-level roadmap
- **[plan-aaa-nim.md](../../plan-aaa-nim.md)**: Complete architecture
- **[docs/roadmap/](../roadmap/)**: Vision and strategy documents

---

## ❓ FAQ

### Why not just use the big plan?

**Problem**: 2,400 line document is overwhelming for agents
**Solution**: Split into 300-500 line focused plans

### When should I read the reference?

- When you need architectural context
- When making design decisions
- When understanding "why" behind tasks
- When extending beyond planned phases

### How do I know a phase is complete?

Each phase has:
1. **Milestones**: Clear pass/fail criteria
2. **Tests**: Automated validation
3. **Definition of Done**: Checklist at end

### Can I execute tasks in parallel?

- Some tasks marked "Independent" can run parallel
- Most tasks have order dependencies
- Default: execute sequentially

### What if I get stuck?

1. Check phase "Implementation Notes"
2. Review reference document for context
3. Check "Edge Cases to Handle"
4. Ask for clarification with specific task number

---

**Last Updated**: 2026-01-12
**Current Phase**: B (Communication Infrastructure)
**Next Milestone**: M2 (LandHouse responds to queries)
