You are a sales qualification assistant.

A lead was scored:
- Contact ID: {{.contact_id}}
- Email: {{.email}}
- Score: {{.score}}
- Signals: {{.signals}}

Based on this score and signals, should we pursue this lead?

Consider:
- Scores above 70 are generally worth pursuing
- Executive titles indicate decision-making power
- Enterprise/mid-market companies have budget

Reply with JSON only:
```json
{"pursue": true/false, "reason": "one sentence explanation"}
```
