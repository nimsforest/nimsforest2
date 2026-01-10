package nims

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/yourusername/nimsforest/internal/core"
	"github.com/yourusername/nimsforest/internal/leaves"
	pkgaiservice "github.com/yourusername/nimsforest/pkg/infrastructure/aiservice"
	aifactory "github.com/yourusername/nimsforest/pkg/integrations/aiservice"

	// Import claude package to trigger init() registration
	_ "github.com/yourusername/nimsforest/pkg/integrations/aiservice/thirdparty/claude"
)

// ClaudeOpusModel is the model identifier for Claude Opus
const ClaudeOpusModel = "claude-3-opus-20240229"

// ClaudeOpusNim is a nim that wraps Claude Opus for AI processing.
// It catches AI request leaves, processes them through Claude Opus,
// and emits AI response leaves.
type ClaudeOpusNim struct {
	*core.BaseNim
	ctx       context.Context
	cancel    context.CancelFunc
	aiService pkgaiservice.AIService
	apiKey    string
}

// NewClaudeOpusNim creates a new ClaudeOpusNim with the given API key.
func NewClaudeOpusNim(wind *core.Wind, humus *core.Humus, soil *core.Soil, apiKey string) (*ClaudeOpusNim, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("claude API key is required")
	}

	baseNim := core.NewBaseNim("claude-opus-nim", wind, humus, soil)

	// Create the Claude service using the factory
	service, err := aifactory.NewService(aifactory.ServiceTypeClaude, apiKey, ClaudeOpusModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create claude service: %w", err)
	}

	return &ClaudeOpusNim{
		BaseNim:   baseNim,
		aiService: service,
		apiKey:    apiKey,
	}, nil
}

// NewClaudeOpusNimWithService creates a ClaudeOpusNim with a custom AI service.
// This is useful for testing with mock services.
func NewClaudeOpusNimWithService(wind *core.Wind, humus *core.Humus, soil *core.Soil, service pkgaiservice.AIService) *ClaudeOpusNim {
	baseNim := core.NewBaseNim("claude-opus-nim", wind, humus, soil)
	return &ClaudeOpusNim{
		BaseNim:   baseNim,
		aiService: service,
	}
}

// Subjects returns the leaf subjects this nim catches.
func (n *ClaudeOpusNim) Subjects() []string {
	return []string{
		"ai.request",        // Generic AI request
		"claude.opus.ask",   // Direct Claude Opus request
	}
}

// Start begins listening for AI request leaves.
func (n *ClaudeOpusNim) Start(ctx context.Context) error {
	n.ctx, n.cancel = context.WithCancel(ctx)

	// Initialize the AI service
	if err := n.aiService.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize claude service: %w", err)
	}

	log.Printf("[ClaudeOpusNim] 🤖 Starting Claude Opus nim")
	log.Printf("[ClaudeOpusNim]     Model: %s", ClaudeOpusModel)
	log.Printf("[ClaudeOpusNim]     Catching: %v", n.Subjects())

	// Register handlers for each subject
	for _, subject := range n.Subjects() {
		if err := n.Catch(subject, func(leaf core.Leaf) {
			if err := n.Handle(n.ctx, leaf); err != nil {
				log.Printf("[ClaudeOpusNim] Error handling %s: %v", leaf.Subject, err)
			}
		}); err != nil {
			return fmt.Errorf("failed to catch %s: %w", subject, err)
		}
	}

	log.Printf("[ClaudeOpusNim] Started listening for AI requests")
	return nil
}

// Handle processes caught leaves and routes them to Claude Opus.
func (n *ClaudeOpusNim) Handle(ctx context.Context, leaf core.Leaf) error {
	log.Printf("[ClaudeOpusNim] 🍃 Caught leaf: %s from %s", leaf.Subject, leaf.Source)

	// Parse the AI request
	var request leaves.AIRequest
	if err := json.Unmarshal(leaf.Data, &request); err != nil {
		return n.emitErrorResponse("", fmt.Errorf("failed to parse AI request: %w", err), nil)
	}

	// Generate request ID if not provided
	if request.RequestID == "" {
		request.RequestID = fmt.Sprintf("req-%d", time.Now().UnixNano())
	}

	// Process the request through Claude Opus
	return n.processRequest(ctx, request)
}

// processRequest sends the prompt to Claude Opus and emits the response.
func (n *ClaudeOpusNim) processRequest(ctx context.Context, request leaves.AIRequest) error {
	log.Printf("[ClaudeOpusNim] 🧠 Processing request: %s", request.RequestID)

	// Build the prompt (include system prompt if provided)
	prompt := request.Prompt
	if request.SystemPrompt != "" {
		prompt = fmt.Sprintf("System: %s\n\nUser: %s", request.SystemPrompt, request.Prompt)
	}

	// Call Claude Opus
	startTime := time.Now()
	response, err := n.aiService.Ask(ctx, prompt)
	if err != nil {
		log.Printf("[ClaudeOpusNim] ❌ Claude Opus error: %v", err)
		return n.emitErrorResponse(request.RequestID, err, request.Metadata)
	}

	duration := time.Since(startTime)
	log.Printf("[ClaudeOpusNim] ✅ Got response in %v", duration)

	// Create and emit the response leaf
	aiResponse := leaves.AIResponse{
		RequestID: request.RequestID,
		Response:  response,
		Model:     ClaudeOpusModel,
		Metadata:  request.Metadata,
		Success:   true,
		Timestamp: time.Now(),
	}

	if err := n.LeafStruct("ai.response", aiResponse); err != nil {
		return fmt.Errorf("failed to emit AI response: %w", err)
	}

	log.Printf("[ClaudeOpusNim] 🍃 Emitted: ai.response for request %s", request.RequestID)

	// Optionally record the interaction in humus for tracking
	if n.GetHumus() != nil {
		record := map[string]interface{}{
			"request_id":    request.RequestID,
			"prompt_length": len(request.Prompt),
			"response_length": len(response),
			"model":         ClaudeOpusModel,
			"duration_ms":   duration.Milliseconds(),
			"timestamp":     time.Now().Format(time.RFC3339),
		}
		recordData, _ := json.Marshal(record)
		if _, err := n.Compost(request.RequestID, "create", recordData); err != nil {
			log.Printf("[ClaudeOpusNim] ⚠️ Failed to record interaction: %v", err)
		}
	}

	return nil
}

// emitErrorResponse emits an error response leaf.
func (n *ClaudeOpusNim) emitErrorResponse(requestID string, err error, metadata map[string]string) error {
	if requestID == "" {
		requestID = fmt.Sprintf("error-%d", time.Now().UnixNano())
	}

	aiResponse := leaves.AIResponse{
		RequestID: requestID,
		Response:  "",
		Model:     ClaudeOpusModel,
		Metadata:  metadata,
		Error:     err.Error(),
		Success:   false,
		Timestamp: time.Now(),
	}

	if emitErr := n.LeafStruct("ai.response", aiResponse); emitErr != nil {
		log.Printf("[ClaudeOpusNim] ❌ Failed to emit error response: %v", emitErr)
		return emitErr
	}

	log.Printf("[ClaudeOpusNim] 🍃 Emitted error response for request %s", requestID)
	return nil
}

// Stop stops the nim from processing leaves.
func (n *ClaudeOpusNim) Stop() error {
	if n.cancel != nil {
		n.cancel()
	}

	// Close the AI service
	if n.aiService != nil {
		if err := n.aiService.Close(context.Background()); err != nil {
			log.Printf("[ClaudeOpusNim] Warning: failed to close AI service: %v", err)
		}
	}

	log.Printf("[ClaudeOpusNim] Stopped")
	return nil
}

// Ask is a convenience method to directly ask Claude Opus a question.
// This can be used for programmatic access without going through the leaf system.
func (n *ClaudeOpusNim) Ask(ctx context.Context, prompt string) (string, error) {
	if n.aiService == nil {
		return "", fmt.Errorf("AI service not initialized")
	}
	return n.aiService.Ask(ctx, prompt)
}

// GetModel returns the Claude model being used.
func (n *ClaudeOpusNim) GetModel() string {
	return ClaudeOpusModel
}
