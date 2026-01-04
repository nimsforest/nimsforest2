# Implementation Plan

## MVP (Do This First)

### 1. Config Loader
- [ ] `pkg/runtime/config.go` - Parse YAML
- [ ] TreeHouse: name, subscribes, publishes, script (path)
- [ ] Nim: name, subscribes, publishes, prompt (path)

### 2. Lua Runtime
- [ ] `pkg/runtime/lua.go` - Lua VM wrapper
- [ ] Load script from file
- [ ] Call `process(input)` → output
- [ ] Helpers: `contains`, `json.encode`, `json.decode`, `log`

### 3. TreeHouse Runtime
- [ ] `pkg/runtime/treehouse.go`
- [ ] Subscribe to NATS subject
- [ ] On message: decode JSON → Lua table → call process() → encode result
- [ ] Publish result to output subject

### 4. Nim Runtime
- [ ] `pkg/runtime/nim.go`
- [ ] Subscribe to NATS subject
- [ ] Load prompt template from `.md` file
- [ ] On message: render template with event data
- [ ] Call brain (Claude)
- [ ] Publish response to output subject

### 5. Main
- [ ] Load `config/forest.yaml`
- [ ] Initialize brain (Claude, from env `CLAUDE_API_KEY`)
- [ ] Start all TreeHouses
- [ ] Start all Nims
- [ ] Wait for SIGINT/SIGTERM → graceful shutdown

### 6. Example ✓
- [x] `config/forest.yaml`
- [x] `scripts/treehouses/scoring.lua`
- [x] `scripts/nims/qualify.md`
- [x] READMEs for each

---

## Post-MVP

### Sources
- [ ] Source interface
- [ ] WebhookSource (generic HTTP)
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

## MVP Summary

```
config/forest.yaml          # What exists
scripts/treehouses/*.lua    # TreeHouse logic
scripts/nims/*.md           # Nim prompts
pkg/runtime/                # Runtime code
cmd/forest/main.go          # Entry point
```

## MVP Flow

```
nats pub contact.created '{...}'
        │
        ▼
   TreeHouse (scoring)
   loads: scripts/treehouses/scoring.lua
   runs: process(contact) → {score, signals}
        │
        ▼
   lead.scored
        │
        ▼
   Nim (qualify)
   loads: scripts/nims/qualify.md
   renders: template with {score, signals}
   calls: Claude
        │
        ▼
   lead.qualified
```
