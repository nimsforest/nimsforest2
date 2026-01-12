# Future Phases - Deferred Work

**Status**: 📋 Planned (not yet prioritized)
**Goal**: Complete AAA infrastructure after first flow works
**Timing**: After Phase C completion

---

## 🎯 Overview

These phases expand beyond the "first working flow" to complete the full AAA architecture described in [plan-aaa-nim.md](../../plan-aaa-nim.md).

**Current Strategy**: Get CoderNim working end-to-end first (Phases A-C), then decide which expansions are needed.

---

## 📋 Potential Future Phases

### Phase D: Human Agent & Songbird

**Goal**: Enable human-in-the-loop via messaging platforms

**Scope**:
- Extend Songbird interface with `Send()` method
- Implement event-driven response handling
- Create HumanAgent implementation
- Add correlation ID tracking
- Test with Telegram (existing Songbird)

**Value**: Enables approval workflows, human review, escalations

**Estimated Time**: 2-3 hours

**Dependencies**: Phase C complete

---

### Phase E: Robot & Browser Agents

**Goal**: Support physical robots and web automation

**Scope**:
- RobotAgent implementation (HTTP API calls)
- BrowserAgent implementation (Playwright in Docker)
- Wire into AgentHouse
- Add agent type routing

**Value**: Enables robotic process automation, UI testing, web scraping

**Estimated Time**: 2-3 hours

**Dependencies**: Phase C complete

**Risk**: May not be needed initially

---

### Phase F: CoderNim.Automate()

**Goal**: Dynamically create TreeHouses and Nims

**Scope**:
- Implement Automate() method
- AI-driven analysis (TreeHouse vs Nim)
- Generate Lua scripts for TreeHouses
- Generate config for runtime Nims
- Human review workflow

**Value**: Self-extending system, AI creates automations

**Estimated Time**: 3-4 hours

**Dependencies**: Phase C complete, possibly Phase D (for review)

**Risk**: Complex, may need architectural refinement

---

### Phase G: Code Reorganization

**Goal**: Clean up misnamed components and examples

**Scope**:
- Move `examples/` to top level
- Migrate domain-specific code from `internal/`
- Rename "Nims" that are actually TreeHouses
- Update imports and tests
- Documentation cleanup

**Value**: Clearer codebase structure, better examples

**Estimated Time**: 1-2 hours

**Dependencies**: None (can do anytime)

**Risk**: Low (mostly mechanical refactoring)

---

### Phase H: Brain Integration

**Goal**: Move Brain to pkg/nim and integrate with Nims

**Scope**:
- Move `pkg/brain/` → `pkg/nim/brain.go`
- Update all imports
- Integrate with CoderNim
- Add memory/retrieval to AAA flow

**Value**: Nims can remember and learn from past interactions

**Estimated Time**: 1-2 hours

**Dependencies**: Phase C complete

**Risk**: May conflict with existing Brain usage

---

### Phase I: Multi-Land Coordination

**Goal**: Nims can discover and use remote Lands

**Scope**:
- Implement Land query broadcast
- Collect multiple responses
- Implement Land selection strategy
- Task queuing and retry
- Load balancing across Lands

**Value**: True distributed execution, horizontal scaling

**Estimated Time**: 4-5 hours

**Dependencies**: Phase C complete, multiple nodes in cluster

**Risk**: Complexity of distributed systems

---

### Phase J: Advanced Agent Features

**Goal**: Production-grade agent execution

**Scope**:
- Agent timeout and cancellation
- Progress streaming
- Task priorities
- Resource quotas
- Failure recovery
- Agent health monitoring

**Value**: Robust, production-ready agent system

**Estimated Time**: 5-6 hours

**Dependencies**: Phase C complete, real-world usage

**Risk**: Scope creep, over-engineering

---

## 🎯 Prioritization Framework

### Immediate Value (After Phase C)
1. **Phase G** (Code Reorganization) - Low effort, high clarity
2. **Phase D** (Human Agent) - Enables real workflows

### High Value (When Needed)
3. **Phase F** (Automate) - Core AAA feature
4. **Phase I** (Multi-Land) - Enables true distribution

### Nice to Have (Future)
5. **Phase E** (Robot/Browser) - Specialized use cases
6. **Phase H** (Brain Integration) - Memory/learning
7. **Phase J** (Advanced Features) - Production hardening

---

## 📊 Estimated Total Effort

| Phase | Hours | Priority |
|-------|-------|----------|
| A (✅) | 1 | Critical |
| B (🎯) | 1.5 | Critical |
| C (⏳) | 3 | Critical |
| **Core Total** | **5.5** | - |
| D | 2-3 | High |
| E | 2-3 | Medium |
| F | 3-4 | High |
| G | 1-2 | High |
| H | 1-2 | Medium |
| I | 4-5 | Medium |
| J | 5-6 | Low |
| **Full Total** | **24-31** | - |

---

## 🤔 Decision Points

### After Phase C: Validate Core
**Questions**:
- Does the event-driven AAA pattern work well?
- Is Docker agent execution reliable?
- What's the performance/latency?

**Possible Outcomes**:
- ✅ Works great → Proceed with expansions
- ⚠️ Needs refinement → Iterate on core
- ❌ Fundamental issues → Rethink architecture

---

### After First Real Use Case
**Questions**:
- Which agent types are actually needed?
- Is Automate() a real requirement?
- Do we need multi-Land coordination?

**Possible Outcomes**:
- Focus on specific phases based on needs
- Discover new requirements not in original plan
- Defer phases that aren't valuable

---

## 📝 Notes on Original Plan

### What We're Deferring (from plan-aaa-nim.md)

**Part 2: Songbird Extensions**
- Send() method for outbound messages
- Slack and Email Songbirds
- Response correlation
- → **Phase D**

**Part 3: Agent Types**
- HumanAgent, RobotAgent, BrowserAgent
- All concrete implementations
- → **Phases D & E**

**Part 4: CoderNim.Automate()**
- TreeHouse generation
- Runtime Nim creation
- AI-driven analysis
- → **Phase F**

**Part 5: Code Reorganization**
- examples/ directory
- Rename misnamed components
- pkg/brain migration
- → **Phases G & H**

**Part 6: Advanced Features**
- Multi-Land queries
- Load balancing
- Agent pools
- → **Phases I & J**

### What We Kept (in Phases A-C)

**Essential Core**:
- Land detection (Phase A) ✅
- LandHouse & AgentHouse (Phase B) 🎯
- AIAgent (Phase C) ⏳
- CoderNim with Advice & Action (Phase C) ⏳
- pkg/nim interfaces (Phase C) ⏳

**Rationale**: Minimum viable AAA system

---

## 🔄 Iterative Approach

```
Phase A → Phase B → Phase C
    ↓
Validate & Learn
    ↓
┌─────────────────────────────────────┐
│ Which phases add most value?        │
│ - Do we need Robot agents?          │
│ - Is Automate() important?          │
│ - Multi-Land or single-node first?  │
└─────────────────────────────────────┘
    ↓
Select Next Phase(s)
    ↓
Implement & Validate
    ↓
Repeat
```

---

## ⚠️ Risks of Big Bang Approach

**Why we split the plan:**

1. **Uncertainty**: Don't know if core design works until tested
2. **Changing Requirements**: Real usage reveals different needs
3. **Wasted Effort**: Building features that aren't used
4. **Complexity**: Harder to debug when everything is new
5. **Integration Issues**: Multiple new pieces increase failure modes

**Mitigation**: Iterative phases with validation gates

---

## ✅ Success Criteria for Phase Selection

Before starting a future phase, answer:

1. **Need**: Is this solving a real problem we have?
2. **Dependencies**: Are prerequisites complete and stable?
3. **Scope**: Can we complete this in one focused session?
4. **Value**: What specific capability does this unlock?
5. **Test**: How will we validate it works?

If any answer is unclear, defer the phase.

---

## 📖 Creating New Phase Plans

When ready to tackle a future phase:

1. **Copy template** from phase-b or phase-c
2. **Define clear objective** (one sentence)
3. **List specific tasks** (4-8 tasks max)
4. **Include code snippets** for key implementations
5. **Specify validation** (tests, integration checks)
6. **Estimate time** based on actual Phase A-C times
7. **Note dependencies** explicitly

**Location**: `docs/plans/phase-{letter}-{name}.md`

---

## 🔗 References

- **Architecture**: [plan-aaa-nim.md](../../plan-aaa-nim.md) - Parts 1-10
- **Roadmap**: [IMPLEMENTATION_ROADMAP.md](../../IMPLEMENTATION_ROADMAP.md)
- **Vision**: [docs/roadmap/VISION.md](../roadmap/VISION.md)

---

**Last Updated**: 2026-01-12
**Current Status**: Documenting deferred work
**Next Review**: After Phase C completion
