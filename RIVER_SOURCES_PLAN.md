# River Sources Planning Document

> **TL;DR**: Add Sources as configurable components that feed external data into River via HTTP webhooks, HTTP polling, or ceremonies. Manageable through `forest.yaml` config and runtime API. Sources are simply connected to River or not - no registry needed.

## Executive Summary

| Aspect | Details |
|--------|---------|
| **Goal** | Enable external systems to push/pull data into NimsForest |
| **Source Types** | HTTP Webhook, HTTP Poll, Ceremony |
| **Configuration** | YAML-based (`forest.yaml`) + Runtime API |
| **Management** | Sources are connected to River directly (no registry) |
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
- No scheduled/ceremony sources for periodic data fetching

---

## Architecture Vision

```
External Systems
      │
      ▼
┌─────────────────────────────────────────────────────────┐
│                      SOURCES                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │ HTTP Webhook │  │  HTTP Poll   │  │   Ceremony   │  │
│  │    Source    │  │    Source    │  │    Source    │  │
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

**Key Design**: Sources hold a reference to River and call `river.Flow()` directly. No registry or intermediary layer - a source is either connected and running, or it's not.

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

### 3. Ceremony Source

Listens to WindWaker's `dance.beat` and counts beats to trigger at intervals. This keeps all timing synchronized through the forest's conductor.

**How it works:**
1. Ceremony source catches `dance.beat` from Wind
2. Counts beats until interval is reached (e.g., 90Hz × 60s = 5400 beats for 1 minute)
3. Flows event into River
4. Resets counter

**Use Cases:**
- Daily report generation
- Scheduled data exports
- Periodic cleanup triggers
- Heartbeat/health check events

**Configuration:**
```yaml
sources:
  hourly-health-check:
    type: ceremony
    interval: 1h                      # Counts beats for 1 hour
    publishes: river.trigger.health_check
    script: scripts/sources/health.lua  # Optional: Lua script to generate payload
    
  heartbeat:
    type: ceremony
    interval: 30s                     # Every 30 seconds (2700 beats at 90Hz)
    publishes: river.system.heartbeat
    
  five-minute-sync:
    type: ceremony
    interval: 5m
    publishes: river.trigger.sync
    payload:                          # Static payload
      type: sync_trigger
```

**Note:** For wall-clock schedules (e.g., "8 AM daily"), a separate scheduler component may be needed, or the ceremony source can check wall-clock time on each beat.

---

## Implementation Plan

### Phase 1: Core Source Infrastructure

#### 1.1 Source Interface

```go
// internal/core/source.go

// Source feeds external data into the River.
// Sources hold a direct reference to River - they're either connected or not.
type Source interface {
    // Name returns the unique identifier for this source
    Name() string
    
    // Type returns the source type (http_webhook, http_poll, ceremony)
    Type() string
    
    // Start begins accepting/fetching data and flowing to River
    Start(ctx context.Context) error
    
    // Stop gracefully shuts down the source
    Stop() error
    
    // IsRunning returns whether the source is active
    IsRunning() bool
}

// BaseSource provides common functionality for all sources.
// Embeds a River reference for direct data flow.
type BaseSource struct {
    name      string
    river     *River
    publishes string  // The river subject to publish to
    running   bool
    mu        sync.Mutex
}

// NewBaseSource creates a base source connected to the given River.
func NewBaseSource(name string, river *River, publishes string) *BaseSource {
    return &BaseSource{
        name:      name,
        river:     river,
        publishes: publishes,
    }
}

// Flow sends data to the River.
func (s *BaseSource) Flow(data []byte) error {
    return s.river.Flow(s.publishes, data)
}

// FlowWithSubject sends data to River with a custom subject suffix.
// E.g., if publishes is "river.stripe" and suffix is "webhook.charge",
// the final subject becomes "river.stripe.webhook.charge"
func (s *BaseSource) FlowWithSubject(suffix string, data []byte) error {
    subject := s.publishes
    if suffix != "" {
        subject = s.publishes + "." + suffix
    }
    return s.river.Flow(subject, data)
}
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

### Phase 4: Ceremony Source

#### 4.1 CeremonySource Implementation

```go
// internal/sources/ceremony.go

type CeremonySourceConfig struct {
    Name      string
    Interval  time.Duration     // How often to trigger (e.g., 30s, 5m, 1h)
    Publishes string
    Payload   map[string]any    // Static payload (optional)
    Script    string            // Lua script path (optional)
}

type CeremonySource struct {
    *core.BaseSource
    config       CeremonySourceConfig
    wind         *core.Wind
    vm           *runtime.LuaVM    // For script-based payloads
    
    // Beat counting
    beatsPerTrigger uint64        // Calculated from interval and Hz
    beatCount       uint64        // Current count
    hz              int           // WindWaker frequency (typically 90)
    sub             *core.Subscription
}

func NewCeremonySource(cfg CeremonySourceConfig, wind *core.Wind, river *core.River) *CeremonySource

// Start subscribes to dance.beat and begins counting
func (s *CeremonySource) Start(ctx context.Context) error {
    // Calculate beats needed: interval_seconds * hz
    s.beatsPerTrigger = uint64(s.config.Interval.Seconds()) * uint64(s.hz)
    
    // Catch dance beats from WindWaker
    s.sub, _ = s.wind.Catch("dance.beat", func(leaf core.Leaf) {
        s.beatCount++
        if s.beatCount >= s.beatsPerTrigger {
            s.trigger()
            s.beatCount = 0
        }
    })
    return nil
}

// trigger flows the event into River
func (s *CeremonySource) trigger() {
    payload := s.buildPayload()  // Static or from Lua script
    s.Flow(payload)
}
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
      
  hourly-report:
    type: ceremony
    interval: 1h
    publishes: river.trigger.hourly_report

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
    Type      string            `yaml:"type"`      // http_webhook, http_poll, ceremony
    
    // HTTP Webhook fields
    Path      string            `yaml:"path,omitempty"`
    Secret    string            `yaml:"secret,omitempty"`
    Headers   []string          `yaml:"headers,omitempty"`
    
    // HTTP Poll / Ceremony fields
    URL       string            `yaml:"url,omitempty"`       // http_poll only
    Method    string            `yaml:"method,omitempty"`    // http_poll only
    Interval  string            `yaml:"interval,omitempty"`  // Duration (e.g., "5m", "1h")
    
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

# Add ceremony source (counts WindWaker beats)
forest add source heartbeat \
  --type=ceremony \
  --interval=30s \
  --publishes=river.system.heartbeat

forest add source hourly-sync \
  --type=ceremony \
  --interval=1h \
  --publishes=river.trigger.sync

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
    
    // Components (sources are just connected to river or not)
    sources       map[string]Source
    trees         map[string]*Tree
    treehouses    map[string]*TreeHouse
    nims          map[string]*Nim
    
    // HTTP server for webhooks
    webhookServer *sources.WebhookServer
    
    mu      sync.Mutex
    running bool
}

func (f *Forest) AddSource(name string, cfg SourceConfig) error
func (f *Forest) RemoveSource(name string) error
func (f *Forest) PauseSource(name string) error
func (f *Forest) ResumeSource(name string) error
func (f *Forest) ListSources() []SourceInfo
```

---

## File Structure

```
nimsforest/
├── internal/
│   ├── core/
│   │   ├── source.go           # Source interface & BaseSource
│   │   └── river.go            # (existing)
│   │
│   └── sources/
│       ├── webhook.go          # HTTP webhook source
│       ├── webhook_verify.go   # Signature verification
│       ├── http_server.go      # Webhook HTTP server
│       ├── poll.go             # HTTP poll source
│       ├── ceremony.go         # Ceremony source (counts WindWaker beats)
│       └── factory.go          # Source factory (creates source from config)
│
├── pkg/runtime/
│   ├── config.go               # Updated with SourceConfig
│   ├── forest.go               # Updated with source management
│   └── api.go                  # Updated with source endpoints
│
├── scripts/sources/            # Lua scripts for ceremony sources
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
      
  # Hourly sync trigger (ceremony counts WindWaker beats)
  hourly-sync:
    type: ceremony
    interval: 1h
    publishes: river.trigger.sync
    
  # System heartbeat (every 30 seconds)
  heartbeat:
    type: ceremony
    interval: 30s
    publishes: river.system.heartbeat

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
- [ ] Add source configuration to `Config`
- [ ] Add source map to `Forest`
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

### Milestone 4: Ceremony Source (Week 4)
- [ ] Implement `CeremonySource` that catches `dance.beat`
- [ ] Beat counting for interval-based triggers
- [ ] Add Lua script support for payload generation
- [ ] Integration tests with WindWaker

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
- Ceremony beat counting accuracy

### Integration Tests
- Webhook source → River flow
- Poll source with mock HTTP server
- Ceremony source timing accuracy
- Source start/stop lifecycle

### E2E Tests
- Complete webhook flow: HTTP POST → Source → River → Tree → Wind → Nim
- Multiple sources feeding same River
- Runtime add/remove of sources via API
- Configuration reload with sources

---

## Success Criteria

1. **Config-based sources**: Sources defined in `forest.yaml` start automatically
2. **Runtime management**: Add/remove/pause/resume sources via API and CLI
3. **Webhook support**: At least Stripe and GitHub signature verification
4. **Poll support**: Basic polling with cursor/pagination
5. **Ceremony support**: Interval-based triggers using WindWaker beats, optional Lua payload
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
