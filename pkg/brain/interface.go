package brain

import (
	"context"
)

// Brain defines the interface for all brain implementations
type Brain interface {
	// Store stores new knowledge in the brain
	Store(ctx context.Context, content string, tags []string) (*Knowledge, error)
	// Retrieve retrieves knowledge by ID
	Retrieve(ctx context.Context, id string) (*Knowledge, error)
	// Search searches for knowledge by tags or content
	Search(ctx context.Context, query string) ([]*Knowledge, error)
	// Update updates existing knowledge
	Update(ctx context.Context, id string, content string) error
	// Delete deletes knowledge by ID
	Delete(ctx context.Context, id string) error
	// List returns all knowledge entries
	List(ctx context.Context) ([]*Knowledge, error)
	// Ask asks a question and returns an answer based on stored knowledge
	Ask(ctx context.Context, question string) (string, error)
	// Initialize initializes the brain
	Initialize(ctx context.Context) error
	// Close closes the brain and releases resources
	Close(ctx context.Context) error
}
