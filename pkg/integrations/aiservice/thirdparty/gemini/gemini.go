package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	pkgaiservice "github.com/yourusername/nimsforest/pkg/infrastructure/aiservice"
	aifactory "github.com/yourusername/nimsforest/pkg/integrations/aiservice"
)

// GeminiService implements the AIService interface for Google's Gemini models.
type GeminiService struct {
	config      aifactory.Config
	initialized bool
	client      *http.Client
}

// Ensure GeminiService implements the public interface
var _ pkgaiservice.AIService = (*GeminiService)(nil)

// init registers the internal constructor with the public registry.
func init() {
	aifactory.RegisterService(aifactory.ServiceTypeGemini, NewGeminiService)
}

// NewGeminiService is the exported constructor.
func NewGeminiService(apiKey, model string) (pkgaiservice.AIService, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("gemini API key is required")
	}
	if model == "" {
		return nil, fmt.Errorf("gemini model name is required")
	}

	// Construct the Gemini-specific API endpoint URL
	// Note: The API key is typically appended as a query parameter for Gemini REST API
	endpointURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", model)

	config := aifactory.Config{
		APIKey:     apiKey,
		APIBaseURL: endpointURL,
		Model:      model,
	}

	return &GeminiService{
		config: config,
		client: &http.Client{},
	}, nil
}

// Initialize is part of the AIService interface.
func (s *GeminiService) Initialize(ctx context.Context) error {
	if s.config.APIKey == "" {
		return errors.New("GeminiService not properly configured: missing API Key")
	}
	s.initialized = true
	return nil
}

// Close is part of the AIService interface.
func (s *GeminiService) Close(ctx context.Context) error {
	if !s.initialized {
		return nil // Idempotent
	}
	s.initialized = false
	return nil
}

// GeminiAPIRequestPart represents a part of the Gemini request content.
type GeminiAPIRequestPart struct {
	Text string `json:"text"`
}

// GeminiAPIRequestContent represents the content structure for a Gemini API request.
type GeminiAPIRequestContent struct {
	Parts []GeminiAPIRequestPart `json:"parts"`
	// Role string `json:"role,omitempty"` // Optional: Gemini API infers role
}

// GeminiAPIRequest represents the overall structure for a Gemini API request.
type GeminiAPIRequest struct {
	Contents []GeminiAPIRequestContent `json:"contents"`
	// Add other parameters like generationConfig, safetySettings if needed
}

// GeminiAPIResponseCandidateContentPart represents a part of the response content.
type GeminiAPIResponseCandidateContentPart struct {
	Text string `json:"text"`
}

// GeminiAPIResponseCandidateContent represents the content structure in the response.
type GeminiAPIResponseCandidateContent struct {
	Parts []GeminiAPIResponseCandidateContentPart `json:"parts"`
	Role  string                                  `json:"role"`
}

// GeminiAPIResponseCandidate represents a single candidate response.
type GeminiAPIResponseCandidate struct {
	Content GeminiAPIResponseCandidateContent `json:"content"`
	// Add other fields like finishReason, safetyRatings if needed
}

// GeminiAPIResponse represents the overall structure of the Gemini API response.
type GeminiAPIResponse struct {
	Candidates []GeminiAPIResponseCandidate `json:"candidates"`
	// Add promptFeedback if needed
}

// Ask implements the AIService interface.
func (s *GeminiService) Ask(ctx context.Context, question string) (string, error) {
	if !s.initialized {
		return "", errors.New("GeminiService is not initialized")
	}

	// Construct the request body specific to Gemini
	requestPayload := GeminiAPIRequest{
		Contents: []GeminiAPIRequestContent{
			{
				Parts: []GeminiAPIRequestPart{
					{Text: question},
				},
				// Role: "user", // Often inferred
			},
		},
	}

	jsonBody, err := json.Marshal(requestPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Gemini request body: %w", err)
	}

	// Append API key to URL for Gemini REST API
	urlWithKey := s.config.APIBaseURL
	if strings.Contains(urlWithKey, "?") {
		urlWithKey += "&key=" + s.config.APIKey
	} else {
		urlWithKey += "?key=" + s.config.APIKey
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlWithKey, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create Gemini request: %w", err)
	}

	// Set Gemini-specific headers (usually just Content-Type)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make Gemini API request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Gemini response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Attempt to parse standard Google API error format
		var googleErr struct {
			Error struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
				Status  string `json:"status"`
			} `json:"error"`
		}
		if parseErr := json.Unmarshal(body, &googleErr); parseErr == nil && googleErr.Error.Message != "" {
			return "", fmt.Errorf("gemini API request failed with status %d: %s", resp.StatusCode, googleErr.Error.Message)
		}
		// Fallback to raw body
		return "", fmt.Errorf("gemini API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response GeminiAPIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal Gemini response: %w", err)
	}

	if len(response.Candidates) == 0 || len(response.Candidates[0].Content.Parts) == 0 || response.Candidates[0].Content.Parts[0].Text == "" {
		return "", fmt.Errorf("no response content found in Gemini response")
	}

	// Assuming the first part of the first candidate's content is the answer
	return response.Candidates[0].Content.Parts[0].Text, nil
}

// Analyze implements the AIService interface.
func (s *GeminiService) Analyze(ctx context.Context, content string) (map[string]interface{}, error) {
	if !s.initialized {
		return nil, errors.New("GeminiService is not initialized")
	}
	return nil, errors.New("Analyze method not implemented for GeminiService")
}
