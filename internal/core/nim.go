package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

// Nim represents a business logic component that reacts to leaves.
// Nims catch leaves from the wind, make decisions, and can emit new leaves
// or compost state changes.
type Nim interface {
	// Name returns the unique identifier for this nim
	Name() string

	// Subjects returns the wind subject patterns this nim listens to
	Subjects() []string

	// Handle processes a caught leaf
	Handle(ctx context.Context, leaf Leaf) error

	// Start begins listening for leaves
	Start(ctx context.Context) error

	// Stop gracefully shuts down the nim
	Stop() error
}

// BaseNim provides common functionality for all nims.
// Concrete nims should embed this and implement the Nim interface.
type BaseNim struct {
	name  string
	wind  *Wind
	humus *Humus
	soil  *Soil
}

// NewBaseNim creates a new base nim with the given name and connections.
func NewBaseNim(name string, wind *Wind, humus *Humus, soil *Soil) *BaseNim {
	return &BaseNim{
		name:  name,
		wind:  wind,
		humus: humus,
		soil:  soil,
	}
}

// Name returns the nim's name.
func (n *BaseNim) Name() string {
	return n.name
}

// Leaf drops a new leaf onto the wind.
// This is used when a nim needs to emit an event.
func (n *BaseNim) Leaf(subject string, data []byte) error {
	leaf := NewLeaf(subject, data, n.name)
	
	if err := n.wind.Drop(*leaf); err != nil {
		return fmt.Errorf("nim %s failed to drop leaf: %w", n.name, err)
	}

	log.Printf("[Nim:%s] Dropped leaf: %s", n.name, subject)
	return nil
}

// LeafStruct drops a leaf with a structured payload.
// The data parameter is marshaled to JSON automatically.
func (n *BaseNim) LeafStruct(subject string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("nim %s failed to marshal leaf data: %w", n.name, err)
	}
	return n.Leaf(subject, jsonData)
}

// Compost sends a state change to humus.
// The decomposer will apply this change to soil.
func (n *BaseNim) Compost(entity string, action string, data []byte) (uint64, error) {
	slot, err := n.humus.Add(n.name, entity, action, data)
	if err != nil {
		return 0, fmt.Errorf("nim %s failed to compost: %w", n.name, err)
	}

	log.Printf("[Nim:%s] Composted: entity=%s, action=%s, slot=%d", n.name, entity, action, slot)
	return slot, nil
}

// CompostStruct sends a state change with structured data.
// The data parameter is marshaled to JSON automatically.
func (n *BaseNim) CompostStruct(entity string, action string, data interface{}) (uint64, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return 0, fmt.Errorf("nim %s failed to marshal compost data: %w", n.name, err)
	}
	return n.Compost(entity, action, jsonData)
}

// Dig reads current state from soil.
// Returns the data, the current revision, and any error.
func (n *BaseNim) Dig(entity string) ([]byte, uint64, error) {
	data, revision, err := n.soil.Dig(entity)
	if err != nil {
		return nil, 0, fmt.Errorf("nim %s failed to dig: %w", n.name, err)
	}

	log.Printf("[Nim:%s] Dug entity: %s (revision: %d)", n.name, entity, revision)
	return data, revision, nil
}

// DigStruct reads state and unmarshals it into the provided struct.
func (n *BaseNim) DigStruct(entity string, target interface{}) (uint64, error) {
	data, revision, err := n.Dig(entity)
	if err != nil {
		return 0, err
	}

	if err := json.Unmarshal(data, target); err != nil {
		return 0, fmt.Errorf("nim %s failed to unmarshal state: %w", n.name, err)
	}

	return revision, nil
}

// Bury writes state to soil with optimistic locking.
// Use revision 0 for new entities, or the current revision for updates.
func (n *BaseNim) Bury(entity string, data []byte, expectedRevision uint64) error {
	err := n.soil.Bury(entity, data, expectedRevision)
	if err != nil {
		return fmt.Errorf("nim %s failed to bury: %w", n.name, err)
	}

	log.Printf("[Nim:%s] Buried entity: %s (expected revision: %d)", n.name, entity, expectedRevision)
	return nil
}

// BuryStruct writes structured state to soil with optimistic locking.
func (n *BaseNim) BuryStruct(entity string, data interface{}, expectedRevision uint64) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("nim %s failed to marshal state: %w", n.name, err)
	}
	return n.Bury(entity, jsonData, expectedRevision)
}

// Catch starts listening for leaves matching the given subject pattern.
// This is a helper method for concrete nims to use in their Start() implementation.
func (n *BaseNim) Catch(subject string, handler func(leaf Leaf)) error {
	_, err := n.wind.Catch(subject, handler)
	if err != nil {
		return fmt.Errorf("nim %s failed to catch %s: %w", n.name, subject, err)
	}

	log.Printf("[Nim:%s] Catching leaves: %s", n.name, subject)
	return nil
}

// CatchWithQueue starts listening with a queue group for load balancing.
func (n *BaseNim) CatchWithQueue(subject, queue string, handler func(leaf Leaf)) error {
	_, err := n.wind.CatchWithQueue(subject, queue, handler)
	if err != nil {
		return fmt.Errorf("nim %s failed to catch %s with queue %s: %w", n.name, subject, queue, err)
	}

	log.Printf("[Nim:%s] Catching leaves with queue: %s (queue: %s)", n.name, subject, queue)
	return nil
}

// GetWind returns the wind connection (for testing or advanced usage).
func (n *BaseNim) GetWind() *Wind {
	return n.wind
}

// GetHumus returns the humus connection (for testing or advanced usage).
func (n *BaseNim) GetHumus() *Humus {
	return n.humus
}

// GetSoil returns the soil connection (for testing or advanced usage).
func (n *BaseNim) GetSoil() *Soil {
	return n.soil
}
