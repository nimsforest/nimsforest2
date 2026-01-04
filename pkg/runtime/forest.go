package runtime

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/yourusername/nimsforest/internal/core"
	"github.com/yourusername/nimsforest/pkg/brain"
)

// Forest is the main runtime that manages all TreeHouses and Nims.
// It uses Wind for all pub/sub operations.
type Forest struct {
	config     *Config
	wind       *core.Wind
	humus      *core.Humus // Optional: for state tracking
	brain      brain.Brain
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

// Start starts all TreeHouses and Nims.
func (f *Forest) Start(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.running {
		return fmt.Errorf("forest already running")
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
	log.Printf("[Forest] Started with %d treehouses and %d nims",
		len(f.treehouses), len(f.nims))
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
