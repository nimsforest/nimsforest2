package windwaker

import (
	"encoding/json"
	"log"

	"github.com/yourusername/nimsforest/internal/core"
)

// Dancer is the interface for components that participate in the synchronized dance.
// Components implement Dance() to perform their per-beat work.
type Dancer interface {
	// Dance is called on each beat. The beat contains sequence and timing info.
	Dance(beat Beat) error

	// ID returns the dancer's unique identifier (for logging/debugging).
	ID() string
}

// DanceFunc is a function type for simple dance handlers.
type DanceFunc func(beat Beat) error

// RegisterDancer subscribes a Dancer to dance beats via the Wind.
// The dancer's Dance() method will be called on each beat.
func RegisterDancer(wind *core.Wind, dancer Dancer) error {
	return wind.Catch(SubjectDanceBeat, func(leaf core.Leaf) {
		var beat Beat
		if err := json.Unmarshal(leaf.Data, &beat); err != nil {
			log.Printf("[Dancer:%s] Failed to unmarshal beat: %v", dancer.ID(), err)
			return
		}

		if err := dancer.Dance(beat); err != nil {
			log.Printf("[Dancer:%s] Dance error: %v", dancer.ID(), err)
		}
	})
}

// OnBeat subscribes a simple function to dance beats.
// This is a convenience for components that don't need the full Dancer interface.
func OnBeat(wind *core.Wind, name string, fn DanceFunc) error {
	return wind.Catch(SubjectDanceBeat, func(leaf core.Leaf) {
		var beat Beat
		if err := json.Unmarshal(leaf.Data, &beat); err != nil {
			log.Printf("[Dancer:%s] Failed to unmarshal beat: %v", name, err)
			return
		}

		if err := fn(beat); err != nil {
			log.Printf("[Dancer:%s] Dance error: %v", name, err)
		}
	})
}

// SimpleDancer wraps a function as a Dancer.
type SimpleDancer struct {
	id string
	fn DanceFunc
}

// NewSimpleDancer creates a Dancer from a function.
func NewSimpleDancer(id string, fn DanceFunc) *SimpleDancer {
	return &SimpleDancer{id: id, fn: fn}
}

// ID returns the dancer's identifier.
func (d *SimpleDancer) ID() string {
	return d.id
}

// Dance calls the wrapped function.
func (d *SimpleDancer) Dance(beat Beat) error {
	return d.fn(beat)
}
