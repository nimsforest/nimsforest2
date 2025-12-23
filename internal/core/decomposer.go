package core

import (
	"context"
	"fmt"
	"log"
)

// Decomposer processes compost entries from humus and applies them to soil.
// It's the worker that maintains the current state by processing the state change log.
type Decomposer struct {
	humus        *Humus
	soil         *Soil
	consumerName string
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewDecomposer creates a new decomposer instance.
func NewDecomposer(humus *Humus, soil *Soil) *Decomposer {
	return &Decomposer{
		humus:        humus,
		soil:         soil,
		consumerName: "decomposer",
	}
}

// NewDecomposerWithConsumer creates a decomposer with a custom consumer name.
// This allows multiple decomposers or different consumption patterns.
func NewDecomposerWithConsumer(humus *Humus, soil *Soil, consumerName string) *Decomposer {
	return &Decomposer{
		humus:        humus,
		soil:         soil,
		consumerName: consumerName,
	}
}

// Start begins processing compost entries.
// This runs in the background and returns immediately.
func (d *Decomposer) Start() error {
	d.ctx, d.cancel = context.WithCancel(context.Background())

	err := d.humus.DecomposeWithConsumer(d.consumerName, func(compost Compost) {
		// Check if we should stop
		select {
		case <-d.ctx.Done():
			return
		default:
		}

		// Process the compost entry
		if err := d.processCompost(compost); err != nil {
			log.Printf("[Decomposer] Error processing compost slot %d: %v", compost.Slot, err)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to start decomposer: %w", err)
	}

	log.Printf("[Decomposer] Started with consumer: %s", d.consumerName)
	return nil
}

// Stop gracefully shuts down the decomposer.
func (d *Decomposer) Stop() {
	if d.cancel != nil {
		d.cancel()
		log.Printf("[Decomposer] Stopped")
	}
}

// processCompost applies a single compost entry to soil.
func (d *Decomposer) processCompost(compost Compost) error {
	log.Printf("[Decomposer] Processing: slot=%d, entity=%s, action=%s, nim=%s",
		compost.Slot, compost.Entity, compost.Action, compost.NimName)

	switch compost.Action {
	case "create":
		// Create new entity in soil
		err := d.soil.Bury(compost.Entity, compost.Data, 0)
		if err != nil {
			// If entity already exists, that's okay - might be a replay
			return fmt.Errorf("failed to create entity %s: %w", compost.Entity, err)
		}
		log.Printf("[Decomposer] Created entity: %s", compost.Entity)

	case "update":
		// Update existing entity
		// Read current state to get revision
		_, currentRevision, err := d.soil.Dig(compost.Entity)
		if err != nil {
			// Entity doesn't exist, create it instead
			log.Printf("[Decomposer] Entity %s not found, creating instead", compost.Entity)
			err = d.soil.Bury(compost.Entity, compost.Data, 0)
			if err != nil {
				return fmt.Errorf("failed to create entity %s during update: %w", compost.Entity, err)
			}
			return nil
		}

		// Update with optimistic locking
		err = d.soil.Bury(compost.Entity, compost.Data, currentRevision)
		if err != nil {
			return fmt.Errorf("failed to update entity %s: %w", compost.Entity, err)
		}
		log.Printf("[Decomposer] Updated entity: %s (revision: %d)", compost.Entity, currentRevision)

	case "delete":
		// Delete entity from soil
		err := d.soil.Delete(compost.Entity)
		if err != nil {
			// If entity doesn't exist, that's okay - might be a replay
			log.Printf("[Decomposer] Entity %s not found for deletion (might be replay)", compost.Entity)
			return nil
		}
		log.Printf("[Decomposer] Deleted entity: %s", compost.Entity)

	default:
		return fmt.Errorf("unknown action: %s", compost.Action)
	}

	return nil
}

// RunDecomposer is a convenience function that creates and starts a decomposer.
// It returns the decomposer instance so the caller can stop it later.
func RunDecomposer(humus *Humus, soil *Soil) (*Decomposer, error) {
	decomposer := NewDecomposer(humus, soil)
	if err := decomposer.Start(); err != nil {
		return nil, err
	}
	return decomposer, nil
}

// RunDecomposerWithConsumer is like RunDecomposer but allows specifying a consumer name.
func RunDecomposerWithConsumer(humus *Humus, soil *Soil, consumerName string) (*Decomposer, error) {
	decomposer := NewDecomposerWithConsumer(humus, soil, consumerName)
	if err := decomposer.Start(); err != nil {
		return nil, err
	}
	return decomposer, nil
}
