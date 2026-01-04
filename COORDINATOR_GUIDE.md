# Coordinator Guide - NimsForest Project

## Quick Reference for Coordinating Multiple Cloud Agents

This guide helps you efficiently assign and coordinate tasks across multiple cloud agents working on the NimsForest prototype.

---

## Document Structure

1. **Cursorinstructions.md** - Original specification (DO NOT MODIFY)
2. **TASK_BREAKDOWN.md** - Detailed task breakdown with dependencies
3. **AGENT_INSTRUCTIONS.md** - Instructions for agents executing tasks
4. **PROGRESS.md** - Real-time progress tracking
5. **COORDINATOR_GUIDE.md** - This file

---

## Quick Start Workflow

### Step 1: Assign First Task

Start with Task 1.1 (Infrastructure Setup) - it has no dependencies.

**Agent Assignment Template**:

```
Task: 1.1 - Project Infrastructure Setup
Reference: TASK_BREAKDOWN.md - Phase 1, Task 1.1
Instructions: AGENT_INSTRUCTIONS.md
Spec: Cursorinstructions.md - Tech Stack & Project Structure sections
Update: PROGRESS.md when starting and completing
```

### Step 2: Parallel Phase 2

Once Task 1.1 is complete, assign all Phase 2 tasks simultaneously to different agents:

- **Agent A**: Task 2.1 (Leaf) - Simple, quick
- **Agent B**: Task 2.3 (River) - Independent
- **Agent C**: Task 2.4 (Soil) - Independent  
- **Agent D**: Task 2.5 (Humus) - Independent

**Agent B-D can work fully in parallel**. Agent A should complete quickly, then move to Task 2.2.

### Step 3: Continue Through Phases

Follow the dependency graph in TASK_BREAKDOWN.md.

---

## Task Assignment Template

When assigning a task to a cloud agent, use this template:

```markdown
# Task Assignment: [Task ID] - [Component Name]

## Task Details
- **Task ID**: [e.g., 2.2]
- **Component**: [e.g., Wind - NATS Core Pub/Sub]
- **Complexity**: [Low/Medium/High]
- **Estimated Time**: [e.g., 4-6 hours]

## Dependencies
[List required completed tasks or state "None"]
- Task X.X - [Component] - Status: [✅/🏃/⏳]

## References
- **Full Spec**: `Cursorinstructions.md` - Section [X]
- **Task Details**: `TASK_BREAKDOWN.md` - Phase [X], Task [X.X]
- **Instructions**: `AGENT_INSTRUCTIONS.md`

## Deliverables
[Copy from TASK_BREAKDOWN.md]

## Acceptance Criteria
[Copy from TASK_BREAKDOWN.md]

## Testing Requirements
- Unit tests required: Yes
- Integration tests required: [Yes/No]
- Minimum coverage: 80%

## Progress Tracking
- Update `PROGRESS.md` when starting (🏃 In Progress)
- Update `PROGRESS.md` when complete (✅ Complete)
- Report any blockers immediately

## Next Steps After Completion
[List tasks that will be unblocked by this task's completion]

---

Start Time: [Leave blank for agent to fill]
Completion Time: [Leave blank for agent to fill]
Agent Notes: [Leave blank for agent to fill]
```

---

## Batch Assignment Strategy

### Batch 1: Foundation (Sequential)

```
Agent 1 → Task 1.1 (Infrastructure)
Wait for completion ✅
```

### Batch 2: Core Components Part 1 (Parallel)

```
Agent 1 → Task 2.1 (Leaf) [Quick]
Agent 2 → Task 2.3 (River)
Agent 3 → Task 2.4 (Soil)
Agent 4 → Task 2.5 (Humus)

Wait for 2.1 ✅
```

### Batch 3: Core Components Part 2 (Parallel)

```
Agent 1 → Task 2.2 (Wind) [needs 2.1]
Agent 2 → Task 3.3 (Decomposer) [needs 2.4, 2.5]
Agent 3 → Task 4.1 (Leaf Types) [needs 2.1]

Wait for 2.2, 2.3, 2.4, 2.5 ✅
```

### Batch 4: Base Interfaces (Parallel)

```
Agent 1 → Task 3.1 (Base Tree) [needs 2.1, 2.2, 2.3]
Agent 2 → Task 3.2 (Base Nim) [needs 2.1, 2.2, 2.4, 2.5]

Wait for 3.1, 3.2, 4.1 ✅
```

### Batch 5: Examples (Parallel)

```
Agent 1 → Task 4.2 (Payment Tree) [needs 3.1, 4.1]
Agent 2 → Task 4.3 (AfterSales Nim) [needs 3.2, 4.1]

Wait for 4.2, 4.3 ✅
```

### Batch 6: Main Application (Sequential)

```
Agent 1 → Task 5.1 (Main)

Wait for 5.1 ✅
```

### Batch 7: Testing & Docs (Parallel)

```
Agent 1 → Task 6.1 (E2E Testing)
Agent 2 → Task 6.2 (Documentation)
```

---

## Dependency Checker

Before assigning a task, verify all dependencies are complete:

**Task 2.2 (Wind)**:

- ✅ Task 1.1 (Infrastructure) - Complete?
- ✅ Task 2.1 (Leaf) - Complete?

**Task 3.1 (Base Tree)**:

- ✅ Task 2.1 (Leaf) - Complete?
- ✅ Task 2.2 (Wind) - Complete?
- ✅ Task 2.3 (River) - Complete?

**Task 3.2 (Base Nim)**:

- ✅ Task 2.1 (Leaf) - Complete?
- ✅ Task 2.2 (Wind) - Complete?
- ✅ Task 2.4 (Soil) - Complete?
- ✅ Task 2.5 (Humus) - Complete?

**Task 4.2 (Payment Tree)**:

- ✅ Task 3.1 (Base Tree) - Complete?
- ✅ Task 4.1 (Leaf Types) - Complete?

**Task 4.3 (AfterSales Nim)**:

- ✅ Task 3.2 (Base Nim) - Complete?
- ✅ Task 4.1 (Leaf Types) - Complete?

**Task 5.1 (Main)**:

- ✅ Task 3.3 (Decomposer) - Complete?
- ✅ Task 4.2 (Payment Tree) - Complete?
- ✅ Task 4.3 (AfterSales Nim) - Complete?

---

## Agent Capability Matching

### Low Complexity Tasks (Good for new agents)

- Task 1.1 - Infrastructure Setup
- Task 2.1 - Leaf Types
- Task 4.1 - Leaf Type Definitions
- Task 6.2 - Documentation

### Medium Complexity Tasks

- Task 2.2 - Wind
- Task 2.3 - River
- Task 2.4 - Soil
- Task 2.5 - Humus
- Task 3.1 - Base Tree
- Task 3.2 - Base Nim
- Task 3.3 - Decomposer
- Task 4.2 - Payment Tree
- Task 4.3 - AfterSales Nim
- Task 5.1 - Main Application
- Task 7.2 - Monitoring

### High Complexity Tasks (Experienced agents)

- Task 6.1 - End-to-End Testing
- Task 7.3 - Performance Testing

---

## Progress Monitoring

### Daily Checklist

- [ ] Review PROGRESS.md for status updates
- [ ] Check for blocked tasks (❌ or ⚠️ status)
- [ ] Verify agents have updated their status
- [ ] Identify next tasks ready to assign
- [ ] Review test results
- [ ] Check for integration issues

### Key Metrics to Track

1. **Tasks Complete**: X / 18
2. **Current Active Tasks**: Count of 🏃
3. **Blocked Tasks**: Count of ❌
4. **Test Pass Rate**: Passing / Total
5. **Overall Progress**: X%

### Red Flags

- ⚠️ Task in progress for >24 hours without update
- ⚠️ Multiple tasks blocked on same dependency
- ⚠️ Tests failing consistently
- ⚠️ Agent unresponsive
- ⚠️ Dependencies out of order

---

## Communication Templates

### Task Completion Notification

```
Task [X.X] Complete ✅

Component: [Name]
Completed by: [Agent]
Test Results: [Pass/Fail]
Coverage: [X%]
Notes: [Any important notes]

Unblocked Tasks:
- Task [Y.Y] - Ready to assign
- Task [Z.Z] - Ready to assign
```

### Blocker Report

```
Task [X.X] Blocked ❌

Component: [Name]
Blocked by: [Agent/Task]
Issue: [Description]
Impact: [Which tasks are affected]
Proposed Solution: [If known]
Priority: [High/Medium/Low]
```

### Issue Escalation

```
Issue with Task [X.X] ⚠️

Component: [Name]
Agent: [Name]
Issue Type: [Technical/Process/Dependency]
Description: [Details]
Attempted Solutions: [What was tried]
Help Needed: [Specific request]
```

---

## Common Issues & Solutions

### Issue: Agent can't connect to NATS

**Solution**: Verify Task 1.1 complete, NATS server running

```bash
ps aux | grep nats-server
make start
curl http://localhost:8222/varz
```

### Issue: Import cycle detected

**Solution**: Check component dependencies, core shouldn't import examples

### Issue: Tests timing out

**Solution**: Add timeouts, check for goroutine leaks, cleanup subscriptions

### Issue: Optimistic locking failures

**Solution**: Implement retry logic, check revision handling

### Issue: Task dependencies unclear

**Solution**: Refer to dependency graph in TASK_BREAKDOWN.md

---

## Quality Gates

Before marking Phase complete, verify:

### Phase 1 Complete

- [ ] go.mod exists and valid
- [ ] NATS server binary installed
- [ ] NATS server can be started with `make start`
- [ ] NATS starts successfully with JetStream
- [ ] Directory structure created
- [ ] Test program verifies connectivity

### Phase 2 Complete

- [ ] All core components implemented
- [ ] Unit tests pass for all
- [ ] Integration tests pass
- [ ] Coverage >80% for each component

### Phase 3 Complete

- [ ] Interfaces defined
- [ ] Base implementations working
- [ ] Decomposer running
- [ ] Tests pass

### Phase 4 Complete

- [ ] Example tree works
- [ ] Example nim works
- [ ] Leaf types defined
- [ ] Tests pass

### Phase 5 Complete

- [ ] Application starts
- [ ] All components initialized
- [ ] Graceful shutdown works
- [ ] Basic integration works

### Phase 6 Complete

- [ ] E2E test passes
- [ ] Documentation complete
- [ ] Setup guide works
- [ ] Examples documented

---

## Rollback Procedure

If a task needs to be redone:

1. **Mark status**: Change to ⏸️ Paused
2. **Document issue**: Add to PROGRESS.md issues section
3. **Notify dependents**: Alert agents with blocked tasks
4. **Reassign**: Assign to same or different agent
5. **Track**: Monitor closely until resolved

---

## Success Metrics

### Definition of Done (Per Task)

- ✅ All deliverables implemented
- ✅ Unit tests written and passing
- ✅ Integration tests (if required) passing
- ✅ Code formatted and linted
- ✅ Documentation added
- ✅ PROGRESS.md updated
- ✅ No blocking issues

### Definition of Done (Project)

- ✅ All Phase 1-5 tasks complete
- ✅ E2E test demonstrates full flow
- ✅ Documentation allows new dev to onboard
- ✅ All tests passing
- ✅ Code quality standards met

---

## Estimated Timeline

With 4 agents working in parallel:

| Phase | Duration | Cumulative |
|-------|----------|------------|
| 1     | 1 hour   | 1 hour     |
| 2     | 8 hours  | 9 hours    |
| 3     | 6 hours  | 15 hours   |
| 4     | 6 hours  | 21 hours   |
| 5     | 4 hours  | 25 hours   |
| 6     | 8 hours  | 33 hours   |

**Total**: ~33 hours (~4-5 days with parallel execution)

With 2 agents: ~6-8 days
With 1 agent: ~2-3 weeks

---

## Agent Onboarding Checklist

New agent joining the project:

- [ ] Read `Cursorinstructions.md` (full context)
- [ ] Read `AGENT_INSTRUCTIONS.md` (how to work)
- [ ] Review `TASK_BREAKDOWN.md` (understand structure)
- [ ] Check `PROGRESS.md` (current state)
- [ ] Setup local environment (Go, NATS binary)
- [ ] Run `make start` to verify setup
- [ ] Receive first task assignment
- [ ] Update PROGRESS.md with name and task

---

## End-of-Day Checklist

- [ ] All agents have updated PROGRESS.md
- [ ] No tasks stuck in 🏃 without updates
- [ ] Tomorrow's task assignments ready
- [ ] Blockers identified and prioritized
- [ ] Test results reviewed
- [ ] Backup/commit all work

---

## Final Delivery Checklist

Before considering project complete:

- [ ] All Phase 1-5 tasks marked ✅
- [ ] All tests passing (unit + integration)
- [ ] E2E test demonstrates: webhook → river → tree → leaf → wind → nim → humus → soil
- [ ] README.md has complete setup instructions
- [ ] New developer can clone and run successfully
- [ ] Code coverage >80%
- [ ] No lint errors
- [ ] Documentation complete
- [ ] Example flows documented and working

---

## Quick Reference Commands

### Check overall status

```bash
grep "✅ Complete" PROGRESS.md | wc -l  # Count completed tasks
grep "🏃 In Progress" PROGRESS.md      # Show active tasks
grep "❌ Blocked" PROGRESS.md          # Show blocked tasks
```

### Run all tests

```bash
make start
go test ./... -v
go test ./... -cover
go test ./... -race
go test ./... -tags=integration
```

### Check code quality

```bash
go fmt ./...
go vet ./...
golangci-lint run
```

---

## Contact & Escalation

For issues requiring human intervention:

1. Document in PROGRESS.md issues section
2. Mark affected tasks with ⚠️
3. Include: Issue description, impact, attempted solutions
4. Continue with non-blocked tasks

---

**Remember**: The goal is working software. Prioritize:

1. ✅ Working code over perfect code
2. ✅ Tests passing over feature complete
3. ✅ Progress over perfection
4. ✅ Communication over surprise

Good luck! 🚀
