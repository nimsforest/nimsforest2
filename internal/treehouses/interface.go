package treehouses

import "github.com/yourusername/nimsforest/internal/core"

// GoTreeHouse is a compile-time TreeHouse implemented in Go.
// Unlike Lua TreeHouses (runtime), these are compiled into the binary.
//
// CRITICAL: Must be deterministic - same input Leaf produces same output Leaf.
// If you need AI/non-deterministic behavior, implement a Nim instead.
//
// Examples of GoTreeHouses:
//   - LandHouse: Responds to land capacity queries
//   - AgentHouse: Dispatches tasks to Docker containers
//   - AfterSalesHouse: Deterministic payment event routing
//   - GeneralHouse: Deterministic general event routing
type GoTreeHouse interface {
	// Name returns the unique name of this TreeHouse
	Name() string

	// Subjects returns the Wind subjects this TreeHouse subscribes to
	Subjects() []string

	// Process handles an incoming Leaf and optionally returns a response Leaf.
	// Return nil if no response should be emitted.
	//
	// IMPORTANT: This method must be deterministic - given the same input Leaf,
	// it must always return the same output Leaf (or nil).
	Process(leaf core.Leaf) *core.Leaf
}
