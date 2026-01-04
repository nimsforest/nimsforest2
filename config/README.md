# Configuration

This file declares what components exist in your forest.

---

## forest.yaml

### TreeHouses

```yaml
treehouses:
  scoring:                        # Name of this TreeHouse
    subscribes: contact.created   # NATS subject to listen to
    publishes: lead.scored        # NATS subject to publish to
    script: scripts/treehouses/scoring.lua  # Lua script path
```

- **subscribes**: When an event hits this subject, the TreeHouse wakes up
- **publishes**: After processing, result goes here
- **script**: Lua file with `process(input)` function

### Nims

```yaml
nims:
  qualify:                        # Name of this Nim
    subscribes: lead.scored       # NATS subject to listen to
    publishes: lead.qualified     # NATS subject to publish to
    prompt: scripts/nims/qualify.md  # Path to prompt file
```

- **subscribes**: When an event hits this subject, the Nim wakes up
- **publishes**: After LLM responds, result goes here
- **prompt**: Path to `.md` file containing the prompt template

---

## Template Syntax

Prompts use Go templates:

| Syntax | What |
|--------|------|
| `{{.field}}` | Insert field value |
| `{{.nested.field}}` | Nested field |
| `{{range .items}}...{{end}}` | Loop |
| `{{if .field}}...{{end}}` | Conditional |

Example incoming event:
```json
{"contact_id": "c123", "score": 85, "signals": ["enterprise", "executive"]}
```

Template:
```
Score: {{.score}}
Signals: {{.signals}}
```

Result:
```
Score: 85
Signals: [enterprise executive]
```
