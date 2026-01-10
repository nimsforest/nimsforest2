package leaves

import "time"

// ChatMessage represents an incoming chat message from any platform.
// This is a platform-agnostic type - the Platform and ChatID fields
// are used for routing responses back through the appropriate songbird.
type ChatMessage struct {
	Platform  string            `json:"platform"` // "telegram", "slack", "whatsapp"
	ChatID    string            `json:"chat_id"`  // Platform-specific chat identifier
	UserID    string            `json:"user_id"`  // Platform-specific user identifier
	Username  string            `json:"username"` // Display name
	Text      string            `json:"text"`     // Message content
	Mentions  []string          `json:"mentions"` // @mentions in message
	ReplyTo   string            `json:"reply_to"` // Message ID being replied to (optional)
	Metadata  map[string]string `json:"metadata"` // Platform-specific extras
	Timestamp time.Time         `json:"timestamp"`
}

// Chirp represents an outgoing chat message to be sent through a songbird.
// The Platform and ChatID are used by the matching songbird to deliver
// the message to the correct destination.
type Chirp struct {
	Platform string `json:"platform"` // "telegram", "slack", "whatsapp"
	ChatID   string `json:"chat_id"`  // Where to send the message
	Text     string `json:"text"`     // Message content to send
	ReplyTo  string `json:"reply_to"` // Optional: reply to specific message
}
