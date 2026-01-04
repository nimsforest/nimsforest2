# TreeHouse Scripts

Lua scripts for deterministic business rules.

---

## How It Works

Each script must have a `process` function:

```lua
function process(input)
    -- your logic here
    return output
end
```

- **input**: Table with event data (from NATS message)
- **output**: Table that gets published to next subject

---

## scoring.lua

Scores incoming contacts based on company size, title, and industry.

### Input

```json
{
  "id": "c123",
  "email": "jane@acme.com",
  "title": "VP Engineering",
  "company_size": 250,
  "industry": "technology"
}
```

### Logic

| Signal | Condition | Points |
|--------|-----------|--------|
| enterprise | company_size > 500 | +50 |
| mid_market | company_size > 100 | +30 |
| smb | company_size > 20 | +10 |
| executive | title contains CEO/CTO/VP | +40 |
| manager | title contains Director/Manager | +20 |
| target_industry | industry is technology/finance | +15 |

### Output

```json
{
  "contact_id": "c123",
  "email": "jane@acme.com",
  "score": 85,
  "signals": ["mid_market", "executive", "target_industry"]
}
```

---

## Available Helpers

| Function | What | Example |
|----------|------|---------|
| `contains(str, sub)` | Check if string contains substring | `contains(contact.title, "VP")` |
| `json.encode(t)` | Table to JSON string | `json.encode({a=1})` → `{"a":1}` |
| `json.decode(s)` | JSON string to table | `json.decode('{"a":1}')` → `{a=1}` |
| `log(msg)` | Print to logs | `log("score: " .. score)` |

---

## Tips

1. **Keep it simple.** Complex logic = hard to debug.
2. **Be explicit.** Name your signals clearly.
3. **Test locally.** Run Lua standalone first.
4. **Same input = same output.** No randomness, no external calls.
