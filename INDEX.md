# NimsForest Document Index

## The Goal

**Automate the route to $1M ARR with 10 FTEs.**

---

## Start Here

| Document | Purpose |
|----------|---------|
| [VISION.md](./VISION.md) | Why we're building this. Core vs Adapters. |
| [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) | What to build, in what order. |

---

## Architecture

```
ADAPTERS (separate, optional)          CORE FRAMEWORK (testable)
─────────────────────────              ─────────────────────────
Stripe   ─┐                            
PayPal   ─┼─► payment.received ──────► TreeHouses (deterministic)
                                              │
HubSpot  ─┐                                   ▼
Salesforce┼─► contact.created ───────► Nims (human/LLM)
                                              │
Zendesk  ─┐                                   ▼
Intercom ─┼─► ticket.created ────────► Soil (state)
```

**Core is vendor-agnostic. Adapters translate external systems.**

---

## Key Concepts

| Concept | What It Is |
|---------|------------|
| **Tree** | Parses raw data → structured event (deterministic) |
| **Tree House** | Applies business rules. Same input = same output. (deterministic) |
| **Nim** | Makes decisions requiring judgment. Human or LLM. (non-deterministic) |
| **Adapter** | Translates external webhook → generic event |

---

## MVP Focus

**Contacts → Qualified Leads → Sales**

| Phase | What |
|-------|------|
| 1 | TreeHouse foundation + generic leaf types |
| 2 | Lead path: Scoring → Qualification → Enrichment |
| 3 | LLM infrastructure |
| 4 | Support path: Routing → Triage → Response |
| 5 | Adapters (Stripe, CRM, Support) |
| 6 | Integration & E2E tests |

---

## Reference Documents

| Document | When to Use |
|----------|-------------|
| [README.md](./README.md) | Technical overview, quick start |
| [Cursorinstructions.md](./Cursorinstructions.md) | Original architecture spec |
| [EXTENSIBILITY_GUIDE.md](./EXTENSIBILITY_GUIDE.md) | Adding new components |

---

## File Structure

```
internal/
├── core/           # Base infrastructure (exists)
├── leaves/         # Generic event types
├── treehouses/     # Deterministic rules
├── nims/           # Human/LLM decisions
└── llm/            # LLM client

adapters/           # External system translations (separate)
├── stripe/
├── crm/
└── support/

test/e2e/           # End-to-end tests (no external services)
```
