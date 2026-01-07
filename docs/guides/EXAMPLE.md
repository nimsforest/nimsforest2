# Example: Lead Scoring Pipeline

Contact comes in → TreeHouse scores it → Nim decides if worth pursuing.

---

## Config

```yaml
# config/forest.yaml

treehouses:
  scoring:
    subscribes: contact.created
    publishes: lead.scored
    script: scripts/treehouses/scoring.lua

nims:
  qualify:
    subscribes: lead.scored
    publishes: lead.qualified
    prompt: scripts/nims/qualify.md
```

## Prompt

```markdown
<!-- scripts/nims/qualify.md -->

You are a sales qualification assistant.

A lead was scored:
- Contact ID: {{.contact_id}}
- Email: {{.email}}
- Score: {{.score}}
- Signals: {{.signals}}

Based on this score and signals, should we pursue this lead?

Reply with JSON only:
{"pursue": true/false, "reason": "one sentence explanation"}
```

---

## Lua Script

```lua
-- scripts/treehouses/scoring.lua

function process(contact)
    local score = 0
    local signals = {}
    
    -- Company size scoring
    if contact.company_size > 500 then
        score = score + 50
        table.insert(signals, "enterprise")
    elseif contact.company_size > 100 then
        score = score + 30
        table.insert(signals, "mid_market")
    elseif contact.company_size > 20 then
        score = score + 10
        table.insert(signals, "smb")
    end
    
    -- Title scoring
    if contains(contact.title, "CEO") or 
       contains(contact.title, "CTO") or
       contains(contact.title, "VP") then
        score = score + 40
        table.insert(signals, "executive")
    elseif contains(contact.title, "Director") or
           contains(contact.title, "Manager") then
        score = score + 20
        table.insert(signals, "manager")
    end
    
    -- Industry bonus
    if contact.industry == "technology" or
       contact.industry == "finance" then
        score = score + 15
        table.insert(signals, "target_industry")
    end
    
    return {
        contact_id = contact.id,
        email = contact.email,
        score = score,
        signals = signals
    }
end
```

---

## Flow

```
1. Publish to NATS:
   Subject: contact.created
   Data: {"id": "c123", "email": "jane@acme.com", "title": "VP Engineering", "company_size": 250, "industry": "technology"}

2. TreeHouse (scoring) receives it, runs Lua:
   Output: {"contact_id": "c123", "email": "jane@acme.com", "score": 85, "signals": ["mid_market", "executive", "target_industry"]}
   Publishes to: lead.scored

3. Nim (qualify) receives it, calls Claude:
   Prompt: "A lead was scored... Score: 85, Signals: [mid_market, executive, target_industry]..."
   Claude responds: {"pursue": true, "reason": "High score with executive title at mid-market tech company"}
   Publishes to: lead.qualified
```

---

## Test It

```bash
# Terminal 1: Run forest
go run cmd/forest/main.go

# Terminal 2: Publish test event
nats pub contact.created '{"id":"c123","email":"jane@acme.com","title":"VP Engineering","company_size":250,"industry":"technology"}'

# Terminal 3: Watch output
nats sub "lead.>"
```

---

## Environment

```bash
export NATS_URL=nats://localhost:4222
export CLAUDE_API_KEY=sk-ant-...
```
