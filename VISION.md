# 🌲 NimsForest Vision

## The Goal

**Automate the route to $1M ARR with 10 FTEs.**

A 10-person company must do the work of 30-50 people. NimsForest is the automation layer that makes this possible.

---

## The Problem

Small teams hit a wall. As customers grow, so does operational load:

| Function | What Happens Without Automation |
|----------|--------------------------------|
| **Support** | Tickets pile up. Response times spike. Customers churn. |
| **Sales** | Leads go cold. Manual qualification wastes time on bad fits. |
| **Onboarding** | Each customer needs hand-holding. Doesn't scale. |
| **Billing** | Failed payments need chasing. Disputes eat time. |
| **Operations** | Admin overhead consumes everyone. No time for growth. |

**The result:** 10 people spend all their time on operations, zero on growth.

---

## The Solution

NimsForest is an event-driven automation system that:

1. **Ingests data** from external systems (payments, CRM, support, email)
2. **Applies rules** deterministically (routing, scoring, thresholds)
3. **Uses AI** for judgment calls (triage, drafting, analysis)
4. **Surfaces exceptions** to humans (approvals, edge cases)

### The Core Principle

> **Humans for exceptions. Machines for rules.**

```
Volume Distribution:

80%+ ─── TREE HOUSES (deterministic)
         Rules-based. Same input = same output.
         No human needed.

~15% ─── NIMS (LLM)
         AI judgment. Human reviews output.
         Drafts, analysis, suggestions.

~5%  ─── NIMS (Human)
         True exceptions. High-value decisions.
         Approvals, relationships, strategy.
```

---

## Architecture

### The Forest Metaphor

| Component | What It Is | Nature |
|-----------|------------|--------|
| **River** | External data flowing in | Raw webhooks, API events |
| **Tree** | Parser that structures data | Deterministic transformation |
| **Leaf** | Structured event | The contract between components |
| **Tree House** | Rules engine | Deterministic. Same in, same out. |
| **Nim** | Decision maker | Non-deterministic. Human or LLM. |
| **Wind** | Event distribution | Carries leaves between components |
| **Humus** | State change log | Audit trail of all changes |
| **Soil** | Current state | Source of truth |

### Data Flow

```
External System (Stripe, Zendesk, HubSpot)
         │
         ▼
      RIVER ──────────────────────────────────────
         │                                        │
         ▼                                        │
       TREE (parse webhook)                       │
         │                                        │ DETERMINISTIC
         ▼                                        │
       LEAF (structured event)                    │
         │                                        │
         ▼                                        │
    TREE HOUSE (apply rules)                      │
         │                                        │
         ▼                                   ─────
       LEAF (enriched/routed)
         │
         ▼                                   ─────
       NIM (LLM/Human decision)                   │
         │                                        │ NON-DETERMINISTIC
         ▼                                        │
    LEAF + COMPOST                           ─────
         │
         ▼
       SOIL (state updated)
```

---

## Key Distinctions

### Trees vs Tree Houses vs Nims

| Component | Deterministic? | State? | Purpose |
|-----------|---------------|--------|---------|
| **Tree** | Yes | No | Parse external data into structured events |
| **Tree House** | Yes | No | Apply business rules. Same input = same output. |
| **Nim** | No | Yes | Make decisions requiring judgment (human or LLM) |

### Tree Houses: Deterministic Rules

Tree Houses are pure functions. They:
- Apply formulas (lead scores, health scores, pricing)
- Route based on conditions (tier, category, region)
- Trigger thresholds (SLA breach, low stock, failed payments)
- Validate data (required fields, formats)
- Enrich with lookups (geo, company data)

**Example:** A support ticket always routes to the same queue given the same inputs.

### Nims: Non-Deterministic Decisions

Nims involve judgment. They:
- Analyze sentiment and intent (LLM)
- Draft responses (LLM)
- Suggest actions (LLM)
- Approve exceptions (Human)
- Handle ambiguity (Human or LLM)

**Example:** An LLM analyzing a support ticket may interpret urgency differently based on context.

---

## Business Impact

### What NimsForest Automates

| Function | Without NimsForest | With NimsForest |
|----------|-------------------|-----------------|
| **Support** | 20-30 tickets/person/day | 100-150 tickets/person/day |
| **Sales** | Manual qualification (hours) | Auto-qualification (minutes) |
| **Billing** | Chase failed payments manually | Auto-retry + dunning sequences |
| **Onboarding** | Hand-hold each customer | Self-serve with smart nudges |
| **Operations** | Reactive firefighting | Proactive alerts on exceptions |

### The 10 FTE Team

| Role | Count | What They Do | What NimsForest Does |
|------|-------|--------------|---------------------|
| Founder/CEO | 1 | Strategy, big deals | Dashboard of exceptions |
| Product/Eng | 3 | Build product, maintain system | Trees + Tree Houses = leverage |
| Sales | 2 | Close qualified deals | Pre-qualified leads, drafted proposals |
| Support | 2 | Handle escalations | LLM drafts 80% of responses |
| Marketing | 1 | Campaigns, content | Lead scoring shows what works |
| Ops/Finance | 1 | Billing exceptions | System handles routine billing |

---

## MVP Scope

### What We're Building

| Layer | Components | Purpose |
|-------|-----------|---------|
| **Trees** | Payment, Support, CRM | Ingest external data |
| **Tree Houses** | Dunning, Routing, Scoring, Threshold, Onboarding | Deterministic rules |
| **Nims** | Triage, Response, Approval | LLM + Human decisions |

### What We're NOT Building (Yet)

- Inventory management
- Calendar/scheduling
- Code/engineering workflows
- Advanced analytics
- Complex multi-step workflows

### Revenue-First Priority

1. **Don't lose money** — Billing automation, payment recovery
2. **Support at scale** — LLM triage and response drafts
3. **Sales efficiency** — Lead scoring and auto-qualification
4. **Customer success** — Automated onboarding, health scores

---

## Success Criteria

NimsForest is successful when:

1. **Support scales:** 2 people handle 500+ customers effectively
2. **Sales focuses:** Reps only talk to qualified, ready-to-buy leads
3. **Billing runs itself:** Failed payments auto-recover, exceptions surface
4. **Onboarding is self-serve:** 80%+ customers succeed without hand-holding
5. **Exceptions surface:** System tells humans what needs attention

---

## Guiding Principles

1. **Automate the rule, surface the exception.** If it can be a rule, it's a Tree House. If it needs judgment, it's a Nim.

2. **LLM for drafts, humans for sends.** AI suggests, humans approve (at least initially).

3. **Deterministic first.** Build Tree Houses before Nims. Rules scale infinitely.

4. **One source of truth.** All state lives in Soil. All changes flow through Humus.

5. **Observe everything.** Every event, decision, and state change is logged.

---

## What's Next

See [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) for the ordered task list.
