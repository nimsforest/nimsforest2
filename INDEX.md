# NimsForest

Event-driven automation framework. Components subscribe to NATS, process events, publish results.

---

## Docs

| Document | What |
|----------|------|
| [VISION.md](./VISION.md) | What NimsForest is. Core primitives. |
| [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) | What to build, in what order. |
| [README.md](./README.md) | Quick start, running the system. |

---

## Core Primitives

| Primitive | What | Nature |
|-----------|------|--------|
| **Tree** | Parses raw data → structured events | Deterministic |
| **TreeHouse** | Applies rules. Same input = same output. | Deterministic |
| **Nim** | Makes decisions. Human or LLM. | Non-deterministic |

---

## How It Works

```
Components subscribe to NATS subjects.
Components publish to NATS subjects.
NATS connects them.
```

No central orchestrator. No registration. Just pub/sub.

---

## File Structure

```
internal/
├── core/           # Primitives (Wind, River, Tree, TreeHouse, Nim, etc.)
├── leaves/         # Event types (Contact, Lead, Ticket, Payment)
├── treehouses/     # Deterministic rules
├── nims/           # LLM/Human decisions
└── llm/            # LLM client

adapters/           # External system translation (Stripe, CRM, etc.)
```

---

## Start

1. Read [VISION.md](./VISION.md)
2. Follow [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md)
3. Start with Phase 1: TreeHouse base
