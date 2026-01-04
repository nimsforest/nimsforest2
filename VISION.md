# 🌲 NimsForest Vision

## The Goal

**Automate the route to $1M ARR with 10 FTEs.**

## The Focus

**Contacts → Qualified Leads → Sales**

Everything else is secondary.

---

## Architecture: Core vs Adapters

### The Problem with Coupling

If Salesforce/HubSpot/Stripe are baked into the framework:
- Can't test without mocking external APIs
- Can't swap providers without rewriting
- Framework is tied to specific vendors

### The Solution: Abstract Core + Pluggable Adapters

```
┌─────────────────────────────────────────────────────────────┐
│                         ADAPTERS                             │
│  (Translate external systems to generic events)              │
│                                                              │
│   Stripe ──┐    HubSpot ──┐    Zendesk ──┐                  │
│   PayPal ──┼──► Payment   │    Salesforce┼──► Contact       │
│   Paddle ──┘    Adapter   │    Pipedrive─┘    Adapter       │
│                    │      │                      │           │
│                    ▼      │                      ▼           │
└────────────────────┼──────┴──────────────────────┼───────────┘
                     │                             │
┌────────────────────┼─────────────────────────────┼───────────┐
│                    ▼         CORE FRAMEWORK      ▼           │
│                                                              │
│   payment.received ─────►  contact.created ─────►            │
│          │                       │                           │
│          ▼                       ▼                           │
│   Tree Houses              Tree Houses                       │
│   (Deterministic)          (Scoring, Qualifying)             │
│          │                       │                           │
│          ▼                       ▼                           │
│       Nims                    Nims                           │
│   (Human/LLM)              (Human/LLM)                       │
│          │                       │                           │
│          ▼                       ▼                           │
│       Soil ◄──────────────────────                          │
│   (State)                                                    │
│                                                              │
│   Generic concepts: Contact, Lead, Ticket, Payment           │
│   No vendor names. Fully testable.                           │
└──────────────────────────────────────────────────────────────┘
```

---

## Core Concepts (Vendor-Agnostic)

| Concept | What It Represents | Not Tied To |
|---------|-------------------|-------------|
| **Contact** | A person/company we know about | HubSpot, Salesforce |
| **Lead** | A contact showing buying intent | Any CRM |
| **Ticket** | A support request | Zendesk, Intercom |
| **Payment** | Money received/failed | Stripe, PayPal |
| **Message** | Communication sent/received | SendGrid, Mailgun |

The core framework only knows about these abstractions.

---

## Component Types

| Component | Nature | Purpose |
|-----------|--------|---------|
| **Adapter** | Translation | Convert external webhook → generic event |
| **Tree** | Deterministic | Parse river data → structured leaf |
| **Tree House** | Deterministic | Apply rules. Same input = same output. |
| **Nim** | Non-deterministic | Human or LLM judgment required |

### Trees vs Tree Houses

Both are deterministic, but:
- **Trees** parse raw data into structured events (edge of system)
- **Tree Houses** apply business rules to structured events (core logic)

### Nims

Non-deterministic. Used when:
- Judgment is needed (LLM analyzes sentiment)
- Human approval is required
- Context matters (different answer for same input)

---

## MVP: Contacts → Leads → Sales

### What We Actually Need

```
Contact enters system
       │
       ▼
   Score it (TreeHouse)
   - Firmographic signals
   - Behavioral signals
       │
       ▼
   Qualify it (TreeHouse)
   - MQL threshold
   - SQL threshold
       │
       ▼
   Surface to sales (Nim)
   - LLM enrichment
   - Human prioritization
       │
       ▼
   Close deal
```

### Secondary: Support That Scales

```
Ticket enters system
       │
       ▼
   Route it (TreeHouse)
   - Category
   - Priority
       │
       ▼
   Triage it (Nim - LLM)
   - Sentiment
   - Urgency
       │
       ▼
   Draft response (Nim - LLM)
   - Context-aware
   - Human reviews
```

---

## What's In vs Out of Core

### IN: Core Framework

- Base interfaces (Tree, TreeHouse, Nim)
- Generic leaf types (Contact, Lead, Ticket, Payment)
- Infrastructure (Wind, River, Humus, Soil)
- Abstract Tree Houses (Scoring, Qualification, Routing)
- Abstract Nims (Triage, Response, Approval)

### OUT: Adapters (Separate Package)

- Stripe adapter
- HubSpot adapter
- Salesforce adapter
- Zendesk adapter
- SendGrid adapter

Adapters are thin. They just translate webhooks to generic events.

---

## Testing Strategy

### Core Framework Tests

```go
// No external services needed
func TestScoringHouse(t *testing.T) {
    // Given a contact with these attributes
    contact := Contact{
        CompanySize: 100,
        Title: "VP Engineering",
        PagesViewed: []string{"/pricing"},
    }
    
    // When scored
    result := scoringHouse.Process(contact)
    
    // Then score is calculated correctly
    assert.Equal(t, 85, result.Score)
}
```

### Adapter Tests

```go
// Test translation only
func TestStripeAdapter(t *testing.T) {
    // Given a Stripe webhook payload
    webhook := `{"type": "charge.succeeded", ...}`
    
    // When translated
    payment := stripeAdapter.Translate(webhook)
    
    // Then generic payment event is correct
    assert.Equal(t, "payment.received", payment.Type)
}
```

### E2E Tests

```go
// Use mock adapters, test full flow
func TestContactToQualifiedLead(t *testing.T) {
    // Send generic contact event (no Salesforce needed)
    river.Flow("contact.created", contact)
    
    // Wait for processing
    // Assert lead is qualified in Soil
}
```

---

## Success Criteria

1. **Core is vendor-agnostic** - No HubSpot/Stripe in framework code
2. **Everything built is used** - No dead code
3. **E2E tests work offline** - No external API calls
4. **Path to leads is clear** - Contact → Score → Qualify → Surface

---

## Guiding Principles

1. **Abstract the vendor, not the concept.** Payment is universal. Stripe is not.

2. **Test the core, mock the edge.** Core framework tests need zero external services.

3. **Build what you'll use.** No speculative features.

4. **Leads first.** Revenue comes from qualified leads, not features.

---

See [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) for the simplified build order.
