# Implementation Plan

Ordered tasks. No timings. Check off as completed.

---

## Phase 1: Runtime Foundation

### 1.1 Lua Runtime
- [ ] Add `github.com/yuin/gopher-lua` dependency
- [ ] Create `pkg/runtime/lua.go` - Lua VM wrapper
- [ ] Implement helper functions: `contains`, `json.encode`, `json.decode`, `log`
- [ ] Create `pkg/runtime/lua_test.go` - Unit tests

### 1.2 Config Loader
- [ ] Create `pkg/runtime/config.go` - YAML config structs
- [ ] Parse TreeHouse definitions
- [ ] Parse Nim definitions
- [ ] Validate config (script exists, valid subjects)
- [ ] Create `pkg/runtime/config_test.go` - Unit tests

### 1.3 Prompt Templates
- [ ] Create `pkg/runtime/prompt.go`
- [ ] Go template support for prompts (e.g., `{{.body}}`)
- [ ] Render Leaf data into prompt
- [ ] Unit tests

---

## Phase 2: Component Runtimes

### 2.1 TreeHouse Runtime
- [ ] Create `pkg/runtime/treehouse.go`
- [ ] Subscribe to configured NATS subject
- [ ] Load Lua script on startup
- [ ] Call `process()` function for each message
- [ ] Publish result to output subject
- [ ] Error handling and logging

### 2.2 Nim Runtime
- [ ] Create `pkg/runtime/nim.go`
- [ ] Subscribe to configured NATS subject
- [ ] Initialize brain on startup (from config)
- [ ] Render prompt template with Leaf data
- [ ] Call `brain.Ask(prompt)`
- [ ] Publish response to output subject

### 2.3 Unit Tests
- [ ] TreeHouse tests with mock NATS (verify Lua execution)
- [ ] Nim tests with mock NATS and `MockBrain`

---

## Phase 3: Main Wiring

### 3.1 Entry Point
- [ ] Update `cmd/forest/main.go`
- [ ] Load config from `config/forest.yaml`
- [ ] Start all TreeHouses
- [ ] Start all Nims
- [ ] Graceful shutdown

### 3.2 CLI Flags
- [ ] `--config` path to config file
- [ ] `--nats` NATS server URL
- [ ] Env var overrides

---

## Phase 4: Examples

### 4.1 TreeHouse Lua Scripts
- [ ] `scripts/treehouses/scoring.lua` - Lead scoring
- [ ] `scripts/treehouses/routing.lua` - Ticket routing

### 4.2 Config Example
- [ ] `config/forest.yaml` - Complete working config
- [ ] Include Sources, TreeHouses, Nims with prompts

---

## Phase 5: Integration Testing

### 5.1 E2E Test
- [ ] Spin up embedded NATS
- [ ] Load test config
- [ ] Publish test events
- [ ] Verify output events
- [ ] Test TreeHouse determinism
- [ ] Test Nim with mock brain

---

## Phase 6: Sources

### 6.1 Source Interface
- [ ] `sources/source.go` - Source interface
- [ ] `Start()`, `Stop()`, `Name()` methods
- [ ] Connect to River (NATS)
- [ ] Translate external data → Leaf events

### 6.2 WebhookSource (Generic)
- [ ] `sources/webhook/webhook.go` - HTTP server
- [ ] Parse incoming webhooks
- [ ] Configurable path → subject mapping

### 6.3 Example Implementations
- [ ] `StripeSource` - Stripe webhooks → payment.*, subscription.*
- [ ] `SalesforceSource` - Salesforce events → contact.*, deal.*
- [ ] `ZendeskSource` - Zendesk webhooks → ticket.*

---

## Summary

| Phase | Description |
|-------|-------------|
| 1 | Lua VM, config loader, brain wrapper |
| 2 | TreeHouse and Nim runtimes |
| 3 | Main entry point and CLI |
| 4 | Example Lua scripts |
| 5 | Integration tests |
| 6 | Example adapters |

MVP = Phases 1-4
