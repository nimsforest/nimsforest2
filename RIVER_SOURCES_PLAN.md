# River Sources Planning Document

> **TL;DR**: Add Sources as configurable components that feed external data into River via HTTP webhooks, HTTP polling, or cron schedules. Manageable through `forest.yaml` config and runtime API.

## Executive Summary

| Aspect | Details |
|--------|---------|
| **Goal** | Enable external systems to push/pull data into NimsForest |
| **Source Types** | HTTP Webhook, HTTP Poll, Cron |
| **Configuration** | YAML-based (`forest.yaml`) + Runtime API |
| **Security** | Signature verification, secret management, rate limiting |
| **Timeline** | 5 weeks for full implementation |

## Overview

This document outlines the plan for adding **River Sources** - components that feed external data into the River stream. Sources are the entry point for external systems to interact with NimsForest.

---

## Current State

### How Data Enters the River Today

1. **Programmatic**: Direct calls to `river.Flow(subject, data)`
2. **Demo mode**: Internal `sendDemoData()` function
3. **NATS CLI**: `nats pub river.stripe.webhook '{...}'`

### Missing Functionality

- No HTTP endpoints to receive webhooks from external services (Stripe, GitHub, etc.)
- No way to configure data sources in `forest.yaml`
- No runtime management of sources via API
- No polling sources for APIs that don't support webhooks
- No scheduled/cron sources for periodic data fetching

---

## Architecture Vision

```
External Systems
      │
      ▼
┌─────────────────────────────────────────────────────────┐
│                      SOURCES                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │ HTTP Webhook │  │  HTTP Poll   │  │    Cron      │  │
│  │    Source    │  │    Source    │  │   Source     │  │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  │
│         │                 │                 │          │
│         └────────────┬────┴────────────────┘          │
│                      ▼                                 │
│              river.Flow(subject, data)                 │
└─────────────────────────────────────────────────────────┘
                       │
                       ▼
                ┌──────────────┐
                │    RIVER     │  (JetStream Stream)
                │   river.>    │
                └──────┬───────┘
                       │
                       ▼
                ┌──────────────┐
                │    TREES     │  (Parse → Leaves)
                └──────────────┘
```

---

## Source Types

### 1. HTTP Webhook Source

Receives HTTP POST requests from external services and flows them into River.

**Use Cases:**
- Stripe payment webhooks
- GitHub repository events
- Slack interactions
- Custom application events

**Configuration:**
```yaml
sources:
  stripe-webhook:
    type: http_webhook
    path: /webhooks/stripe          # HTTP endpoint path
    publishes: river.stripe.webhook  # River subject
    secret: ${STRIPE_WEBHOOK_SECRET} # Optional: webhook signature verification
    
  github-webhook:
    type: http_webhook
    path: /webhooks/github
    publishes: river.github.events
    secret: ${GITHUB_WEBHOOK_SECRET}
    headers:                         # Optional: extract headers to include
      - X-GitHub-Event
      - X-GitHub-Delivery
```

### 2. HTTP Poll Source

Periodically fetches data from an HTTP API and flows it into River.

**Use Cases:**
- APIs without webhook support
- Data aggregation from multiple sources
- Monitoring external systems

**Configuration:**
```yaml
sources:
  salesforce-contacts:
    type: http_poll
    url: https://api.salesforce.com/v1/contacts
    publishes: river.salesforce.contacts
    interval: 5m                     # Poll interval
    method: GET
    headers:
      Authorization: Bearer ${SF_TOKEN}
    # Optional: track cursor/pagination
    cursor:
      param: since
      extract: $.lastModified
      
  weather-data:
    type: http_poll
    url: https://api.weather.com/current
    publishes: river.weather.current
    interval: 15m
```

### 3. Cron Source

Runs on a schedule and can execute scripts or make HTTP calls.

**Use Cases:**
- Daily report generation
- Scheduled data exports
- Periodic cleanup triggers

**Configuration:**
```yaml
sources:
  daily-summary:
    type: cron
    schedule: "0 8 * * *"            # 8 AM daily
    publishes: river.trigger.daily_summary
    payload:                          # Static payload
      type: daily_summary
      
  hourly-health-check:
    type: cron
    schedule: "0 * * * *"            # Every hour
    publishes: river.trigger.health_check
    script: scripts/sources/health.lua  # Optional: Lua script to generate payload
```

### 4. NATS Bridge Source (Future)

Bridges events from other NATS subjects or clusters.

**Configuration:**
```yaml
sources:
  legacy-events:
    type: nats_bridge
    subscribes: legacy.events.>      # Source NATS subject
    publishes: river.legacy.events   # Target River subject
    cluster: nats://legacy-cluster:4222
```

---

## Implementation Plan

### Phase 1: Core Source Infrastructure

#### 1.1 Source Interface

```go
// internal/core/source.go

// Source feeds external data into the River.
type Source interface {
    // Name returns the unique identifier for this source
    Name() string
    
    // Type returns the source type (http_webhook, http_poll, cron, etc.)
    Type() string
    
    // Start begins accepting/fetching data
    Start(ctx context.Context) error
    
    // Stop gracefully shuts down the source
    Stop() error
    
    // IsRunning returns whether the source is active
    IsRunning() bool
}

// BaseSource provides common functionality for all sources.
type BaseSource struct {
    name   string
    river  *River
    logger *log.Logger
}

// Flow sends data to the River with the given subject.
func (s *BaseSource) Flow(subject string, data []byte) error {
    return s.river.Flow(subject, data)
}
```

#### 1.2 Source Registry

```go
// internal/core/source_registry.go

// SourceRegistry manages all active sources.
type SourceRegistry struct {
    sources map[string]Source
    river   *River
    mu      sync.RWMutex
}

func NewSourceRegistry(river *River) *SourceRegistry
func (r *SourceRegistry) Register(source Source) error
func (r *SourceRegistry) Unregister(name string) error
func (r *SourceRegistry) Get(name string) (Source, bool)
func (r *SourceRegistry) List() []SourceInfo
func (r *SourceRegistry) StartAll(ctx context.Context) error
func (r *SourceRegistry) StopAll() error
```

### Phase 2: HTTP Webhook Source

#### 2.1 WebhookSource Implementation

```go
// internal/sources/webhook.go

type WebhookSourceConfig struct {
    Name     string
    Path     string            // e.g., "/webhooks/stripe"
    Publishes string           // e.g., "river.stripe.webhook"
    Secret   string            // For signature verification (optional)
    Headers  []string          // Headers to include in payload
}

type WebhookSource struct {
    *core.BaseSource
    config  WebhookSourceConfig
    handler http.HandlerFunc
}

func NewWebhookSource(cfg WebhookSourceConfig, river *core.River) *WebhookSource

// Handler returns the HTTP handler for this webhook endpoint
func (s *WebhookSource) Handler() http.HandlerFunc
```

#### 2.2 HTTP Server for Webhooks

```go
// internal/sources/http_server.go

type WebhookServer struct {
    server   *http.Server
    mux      *http.ServeMux
    sources  map[string]*WebhookSource
    address  string
}

func NewWebhookServer(address string) *WebhookServer
func (s *WebhookServer) Mount(source *WebhookSource) error
func (s *WebhookServer) Unmount(name string) error
func (s *WebhookServer) Start() error
func (s *WebhookServer) Stop(ctx context.Context) error
```

#### 2.3 Webhook Signature Verification

```go
// internal/sources/webhook_verify.go

type SignatureVerifier interface {
    Verify(payload []byte, signature string) error
}

// Built-in verifiers for common providers
func NewStripeVerifier(secret string) SignatureVerifier
func NewGitHubVerifier(secret string) SignatureVerifier
func NewSlackVerifier(secret string) SignatureVerifier
func NewHMACVerifier(secret string, algo string) SignatureVerifier
```

### Phase 3: HTTP Poll Source

#### 3.1 PollSource Implementation

```go
// internal/sources/poll.go

type PollSourceConfig struct {
    Name      string
    URL       string
    Publishes string
    Interval  time.Duration
    Method    string            // GET, POST, etc.
    Headers   map[string]string
    Body      []byte            // For POST requests
    Cursor    *CursorConfig     // For pagination/cursor tracking
}

type CursorConfig struct {
    Param   string  // Query param name for cursor
    Extract string  // JSONPath to extract next cursor from response
    Store   string  // KV key to persist cursor (uses Soil)
}

type PollSource struct {
    *core.BaseSource
    config   PollSourceConfig
    client   *http.Client
    ticker   *time.Ticker
    cursor   string
}

func NewPollSource(cfg PollSourceConfig, river *core.River, soil *core.Soil) *PollSource
```

### Phase 4: Cron Source

#### 4.1 CronSource Implementation

```go
// internal/sources/cron.go

type CronSourceConfig struct {
    Name      string
    Schedule  string            // Cron expression
    Publishes string
    Payload   map[string]any    // Static payload (optional)
    Script    string            // Lua script path (optional)
}

type CronSource struct {
    *core.BaseSource
    config    CronSourceConfig
    scheduler *cron.Cron
    entryID   cron.EntryID
    vm        *runtime.LuaVM    // For script-based payloads
}

func NewCronSource(cfg CronSourceConfig, river *core.River) *CronSource
```

### Phase 5: Configuration Integration

#### 5.1 Update forest.yaml Schema

```yaml
# config/forest.yaml

# Sources - Entry points for external data
sources:
  stripe:
    type: http_webhook
    path: /webhooks/stripe
    publishes: river.stripe.webhook
    secret: ${STRIPE_WEBHOOK_SECRET}
    
  github:
    type: http_webhook
    path: /webhooks/github
    publishes: river.github.events
    
  crm-sync:
    type: http_poll
    url: ${CRM_API_URL}/contacts
    publishes: river.crm.contacts
    interval: 10m
    headers:
      Authorization: Bearer ${CRM_TOKEN}
      
  daily-report:
    type: cron
    schedule: "0 9 * * *"
    publishes: river.trigger.daily_report

# Trees - Parse River data into Leaves
trees:
  stripe-parser:
    watches: river.stripe.webhook
    publishes: payment.completed
    script: scripts/trees/stripe_parser.lua

# TreeHouses - Lua transformers
treehouses:
  scoring:
    subscribes: contact.created
    publishes: lead.scored
    script: scripts/treehouses/scoring.lua

# Nims - AI processors
nims:
  qualify:
    subscribes: lead.scored
    publishes: lead.qualified
    prompt: scripts/nims/qualify.md
```

#### 5.2 Update Config Loader

```go
// pkg/runtime/config.go

type Config struct {
    Sources    map[string]SourceConfig    `yaml:"sources"`
    Trees      map[string]TreeConfig      `yaml:"trees"`
    TreeHouses map[string]TreeHouseConfig `yaml:"treehouses"`
    Nims       map[string]NimConfig       `yaml:"nims"`
    BaseDir    string                     `yaml:"-"`
}

type SourceConfig struct {
    Name      string            `yaml:"-"`
    Type      string            `yaml:"type"`      // http_webhook, http_poll, cron
    
    // HTTP Webhook fields
    Path      string            `yaml:"path,omitempty"`
    Secret    string            `yaml:"secret,omitempty"`
    Headers   []string          `yaml:"headers,omitempty"`
    
    // HTTP Poll fields
    URL       string            `yaml:"url,omitempty"`
    Interval  string            `yaml:"interval,omitempty"`
    Method    string            `yaml:"method,omitempty"`
    
    // Cron fields
    Schedule  string            `yaml:"schedule,omitempty"`
    
    // Common fields
    Publishes string            `yaml:"publishes"`
    Payload   map[string]any    `yaml:"payload,omitempty"`
    Script    string            `yaml:"script,omitempty"`
}
```

### Phase 6: Runtime Management API

#### 6.1 API Endpoints

```
GET    /api/v1/sources              # List all sources
GET    /api/v1/sources/{name}       # Get source details
POST   /api/v1/sources              # Add new source
DELETE /api/v1/sources/{name}       # Remove source
PUT    /api/v1/sources/{name}/pause # Pause source
PUT    /api/v1/sources/{name}/resume # Resume source
```

#### 6.2 CLI Commands

```bash
# List sources
forest list sources

# Add webhook source
forest add source stripe-webhook \
  --type=http_webhook \
  --path=/webhooks/stripe \
  --publishes=river.stripe.webhook \
  --secret=${STRIPE_WEBHOOK_SECRET}

# Add poll source
forest add source crm-contacts \
  --type=http_poll \
  --url=https://api.crm.com/contacts \
  --publishes=river.crm.contacts \
  --interval=5m

# Add cron source
forest add source daily-cleanup \
  --type=cron \
  --schedule="0 2 * * *" \
  --publishes=river.trigger.cleanup

# Remove source
forest remove source stripe-webhook

# Pause/resume source
forest pause source crm-contacts
forest resume source crm-contacts
```

### Phase 7: Forest Integration

#### 7.1 Update Forest to Manage Sources

```go
// pkg/runtime/forest.go

type Forest struct {
    config        *Config
    wind          *core.Wind
    river         *core.River
    humus         *core.Humus
    brain         brain.Brain
    
    // Component registries
    sources       *core.SourceRegistry
    trees         map[string]*Tree
    treehouses    map[string]*TreeHouse
    nims          map[string]*Nim
    
    // HTTP servers
    webhookServer *sources.WebhookServer
    
    mu      sync.Mutex
    running bool
}

func (f *Forest) AddSource(name string, cfg SourceConfig) error
func (f *Forest) RemoveSource(name string) error
func (f *Forest) PauseSource(name string) error
func (f *Forest) ResumeSource(name string) error
func (f *Forest) SourceStatus() []SourceInfo
```

---

## File Structure

```
nimsforest/
├── internal/
│   ├── core/
│   │   ├── source.go           # Source interface & BaseSource
│   │   ├── source_registry.go  # Source registry
│   │   └── river.go            # (existing)
│   │
│   └── sources/
│       ├── webhook.go          # HTTP webhook source
│       ├── webhook_verify.go   # Signature verification
│       ├── http_server.go      # Webhook HTTP server
│       ├── poll.go             # HTTP poll source
│       ├── cron.go             # Cron source
│       └── factory.go          # Source factory
│
├── pkg/runtime/
│   ├── config.go               # Updated with SourceConfig
│   ├── forest.go               # Updated with source management
│   └── api.go                  # Updated with source endpoints
│
├── scripts/sources/            # Lua scripts for cron sources
│   └── example.lua
│
└── config/
    └── forest.yaml             # Updated with sources section
```

---

## Security Considerations

### Webhook Security

1. **Signature Verification**: All webhook sources SHOULD configure signature verification
2. **HTTPS Only**: Webhook server should support TLS in production
3. **Rate Limiting**: Prevent abuse with configurable rate limits
4. **IP Allowlisting**: Optional IP-based access control

### Secret Management

1. **Environment Variables**: Secrets reference env vars via `${VAR_NAME}` syntax
2. **No Plaintext Secrets**: Never store secrets in config files
3. **Secret Rotation**: Support for rotating webhook secrets without restart

### Network Security

1. **Localhost by Default**: Webhook server binds to localhost by default
2. **Configurable Bind Address**: Allow binding to specific interfaces
3. **Separate Port**: Use separate port from management API (e.g., 8081)

---

## Configuration Example

### Complete forest.yaml with Sources

```yaml
# =============================================================================
# SOURCES - Entry points for external data
# =============================================================================
sources:
  # Stripe payment webhooks
  stripe:
    type: http_webhook
    path: /webhooks/stripe
    publishes: river.stripe.webhook
    secret: ${STRIPE_WEBHOOK_SECRET}
    
  # GitHub repository events
  github:
    type: http_webhook
    path: /webhooks/github
    publishes: river.github.events
    headers:
      - X-GitHub-Event
      - X-GitHub-Delivery
    
  # Poll CRM API for new contacts
  crm-sync:
    type: http_poll
    url: ${CRM_API_URL}/contacts
    publishes: river.crm.contacts
    interval: 10m
    headers:
      Authorization: Bearer ${CRM_TOKEN}
    cursor:
      param: since
      extract: $.meta.last_updated
      
  # Daily summary trigger
  daily-summary:
    type: cron
    schedule: "0 9 * * *"
    publishes: river.trigger.daily_summary

# =============================================================================
# TREES - Parse River data into structured Leaves
# =============================================================================
trees:
  stripe-parser:
    watches: river.stripe.webhook
    publishes: payment.completed
    script: scripts/trees/stripe_parser.lua
    
  github-parser:
    watches: river.github.events
    publishes: repo.events
    script: scripts/trees/github_parser.lua

# =============================================================================
# TREEHOUSES - Lua-based transformers
# =============================================================================
treehouses:
  scoring:
    subscribes: contact.created
    publishes: lead.scored
    script: scripts/treehouses/scoring.lua

# =============================================================================
# NIMS - AI-powered processors
# =============================================================================
nims:
  qualify:
    subscribes: lead.scored
    publishes: lead.qualified
    prompt: scripts/nims/qualify.md
```

---

## Environment Variables

```bash
# Webhook server configuration
NIMSFOREST_WEBHOOK_ADDR=0.0.0.0:8081  # Webhook server bind address
NIMSFOREST_WEBHOOK_TLS_CERT=/path/to/cert.pem
NIMSFOREST_WEBHOOK_TLS_KEY=/path/to/key.pem

# Secrets for webhook signature verification
STRIPE_WEBHOOK_SECRET=whsec_...
GITHUB_WEBHOOK_SECRET=...
SLACK_SIGNING_SECRET=...

# Secrets for poll sources
CRM_API_URL=https://api.crm.com/v1
CRM_TOKEN=...
```

---

## Implementation Timeline

### Milestone 1: Core Infrastructure (Week 1)
- [ ] Create `Source` interface and `BaseSource`
- [ ] Implement `SourceRegistry`
- [ ] Add source configuration to `Config`
- [ ] Unit tests for core infrastructure

### Milestone 2: HTTP Webhook Source (Week 2)
- [ ] Implement `WebhookSource`
- [ ] Implement `WebhookServer`
- [ ] Add signature verification for Stripe, GitHub
- [ ] Integration tests with mock webhooks
- [ ] Update CLI for source management

### Milestone 3: HTTP Poll Source (Week 3)
- [ ] Implement `PollSource`
- [ ] Add cursor/pagination support
- [ ] Add Soil integration for cursor persistence
- [ ] Integration tests

### Milestone 4: Cron Source (Week 4)
- [ ] Implement `CronSource`
- [ ] Add Lua script support for payload generation
- [ ] Integration tests

### Milestone 5: Full Integration (Week 5)
- [ ] Integrate all sources into Forest
- [ ] Update management API with source endpoints
- [ ] Update CLI with full source commands
- [ ] E2E tests with complete pipeline
- [ ] Documentation

---

## Testing Strategy

### Unit Tests
- Source interface compliance
- Configuration parsing and validation
- Signature verification algorithms
- Cron expression parsing

### Integration Tests
- Webhook → River → Tree → Leaf flow
- Poll source with mock HTTP server
- Cron source timing accuracy
- Runtime add/remove of sources

### E2E Tests
- Complete webhook flow: HTTP POST → River → Tree → Wind → Nim → Humus → Soil
- Multi-source scenario with different types
- Source pause/resume functionality
- Configuration reload with sources

---

## Success Criteria

1. **Config-based sources**: Sources defined in `forest.yaml` start automatically
2. **Runtime management**: Add/remove/pause/resume sources via API and CLI
3. **Webhook support**: At least Stripe and GitHub signature verification
4. **Poll support**: Basic polling with cursor/pagination
5. **Cron support**: Cron expressions with optional Lua payload generation
6. **Security**: Signature verification, secret management, rate limiting
7. **Observability**: Logging, metrics for each source
8. **Documentation**: Config examples, API docs, security guidelines

---

## Open Questions

1. **Port allocation**: Should webhooks use the same port as management API or separate?
   - Recommendation: Separate port (8081) for security isolation

2. **Queue vs immediate**: Should sources queue data or flow immediately?
   - Recommendation: Flow immediately; River provides the queue

3. **Backpressure**: How to handle when River/NATS is overloaded?
   - Recommendation: HTTP 503 for webhooks, skip poll cycle for polls

4. **Retries**: Should poll sources retry failed requests?
   - Recommendation: Yes, with exponential backoff (configurable)

5. **Monitoring**: What metrics should sources expose?
   - Recommendations: requests_total, errors_total, latency_histogram, last_success_time

---

## References

- [NATS JetStream Documentation](https://docs.nats.io/nats-concepts/jetstream)
- [Stripe Webhook Security](https://stripe.com/docs/webhooks/signatures)
- [GitHub Webhook Security](https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries)
- [Cron Expression Format](https://pkg.go.dev/github.com/robfig/cron/v3)
