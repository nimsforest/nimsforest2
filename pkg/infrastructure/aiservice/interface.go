package aiservice

import (
	"context"
)

// AIService defines a standard interface for interacting with various AI services,
// including language models and potentially other types of AI.
type AIService interface {
	// Initialize sets up the service, potentially authenticating or loading resources.
	Initialize(ctx context.Context) error

	// Close releases any resources held by the service.
	Close(ctx context.Context) error

	// Ask sends a question or prompt to the AI service and expects a textual response.
	// This is typical for LLM interactions.
	Ask(ctx context.Context, question string) (string, error)

	// Analyze processes the given content and returns structured insights.
	// The specific structure of the insights map can vary by implementation.
	Analyze(ctx context.Context, content string) (map[string]interface{}, error)
}
