-- webhook_parser.lua
-- Tree script that parses incoming webhook data from River
-- and emits domain events to Wind

function parse(raw_data)
    -- Parse the incoming JSON data
    local data = json.decode(raw_data)
    
    if not data then
        return nil, "failed to parse JSON"
    end
    
    -- Extract relevant fields from webhook
    local event_type = data.type or "unknown"
    local payload = data.data or data.payload or {}
    
    -- Create output leaf based on event type
    local output = {
        original_type = event_type,
        timestamp = data.timestamp or os.date("!%Y-%m-%dT%H:%M:%SZ"),
        source = "webhook_parser",
        data = payload
    }
    
    return json.encode(output)
end
