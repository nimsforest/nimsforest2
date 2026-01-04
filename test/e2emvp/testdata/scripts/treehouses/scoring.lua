-- Test scoring script (same as main example)

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
