# 🌲 NimsForest Implementation Plan

## Principle

**Build only what's used. Test without external services.**

---

## Architecture Overview

```
┌──────────────────────────────────────────────────────┐
│                     ADAPTERS                          │
│            (separate package, optional)               │
│                                                       │
│   adapters/stripe/     → payment.received            │
│   adapters/hubspot/    → contact.created             │
│   adapters/zendesk/    → ticket.created              │
└───────────────────────────┬──────────────────────────┘
                            │ Generic events
                            ▼
┌──────────────────────────────────────────────────────┐
│                   CORE FRAMEWORK                      │
│                                                       │
│   internal/core/        Base infrastructure          │
│   internal/leaves/      Generic event types          │
│   internal/treehouses/  Deterministic rules          │
│   internal/nims/        Human/LLM decisions          │
│   internal/llm/         LLM integration              │
└──────────────────────────────────────────────────────┘
```

---

## What Already Exists

- [x] `internal/core/` - Wind, River, Humus, Soil, Decomposer
- [x] `internal/core/tree.go` - Base Tree interface
- [x] `internal/core/nim.go` - Base Nim interface
- [x] `internal/leaves/types.go` - PaymentCompleted, PaymentFailed
- [x] `internal/trees/payment.go` - Example tree (Stripe-specific, will move to adapter)
- [x] `internal/nims/aftersales.go` - Example nim

---

## Phase 1: Core Foundation

**Goal:** Tree House infrastructure + generic leaf types.

### 1.1 TreeHouse Base Interface

File: `internal/core/treehouse.go`

```go
type TreeHouse interface {
    Name() string
    InputSubjects() []string
    Process(leaf Leaf) ([]Leaf, error)  // Must be deterministic
    Start(ctx context.Context) error
    Stop() error
}

type BaseTreeHouse struct {
    name string
    wind *Wind
}
```

- [ ] Create interface
- [ ] Create BaseTreeHouse with Wind integration
- [ ] Catch input subjects, call Process, drop output leaves
- [ ] Unit tests proving determinism

### 1.2 Generic Leaf Types

File: `internal/leaves/types.go`

```go
// Contact - a person or company
type Contact struct {
    ID          string
    Email       string
    Name        string
    Company     string
    CompanySize int
    Title       string
    Source      string    // "inbound", "outbound", "referral"
    CreatedAt   time.Time
}

// Lead - a contact with a score
type Lead struct {
    ContactID   string
    Score       int
    Qualification string  // "none", "mql", "sql"
    Signals     []string  // what contributed to score
    ScoredAt    time.Time
}

// Ticket - a support request
type Ticket struct {
    ID          string
    ContactID   string
    Subject     string
    Body        string
    Channel     string    // "email", "chat", "form"
    Priority    string    // "low", "normal", "high", "urgent"
    Status      string
    CreatedAt   time.Time
}

// Payment - money movement
type Payment struct {
    ID          string
    ContactID   string
    Amount      float64
    Currency    string
    Status      string    // "succeeded", "failed", "refunded"
    FailReason  string
    CreatedAt   time.Time
}
```

- [ ] Define Contact
- [ ] Define Lead  
- [ ] Define Ticket
- [ ] Define Payment
- [ ] Define supporting types (Qualification, TicketTriage, etc.)

---

## Phase 2: Lead Generation Path

**Goal:** Contact → Scored → Qualified → Ready for Sales

### 2.1 ScoringHouse

File: `internal/treehouses/scoring.go`

Input: `contact.created`, `contact.activity`
Output: `lead.scored`

Rules (configurable):
```go
type ScoringRules struct {
    CompanySizeScore    map[string]int  // "1-10": 5, "11-50": 15, "51-200": 25, "200+": 40
    TitleScore          map[string]int  // "engineer": 10, "manager": 20, "director": 30, "vp": 40, "c-level": 50
    ActivityScore       map[string]int  // "page_view": 1, "pricing_view": 10, "demo_request": 50
}
```

- [ ] Create ScoringHouse
- [ ] Configurable scoring rules (no hardcoded values)
- [ ] Calculate total score from signals
- [ ] Emit `lead.scored` with score and contributing signals
- [ ] Unit tests with various contact profiles

### 2.2 QualificationHouse

File: `internal/treehouses/qualification.go`

Input: `lead.scored`
Output: `lead.qualified`

Rules:
```go
type QualificationRules struct {
    MQLThreshold int  // Score >= this = MQL
    SQLThreshold int  // Score >= this = SQL
    RequiredSignals []string  // Must have these to qualify
}
```

- [ ] Create QualificationHouse
- [ ] Apply MQL/SQL thresholds
- [ ] Check required signals
- [ ] Emit `lead.qualified` with qualification level
- [ ] Unit tests for threshold edge cases

### 2.3 LeadNim (LLM Enrichment)

File: `internal/nims/lead.go`

Input: `lead.qualified` (SQL only)
Output: `lead.enriched`

LLM analyzes:
- Company research summary
- Suggested talking points
- Potential pain points based on title/industry

- [ ] Create LeadNim
- [ ] LLM prompt for enrichment
- [ ] Emit enriched lead for sales
- [ ] Tests with mock LLM

---

## Phase 3: LLM Infrastructure

**Goal:** Enable Nims to use LLM reasoning.

### 3.1 LLM Client Interface

File: `internal/llm/client.go`

```go
type Client interface {
    Complete(ctx context.Context, prompt string) (string, error)
    CompleteJSON(ctx context.Context, prompt string, schema any) error
}
```

- [ ] Define interface
- [ ] Support both freeform and structured responses

### 3.2 OpenAI Implementation

File: `internal/llm/openai.go`

- [ ] Implement Client interface
- [ ] Handle API key from environment
- [ ] Handle rate limits and retries
- [ ] Configurable model selection

### 3.3 Mock Implementation

File: `internal/llm/mock.go`

- [ ] Implement Client interface
- [ ] Return configured responses for testing
- [ ] Record calls for assertions

---

## Phase 4: Support Path

**Goal:** Ticket → Routed → Triaged → Response Drafted

### 4.1 RoutingHouse

File: `internal/treehouses/routing.go`

Input: `ticket.created`
Output: `ticket.routed`

Rules:
```go
type RoutingRules struct {
    CategoryKeywords map[string][]string  // "billing": ["invoice", "charge", "payment"]
    PriorityRules    []PriorityRule       // If contains "urgent" → high
    DefaultQueue     string
}
```

- [ ] Create RoutingHouse
- [ ] Keyword-based categorization
- [ ] Priority rules
- [ ] Emit `ticket.routed` with queue and priority
- [ ] Unit tests

### 4.2 TriageNim

File: `internal/nims/triage.go`

Input: `ticket.routed`
Output: `ticket.triaged`

LLM analyzes:
- Sentiment (positive, neutral, negative, angry)
- Urgency (low, medium, high, critical)
- Intent (question, complaint, request, praise)
- Summary (one line)

- [ ] Create TriageNim
- [ ] LLM prompt for analysis
- [ ] Structured output parsing
- [ ] Tests with mock LLM

### 4.3 ResponseNim

File: `internal/nims/response.go`

Input: `ticket.triaged`
Output: `response.drafted`

LLM drafts:
- Empathetic response matching sentiment
- Addresses the core issue
- Suggests next steps

- [ ] Create ResponseNim
- [ ] LLM prompt for drafting
- [ ] Include ticket context in prompt
- [ ] Tests with mock LLM

---

## Phase 5: Adapters (Separate from Core)

**Goal:** Translate external webhooks to generic events.

Directory: `adapters/`

### 5.1 Adapter Interface

File: `adapters/adapter.go`

```go
type Adapter interface {
    Name() string
    // Translate converts raw webhook to generic leaf
    Translate(subject string, data []byte) (*core.Leaf, error)
}
```

### 5.2 Stripe Adapter

File: `adapters/stripe/adapter.go`

- [ ] Parse Stripe webhook format
- [ ] `charge.succeeded` → `payment.received`
- [ ] `charge.failed` → `payment.failed`
- [ ] Move existing PaymentTree logic here

### 5.3 CRM Adapter (Generic)

File: `adapters/crm/adapter.go`

- [ ] Parse common CRM webhook formats
- [ ] Contact created/updated events
- [ ] Deal/opportunity events
- [ ] Can be extended for HubSpot, Salesforce, etc.

### 5.4 Support Adapter (Generic)

File: `adapters/support/adapter.go`

- [ ] Parse common support webhook formats
- [ ] Ticket created/updated events
- [ ] Can be extended for Zendesk, Intercom, etc.

---

## Phase 6: Integration

**Goal:** Wire everything together, prove it works.

### 6.1 Main.go Updates

- [ ] Start all TreeHouses
- [ ] Start all Nims
- [ ] Adapters loaded based on config

### 6.2 E2E Test: Lead Path

File: `test/e2e/lead_test.go`

```go
func TestContactToQualifiedLead(t *testing.T) {
    // No external services - all generic events
    
    // Create contact
    river.Flow("contact.created", Contact{...})
    
    // Simulate activity
    river.Flow("contact.activity", Activity{Type: "pricing_view"})
    
    // Assert lead is qualified in Soil
    lead := soil.Dig("lead:contact-123")
    assert.Equal(t, "sql", lead.Qualification)
}
```

- [ ] Test full lead path
- [ ] Test scoring rules
- [ ] Test qualification thresholds

### 6.3 E2E Test: Support Path

File: `test/e2e/support_test.go`

```go
func TestTicketToResponse(t *testing.T) {
    // Create ticket
    river.Flow("ticket.created", Ticket{...})
    
    // Assert routing happened
    // Assert triage happened (mock LLM)
    // Assert response drafted
}
```

- [ ] Test full support path
- [ ] Test routing rules
- [ ] Test LLM integration (mocked)

---

## File Checklist

### Core Framework

```
internal/
├── core/
│   ├── treehouse.go           # NEW
│   └── treehouse_test.go      # NEW
├── leaves/
│   └── types.go               # EXPAND
├── treehouses/
│   ├── scoring.go             # NEW
│   ├── scoring_test.go        # NEW
│   ├── qualification.go       # NEW
│   ├── qualification_test.go  # NEW
│   ├── routing.go             # NEW
│   └── routing_test.go        # NEW
├── nims/
│   ├── lead.go                # NEW
│   ├── lead_test.go           # NEW
│   ├── triage.go              # NEW
│   ├── triage_test.go         # NEW
│   ├── response.go            # NEW
│   └── response_test.go       # NEW
└── llm/
    ├── client.go              # NEW
    ├── openai.go              # NEW
    └── mock.go                # NEW
```

### Adapters (Separate)

```
adapters/
├── adapter.go                 # Interface
├── stripe/
│   ├── adapter.go
│   └── adapter_test.go
├── crm/
│   ├── adapter.go
│   └── adapter_test.go
└── support/
    ├── adapter.go
    └── adapter_test.go
```

### Tests

```
test/
└── e2e/
    ├── lead_test.go           # NEW
    └── support_test.go        # NEW
```

---

## Dependency Graph

```
Phase 1.1 (TreeHouse base)
    │
    ├──► Phase 2.1 (ScoringHouse)
    │        │
    │        └──► Phase 2.2 (QualificationHouse)
    │                 │
    │                 └──► Phase 2.3 (LeadNim) ◄── Phase 3 (LLM)
    │
    └──► Phase 4.1 (RoutingHouse)
             │
             └──► Phase 4.2 (TriageNim) ◄── Phase 3 (LLM)
                      │
                      └──► Phase 4.3 (ResponseNim)

Phase 5 (Adapters) ── Independent, can be done anytime

Phase 6 (Integration) ── After all above complete
```

---

## What's NOT Being Built (Yet)

- Dunning/payment recovery (add when revenue justifies)
- Onboarding automation (add after leads convert)
- Health scores (add after customer base grows)
- Complex workflows (add when needed)
- Multiple LLM providers (add when OpenAI isn't enough)

---

## Start Here

**Phase 1.1:** Create TreeHouse base interface

This unlocks all TreeHouse development and establishes the deterministic processing pattern.
