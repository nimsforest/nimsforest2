package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	pkgaiservice "github.com/yourusername/nimsforest/pkg/infrastructure/aiservice"
	aifactory "github.com/yourusername/nimsforest/pkg/integrations/aiservice"
)

// ClaudeService implements the AIService interface for Anthropic's Claude models.
type ClaudeService struct {
	config      aifactory.Config // Use the Config from integrations/aiservice
	initialized bool
	client      *http.Client // Add http client
	model       string
}

// Ensure ClaudeService implements the public interface
var _ pkgaiservice.AIService = (*ClaudeService)(nil)

// DefaultClaudeModel is the default model to use if not specified
const DefaultClaudeModel = "claude-3-haiku-20240307"

// init registers the internal constructor with the public registry.
func init() {
	// Use the factory package for registration and service type
	aifactory.RegisterService(aifactory.ServiceTypeClaude, NewClaudeService)
}

// newClaudeService creates a new instance from a config (used for testing)
func newClaudeService(config aifactory.Config) (*ClaudeService, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Claude API key is required")
	}

	model := config.Model
	if model == "" {
		model = DefaultClaudeModel // Use default model if not specified
	}

	return &ClaudeService{
		config: config,
		client: &http.Client{}, // Initialize client
		model:  model,
	}, nil
}

// NewClaudeService is the exported constructor.
func NewClaudeService(apiKey, model string) (pkgaiservice.AIService, error) {
	// Convert params to config and delegate to internal constructor
	config := aifactory.Config{
		APIKey:     apiKey,
		APIBaseURL: "https://api.anthropic.com/v1/messages", // Claude specific URL
		Model:      model,
	}

	return newClaudeService(config)
}

// Initialize is part of the AIService interface.
func (s *ClaudeService) Initialize(ctx context.Context) error {
	// Basic initialization check, could add a test ping to API if needed
	if s.config.APIKey == "" {
		return errors.New("ClaudeService not properly configured: missing API Key")
	}
	s.initialized = true
	return nil
}

// Close is part of the AIService interface.
func (s *ClaudeService) Close(ctx context.Context) error {
	if !s.initialized {
		return nil // Idempotent
	}
	s.initialized = false
	return nil
}

// ClaudeAPIRequest represents the structure for a Claude API request.
// Moved here from the old types package structure for clarity.
type ClaudeAPIRequest struct {
	Model     string              `json:"model"`
	Messages  []aifactory.Message `json:"messages"`
	MaxTokens int                 `json:"max_tokens"`
	// Add other parameters like temperature, stream if needed
}

// ClaudeAPIResponse represents the structure of the Claude API response.
// Moved here from the old types package structure for clarity.
type ClaudeAPIResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	// Add other fields like Usage, StopReason if needed
}

// Ask implements the AIService interface.
func (s *ClaudeService) Ask(ctx context.Context, question string) (string, error) {
	if !s.initialized {
		return "", errors.New("ClaudeService is not initialized")
	}

	// Use aifactory.Message
	messages := []aifactory.Message{
		{
			Role:    "user",
			Content: question,
		},
	}

	// Construct the request body specific to Claude
	requestPayload := ClaudeAPIRequest{
		Model:     s.config.Model,
		Messages:  messages,
		MaxTokens: 1000, // Example value, make configurable if needed
	}

	jsonBody, err := json.Marshal(requestPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Claude request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.config.APIBaseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create Claude request: %w", err)
	}

	// Set Claude-specific headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01") // Use appropriate version

	// Make the request using the service's client
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make Claude API request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Claude response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Claude API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response ClaudeAPIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal Claude response: %w", err)
	}

	if len(response.Content) == 0 {
		// Consider checking response.StopReason or other fields for more context
		return "", fmt.Errorf("no response content found in Claude response")
	}

	return response.Content[0].Text, nil
}

// Analyze implements the AIService interface.
func (s *ClaudeService) Analyze(ctx context.Context, content string) (map[string]interface{}, error) {
	if !s.initialized {
		return nil, errors.New("ClaudeService is not initialized")
	}
	// Keep Analyze unimplemented for now
	return nil, errors.New("Analyze method not implemented for ClaudeService")
}
