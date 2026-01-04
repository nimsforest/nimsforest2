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
    script: scripts/scoring.lua    # Lua for rules

nims:
  triage:
    subscribes: ticket.routed
    publishes: ticket.triaged
    brain: openai
    prompt: |                       # Prompt for LLM
      Analyze this ticket: {{.body}}
```

```lua
-- scripts/scoring.lua - deterministic rules
function process(contact)
    local score = 0
    if contact.company_size > 200 then
        score = score + 40
    end
    return { contact_id = contact.id, score = score }
end
```

- **TreeHouses** → Lua scripts (deterministic)
- **Nims** → Prompts to brain (non-deterministic)
- **River** → NATS connects everything

---

## Core Primitives

| Primitive | Nature | What It Does |
|-----------|--------|--------------|
| **River** | Infrastructure | Event stream (NATS). Events flow through the forest. |
| **Source** | Interface | Feeds external data into the River. |
| **TreeHouse** | Deterministic | Applies business rules (Lua). Same input = same output. |
| **Nim** | Non-deterministic | Calls brain (LLM) with prompt. No script - just config. |
| **Leaf** | Data | An event flowing through the River. |

### Source Implementations

| Implementation | What It Connects |
|----------------|------------------|
| `SalesforceSource` | Salesforce CRM |
| `HubSpotSource` | HubSpot CRM |
| `StripeSource` | Stripe payments |
| `ZendeskSource` | Zendesk support |
| `WebhookSource` | Generic webhooks |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                       SOURCES                                │
│              (implementations feed the River)                │
│                                                              │
│   StripeSource     ─┐                                       │
│   SalesforceSource ─┼──►  River (NATS)                      │
│   ZendeskSource    ─┘                                       │
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

# Sources - feed external data into the River
sources:
  payments:
    type: stripe
    webhook_path: /webhooks/stripe
    webhook_secret: ${STRIPE_WEBHOOK_SECRET}
    
  crm:
    type: salesforce
    instance_url: ${SALESFORCE_INSTANCE_URL}
    client_id: ${SALESFORCE_CLIENT_ID}
    
  support:
    type: zendesk
    subdomain: ${ZENDESK_SUBDOMAIN}
    webhook_path: /webhooks/zendesk

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

# Nims - call brain (LLM) with prompt, no script
nims:
  triage:
    subscribes: ticket.routed
    publishes: ticket.triaged
    brain: openai
    model: gpt-4o
    prompt: |
      Analyze this support ticket and return JSON with:
      - sentiment: positive/neutral/negative
      - urgency: low/medium/high/critical
      - category: billing/technical/general
      - summary: one sentence
      
      Ticket: {{.body}}
    
  response:
    subscribes: ticket.triaged
    publishes: response.drafted
    brain: claude
    model: claude-3-haiku-20240307
    prompt: |
      Draft a helpful response to this support ticket.
      Be empathetic and concise.
      
      Ticket: {{.body}}
      Category: {{.category}}
      Sentiment: {{.sentiment}}
```

---

## Lua Scripts (TreeHouses Only)

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

```lua
-- scripts/treehouses/routing.lua

function process(ticket)
    local team = "general"
    
    if contains(ticket.subject, "billing") or 
       contains(ticket.subject, "invoice") then
        team = "billing"
    elseif contains(ticket.subject, "bug") or 
           contains(ticket.subject, "error") then
        team = "engineering"
    end
    
    return {
        ticket_id = ticket.id,
        team = team,
        original_subject = ticket.subject
    }
end
```

---

## Lua Helpers (TreeHouses)

| Helper | What |
|--------|------|
| `contains(str, substr)` | String contains |
| `json.encode(table)` | Table to JSON |
| `json.decode(str)` | JSON to table |
| `log(msg)` | Logging |

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
│   └── treehouses/           # Deterministic Lua scripts
│       ├── scoring.lua
│       └── routing.lua
│
└── sources/                  # Source implementations
    ├── source.go             # Source interface
    ├── salesforce/           # SalesforceSource
    ├── hubspot/              # HubSpotSource
    ├── stripe/               # StripeSource
    ├── zendesk/              # ZendeskSource
    └── webhook/              # WebhookSource (generic)
```

---

## Principles

1. **Declarative config.** YAML defines Sources, TreeHouses, Nims.

2. **TreeHouses use Lua.** Deterministic rules. Same input = same output.

3. **Nims use prompts.** Non-deterministic. Brain (LLM) makes the call.

4. **Components subscribe to River.** No registration, no orchestrator.

5. **Sources are separate.** Vendor-specific implementations (SalesforceSource, StripeSource, etc.).
