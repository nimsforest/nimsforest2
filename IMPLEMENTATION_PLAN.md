# Implementation Plan

## MVP (Do This First)

### 1. Config Loader
- [ ] `pkg/runtime/config.go` - Parse YAML
- [ ] TreeHouse: name, subscribes, publishes, script
- [ ] Nim: name, subscribes, publishes, prompt, model

### 2. Lua Runtime
- [ ] `pkg/runtime/lua.go` - Lua VM wrapper
- [ ] `process(input)` → output
- [ ] Helpers: `contains`, `json.encode`, `json.decode`

### 3. TreeHouse Runtime
- [ ] `pkg/runtime/treehouse.go`
- [ ] Subscribe to NATS subject
- [ ] Load + run Lua script
- [ ] Publish result

### 4. Nim Runtime
- [ ] `pkg/runtime/nim.go`
- [ ] Subscribe to NATS subject
- [ ] Render prompt template with `{{.field}}`
- [ ] Call brain (Claude for MVP)
- [ ] Publish result

### 5. Main
- [ ] Load `config/forest.yaml`
- [ ] Start TreeHouses
- [ ] Start Nims
- [ ] Wait for shutdown

### 6. Example
- [ ] `config/forest.yaml` - one TreeHouse, one Nim
- [ ] `scripts/treehouses/scoring.lua`

---

## Post-MVP

### Sources
- [ ] Source interface
- [ ] WebhookSource (generic)
- [ ] StripeSource
- [ ] SalesforceSource

### More Brains
- [ ] OpenAI
- [ ] Gemini

### CLI
- [ ] `--config` flag
- [ ] `--nats` flag

### Tests
- [ ] Unit tests
- [ ] Integration tests

---

## MVP Flow

```
Manual publish to NATS     TreeHouse (Lua)        Nim (Claude)
        │                       │                      │
        │  contact.created      │                      │
        ├──────────────────────►│                      │
        │                       │  lead.scored         │
        │                       ├─────────────────────►│
        │                       │                      │  lead.qualified
        │                       │                      ├────────────────►
```

For MVP: publish test events directly to NATS. Sources come later.
