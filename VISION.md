# 🌲 NimsForest

Event-driven automation. Lua rules. LLM decisions.

---

## What It Is

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
    prompt: |
      Score: {{.score}}
      Should we pursue? Reply YES or NO with reason.
```

```lua
-- scripts/scoring.lua
function process(contact)
    local score = 0
    if contact.company_size > 200 then score = score + 40 end
    if contains(contact.title, "VP") then score = score + 30 end
    return { contact_id = contact.id, score = score }
end
```

---

## Core Primitives

| Primitive | What |
|-----------|------|
| **River** | NATS. Events flow through. |
| **Source** | Feeds external data into River. |
| **TreeHouse** | Lua script. Deterministic. |
| **Nim** | Prompt → LLM. Non-deterministic. |
| **Leaf** | An event. |

---

## Architecture

```
Source ──► River (NATS) ──► TreeHouse (Lua) ──► Nim (LLM) ──► River
```

Components subscribe to subjects. That's it.

---

## TreeHouse (Lua)

Deterministic rules. Same input = same output.

```lua
function process(input)
    -- your logic
    return output
end
```

Helpers: `contains(str, sub)`, `json.encode(t)`, `json.decode(s)`, `log(msg)`

---

## Nim (Prompt)

Non-deterministic. LLM decides.

### MVP: One-shot

```yaml
nims:
  triage:
    subscribes: ticket.created
    publishes: ticket.triaged
    prompt: scripts/nims/triage.md
```

Receives event → calls Claude → publishes response.

### Future: Agentic

```yaml
nims:
  resolver:
    subscribes: ticket.escalated
    publishes: ticket.resolved
    prompt: scripts/nims/resolver.md
    tools: [search_kb, read_docs, draft_response]
    checkpoint: human_approval
```

Reason → use tools → checkpoint for human → continue → complete.

Like Cursor: autonomous work with human oversight.

---

## Config

```yaml
# config/forest.yaml

treehouses:
  name:
    subscribes: subject.in
    publishes: subject.out
    script: path/to/script.lua

nims:
  name:
    subscribes: subject.in
    publishes: subject.out
    prompt: scripts/nims/name.md   # Path to prompt file
```

---

## MVP File Structure

```
nimsforest/
├── cmd/forest/main.go
├── pkg/
│   ├── brain/              # LLM (exists)
│   └── runtime/
│       ├── config.go       # YAML loader
│       ├── lua.go          # Lua VM
│       ├── treehouse.go    # TreeHouse runtime
│       └── nim.go          # Nim runtime
├── config/
│   └── forest.yaml
└── scripts/
    ├── treehouses/
    │   └── scoring.lua
    └── nims/
        └── qualify.md
```

---

## Principles

1. **TreeHouses are Lua.** Deterministic. Testable.
2. **Nims are prompts.** LLM does the thinking.
3. **Everything subscribes to River.** No orchestrator.
4. **Config declares what exists.** Code implements how.
