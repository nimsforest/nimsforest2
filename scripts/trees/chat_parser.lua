-- chat_parser.lua
-- Tree script that parses incoming chat webhooks into platform-agnostic ChatMessage format.
-- Supports: Telegram (more platforms can be added)
--
-- Input: TelegramPayload table (already decoded by tree runtime)
--   - body: raw Telegram update (table or JSON string)
--   - timestamp: when received
--   - source: source name

function process(payload)
    -- Get the raw webhook body
    local body = payload.body

    -- If body is a string, decode it
    if type(body) == "string" then
        body = json.decode(body)
    end

    if not body then
        return nil -- skip, no body
    end

    -- Detect platform and parse accordingly
    local chat_msg = nil

    -- Telegram detection: has update_id and message
    if body.update_id and body.message then
        chat_msg = parse_telegram(body)
    else
        return nil -- unknown platform
    end

    return chat_msg
end

-- Parse Telegram webhook update into ChatMessage
function parse_telegram(update)
    local msg = update.message
    if not msg then
        return nil
    end

    -- Extract user info
    local user_id = ""
    local username = ""
    if msg.from then
        user_id = tostring(msg.from.id)
        username = msg.from.username or msg.from.first_name or ""
    end

    -- Extract chat info
    local chat_id = ""
    if msg.chat then
        chat_id = tostring(msg.chat.id)
    end

    -- Extract mentions from entities
    local mentions = {}
    if msg.entities then
        for _, entity in ipairs(msg.entities) do
            if entity.type == "mention" or entity.type == "text_mention" then
                local mention = string.sub(msg.text or "", entity.offset + 1, entity.offset + entity.length)
                table.insert(mentions, mention)
            end
        end
    end

    -- Extract reply_to message ID if present
    local reply_to = ""
    if msg.reply_to_message then
        reply_to = tostring(msg.reply_to_message.message_id)
    end

    -- Build metadata
    local metadata = {
        message_id = tostring(msg.message_id),
        chat_type = msg.chat and msg.chat.type or "unknown"
    }
    if msg.chat and msg.chat.title then
        metadata.chat_title = msg.chat.title
    end

    -- Build ChatMessage
    return {
        platform = "telegram",
        chat_id = chat_id,
        user_id = user_id,
        username = username,
        text = msg.text or "",
        mentions = mentions,
        reply_to = reply_to,
        metadata = metadata,
        timestamp = os.date("!%Y-%m-%dT%H:%M:%SZ", msg.date or os.time())
    }
end
