package brain

import (
	"context"
	"fmt"
	"time"
)

// MockBrain implements Brain interface for testing purposes
type MockBrain struct {
	// Optional fields for controlling mock behavior
	AskResponse    string
	AskResponseRaw bool // If true, return AskResponse as-is without prepending question
	AskError       error
	StoreError     error
	UpdateError    error
	DeleteError    error
	InitError      error
	CloseError     error
}

// NewMockBrain creates a new MockBrain with default responses
func NewMockBrain() *MockBrain {
	return &MockBrain{
		AskResponse: "Mock brain response",
	}
}

// Store implements Brain.Store
func (m *MockBrain) Store(ctx context.Context, content string, tags []string) (*Knowledge, error) {
	if m.StoreError != nil {
		return nil, m.StoreError
	}
	
	now := time.Now()
	return &Knowledge{
		ID:        "mock-id",
		Content:   content,
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Retrieve implements Brain.Retrieve
func (m *MockBrain) Retrieve(ctx context.Context, id string) (*Knowledge, error) {
	now := time.Now()
	return &Knowledge{
		ID:        id,
		Content:   "mock content",
		Tags:      []string{"test"},
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Search implements Brain.Search
func (m *MockBrain) Search(ctx context.Context, query string) ([]*Knowledge, error) {
	now := time.Now()
	return []*Knowledge{{
		ID:        "mock-id",
		Content:   "mock content for: " + query,
		Tags:      []string{"test"},
		CreatedAt: now,
		UpdatedAt: now,
	}}, nil
}

// Update implements Brain.Update
func (m *MockBrain) Update(ctx context.Context, id string, content string) error {
	return m.UpdateError
}

// Delete implements Brain.Delete
func (m *MockBrain) Delete(ctx context.Context, id string) error {
	return m.DeleteError
}

// List implements Brain.List
func (m *MockBrain) List(ctx context.Context) ([]*Knowledge, error) {
	now := time.Now()
	return []*Knowledge{{
		ID:        "mock-id",
		Content:   "mock content",
		Tags:      []string{"test"},
		CreatedAt: now,
		UpdatedAt: now,
	}}, nil
}

// Ask implements Brain.Ask
func (m *MockBrain) Ask(ctx context.Context, question string) (string, error) {
	if m.AskError != nil {
		return "", m.AskError
	}

	if m.AskResponse != "" {
		if m.AskResponseRaw {
			return m.AskResponse, nil
		}
		return fmt.Sprintf("%s: %s", m.AskResponse, question), nil
	}

	return fmt.Sprintf("Mock response to: %s", question), nil
}

// SetRawResponse sets the response to be returned as-is (useful for JSON responses)
func (m *MockBrain) SetRawResponse(response string) {
	m.AskResponse = response
	m.AskResponseRaw = true
}

// Initialize implements Brain.Initialize
func (m *MockBrain) Initialize(ctx context.Context) error {
	return m.InitError
}

// Close implements Brain.Close
func (m *MockBrain) Close(ctx context.Context) error {
	return m.CloseError
}

// Compile-time interface compliance check
var _ Brain = (*MockBrain)(nil)