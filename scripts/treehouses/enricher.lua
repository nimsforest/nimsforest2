-- enricher.lua
-- TreeHouse script that enriches lead data
-- Subscribes to lead.scored, publishes lead.enriched

function process(leaf_json)
    local leaf = json.decode(leaf_json)
    if not leaf then
        return nil, "failed to parse leaf"
    end
    
    local data = leaf.data or {}
    
    -- Add enrichment data
    local enriched = {
        original = data,
        enriched_at = os.date("!%Y-%m-%dT%H:%M:%SZ"),
        tier = "unknown"
    }
    
    -- Determine tier based on score
    local score = data.score or 0
    if score >= 80 then
        enriched.tier = "enterprise"
        enriched.priority = "high"
    elseif score >= 50 then
        enriched.tier = "mid-market"
        enriched.priority = "medium"
    else
        enriched.tier = "startup"
        enriched.priority = "low"
    end
    
    return json.encode(enriched)
end
