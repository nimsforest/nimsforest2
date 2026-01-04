# 🌲 NimsForest

## What It Is

An event-driven automation framework. Components subscribe to NATS subjects, process events, publish results. That's it.

---

## Core Primitives

| Primitive | What It Does | Nature |
|-----------|--------------|--------|
| **Tree** | Parses raw external data → structured events | Deterministic |
| **TreeHouse** | Applies business rules. Same input = same output. | Deterministic |
| **Nim** | Makes decisions requiring judgment. Human or LLM. | Non-deterministic |

### Infrastructure

| Component | Purpose |
|-----------|---------|
| **Wind** | NATS pub/sub. Carries events between components. |
| **River** | NATS JetStream. Ingests external data. |
| **Humus** | NATS JetStream. Logs state changes. |
| **Soil** | NATS KV. Stores current state. |
| **Decomposer** | Applies state changes from Humus to Soil. |

---

## How Components Connect

Components don't register with a central system. They subscribe to NATS subjects.

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  ScoringHouse   │     │ QualificationH. │     │   TriageNim     │
│                 │     │                 │     │                 │
│ subscribes to:  │     │ subscribes to:  │     │ subscribes to:  │
│ contact.created │     │ lead.scored     │     │ ticket.routed   │
│                 │     │                 │     │                 │
│ publishes:      │     │ publishes:      │     │ publishes:      │
│ lead.scored     │     │ lead.qualified  │     │ ticket.triaged  │
└────────┬────────┘     └────────┬────────┘     └────────┬────────┘
         │                       │                       │
         └───────────────────────┴───────────────────────┘
                                 │
                                 ▼
                              NATS
                       (the only connection)
```

---

## Trees vs TreeHouses vs Nims

All three subscribe to subjects and publish results. The difference:

| Type | Input | Processing | Output |
|------|-------|------------|--------|
| **Tree** | Raw data (River) | Parse/structure | Structured event (Wind) |
| **TreeHouse** | Structured event | Deterministic rules | Enriched/routed event |
| **Nim** | Structured event | Human or LLM judgment | Decision/action |

### Deterministic vs Non-Deterministic

**TreeHouses** are deterministic:
- Same input always produces same output
- Unit testable with fixed inputs
- No external calls, no randomness

**Nims** are non-deterministic:
- May produce different output for same input
- Involve LLM reasoning or human judgment
- Test with mocked LLM or approval stubs

---

## Adapters

Adapters translate external system webhooks to generic events.

```
External Systems              Adapters                    Generic Events
────────────────              ────────                    ──────────────
Stripe webhook     ────►      Stripe adapter    ────►    payment.received
HubSpot webhook    ────►      CRM adapter       ────►    contact.created
Zendesk webhook    ────►      Support adapter   ────►    ticket.created
```

Adapters are thin. They just translate format. No business logic.

Core framework only sees generic events. Easily testable without external services.

---

## Organization

The framework has no opinion on how you organize components.

**Folder structure is up to you:**

```
internal/
├── growth/           # Or "sales/", "acquisition/", whatever
├── ops/              # Or "support/", "delivery/", whatever
└── direction/        # Or "insights/", "analytics/", whatever
```

**Subject naming is up to you:**

```
growth.contact.created
growth.lead.scored
ops.ticket.created
ops.ticket.triaged
```

The framework doesn't care. Components subscribe to subjects. That's the only contract.

---

## Example Use Case: SME Scaling

One way to use NimsForest. Not the only way.

**Goal:** Automate route to $1M ARR with 10 FTEs.

**Domains:**
- Growth: Contact → Customer
- Ops: Customer → Value  
- Direction: Strategy + Insights

**Components:**

| Domain | Component | Type | Does |
|--------|-----------|------|------|
| Growth | ScoringHouse | TreeHouse | Score leads |
| Growth | QualificationHouse | TreeHouse | MQL/SQL |
| Growth | EnrichNim | Nim | LLM research |
| Ops | RoutingHouse | TreeHouse | Route tickets |
| Ops | TriageNim | Nim | LLM sentiment |
| Ops | ResponseNim | Nim | LLM draft |
| Direction | MetricsHouse | TreeHouse | Aggregate |
| Direction | AnalyzeNim | Nim | LLM insights |

This is an example. Your use case may be different.

---

## Principles

1. **Components subscribe, that's it.** No registration, no central orchestrator.

2. **Deterministic where possible.** TreeHouses handle the volume. Nims handle exceptions.

3. **Vendor-agnostic core.** Adapters translate. Core sees generic events.

4. **Test without external services.** Mock adapters, mock LLM, test core logic.

5. **Organize however you want.** Framework doesn't impose structure.
