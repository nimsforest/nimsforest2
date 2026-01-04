# NimsForest

Event-driven automation. Lua rules. LLM decisions.

---

## Start Here

1. **[VISION.md](./VISION.md)** — What it is
2. **[EXAMPLE.md](./EXAMPLE.md)** — Working example
3. **[IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md)** — What to build

---

## Core Concepts

| Concept | What |
|---------|------|
| **River** | NATS event stream |
| **Source** | Feeds data into River |
| **TreeHouse** | Lua script (deterministic) |
| **Nim** | Prompt → LLM (non-deterministic) |

---

## MVP

```yaml
# forest.yaml
treehouses:
  scoring:
    subscribes: contact.created
    publishes: lead.scored
    script: scripts/scoring.lua

nims:
  qualify:
    subscribes: lead.scored
    publishes: lead.qualified
    prompt: "Score: {{.score}}. Pursue? YES/NO."
```

That's it. Config + Lua scripts + prompts.
