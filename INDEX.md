# NimsForest Document Index

**Purpose**: Automate the route to $1M ARR with 10 FTEs  
**Status**: Implementation ready

---

## 📋 Document Overview

NimsForest is an event-driven automation system that lets a small team operate at the scale of a much larger company.

---

## 🎯 Start Here

| If you want to... | Read this |
|-------------------|-----------|
| **Understand the vision** | [VISION.md](./VISION.md) |
| **See the implementation plan** | [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) |
| **Understand the architecture** | [README.md](./README.md) |
| **See technical specs** | [Cursorinstructions.md](./Cursorinstructions.md) |
| **Start implementing** | [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) → Phase 1 |

---

## 📚 Key Documents

### Strategic

| Document | Purpose |
|----------|---------|
| [VISION.md](./VISION.md) | Why we're building this. The goal, the problem, the solution. |
| [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) | Ordered task list. What to build, in what order. |

### Technical

| Document | Purpose |
|----------|---------|
| [README.md](./README.md) | Project overview, architecture, quick start. |
| [Cursorinstructions.md](./Cursorinstructions.md) | Detailed technical specification. |
| [EXTENSIBILITY_GUIDE.md](./EXTENSIBILITY_GUIDE.md) | How to add new trees and nims. |

---

## 📚 Complete Document List

### Core Documents (Read These)

1. **README.md**
   - **Purpose**: Project overview and entry point
   - **Audience**: Everyone
   - **Length**: Short (~5 min read)
   - **When**: Start here, always

2. **Cursorinstructions.md**
   - **Purpose**: Original technical specification
   - **Audience**: Agents implementing components
   - **Length**: Long (~30 min read)
   - **When**: Reference during implementation

3. **TASK_BREAKDOWN.md**
   - **Purpose**: Detailed task list with dependencies and deliverables
   - **Audience**: Coordinators and agents
   - **Length**: Long (~45 min read)
   - **When**: Planning and task assignment

4. **PROGRESS.md**
   - **Purpose**: Real-time status tracking
   - **Audience**: Everyone
   - **Length**: Living document
   - **When**: Update continuously, check daily

---

### Guide Documents (How-To)

5. **AGENT_INSTRUCTIONS.md**
   - **Purpose**: Step-by-step guide for agents executing tasks
   - **Audience**: Cloud agents
   - **Length**: Medium (~20 min read)
   - **When**: Before starting first task

6. **COORDINATOR_GUIDE.md**
   - **Purpose**: Guide for assigning and coordinating tasks
   - **Audience**: Project coordinators
   - **Length**: Medium (~25 min read)
   - **When**: Before assigning tasks

7. **QUICK_REFERENCE.md**
   - **Purpose**: Cheat sheet with quick lookups
   - **Audience**: Everyone
   - **Length**: Reference (skim as needed)
   - **When**: Keep open while working

---

### Supporting Documents

8. **SAMPLE_TASK_ASSIGNMENTS.md**
   - **Purpose**: Copy-paste ready task assignments
   - **Audience**: Coordinators
   - **Length**: Reference
   - **When**: Assigning tasks to agents

9. **INDEX.md**
   - **Purpose**: This file - navigation guide
   - **Audience**: Everyone
   - **Length**: Short (~5 min read)
   - **When**: Need to find a document

---

## 🗺️ Document Relationships

```
                    INDEX.md (You are here)
                         ↓
                    README.md
                    (Start here)
                    ↙         ↘
     COORDINATOR_GUIDE.md    AGENT_INSTRUCTIONS.md
              ↓                       ↓
    TASK_BREAKDOWN.md          Cursorinstructions.md
              ↓                       ↓
SAMPLE_TASK_ASSIGNMENTS.md     Implementation
              ↓                       ↓
         PROGRESS.md ←──────────────┘
              ↑
    (Updated by everyone)

    QUICK_REFERENCE.md
    (Reference anytime)
```

---

## 📖 Reading Paths

### Path 1: Coordinator Getting Started

1. `INDEX.md` ← You are here
2. `README.md` - Understand the project
3. `COORDINATOR_GUIDE.md` - Learn coordination process
4. `TASK_BREAKDOWN.md` - Understand all tasks
5. `SAMPLE_TASK_ASSIGNMENTS.md` - Get assignment templates
6. Assign Task 1.1 to first agent
7. Monitor `PROGRESS.md` daily

### Path 2: Agent Getting Started

1. `INDEX.md` ← You are here
2. `README.md` - Understand the project
3. `AGENT_INSTRUCTIONS.md` - Learn the process
4. Receive task assignment from coordinator
5. Read relevant section in `Cursorinstructions.md`
6. Check dependencies in `TASK_BREAKDOWN.md`
7. Implement and test
8. Update `PROGRESS.md`

### Path 3: Quick Lookup While Working

1. `QUICK_REFERENCE.md` - Find what you need fast
2. Jump to relevant detailed doc if needed

---

## 🎯 Document Purpose Matrix

| Document | Planning | Implementation | Coordination | Reference | Tracking |
|----------|----------|----------------|--------------|-----------|----------|
| README.md | ✅ | ✅ | ✅ | ✅ | - |
| Cursorinstructions.md | ✅ | ✅ | - | ✅ | - |
| TASK_BREAKDOWN.md | ✅ | - | ✅ | ✅ | - |
| PROGRESS.md | - | ✅ | ✅ | - | ✅ |
| AGENT_INSTRUCTIONS.md | - | ✅ | - | ✅ | - |
| COORDINATOR_GUIDE.md | ✅ | - | ✅ | ✅ | - |
| QUICK_REFERENCE.md | - | ✅ | ✅ | ✅ | - |
| SAMPLE_TASK_ASSIGNMENTS.md | - | - | ✅ | ✅ | - |
| INDEX.md | ✅ | - | - | ✅ | - |

---

## 📊 Information Architecture

### By Information Type

**Specifications** (What to build):

- `Cursorinstructions.md` - Detailed technical spec
- `TASK_BREAKDOWN.md` - Task-level specifications

**Processes** (How to build):

- `AGENT_INSTRUCTIONS.md` - Agent workflow
- `COORDINATOR_GUIDE.md` - Coordination workflow

**Status** (What's built):

- `PROGRESS.md` - Current state
- Test results in progress tracker

**Reference** (Quick help):

- `QUICK_REFERENCE.md` - Quick lookups
- `SAMPLE_TASK_ASSIGNMENTS.md` - Templates

**Navigation** (Where is it):

- `README.md` - Overview and entry
- `INDEX.md` - This file

---

## 🔍 Find Information By Question

| Question | Document | Section |
|----------|----------|---------|
| What is this project? | README.md | Overview |
| How do I start? | README.md | Quick Start |
| What's the architecture? | Cursorinstructions.md | Architecture |
| What are all the tasks? | TASK_BREAKDOWN.md | All phases |
| How do I implement X? | Cursorinstructions.md | Component sections |
| What's the status? | PROGRESS.md | Task status table |
| How do I execute a task? | AGENT_INSTRUCTIONS.md | Execute Your Task |
| How do I assign tasks? | COORDINATOR_GUIDE.md | Task Assignment |
| What depends on what? | TASK_BREAKDOWN.md | Dependency Graph |
| What's the timeline? | COORDINATOR_GUIDE.md | Estimated Timeline |
| Quick command reference? | QUICK_REFERENCE.md | Common Commands |
| Task assignment template? | SAMPLE_TASK_ASSIGNMENTS.md | Relevant task |

---

## 📐 Document Lengths

For planning your reading time:

| Document | Word Count | Reading Time | Type |
|----------|------------|--------------|------|
| README.md | ~1,000 | 5 min | Overview |
| INDEX.md | ~1,000 | 5 min | Navigation |
| Cursorinstructions.md | ~5,000 | 30 min | Specification |
| TASK_BREAKDOWN.md | ~7,000 | 45 min | Planning |
| AGENT_INSTRUCTIONS.md | ~3,500 | 20 min | Guide |
| COORDINATOR_GUIDE.md | ~4,000 | 25 min | Guide |
| QUICK_REFERENCE.md | ~2,000 | Skim | Reference |
| SAMPLE_TASK_ASSIGNMENTS.md | ~4,000 | Skim | Templates |
| PROGRESS.md | Variable | 2 min | Tracking |

**Total**: ~28,500 words, ~2.5 hours to read everything

**Minimum**: Read README.md (5 min) + your role's guide (20-25 min) = 30 min to start

---

## 🎯 Daily Usage Patterns

### Coordinator Daily Workflow

1. Morning: Check `PROGRESS.md` for updates
2. Review: Use `COORDINATOR_GUIDE.md` for decisions
3. Assign: Use `SAMPLE_TASK_ASSIGNMENTS.md` for templates
4. Reference: Use `QUICK_REFERENCE.md` for quick lookups
5. Evening: Update `PROGRESS.md` summary

### Agent Daily Workflow

1. Morning: Check task assignment and `PROGRESS.md`
2. Implement: Reference `Cursorinstructions.md` for details
3. Help: Use `AGENT_INSTRUCTIONS.md` for processes
4. Quick Lookup: Use `QUICK_REFERENCE.md` for commands
5. Complete: Update `PROGRESS.md`

---

## 🔄 Document Update Frequency

| Document | Updated By | Frequency | Version Control |
|----------|------------|-----------|-----------------|
| README.md | Coordinator | At milestones | Track changes |
| Cursorinstructions.md | No one | Never (source of truth) | Read-only |
| TASK_BREAKDOWN.md | Coordinator | If scope changes | Track changes |
| PROGRESS.md | Everyone | Continuously | Real-time |
| AGENT_INSTRUCTIONS.md | Coordinator | As needed | Track changes |
| COORDINATOR_GUIDE.md | Coordinator | As needed | Track changes |
| QUICK_REFERENCE.md | Coordinator | As needed | Track changes |
| SAMPLE_TASK_ASSIGNMENTS.md | Coordinator | As needed | Track changes |
| INDEX.md | Coordinator | Rarely | Track changes |

---

## 📱 Quick Access Bookmarks

### Most Frequently Accessed

1. `PROGRESS.md` - Check status (daily)
2. `QUICK_REFERENCE.md` - Quick lookups (hourly)
3. `Cursorinstructions.md` - Implementation details (during work)

### Occasional Reference

4. `AGENT_INSTRUCTIONS.md` - When stuck
5. `TASK_BREAKDOWN.md` - Check dependencies
6. `COORDINATOR_GUIDE.md` - Assignment help

### One-Time Reads

7. `README.md` - Initial orientation
8. `INDEX.md` - Navigation help
9. `SAMPLE_TASK_ASSIGNMENTS.md` - Copy templates

---

## 🎓 Learning Curve

### First Day

- Read: `README.md`, your role's guide
- Understand: Project goals, your workflow
- Setup: Development environment
- Start: Task 1.1 or receive assignment

### First Week

- Familiar with: Core documents, task structure
- Comfortable: Executing tasks, updating progress
- Reference: Spec and quick reference as needed

### Ongoing

- Master: Your domain (core/trees/nims)
- Contribute: Improvements to processes
- Help: Onboard new agents

---

## 🏁 Getting Started Checklist

### For Coordinators

- [ ] Read `README.md`
- [ ] Read `COORDINATOR_GUIDE.md`
- [ ] Review `TASK_BREAKDOWN.md`
- [ ] Prepare `SAMPLE_TASK_ASSIGNMENTS.md`
- [ ] Setup `PROGRESS.md` tracking
- [ ] Assign first task (1.1)

### For Agents

- [ ] Read `README.md`
- [ ] Read `AGENT_INSTRUCTIONS.md`
- [ ] Setup development environment
- [ ] Receive first task assignment
- [ ] Read relevant spec in `Cursorinstructions.md`
- [ ] Bookmark `QUICK_REFERENCE.md`
- [ ] Start implementing

---

## 💡 Pro Tips

1. **Keep `QUICK_REFERENCE.md` open** in a side tab while working
2. **Update `PROGRESS.md` immediately** when status changes
3. **Bookmark frequently accessed sections** in your browser
4. **Use Ctrl+F (Find)** to search within long documents
5. **Read relevant sections only** - you don't need to read everything
6. **Ask for clarification** rather than guessing
7. **Document discoveries** for the next person

---

## 🆘 Troubleshooting

**Can't find what you need?**

1. Check `QUICK_REFERENCE.md` first
2. Use document matrix above
3. Search in likely document with Ctrl+F
4. Check INDEX.md (this file)

**Document seems out of date?**

1. Check "Last Updated" date
2. Notify coordinator
3. Refer to `Cursorinstructions.md` as source of truth

**Conflicting information?**

1. `Cursorinstructions.md` is always correct (source of truth)
2. Other docs summarize/organize it
3. Notify coordinator of conflicts

---

## 📞 Document Maintainers

| Document | Primary Owner | Update Frequency |
|----------|--------------|------------------|
| README.md | Coordinator | Milestones |
| Cursorinstructions.md | Product/Architect | Never (fixed spec) |
| TASK_BREAKDOWN.md | Coordinator | Rarely |
| PROGRESS.md | All agents | Continuous |
| AGENT_INSTRUCTIONS.md | Coordinator | As needed |
| COORDINATOR_GUIDE.md | Coordinator | As needed |
| QUICK_REFERENCE.md | Coordinator | As needed |
| SAMPLE_TASK_ASSIGNMENTS.md | Coordinator | As needed |
| INDEX.md | Coordinator | Rarely |

---

## ✅ Document Quality Checklist

Each document should be:

- [ ] Clear and concise
- [ ] Up to date
- [ ] Cross-referenced where relevant
- [ ] Accessible to target audience
- [ ] Actionable (where applicable)
- [ ] Versioned or dated

---

## 🎯 Success Metrics

You're using the documentation well when:

- ✅ You can find information quickly
- ✅ Tasks are clear and actionable
- ✅ Progress is visible to all
- ✅ Fewer questions about "where is X?"
- ✅ New agents onboard smoothly
- ✅ Work proceeds without blockers

---

**Remember**: These documents exist to make your work easier. If something is unclear or hard to find, that's a documentation bug - report it!

**Navigation Tip**: Most documents link to each other. Follow the links to jump between related content.

**Happy Building!** 🚀

---

Last Updated: 2025-12-23
