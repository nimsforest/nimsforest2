package songbirds

import (
	"context"

	"github.com/yourusername/nimsforest/internal/core"
)

// Songbird listens for patterns on the wind and carries messages out
// to external platforms (Telegram, Slack, WhatsApp, etc.).
// Each songbird knows how to reach one destination.
type Songbird interface {
	// Name returns the unique identifier for this songbird
	Name() string

	// Type returns the songbird type (e.g., "telegram", "slack")
	Type() string

	// Pattern returns the wind subject pattern this songbird listens for
	// e.g., "song.telegram.>" to catch all Telegram-bound messages
	Pattern() string

	// Start begins listening for chirps on the wind
	Start(ctx context.Context) error

	// Stop gracefully shuts down the songbird
	Stop() error

	// IsRunning returns whether the songbird is currently active
	IsRunning() bool
}

// BaseSongbird provides common functionality for all songbirds.
// Concrete songbirds should embed this and implement the Songbird interface.
type BaseSongbird struct {
	name    string
	pattern string
	wind    *core.Wind
	running bool
}

// NewBaseSongbird creates a new base songbird with the given name, pattern, and wind connection.
func NewBaseSongbird(name string, pattern string, wind *core.Wind) *BaseSongbird {
	return &BaseSongbird{
		name:    name,
		pattern: pattern,
		wind:    wind,
	}
}

// Name returns the songbird's name.
func (s *BaseSongbird) Name() string {
	return s.name
}

// Pattern returns the subject pattern this songbird listens for.
func (s *BaseSongbird) Pattern() string {
	return s.pattern
}

// IsRunning returns whether the songbird is currently active.
func (s *BaseSongbird) IsRunning() bool {
	return s.running
}

// SetRunning sets the running state (for use by concrete implementations).
func (s *BaseSongbird) SetRunning(running bool) {
	s.running = running
}

// GetWind returns the wind connection.
func (s *BaseSongbird) GetWind() *core.Wind {
	return s.wind
}
