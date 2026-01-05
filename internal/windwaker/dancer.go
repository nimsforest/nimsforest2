package windwaker

import (
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
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

// CatchBeat subscribes a Dancer to dance beats via Wind.Catch().
// The dancer's Dance() method will be called on each beat.
// Returns the subscription for later cleanup.
func CatchBeat(wind *core.Wind, dancer Dancer) (*nats.Subscription, error) {
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

// CatchBeatFunc subscribes a simple function to dance beats via Wind.Catch().
// This is a convenience for components that don't need the full Dancer interface.
// Returns the subscription for later cleanup.
func CatchBeatFunc(wind *core.Wind, name string, fn DanceFunc) (*nats.Subscription, error) {
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

// SimpleDancer wraps a DanceFunc as a Dancer interface.
type SimpleDancer struct {
	id string
	fn DanceFunc
}

// NewDancer creates a Dancer from an ID and function.
func NewDancer(id string, fn DanceFunc) *SimpleDancer {
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
