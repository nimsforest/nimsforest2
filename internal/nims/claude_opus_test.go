package nims

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/yourusername/nimsforest/internal/core"
	"github.com/yourusername/nimsforest/internal/leaves"
)

// MockAIService is a mock implementation of AIService for testing.
type MockAIService struct {
	initialized   bool
	response      string
	err           error
	askCallCount  int
	lastQuestion  string
}

func NewMockAIService(response string, err error) *MockAIService {
	return &MockAIService{
		response: response,
		err:      err,
	}
}

func (m *MockAIService) Initialize(ctx context.Context) error {
	m.initialized = true
	return nil
}

func (m *MockAIService) Close(ctx context.Context) error {
	m.initialized = false
	return nil
}

func (m *MockAIService) Ask(ctx context.Context, question string) (string, error) {
	m.askCallCount++
	m.lastQuestion = question
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func (m *MockAIService) Analyze(ctx context.Context, content string) (map[string]interface{}, error) {
	return nil, nil
}

func TestClaudeOpusNim_Subjects(t *testing.T) {
	mockService := NewMockAIService("test response", nil)
	nim := NewClaudeOpusNimWithService(nil, nil, nil, mockService)
	subjects := nim.Subjects()

	if len(subjects) != 2 {
		t.Errorf("Expected 2 subjects, got %d", len(subjects))
	}

	expected := map[string]bool{
		"ai.request":      true,
		"claude.opus.ask": true,
	}

	for _, subject := range subjects {
		if !expected[subject] {
			t.Errorf("Unexpected subject: %s", subject)
		}
	}
}

func TestClaudeOpusNim_GetModel(t *testing.T) {
	mockService := NewMockAIService("test response", nil)
	nim := NewClaudeOpusNimWithService(nil, nil, nil, mockService)

	model := nim.GetModel()
	if model != ClaudeOpusModel {
		t.Errorf("Expected model %s, got %s", ClaudeOpusModel, model)
	}
}

func TestClaudeOpusNim_HandleAIRequest(t *testing.T) {
	nc, js := core.SetupTestNATS(t)
	defer nc.Close()

	wind := core.NewWind(nc)
	humus, err := core.NewHumus(js)
	if err != nil {
		t.Fatalf("Failed to create humus: %v", err)
	}

	soil, err := core.NewSoil(js)
	if err != nil {
		t.Fatalf("Failed to create soil: %v", err)
	}

	mockService := NewMockAIService("This is a test response from Claude Opus", nil)
	nim := NewClaudeOpusNimWithService(wind, humus, soil, mockService)

	// Create an AI request leaf
	request := leaves.AIRequest{
		RequestID: "test-req-123",
		Prompt:    "What is the meaning of life?",
		Timestamp: time.Now(),
	}

	requestData, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	leaf := core.NewLeaf("ai.request", requestData, "test")

	// Start the nim first to initialize the AI service
	ctx := context.Background()
	if err := nim.Start(ctx); err != nil {
		t.Fatalf("Failed to start nim: %v", err)
	}
	defer nim.Stop()

	// Subscribe to AI responses
	responseReceived := make(chan leaves.AIResponse, 1)
	_, err = wind.Catch("ai.response", func(leaf core.Leaf) {
		var response leaves.AIResponse
		if err := json.Unmarshal(leaf.Data, &response); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
			return
		}
		responseReceived <- response
	})
	if err != nil {
		t.Fatalf("Failed to catch ai.response: %v", err)
	}

	// Handle the leaf directly
	if err := nim.Handle(ctx, *leaf); err != nil {
		t.Fatalf("Failed to handle AI request: %v", err)
	}

	// Wait for response
	select {
	case response := <-responseReceived:
		if response.RequestID != "test-req-123" {
			t.Errorf("Expected request ID 'test-req-123', got '%s'", response.RequestID)
		}
		if !response.Success {
			t.Errorf("Expected success=true, got false. Error: %s", response.Error)
		}
		if response.Response != "This is a test response from Claude Opus" {
			t.Errorf("Unexpected response: %s", response.Response)
		}
		if response.Model != ClaudeOpusModel {
			t.Errorf("Expected model %s, got %s", ClaudeOpusModel, response.Model)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for AI response")
	}

	// Verify the mock was called correctly
	if mockService.askCallCount != 1 {
		t.Errorf("Expected Ask to be called 1 time, got %d", mockService.askCallCount)
	}
	if mockService.lastQuestion != "What is the meaning of life?" {
		t.Errorf("Expected question 'What is the meaning of life?', got '%s'", mockService.lastQuestion)
	}
}

func TestClaudeOpusNim_HandleWithSystemPrompt(t *testing.T) {
	nc, js := core.SetupTestNATS(t)
	defer nc.Close()

	wind := core.NewWind(nc)
	humus, err := core.NewHumus(js)
	if err != nil {
		t.Fatalf("Failed to create humus: %v", err)
	}

	soil, err := core.NewSoil(js)
	if err != nil {
		t.Fatalf("Failed to create soil: %v", err)
	}

	mockService := NewMockAIService("Response with system context", nil)
	nim := NewClaudeOpusNimWithService(wind, humus, soil, mockService)

	// Create an AI request with system prompt
	request := leaves.AIRequest{
		RequestID:    "test-req-sys",
		Prompt:       "Hello",
		SystemPrompt: "You are a helpful assistant",
		Timestamp:    time.Now(),
	}

	requestData, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	leaf := core.NewLeaf("ai.request", requestData, "test")

	// Start the nim
	ctx := context.Background()
	if err := nim.Start(ctx); err != nil {
		t.Fatalf("Failed to start nim: %v", err)
	}
	defer nim.Stop()

	// Subscribe to responses
	responseReceived := make(chan leaves.AIResponse, 1)
	_, err = wind.Catch("ai.response", func(leaf core.Leaf) {
		var response leaves.AIResponse
		json.Unmarshal(leaf.Data, &response)
		responseReceived <- response
	})
	if err != nil {
		t.Fatalf("Failed to catch ai.response: %v", err)
	}

	// Handle the leaf
	if err := nim.Handle(ctx, *leaf); err != nil {
		t.Fatalf("Failed to handle AI request: %v", err)
	}

	// Wait for response and verify system prompt was included
	select {
	case <-responseReceived:
		expectedPrompt := "System: You are a helpful assistant\n\nUser: Hello"
		if mockService.lastQuestion != expectedPrompt {
			t.Errorf("Expected prompt with system context:\n%s\n\nGot:\n%s", expectedPrompt, mockService.lastQuestion)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for AI response")
	}
}

func TestClaudeOpusNim_HandleInvalidJSON(t *testing.T) {
	nc, js := core.SetupTestNATS(t)
	defer nc.Close()

	wind := core.NewWind(nc)
	humus, err := core.NewHumus(js)
	if err != nil {
		t.Fatalf("Failed to create humus: %v", err)
	}

	soil, err := core.NewSoil(js)
	if err != nil {
		t.Fatalf("Failed to create soil: %v", err)
	}

	mockService := NewMockAIService("test", nil)
	nim := NewClaudeOpusNimWithService(wind, humus, soil, mockService)

	leaf := core.NewLeaf("ai.request", []byte("invalid json"), "test")

	ctx := context.Background()
	if err := nim.Start(ctx); err != nil {
		t.Fatalf("Failed to start nim: %v", err)
	}
	defer nim.Stop()

	// Subscribe to error responses
	responseReceived := make(chan leaves.AIResponse, 1)
	_, err = wind.Catch("ai.response", func(leaf core.Leaf) {
		var response leaves.AIResponse
		json.Unmarshal(leaf.Data, &response)
		responseReceived <- response
	})
	if err != nil {
		t.Fatalf("Failed to catch ai.response: %v", err)
	}

	// Handle should emit an error response
	nim.Handle(ctx, *leaf)

	select {
	case response := <-responseReceived:
		if response.Success {
			t.Error("Expected success=false for invalid JSON")
		}
		if response.Error == "" {
			t.Error("Expected error message for invalid JSON")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for error response")
	}
}

func TestClaudeOpusNim_HandleAIServiceError(t *testing.T) {
	nc, js := core.SetupTestNATS(t)
	defer nc.Close()

	wind := core.NewWind(nc)
	humus, err := core.NewHumus(js)
	if err != nil {
		t.Fatalf("Failed to create humus: %v", err)
	}

	soil, err := core.NewSoil(js)
	if err != nil {
		t.Fatalf("Failed to create soil: %v", err)
	}

	// Create mock that returns an error
	mockService := NewMockAIService("", fmt.Errorf("API rate limit exceeded"))
	nim := NewClaudeOpusNimWithService(wind, humus, soil, mockService)

	request := leaves.AIRequest{
		RequestID: "test-error-req",
		Prompt:    "Test prompt",
		Timestamp: time.Now(),
	}

	requestData, _ := json.Marshal(request)
	leaf := core.NewLeaf("ai.request", requestData, "test")

	ctx := context.Background()
	if err := nim.Start(ctx); err != nil {
		t.Fatalf("Failed to start nim: %v", err)
	}
	defer nim.Stop()

	// Subscribe to responses
	responseReceived := make(chan leaves.AIResponse, 1)
	_, err = wind.Catch("ai.response", func(leaf core.Leaf) {
		var response leaves.AIResponse
		json.Unmarshal(leaf.Data, &response)
		responseReceived <- response
	})
	if err != nil {
		t.Fatalf("Failed to catch ai.response: %v", err)
	}

	// Handle the leaf
	nim.Handle(ctx, *leaf)

	select {
	case response := <-responseReceived:
		if response.Success {
			t.Error("Expected success=false for API error")
		}
		if response.Error != "API rate limit exceeded" {
			t.Errorf("Expected error 'API rate limit exceeded', got '%s'", response.Error)
		}
		if response.RequestID != "test-error-req" {
			t.Errorf("Expected request ID 'test-error-req', got '%s'", response.RequestID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for error response")
	}
}

func TestClaudeOpusNim_DirectAsk(t *testing.T) {
	mockService := NewMockAIService("Direct response", nil)
	nim := NewClaudeOpusNimWithService(nil, nil, nil, mockService)

	// Initialize the service
	ctx := context.Background()
	mockService.Initialize(ctx)

	response, err := nim.Ask(ctx, "Direct question")
	if err != nil {
		t.Fatalf("Direct Ask failed: %v", err)
	}

	if response != "Direct response" {
		t.Errorf("Expected 'Direct response', got '%s'", response)
	}

	if mockService.lastQuestion != "Direct question" {
		t.Errorf("Expected question 'Direct question', got '%s'", mockService.lastQuestion)
	}
}

func TestClaudeOpusNim_MetadataPassthrough(t *testing.T) {
	nc, js := core.SetupTestNATS(t)
	defer nc.Close()

	wind := core.NewWind(nc)
	humus, err := core.NewHumus(js)
	if err != nil {
		t.Fatalf("Failed to create humus: %v", err)
	}

	soil, err := core.NewSoil(js)
	if err != nil {
		t.Fatalf("Failed to create soil: %v", err)
	}

	mockService := NewMockAIService("Response with metadata", nil)
	nim := NewClaudeOpusNimWithService(wind, humus, soil, mockService)

	// Create request with metadata
	request := leaves.AIRequest{
		RequestID: "test-meta-req",
		Prompt:    "Test prompt",
		Metadata: map[string]string{
			"user_id":    "user123",
			"session_id": "session456",
		},
		Timestamp: time.Now(),
	}

	requestData, _ := json.Marshal(request)
	leaf := core.NewLeaf("ai.request", requestData, "test")

	ctx := context.Background()
	if err := nim.Start(ctx); err != nil {
		t.Fatalf("Failed to start nim: %v", err)
	}
	defer nim.Stop()

	// Subscribe to responses
	responseReceived := make(chan leaves.AIResponse, 1)
	_, err = wind.Catch("ai.response", func(leaf core.Leaf) {
		var response leaves.AIResponse
		json.Unmarshal(leaf.Data, &response)
		responseReceived <- response
	})
	if err != nil {
		t.Fatalf("Failed to catch ai.response: %v", err)
	}

	// Handle the leaf
	nim.Handle(ctx, *leaf)

	select {
	case response := <-responseReceived:
		if response.Metadata["user_id"] != "user123" {
			t.Errorf("Expected user_id 'user123', got '%s'", response.Metadata["user_id"])
		}
		if response.Metadata["session_id"] != "session456" {
			t.Errorf("Expected session_id 'session456', got '%s'", response.Metadata["session_id"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for response")
	}
}

// Integration test: Full flow with nim catching leaves
func TestClaudeOpusNim_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	nc, js := core.SetupTestNATS(t)
	defer nc.Close()

	wind := core.NewWind(nc)
	humus, err := core.NewHumus(js)
	if err != nil {
		t.Fatalf("Failed to create humus: %v", err)
	}

	soil, err := core.NewSoil(js)
	if err != nil {
		t.Fatalf("Failed to create soil: %v", err)
	}

	mockService := NewMockAIService("Integration test response", nil)
	nim := NewClaudeOpusNimWithService(wind, humus, soil, mockService)

	// Subscribe to AI responses before starting nim
	responseReceived := make(chan leaves.AIResponse, 1)
	_, err = wind.Catch("ai.response", func(leaf core.Leaf) {
		var response leaves.AIResponse
		if err := json.Unmarshal(leaf.Data, &response); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
			return
		}
		responseReceived <- response
	})
	if err != nil {
		t.Fatalf("Failed to catch ai.response: %v", err)
	}

	// Start the nim
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := nim.Start(ctx); err != nil {
		t.Fatalf("Failed to start nim: %v", err)
	}
	defer nim.Stop()

	time.Sleep(100 * time.Millisecond)

	// Drop an AI request leaf
	request := leaves.AIRequest{
		RequestID: "integration-test-req",
		Prompt:    "What is 2+2?",
		Timestamp: time.Now(),
	}

	requestData, _ := json.Marshal(request)
	if err := wind.Drop(*core.NewLeaf("ai.request", requestData, "integration-test")); err != nil {
		t.Fatalf("Failed to drop AI request leaf: %v", err)
	}

	// Wait for response
	select {
	case response := <-responseReceived:
		if response.RequestID != "integration-test-req" {
			t.Errorf("Expected request ID 'integration-test-req', got '%s'", response.RequestID)
		}
		if !response.Success {
			t.Errorf("Expected success, got error: %s", response.Error)
		}
		if response.Response != "Integration test response" {
			t.Errorf("Expected 'Integration test response', got '%s'", response.Response)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for AI response")
	}
}
