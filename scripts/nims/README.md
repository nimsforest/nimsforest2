# Nim Prompts

Prompt templates for Nims (LLM-powered decisions).

---

## How It Works

Each `.md` file is a prompt template sent to Claude.

```
scripts/nims/
├── qualify.md      # Lead qualification prompt
└── README.md       # This file
```

Config references the file:

```yaml
nims:
  qualify:
    subscribes: lead.scored
    publishes: lead.qualified
    prompt: scripts/nims/qualify.md   # Path to prompt file
```

---

## Template Syntax

Use Go templates to insert event data:

| Syntax | What |
|--------|------|
| `{{.field}}` | Insert field value |
| `{{.nested.field}}` | Nested field |
| `{{range .items}}{{.}}{{end}}` | Loop over array |
| `{{if .field}}...{{end}}` | Conditional |

---

## qualify.md

Decides whether to pursue a scored lead.

### Input (from lead.scored)

```json
{
  "contact_id": "c123",
  "email": "jane@acme.com", 
  "score": 85,
  "signals": ["mid_market", "executive", "target_industry"]
}
```

### What Claude Sees

```
You are a sales qualification assistant.

A lead was scored:
- Contact ID: c123
- Email: jane@acme.com
- Score: 85
- Signals: [mid_market executive target_industry]

Based on this score and signals, should we pursue this lead?
...
```

### Output (published to lead.qualified)

```json
{
  "pursue": true,
  "reason": "High score with executive title at mid-market tech company"
}
```

---

## Tips

1. **Be specific.** Tell Claude exactly what format you want.
2. **Ask for JSON.** Easier to parse downstream.
3. **Give context.** Explain what the signals mean.
4. **Keep it focused.** One decision per Nim.
