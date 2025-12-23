package core

import (
	"context"
	"fmt"
	"log"
)

// Tree represents a component that watches the river and produces structured leaves.
// Trees are the edge layer that parse unstructured external data into typed events.
type Tree interface {
	// Name returns the unique identifier for this tree
	Name() string

	// Patterns returns the river subject patterns this tree watches
	Patterns() []string

	// Parse attempts to match and structure river data into a leaf.
	// Returns nil if the data doesn't match this tree's patterns.
	Parse(subject string, data []byte) *Leaf

	// Start begins watching the river and producing leaves
	Start(ctx context.Context) error

	// Stop gracefully shuts down the tree
	Stop() error
}

// BaseTree provides common functionality for all trees.
// Concrete trees should embed this and implement the Tree interface.
type BaseTree struct {
	name  string
	wind  *Wind
	river *River
}

// NewBaseTree creates a new base tree with the given name and wind connection.
func NewBaseTree(name string, wind *Wind, river *River) *BaseTree {
	return &BaseTree{
		name:  name,
		wind:  wind,
		river: river,
	}
}

// Name returns the tree's name.
func (t *BaseTree) Name() string {
	return t.name
}

// Drop sends a structured leaf onto the wind.
// This is the primary way trees emit parsed events.
func (t *BaseTree) Drop(leaf Leaf) error {
	if leaf.Source == "" {
		leaf.Source = t.name
	}

	if err := t.wind.Drop(leaf); err != nil {
		return fmt.Errorf("tree %s failed to drop leaf: %w", t.name, err)
	}

	log.Printf("[Tree:%s] Dropped leaf: %s", t.name, leaf.Subject)
	return nil
}

// Watch starts observing a river pattern and calls the handler for each data item.
// This is a helper method for concrete trees to use in their Start() implementation.
func (t *BaseTree) Watch(pattern string, handler func(data RiverData)) error {
	if t.river == nil {
		return fmt.Errorf("tree %s has no river connection", t.name)
	}

	err := t.river.Observe(pattern, handler)
	if err != nil {
		return fmt.Errorf("tree %s failed to watch pattern %s: %w", t.name, pattern, err)
	}

	log.Printf("[Tree:%s] Watching pattern: %s", t.name, pattern)
	return nil
}

// GetWind returns the wind connection (for testing or advanced usage).
func (t *BaseTree) GetWind() *Wind {
	return t.wind
}

// GetRiver returns the river connection (for testing or advanced usage).
func (t *BaseTree) GetRiver() *River {
	return t.river
}
