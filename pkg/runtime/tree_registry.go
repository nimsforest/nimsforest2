// Package runtime provides the tree registry for dynamic tree management.
package runtime

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/yourusername/nimsforest/internal/core"
)

// TreeFactory is a function that creates a new Tree instance.
type TreeFactory func(wind *core.Wind, river *core.River) core.Tree

// TreeInfo contains information about a registered tree type.
type TreeInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Patterns    []string `json:"patterns"`
}

// treeRegistry holds registered tree factories.
var (
	treeFactories = make(map[string]TreeFactory)
	treeInfos     = make(map[string]TreeInfo)
	treeMu        sync.RWMutex
)

// RegisterTree registers a tree factory with the given name.
// This is typically called in init() functions.
func RegisterTree(name string, description string, patterns []string, factory TreeFactory) {
	treeMu.Lock()
	defer treeMu.Unlock()

	if _, exists := treeFactories[name]; exists {
		panic(fmt.Sprintf("tree type '%s' already registered", name))
	}

	treeFactories[name] = factory
	treeInfos[name] = TreeInfo{
		Name:        name,
		Description: description,
		Patterns:    patterns,
	}

	log.Printf("[TreeRegistry] Registered tree type: %s", name)
}

// GetTreeFactory returns the factory for a tree type.
func GetTreeFactory(name string) (TreeFactory, bool) {
	treeMu.RLock()
	defer treeMu.RUnlock()
	factory, ok := treeFactories[name]
	return factory, ok
}

// ListTreeTypes returns all registered tree types.
func ListTreeTypes() []TreeInfo {
	treeMu.RLock()
	defer treeMu.RUnlock()

	types := make([]TreeInfo, 0, len(treeInfos))
	for _, info := range treeInfos {
		types = append(types, info)
	}
	return types
}

// TreeConfig defines configuration for a tree instance.
type TreeConfig struct {
	Name string `yaml:"name" json:"name"` // Instance name
	Type string `yaml:"type" json:"type"` // Tree type (from registry)
}

// TreeInstance wraps a running tree with its configuration.
type TreeInstance struct {
	Config  TreeConfig
	Tree    core.Tree
	Running bool
}

// =============================================================================
// Forest Tree Management
// =============================================================================

// AddTree adds a tree instance to the forest.
// The tree type must be registered in the tree registry.
func (f *Forest) AddTree(name string, treeType string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Check if tree already exists
	if _, exists := f.trees[name]; exists {
		return fmt.Errorf("tree '%s' already exists", name)
	}

	// Get factory from registry
	factory, ok := GetTreeFactory(treeType)
	if !ok {
		return fmt.Errorf("unknown tree type '%s' (available: %v)", treeType, getTreeTypeNames())
	}

	// We need Wind and River to create a tree
	if f.wind == nil {
		return fmt.Errorf("forest has no wind connection")
	}
	if f.river == nil {
		return fmt.Errorf("forest has no river connection (trees require JetStream)")
	}

	// Create tree instance
	tree := factory(f.wind, f.river)

	// Start if forest is running
	if f.running {
		if err := tree.Start(context.Background()); err != nil {
			return fmt.Errorf("failed to start tree: %w", err)
		}
	}

	// Store
	f.trees[name] = &TreeInstance{
		Config: TreeConfig{
			Name: name,
			Type: treeType,
		},
		Tree:    tree,
		Running: f.running,
	}

	log.Printf("[Forest] Added tree '%s' (type: %s)", name, treeType)
	return nil
}

// RemoveTree removes a tree instance from the forest.
func (f *Forest) RemoveTree(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	instance, exists := f.trees[name]
	if !exists {
		return fmt.Errorf("tree '%s' not found", name)
	}

	// Stop the tree
	if err := instance.Tree.Stop(); err != nil {
		log.Printf("[Forest] Warning: error stopping tree '%s': %v", name, err)
	}

	delete(f.trees, name)
	log.Printf("[Forest] Removed tree '%s'", name)
	return nil
}

// ListTrees returns information about all running trees.
func (f *Forest) ListTrees() []TreeInstanceInfo {
	f.mu.Lock()
	defer f.mu.Unlock()

	infos := make([]TreeInstanceInfo, 0, len(f.trees))
	for name, instance := range f.trees {
		infos = append(infos, TreeInstanceInfo{
			Name:     name,
			Type:     instance.Config.Type,
			Patterns: instance.Tree.Patterns(),
			Running:  instance.Running,
		})
	}
	return infos
}

// TreeInstanceInfo contains information about a running tree instance.
type TreeInstanceInfo struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Patterns []string `json:"patterns"`
	Running  bool     `json:"running"`
}

// Helper function
func getTreeTypeNames() []string {
	treeMu.RLock()
	defer treeMu.RUnlock()

	names := make([]string, 0, len(treeFactories))
	for name := range treeFactories {
		names = append(names, name)
	}
	return names
}
