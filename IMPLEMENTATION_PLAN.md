# 🌲 NimsForest Implementation Plan

## What Exists

- [x] `internal/core/wind.go` - NATS pub/sub
- [x] `internal/core/river.go` - NATS JetStream ingestion
- [x] `internal/core/humus.go` - NATS JetStream state log
- [x] `internal/core/soil.go` - NATS KV state store
- [x] `internal/core/decomposer.go` - Applies humus to soil
- [x] `internal/core/leaf.go` - Event structure
- [x] `internal/core/tree.go` - Parser base
- [x] `internal/core/nim.go` - Decision base
- [x] `internal/trees/payment.go` - Stripe parser (move to adapter)
- [x] `internal/nims/aftersales.go` - Example nim

---

## What To Build

### Phase 1: TreeHouse Base

**Goal:** Enable deterministic rule processing.

**File:** `internal/core/treehouse.go`

```go
// TreeHouse applies deterministic rules to events
// Same input MUST produce same output
type TreeHouse interface {
    Name() string
    InputSubjects() []string
    Process(leaf Leaf) ([]Leaf, error)
}

// BaseTreeHouse handles subscription and publishing
type BaseTreeHouse struct {
    name string
    wind *Wind
}

func (h *BaseTreeHouse) Start(ctx context.Context) error {
    for _, subject := range h.InputSubjects() {
        h.wind.Catch(subject, func(leaf Leaf) {
            results, err := h.Process(leaf)
            if err != nil {
                log.Printf("[%s] error: %v", h.name, err)
                return
            }
            for _, result := range results {
                h.wind.Drop(result)
            }
        })
    }
    return nil
}
```

Tasks:
- [ ] Create TreeHouse interface
- [ ] Create BaseTreeHouse with Wind subscription
- [ ] Unit tests asserting determinism

---

### Phase 2: LLM Client

**Goal:** Enable Nims to use LLM reasoning.

**Files:** `internal/llm/`

```go
// internal/llm/client.go
type Client interface {
    Complete(ctx context.Context, prompt string) (string, error)
}

// internal/llm/openai.go
type OpenAI struct {
    apiKey string
    model  string
}

// internal/llm/mock.go
type Mock struct {
    responses map[string]string
}
```

Tasks:
- [ ] Create Client interface
- [ ] Create OpenAI implementation
- [ ] Create Mock for testing

---

### Phase 3: Generic Leaf Types

**Goal:** Define vendor-agnostic event types.

**File:** `internal/leaves/types.go`

```go
// Contact - a person or company
type Contact struct {
    ID        string
    Email     string
    Name      string
    Company   string
    Source    string
    CreatedAt time.Time
}

// Lead - scored contact
type Lead struct {
    ContactID     string
    Score         int
    Qualification string  // "none", "mql", "sql"
    Signals       []string
}

// Ticket - support request
type Ticket struct {
    ID        string
    ContactID string
    Subject   string
    Body      string
    Priority  string
    Status    string
}

// Payment - money movement
type Payment struct {
    ID        string
    ContactID string
    Amount    float64
    Currency  string
    Status    string  // "succeeded", "failed"
}
```

Tasks:
- [ ] Define Contact
- [ ] Define Lead
- [ ] Define Ticket
- [ ] Define Payment

---

### Phase 4: Example TreeHouses

**Goal:** Demonstrate deterministic rule processing.

#### 4.1 ScoringHouse

**File:** `internal/treehouses/scoring.go`

```go
type ScoringHouse struct {
    *core.BaseTreeHouse
    rules ScoringRules
}

func (h *ScoringHouse) InputSubjects() []string {
    return []string{"contact.created"}
}

func (h *ScoringHouse) Process(leaf Leaf) ([]Leaf, error) {
    var contact Contact
    json.Unmarshal(leaf.Data, &contact)
    
    score := h.calculateScore(contact)
    lead := Lead{
        ContactID: contact.ID,
        Score:     score,
        Signals:   h.getSignals(contact),
    }
    
    return []Leaf{NewLeaf("lead.scored", lead)}, nil
}
```

Tasks:
- [ ] Create ScoringHouse
- [ ] Configurable scoring rules
- [ ] Unit tests with various inputs

#### 4.2 QualificationHouse

**File:** `internal/treehouses/qualification.go`

```go
func (h *QualificationHouse) InputSubjects() []string {
    return []string{"lead.scored"}
}

func (h *QualificationHouse) Process(leaf Leaf) ([]Leaf, error) {
    var lead Lead
    json.Unmarshal(leaf.Data, &lead)
    
    if lead.Score >= h.rules.SQLThreshold {
        lead.Qualification = "sql"
    } else if lead.Score >= h.rules.MQLThreshold {
        lead.Qualification = "mql"
    }
    
    return []Leaf{NewLeaf("lead.qualified", lead)}, nil
}
```

Tasks:
- [ ] Create QualificationHouse
- [ ] Configurable thresholds
- [ ] Unit tests for edge cases

#### 4.3 RoutingHouse

**File:** `internal/treehouses/routing.go`

```go
func (h *RoutingHouse) InputSubjects() []string {
    return []string{"ticket.created"}
}

func (h *RoutingHouse) Process(leaf Leaf) ([]Leaf, error) {
    var ticket Ticket
    json.Unmarshal(leaf.Data, &ticket)
    
    ticket.Priority = h.calculatePriority(ticket)
    category := h.categorize(ticket)
    
    return []Leaf{NewLeaf("ticket.routed", ticket)}, nil
}
```

Tasks:
- [ ] Create RoutingHouse
- [ ] Keyword-based categorization
- [ ] Priority rules
- [ ] Unit tests

---

### Phase 5: Example Nims

**Goal:** Demonstrate LLM-powered decisions.

#### 5.1 TriageNim

**File:** `internal/nims/triage.go`

```go
type TriageNim struct {
    *core.BaseNim
    llm llm.Client
}

func (n *TriageNim) Subjects() []string {
    return []string{"ticket.routed"}
}

func (n *TriageNim) Handle(ctx context.Context, leaf Leaf) error {
    var ticket Ticket
    json.Unmarshal(leaf.Data, &ticket)
    
    // LLM analyzes sentiment, urgency, intent
    analysis, err := n.llm.Complete(ctx, n.buildPrompt(ticket))
    if err != nil {
        return err
    }
    
    triage := parseTriage(analysis)
    return n.Leaf("ticket.triaged", triage)
}
```

Tasks:
- [ ] Create TriageNim
- [ ] Prompt for sentiment/urgency analysis
- [ ] Tests with mock LLM

#### 5.2 ResponseNim

**File:** `internal/nims/response.go`

```go
func (n *ResponseNim) Subjects() []string {
    return []string{"ticket.triaged"}
}

func (n *ResponseNim) Handle(ctx context.Context, leaf Leaf) error {
    // LLM drafts response based on ticket + triage
    draft, err := n.llm.Complete(ctx, n.buildPrompt(leaf))
    if err != nil {
        return err
    }
    
    return n.Leaf("response.drafted", draft)
}
```

Tasks:
- [ ] Create ResponseNim
- [ ] Prompt for response drafting
- [ ] Tests with mock LLM

---

### Phase 6: Adapters

**Goal:** Translate external webhooks to generic events.

**Directory:** `adapters/`

```go
// adapters/adapter.go
type Adapter interface {
    Translate(subject string, data []byte) (*Leaf, error)
}

// adapters/stripe/adapter.go
func (a *StripeAdapter) Translate(subject string, data []byte) (*Leaf, error) {
    var webhook StripeWebhook
    json.Unmarshal(data, &webhook)
    
    switch webhook.Type {
    case "charge.succeeded":
        payment := Payment{...}
        return NewLeaf("payment.received", payment), nil
    }
}
```

Tasks:
- [ ] Create Adapter interface
- [ ] Create Stripe adapter (move from trees/payment.go)
- [ ] Create generic CRM adapter
- [ ] Create generic Support adapter

---

### Phase 7: Integration

**Goal:** Wire it together, prove it works.

#### 7.1 Main.go

```go
func main() {
    // Infrastructure
    nc, _ := nats.Connect(natsURL)
    js, _ := nc.JetStream()
    
    wind := core.NewWind(nc)
    river, _ := core.NewRiver(js)
    humus, _ := core.NewHumus(js)
    soil, _ := core.NewSoil(js)
    
    go core.RunDecomposer(humus, soil)
    
    // LLM
    llmClient := llm.NewOpenAI(os.Getenv("OPENAI_API_KEY"))
    
    // TreeHouses (subscribe on creation)
    treehouses.NewScoringHouse(wind, scoringRules)
    treehouses.NewQualificationHouse(wind, qualRules)
    treehouses.NewRoutingHouse(wind, routingRules)
    
    // Nims (subscribe on creation)
    nims.NewTriageNim(wind, humus, soil, llmClient)
    nims.NewResponseNim(wind, humus, soil, llmClient)
    
    // Running. Components subscribed to NATS.
    select{}
}
```

Tasks:
- [ ] Update main.go with new components
- [ ] Configuration loading

#### 7.2 E2E Tests

```go
func TestLeadScoring(t *testing.T) {
    // Publish contact.created
    wind.Drop(NewLeaf("contact.created", contact))
    
    // Wait for lead.qualified
    // Assert score and qualification
}

func TestTicketTriage(t *testing.T) {
    // Use mock LLM
    // Publish ticket.created
    // Assert ticket.triaged has sentiment
}
```

Tasks:
- [ ] E2E test for lead flow
- [ ] E2E test for ticket flow
- [ ] All tests pass without external services

---

## File Checklist

### Core
- [ ] `internal/core/treehouse.go`
- [ ] `internal/core/treehouse_test.go`

### LLM
- [ ] `internal/llm/client.go`
- [ ] `internal/llm/openai.go`
- [ ] `internal/llm/mock.go`

### Leaves
- [ ] `internal/leaves/types.go` (expand)

### TreeHouses
- [ ] `internal/treehouses/scoring.go`
- [ ] `internal/treehouses/scoring_test.go`
- [ ] `internal/treehouses/qualification.go`
- [ ] `internal/treehouses/qualification_test.go`
- [ ] `internal/treehouses/routing.go`
- [ ] `internal/treehouses/routing_test.go`

### Nims
- [ ] `internal/nims/triage.go`
- [ ] `internal/nims/triage_test.go`
- [ ] `internal/nims/response.go`
- [ ] `internal/nims/response_test.go`

### Adapters
- [ ] `adapters/adapter.go`
- [ ] `adapters/stripe/adapter.go`
- [ ] `adapters/stripe/adapter_test.go`

### Tests
- [ ] `test/e2e/lead_test.go`
- [ ] `test/e2e/ticket_test.go`

---

## Dependency Order

```
Phase 1 (TreeHouse base)
    │
    ├──► Phase 4.1 (ScoringHouse)
    │         │
    │         └──► Phase 4.2 (QualificationHouse)
    │
    └──► Phase 4.3 (RoutingHouse)
              │
              └──► Phase 5.1 (TriageNim) ◄── Phase 2 (LLM)
                        │
                        └──► Phase 5.2 (ResponseNim)

Phase 3 (Leaf types) ── needed by Phase 4+
Phase 6 (Adapters) ── independent
Phase 7 (Integration) ── after all above
```

---

## Start Here

**Phase 1:** Create TreeHouse interface and BaseTreeHouse.

This enables all deterministic rule processing.
