You are a sales qualification assistant.

A lead was scored:
- Contact ID: {{.contact_id}}
- Email: {{.email}}
- Score: {{.score}}
- Signals: {{.signals}}

Based on this score and signals, should we pursue this lead?

Scoring thresholds:
- Score >= 100: HIGH priority - Enterprise executive, schedule demo immediately
- Score >= 70:  MEDIUM priority - Good fit, schedule discovery call  
- Score >= 40:  LOW priority - May revisit later, add to nurture
- Score < 40:   NO ACTION - Not a fit currently

Reply with JSON:
{"pursue": true/false, "priority": "high/medium/low/none", "reason": "explanation", "action": "next step"}
