# 🌲 NimsForest

Event-driven automation framework. Declarative config. Lua scripts. LLM-powered decisions.

---

## What It Is

```yaml
# forest.yaml - declare what exists
treehouses:
  scoring:
    subscribes: contact.created
    publishes: lead.scored
    script: scripts/scoring.lua

nims:
  triage:
    subscribes: ticket.routed
    publishes: ticket.triaged
    script: scripts/triage.lua
    brain: openai
```

```lua
-- scripts/scoring.lua - define the logic
function process(contact)
    local score = 0
    
    if contact.company_size > 200 then
        score = score + 40
    end
    
    return { contact_id = contact.id, score = score }
end
```

Components subscribe to NATS. NATS connects them. That's it.

---

## Core Primitives

| Primitive | Nature | What It Does |
|-----------|--------|--------------|
| **River** | Infrastructure | Event stream (NATS). Events flow through the forest. |
| **River Source** | Deterministic | Feeds external data into the River (webhooks, APIs). |
| **TreeHouse** | Deterministic | Applies business rules (Lua). Same input = same output. |
| **Nim** | Non-deterministic | Makes decisions using `pkg/brain` (LLM). |
| **Leaf** | Data | An event flowing through the River. |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    RIVER SOURCES                             │
│                 (adapters feed the river)                    │
│                                                              │
│   Stripe webhook  ─┐                                        │
│   CRM webhook     ─┼──►  River (NATS)                       │
│   Support webhook ─┘                                        │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    CORE FRAMEWORK                            │
│                                                              │
│   forest.yaml ─────► Runtime loads config                   │
│                           │                                  │
│                           ▼                                  │
│   scripts/*.lua ───► Lua VM executes logic                  │
│                           │                                  │
│                           ▼                                  │
│   pkg/brain ───────► LLM for Nims (OpenAI, Claude, Gemini)  │
│                           │                                  │
│                           ▼                                  │
│   River (NATS) ────► Pub/sub connects everything            │
└─────────────────────────────────────────────────────────────┘
```

---

## How Components Connect

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  ScoringHouse   │     │ QualificationH. │     │   TriageNim     │
│   (Lua)         │     │   (Lua)         │     │  (Lua + Brain)  │
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
```

No registration. No orchestrator. Components subscribe to NATS subjects.

---

## Declarative Config

```yaml
# config/forest.yaml

# River Sources - feed external data into the River
sources:
  stripe:
    type: webhook
    path: /webhooks/stripe
    publishes: payment.*
    
  crm:
    type: webhook
    path: /webhooks/crm
    publishes: contact.*, deal.*
    
  support:
    type: webhook
    path: /webhooks/support
    publishes: ticket.created

# TreeHouses - deterministic Lua scripts
treehouses:
  scoring:
    subscribes: contact.created
    publishes: lead.scored
    script: scripts/treehouses/scoring.lua
    
  qualification:
    subscribes: lead.scored
    publishes: lead.qualified
    script: scripts/treehouses/qualification.lua
    
  routing:
    subscribes: ticket.created
    publishes: ticket.routed
    script: scripts/treehouses/routing.lua

# Nims - Lua scripts with brain (LLM) access
nims:
  triage:
    subscribes: ticket.routed
    publishes: ticket.triaged
    script: scripts/nims/triage.lua
    brain: openai
    model: gpt-4o
    
  response:
    subscribes: ticket.triaged
    publishes: response.drafted
    script: scripts/nims/response.lua
    brain: claude
    model: claude-3-haiku-20240307
```

---

## Lua Scripts

### TreeHouse (Deterministic)

```lua
-- scripts/treehouses/scoring.lua

function process(contact)
    local score = 0
    local signals = {}
    
    if contact.company_size > 200 then
        score = score + 40
        table.insert(signals, "large_company")
    elseif contact.company_size > 50 then
        score = score + 20
        table.insert(signals, "medium_company")
    end
    
    if contains(contact.title, "VP") or 
       contains(contact.title, "Director") then
        score = score + 30
        table.insert(signals, "decision_maker")
    end
    
    return {
        contact_id = contact.id,
        score = score,
        signals = signals
    }
end
```

### Nim (With Brain Access)

```lua
-- scripts/nims/triage.lua

function process(ticket)
    -- Call the brain (LLM)
    local analysis = brain.ask(
        "Analyze this support ticket. " ..
        "Return JSON with: sentiment, urgency, category, summary.\n\n" ..
        "Ticket: " .. ticket.body
    )
    
    local result = json.decode(analysis)
    
    return {
        ticket_id = ticket.id,
        sentiment = result.sentiment,
        urgency = result.urgency,
        category = result.category,
        summary = result.summary,
        priority = calculate_priority(result)
    }
end

function calculate_priority(analysis)
    if analysis.urgency == "critical" then return "p1"
    elseif analysis.sentiment == "angry" then return "p2"
    else return "p3"
    end
end
```

---

## Lua Helpers Available

| Helper | What | Available In |
|--------|------|--------------|
| `contains(str, substr)` | String contains | All |
| `json.encode(table)` | Table to JSON | All |
| `json.decode(str)` | JSON to table | All |
| `log(msg)` | Logging | All |
| `brain.ask(prompt)` | Call LLM | Nims only |

---

## pkg/brain Integration

```
pkg/
├── brain/
│   ├── brain.go          # Factory, NewGenerativeBrainWithService
│   ├── interface.go      # Brain interface
│   └── testutil.go       # MockBrain for testing
│
└── integrations/
    └── aiservice/
        ├── factory.go    # Service registry
        └── thirdparty/
            ├── openai/   # OpenAI implementation
            ├── claude/   # Claude implementation
            └── gemini/   # Gemini implementation
```

Usage in Go runtime:
```go
b, _ := brain.NewGenerativeBrainWithService(
    brain.LLMServiceTypeOpenAI,
    os.Getenv("OPENAI_API_KEY"),
    "gpt-4o",
)
// Exposed to Lua as brain.ask()
```

---

## File Structure

```
nimsforest/
├── cmd/forest/main.go        # Entry point
│
├── pkg/
│   ├── brain/                # LLM integration (poached)
│   ├── infrastructure/       # AI service interface
│   ├── integrations/         # OpenAI, Claude, Gemini
│   └── runtime/              # Lua runtime, config loader
│       ├── config.go         # YAML config parser
│       ├── lua.go            # Lua VM wrapper
│       ├── treehouse.go      # TreeHouse runtime
│       ├── nim.go            # Nim runtime
│       └── helpers.go        # Lua helper functions
│
├── internal/core/            # NATS wrappers (existing)
│
├── config/
│   └── forest.yaml           # Declarative config
│
├── scripts/
│   ├── treehouses/           # Deterministic Lua
│   │   ├── scoring.lua
│   │   └── routing.lua
│   └── nims/                 # LLM-powered Lua
│       ├── triage.lua
│       └── response.lua
│
└── sources/                  # River sources (feed external data into River)
    ├── webhook/              # HTTP webhook receiver
    ├── stripe/               # Stripe events
    └── crm/                  # CRM events
```

---

## Principles

1. **Declarative config, Lua logic.** YAML says what exists. Lua says how it works.

2. **Components subscribe, that's it.** No registration, no orchestrator.

3. **TreeHouses are deterministic.** Same input = same output. Testable.

4. **Nims use brains.** LLM for judgment calls. `brain.ask()` in Lua.

5. **Adapters are separate.** Core is vendor-agnostic.
