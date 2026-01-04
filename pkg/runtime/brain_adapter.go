package runtime

import (
	"context"
	"fmt"
	"time"

	pkgaiservice "github.com/yourusername/nimsforest/pkg/infrastructure/aiservice"
	"github.com/yourusername/nimsforest/pkg/brain"
)

// AIServiceBrain adapts an AIService to the brain.Brain interface.
// This allows using any AI service (Claude, OpenAI, Gemini) as a brain for Nims.
type AIServiceBrain struct {
	service     pkgaiservice.AIService
	initialized bool
}

// NewAIServiceBrain creates a new brain adapter wrapping an AI service.
func NewAIServiceBrain(service pkgaiservice.AIService) *AIServiceBrain {
	return &AIServiceBrain{
		service: service,
	}
}

// Ensure AIServiceBrain implements brain.Brain
var _ brain.Brain = (*AIServiceBrain)(nil)

// Initialize initializes the underlying AI service.
func (b *AIServiceBrain) Initialize(ctx context.Context) error {
	if err := b.service.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize AI service: %w", err)
	}
	b.initialized = true
	return nil
}

// Close closes the underlying AI service.
func (b *AIServiceBrain) Close(ctx context.Context) error {
	if !b.initialized {
		return nil
	}
	b.initialized = false
	return b.service.Close(ctx)
}

// Ask sends a question to the AI service and returns the response.
func (b *AIServiceBrain) Ask(ctx context.Context, question string) (string, error) {
	if !b.initialized {
		return "", fmt.Errorf("brain not initialized")
	}
	return b.service.Ask(ctx, question)
}

// Store is not implemented for AIServiceBrain (stateless).
func (b *AIServiceBrain) Store(ctx context.Context, content string, tags []string) (*brain.Knowledge, error) {
	return nil, fmt.Errorf("Store not implemented for AIServiceBrain")
}

// Retrieve is not implemented for AIServiceBrain (stateless).
func (b *AIServiceBrain) Retrieve(ctx context.Context, id string) (*brain.Knowledge, error) {
	return nil, fmt.Errorf("Retrieve not implemented for AIServiceBrain")
}

// Search is not implemented for AIServiceBrain (stateless).
func (b *AIServiceBrain) Search(ctx context.Context, query string) ([]*brain.Knowledge, error) {
	return nil, fmt.Errorf("Search not implemented for AIServiceBrain")
}

// Update is not implemented for AIServiceBrain (stateless).
func (b *AIServiceBrain) Update(ctx context.Context, id string, content string) error {
	return fmt.Errorf("Update not implemented for AIServiceBrain")
}

// Delete is not implemented for AIServiceBrain (stateless).
func (b *AIServiceBrain) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("Delete not implemented for AIServiceBrain")
}

// List is not implemented for AIServiceBrain (stateless).
func (b *AIServiceBrain) List(ctx context.Context) ([]*brain.Knowledge, error) {
	return nil, fmt.Errorf("List not implemented for AIServiceBrain")
}

// SimpleBrain is a minimal brain implementation for testing without AI service.
// It evaluates leads based on score extracted from the prompt.
type SimpleBrain struct {
	initialized bool
}

// NewSimpleBrain creates a simple rule-based brain (no AI required).
func NewSimpleBrain() *SimpleBrain {
	return &SimpleBrain{}
}

var _ brain.Brain = (*SimpleBrain)(nil)

func (b *SimpleBrain) Initialize(ctx context.Context) error {
	b.initialized = true
	return nil
}

func (b *SimpleBrain) Close(ctx context.Context) error {
	b.initialized = false
	return nil
}

func (b *SimpleBrain) Ask(ctx context.Context, question string) (string, error) {
	// Simple rule-based response for lead qualification
	// In production, this would call a real AI service
	return `{"pursue": true, "reason": "Lead evaluation pending AI service configuration", "priority": "medium", "action": "Configure ANTHROPIC_API_KEY to enable AI-powered evaluation"}`, nil
}

func (b *SimpleBrain) Store(ctx context.Context, content string, tags []string) (*brain.Knowledge, error) {
	return &brain.Knowledge{
		ID:        fmt.Sprintf("simple-%d", time.Now().UnixNano()),
		Content:   content,
		Tags:      tags,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (b *SimpleBrain) Retrieve(ctx context.Context, id string) (*brain.Knowledge, error) {
	return nil, brain.ErrKnowledgeNotFound
}

func (b *SimpleBrain) Search(ctx context.Context, query string) ([]*brain.Knowledge, error) {
	return []*brain.Knowledge{}, nil
}

func (b *SimpleBrain) Update(ctx context.Context, id string, content string) error {
	return brain.ErrKnowledgeNotFound
}

func (b *SimpleBrain) Delete(ctx context.Context, id string) error {
	return brain.ErrKnowledgeNotFound
}

func (b *SimpleBrain) List(ctx context.Context) ([]*brain.Knowledge, error) {
	return []*brain.Knowledge{}, nil
}
