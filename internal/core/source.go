// Package core provides source interfaces for feeding external data into River.
package core

import (
	"context"
	"sync"
)

// Source feeds external data into the River.
// Sources hold a direct reference to River - they're either connected or not.
type Source interface {
	// Name returns the unique identifier for this source
	Name() string

	// Type returns the source type (http_webhook, http_poll, ceremony)
	Type() string

	// Start begins accepting/fetching data and flowing to River
	Start(ctx context.Context) error

	// Stop gracefully shuts down the source
	Stop() error

	// IsRunning returns whether the source is active
	IsRunning() bool
}

// BaseSource provides common functionality for all sources.
// Embeds a River reference for direct data flow.
type BaseSource struct {
	name      string
	river     *River
	publishes string // The river subject to publish to
	running   bool
	mu        sync.Mutex
}

// NewBaseSource creates a base source connected to the given River.
func NewBaseSource(name string, river *River, publishes string) *BaseSource {
	return &BaseSource{
		name:      name,
		river:     river,
		publishes: publishes,
	}
}

// Name returns the source name.
func (s *BaseSource) Name() string {
	return s.name
}

// Publishes returns the subject this source publishes to.
func (s *BaseSource) Publishes() string {
	return s.publishes
}

// IsRunning returns whether the source is currently running.
func (s *BaseSource) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// SetRunning sets the running state.
func (s *BaseSource) SetRunning(running bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = running
}

// Flow sends data to the River.
func (s *BaseSource) Flow(data []byte) error {
	return s.river.Flow(s.publishes, data)
}

// FlowWithSubject sends data to River with a custom subject suffix.
// E.g., if publishes is "river.stripe" and suffix is "webhook.charge",
// the final subject becomes "river.stripe.webhook.charge"
func (s *BaseSource) FlowWithSubject(suffix string, data []byte) error {
	subject := s.publishes
	if suffix != "" {
		subject = s.publishes + "." + suffix
	}
	return s.river.Flow(subject, data)
}

// River returns the underlying River reference.
func (s *BaseSource) River() *River {
	return s.river
}
