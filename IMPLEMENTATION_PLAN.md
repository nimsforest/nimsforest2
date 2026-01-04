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

### 7. E2E Test ✓
- [x] `test/e2emvp/e2e_test.go` - test skeleton
- [x] `test/e2emvp/testdata/` - test config + scripts
- [x] `make test-e2emvp` - run command

Run: `make test-e2emvp`
Passes when: all TODOs in test are implemented

---

## Post-MVP

### Agentic Nims
Current MVP: prompt → response (one shot)

Full vision:
- [ ] Multi-step reasoning loops
- [ ] Tool use (read files, search, call APIs)
- [ ] Human checkpoints (pause, await approval, resume)
- [ ] State persistence (survive restarts)
- [ ] Publish intermediate status to River
- [ ] Configurable autonomy level (full auto → human approval → human only)

Example flow:
```
ticket.created
    │
    ▼
Nim: analyze ticket
    │ (reason)
    ▼
Nim: search knowledge base (tool)
    │ (reason)
    ▼
Nim: draft response
    │
    ▼
response.drafted (checkpoint: await human approval)
    │
    ▼
Human approves/edits
    │
    ▼
response.sent
```

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
