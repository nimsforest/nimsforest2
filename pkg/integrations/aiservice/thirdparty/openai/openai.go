package openai

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

// OpenAIService implements the AIService interface for OpenAI models.
type OpenAIService struct {
	config      aifactory.Config
	initialized bool
	client      *http.Client
}

// Ensure OpenAIService implements the public interface
var _ pkgaiservice.AIService = (*OpenAIService)(nil)

// init registers the internal constructor with the public registry.
func init() {
	aifactory.RegisterService(aifactory.ServiceTypeOpenAI, NewOpenAIService)
}

// NewOpenAIService is the exported constructor.
func NewOpenAIService(apiKey, model string) (pkgaiservice.AIService, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}
	if model == "" {
		return nil, fmt.Errorf("OpenAI model name is required")
	}

	config := aifactory.Config{
		APIKey:     apiKey,
		APIBaseURL: "https://api.openai.com/v1/chat/completions", // OpenAI specific URL
		Model:      model,
	}

	return &OpenAIService{
		config: config,
		client: &http.Client{},
	}, nil
}

// Initialize is part of the AIService interface.
func (s *OpenAIService) Initialize(ctx context.Context) error {
	if s.config.APIKey == "" {
		return errors.New("OpenAIService not properly configured: missing API Key")
	}
	s.initialized = true
	return nil
}

// Close is part of the AIService interface.
func (s *OpenAIService) Close(ctx context.Context) error {
	if !s.initialized {
		return nil // Idempotent
	}
	s.initialized = false
	return nil
}

// OpenAIAPIRequest represents the structure for an OpenAI Chat Completions API request.
type OpenAIAPIRequest struct {
	Model    string              `json:"model"`
	Messages []aifactory.Message `json:"messages"`
	// Add other parameters like temperature, max_tokens, stream if needed
}

// OpenAIAPIResponse represents the structure of the OpenAI Chat Completions API response.
type OpenAIAPIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	// Add other fields like Usage if needed
}

// Ask implements the AIService interface.
func (s *OpenAIService) Ask(ctx context.Context, question string) (string, error) {
	if !s.initialized {
		return "", errors.New("OpenAIService is not initialized")
	}

	messages := []aifactory.Message{
		{
			Role:    "user",
			Content: question,
		},
	}

	// Construct the request body specific to OpenAI
	requestPayload := OpenAIAPIRequest{
		Model:    s.config.Model,
		Messages: messages,
	}

	jsonBody, err := json.Marshal(requestPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal OpenAI request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.config.APIBaseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create OpenAI request: %w", err)
	}

	// Set OpenAI-specific headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.APIKey))

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make OpenAI API request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read OpenAI response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response OpenAIAPIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal OpenAI response: %w", err)
	}

	if len(response.Choices) == 0 || response.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("no response content found in OpenAI response")
	}

	return response.Choices[0].Message.Content, nil
}

// Analyze implements the AIService interface.
func (s *OpenAIService) Analyze(ctx context.Context, content string) (map[string]interface{}, error) {
	if !s.initialized {
		return nil, errors.New("OpenAIService is not initialized")
	}
	return nil, errors.New("Analyze method not implemented for OpenAIService")
}
