package leaves

import "time"

// AIRequest represents a request for AI processing.
// This leaf is caught by AI wrapper nims (like ClaudeOpusNim) for processing.
type AIRequest struct {
	RequestID   string            `json:"request_id"`            // Unique identifier for this request
	Prompt      string            `json:"prompt"`                // The prompt/question to process
	SystemPrompt string           `json:"system_prompt,omitempty"` // Optional system prompt for context
	MaxTokens   int               `json:"max_tokens,omitempty"`  // Optional max tokens for response
	Temperature float64           `json:"temperature,omitempty"` // Optional temperature (0-1)
	Metadata    map[string]string `json:"metadata,omitempty"`    // Optional metadata to pass through
	Timestamp   time.Time         `json:"timestamp"`
}

// AIResponse represents a response from AI processing.
// This leaf is emitted by AI wrapper nims after processing a request.
type AIResponse struct {
	RequestID   string            `json:"request_id"`           // Matches the original request
	Response    string            `json:"response"`             // The AI-generated response
	Model       string            `json:"model"`                // The model that generated the response
	TokensUsed  int               `json:"tokens_used,omitempty"` // Optional token count
	Metadata    map[string]string `json:"metadata,omitempty"`   // Passed through from request
	Error       string            `json:"error,omitempty"`      // Error message if processing failed
	Success     bool              `json:"success"`              // Whether the request was successful
	Timestamp   time.Time         `json:"timestamp"`
}
