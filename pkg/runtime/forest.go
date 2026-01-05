package runtime

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/yourusername/nimsforest/internal/core"
	"github.com/yourusername/nimsforest/pkg/brain"
)

// Forest is the main runtime that manages all Trees, TreeHouses and Nims.
// It uses Wind for all pub/sub operations.
type Forest struct {
	config     *Config
	wind       *core.Wind
	river      *core.River // Optional: for Trees (requires JetStream)
	humus      *core.Humus // Optional: for state tracking
	brain      brain.Brain
	trees      map[string]*TreeInstance
	treehouses map[string]*TreeHouse
	nims       map[string]*Nim

	mu      sync.Mutex
	running bool
}

// NewForest creates a new Forest runtime from a configuration file.
func NewForest(configPath string, wind *core.Wind, b brain.Brain) (*Forest, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return NewForestFromConfig(cfg, wind, b)
}

// NewForestFromConfig creates a new Forest runtime from an existing config.
func NewForestFromConfig(cfg *Config, wind *core.Wind, b brain.Brain) (*Forest, error) {
	if wind == nil {
		return nil, fmt.Errorf("wind is required")
	}
	if b == nil {
		return nil, fmt.Errorf("brain is required")
	}

	f := &Forest{
		config:     cfg,
		wind:       wind,
		brain:      b,
		trees:      make(map[string]*TreeInstance),
		treehouses: make(map[string]*TreeHouse),
		nims:       make(map[string]*Nim),
	}

	// Create TreeHouses
	for name, thCfg := range cfg.TreeHouses {
		scriptPath := cfg.ResolvePath(thCfg.Script)
		th, err := NewTreeHouse(thCfg, wind, scriptPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create treehouse %s: %w", name, err)
		}
		f.treehouses[name] = th
	}

	// Create Nims
	for name, nimCfg := range cfg.Nims {
		promptPath := cfg.ResolvePath(nimCfg.Prompt)
		nim, err := NewNim(nimCfg, wind, b, promptPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create nim %s: %w", name, err)
		}
		f.nims[name] = nim
	}

	return f, nil
}

// NewForestWithHumus creates a Forest with Humus for state change tracking.
func NewForestWithHumus(cfg *Config, wind *core.Wind, humus *core.Humus, b brain.Brain) (*Forest, error) {
	if wind == nil {
		return nil, fmt.Errorf("wind is required")
	}
	if b == nil {
		return nil, fmt.Errorf("brain is required")
	}

	f := &Forest{
		config:     cfg,
		wind:       wind,
		humus:      humus,
		brain:      b,
		trees:      make(map[string]*TreeInstance),
		treehouses: make(map[string]*TreeHouse),
		nims:       make(map[string]*Nim),
	}

	// Create TreeHouses
	for name, thCfg := range cfg.TreeHouses {
		scriptPath := cfg.ResolvePath(thCfg.Script)
		th, err := NewTreeHouse(thCfg, wind, scriptPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create treehouse %s: %w", name, err)
		}
		f.treehouses[name] = th
	}

	// Create Nims (with Humus if provided)
	for name, nimCfg := range cfg.Nims {
		promptPath := cfg.ResolvePath(nimCfg.Prompt)
		var nim *Nim
		var err error

		if humus != nil {
			nim, err = NewNimWithHumus(nimCfg, wind, humus, b, promptPath)
		} else {
			nim, err = NewNim(nimCfg, wind, b, promptPath)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to create nim %s: %w", name, err)
		}
		f.nims[name] = nim
	}

	return f, nil
}

// Start starts all Trees, TreeHouses and Nims.
func (f *Forest) Start(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.running {
		return fmt.Errorf("forest already running")
	}

	// Start Trees
	for name, instance := range f.trees {
		if err := instance.Tree.Start(ctx); err != nil {
			f.stopAll()
			return fmt.Errorf("failed to start tree %s: %w", name, err)
		}
		instance.Running = true
	}

	// Start TreeHouses
	for name, th := range f.treehouses {
		if err := th.Start(ctx); err != nil {
			// Stop any already started
			f.stopAll()
			return fmt.Errorf("failed to start treehouse %s: %w", name, err)
		}
	}

	// Start Nims
	for name, nim := range f.nims {
		if err := nim.Start(ctx); err != nil {
			// Stop any already started
			f.stopAll()
			return fmt.Errorf("failed to start nim %s: %w", name, err)
		}
	}

	f.running = true
	log.Printf("[Forest] Started with %d trees, %d treehouses and %d nims",
		len(f.trees), len(f.treehouses), len(f.nims))
	return nil
}

// Stop stops all TreeHouses and Nims.
func (f *Forest) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.running {
		return nil
	}

	f.stopAll()
	f.running = false
	log.Printf("[Forest] Stopped")
	return nil
}

// stopAll stops all components without locking (internal use).
func (f *Forest) stopAll() {
	for _, instance := range f.trees {
		instance.Tree.Stop()
		instance.Running = false
	}
	for _, th := range f.treehouses {
		th.Stop()
	}
	for _, nim := range f.nims {
		nim.Stop()
	}
}

// TreeHouse returns a TreeHouse by name.
func (f *Forest) TreeHouse(name string) *TreeHouse {
	return f.treehouses[name]
}

// Nim returns a Nim by name.
func (f *Forest) Nim(name string) *Nim {
	return f.nims[name]
}

// Config returns the forest configuration.
func (f *Forest) Config() *Config {
	return f.config
}

// IsRunning returns whether the forest is currently running.
func (f *Forest) IsRunning() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.running
}

// SetRiver sets the River connection for tree support.
// Must be called before adding trees.
func (f *Forest) SetRiver(river *core.River) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.river = river
}

// =============================================================================
// Runtime Component Management
// =============================================================================

// ComponentInfo provides information about a running component.
type ComponentInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"` // "treehouse" or "nim"
	Subscribes string `json:"subscribes"`
	Publishes  string `json:"publishes"`
	Script     string `json:"script,omitempty"` // TreeHouse only
	Prompt     string `json:"prompt,omitempty"` // Nim only
	Running    bool   `json:"running"`
}

// ForestStatus provides a snapshot of the forest state.
type ForestStatus struct {
	Running     bool               `json:"running"`
	Trees       []TreeInstanceInfo `json:"trees"`
	TreeHouses  []ComponentInfo    `json:"treehouses"`
	Nims        []ComponentInfo    `json:"nims"`
	ConfigPath  string             `json:"config_path,omitempty"`
}

// Status returns the current status of the forest.
func (f *Forest) Status() ForestStatus {
	f.mu.Lock()
	defer f.mu.Unlock()

	status := ForestStatus{
		Running:    f.running,
		Trees:      make([]TreeInstanceInfo, 0, len(f.trees)),
		TreeHouses: make([]ComponentInfo, 0, len(f.treehouses)),
		Nims:       make([]ComponentInfo, 0, len(f.nims)),
	}

	for name, instance := range f.trees {
		status.Trees = append(status.Trees, TreeInstanceInfo{
			Name:     name,
			Type:     instance.Config.Type,
			Patterns: instance.Tree.Patterns(),
			Running:  instance.Running,
		})
	}

	for name, th := range f.treehouses {
		cfg := f.config.TreeHouses[name]
		status.TreeHouses = append(status.TreeHouses, ComponentInfo{
			Name:       name,
			Type:       "treehouse",
			Subscribes: cfg.Subscribes,
			Publishes:  cfg.Publishes,
			Script:     cfg.Script,
			Running:    th.IsRunning(),
		})
	}

	for name, nim := range f.nims {
		cfg := f.config.Nims[name]
		status.Nims = append(status.Nims, ComponentInfo{
			Name:       name,
			Type:       "nim",
			Subscribes: cfg.Subscribes,
			Publishes:  cfg.Publishes,
			Prompt:     cfg.Prompt,
			Running:    nim.IsRunning(),
		})
	}

	return status
}

// AddTreeHouse adds a new TreeHouse at runtime.
func (f *Forest) AddTreeHouse(name string, cfg TreeHouseConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.treehouses[name]; exists {
		return fmt.Errorf("treehouse '%s' already exists", name)
	}

	// Ensure name is set
	cfg.Name = name

	// Resolve script path
	scriptPath := f.config.ResolvePath(cfg.Script)

	// Create the TreeHouse
	th, err := NewTreeHouse(cfg, f.wind, scriptPath)
	if err != nil {
		return fmt.Errorf("failed to create treehouse: %w", err)
	}

	// Start it if the forest is running
	if f.running {
		if err := th.Start(context.Background()); err != nil {
			return fmt.Errorf("failed to start treehouse: %w", err)
		}
	}

	// Add to maps
	f.treehouses[name] = th
	if f.config.TreeHouses == nil {
		f.config.TreeHouses = make(map[string]TreeHouseConfig)
	}
	f.config.TreeHouses[name] = cfg

	log.Printf("[Forest] Added treehouse '%s' (subscribes: %s, publishes: %s)",
		name, cfg.Subscribes, cfg.Publishes)
	return nil
}

// RemoveTreeHouse removes a TreeHouse at runtime.
func (f *Forest) RemoveTreeHouse(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	th, exists := f.treehouses[name]
	if !exists {
		return fmt.Errorf("treehouse '%s' not found", name)
	}

	// Stop it
	if err := th.Stop(); err != nil {
		log.Printf("[Forest] Warning: error stopping treehouse '%s': %v", name, err)
	}

	// Remove from maps
	delete(f.treehouses, name)
	delete(f.config.TreeHouses, name)

	log.Printf("[Forest] Removed treehouse '%s'", name)
	return nil
}

// AddNim adds a new Nim at runtime.
func (f *Forest) AddNim(name string, cfg NimConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.nims[name]; exists {
		return fmt.Errorf("nim '%s' already exists", name)
	}

	// Ensure name is set
	cfg.Name = name

	// Resolve prompt path
	promptPath := f.config.ResolvePath(cfg.Prompt)

	// Create the Nim
	var nim *Nim
	var err error

	if f.humus != nil {
		nim, err = NewNimWithHumus(cfg, f.wind, f.humus, f.brain, promptPath)
	} else {
		nim, err = NewNim(cfg, f.wind, f.brain, promptPath)
	}

	if err != nil {
		return fmt.Errorf("failed to create nim: %w", err)
	}

	// Start it if the forest is running
	if f.running {
		if err := nim.Start(context.Background()); err != nil {
			return fmt.Errorf("failed to start nim: %w", err)
		}
	}

	// Add to maps
	f.nims[name] = nim
	if f.config.Nims == nil {
		f.config.Nims = make(map[string]NimConfig)
	}
	f.config.Nims[name] = cfg

	log.Printf("[Forest] Added nim '%s' (subscribes: %s, publishes: %s)",
		name, cfg.Subscribes, cfg.Publishes)
	return nil
}

// RemoveNim removes a Nim at runtime.
func (f *Forest) RemoveNim(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	nim, exists := f.nims[name]
	if !exists {
		return fmt.Errorf("nim '%s' not found", name)
	}

	// Stop it
	if err := nim.Stop(); err != nil {
		log.Printf("[Forest] Warning: error stopping nim '%s': %v", name, err)
	}

	// Remove from maps
	delete(f.nims, name)
	delete(f.config.Nims, name)

	log.Printf("[Forest] Removed nim '%s'", name)
	return nil
}

// Reload reloads the forest configuration, adding/removing components as needed.
func (f *Forest) Reload(newCfg *Config) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Find TreeHouses to remove (in old config but not in new)
	for name := range f.treehouses {
		if _, exists := newCfg.TreeHouses[name]; !exists {
			if th := f.treehouses[name]; th != nil {
				th.Stop()
			}
			delete(f.treehouses, name)
			log.Printf("[Forest] Removed treehouse '%s' (not in new config)", name)
		}
	}

	// Find TreeHouses to add (in new config but not running)
	for name, cfg := range newCfg.TreeHouses {
		if _, exists := f.treehouses[name]; !exists {
			cfg.Name = name
			scriptPath := newCfg.ResolvePath(cfg.Script)
			th, err := NewTreeHouse(cfg, f.wind, scriptPath)
			if err != nil {
				log.Printf("[Forest] Warning: failed to create treehouse '%s': %v", name, err)
				continue
			}
			if f.running {
				if err := th.Start(context.Background()); err != nil {
					log.Printf("[Forest] Warning: failed to start treehouse '%s': %v", name, err)
					continue
				}
			}
			f.treehouses[name] = th
			log.Printf("[Forest] Added treehouse '%s' from new config", name)
		}
	}

	// Find Nims to remove
	for name := range f.nims {
		if _, exists := newCfg.Nims[name]; !exists {
			if nim := f.nims[name]; nim != nil {
				nim.Stop()
			}
			delete(f.nims, name)
			log.Printf("[Forest] Removed nim '%s' (not in new config)", name)
		}
	}

	// Find Nims to add
	for name, cfg := range newCfg.Nims {
		if _, exists := f.nims[name]; !exists {
			cfg.Name = name
			promptPath := newCfg.ResolvePath(cfg.Prompt)
			var nim *Nim
			var err error
			if f.humus != nil {
				nim, err = NewNimWithHumus(cfg, f.wind, f.humus, f.brain, promptPath)
			} else {
				nim, err = NewNim(cfg, f.wind, f.brain, promptPath)
			}
			if err != nil {
				log.Printf("[Forest] Warning: failed to create nim '%s': %v", name, err)
				continue
			}
			if f.running {
				if err := nim.Start(context.Background()); err != nil {
					log.Printf("[Forest] Warning: failed to start nim '%s': %v", name, err)
					continue
				}
			}
			f.nims[name] = nim
			log.Printf("[Forest] Added nim '%s' from new config", name)
		}
	}

	// Update config reference
	f.config = newCfg

	log.Printf("[Forest] Reloaded with %d treehouses and %d nims",
		len(f.treehouses), len(f.nims))
	return nil
}
