package aiservice

// Message represents a message in the AI conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Config holds configuration for AI services
type Config struct {
	APIKey     string
	APIBaseURL string
	Model      string
}
