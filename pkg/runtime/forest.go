package runtime

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/yourusername/nimsforest/pkg/brain"
)

// Forest is the main runtime that manages all TreeHouses and Nims.
type Forest struct {
	config     *Config
	nc         *nats.Conn
	brain      brain.Brain
	treehouses map[string]*TreeHouse
	nims       map[string]*Nim

	mu      sync.Mutex
	running bool
}

// NewForest creates a new Forest runtime from a configuration file.
func NewForest(configPath string, nc *nats.Conn, b brain.Brain) (*Forest, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return NewForestFromConfig(cfg, nc, b)
}

// NewForestFromConfig creates a new Forest runtime from an existing config.
func NewForestFromConfig(cfg *Config, nc *nats.Conn, b brain.Brain) (*Forest, error) {
	f := &Forest{
		config:     cfg,
		nc:         nc,
		brain:      b,
		treehouses: make(map[string]*TreeHouse),
		nims:       make(map[string]*Nim),
	}

	// Create TreeHouses
	for name, thCfg := range cfg.TreeHouses {
		scriptPath := cfg.ResolvePath(thCfg.Script)
		th, err := NewTreeHouse(thCfg, nc, scriptPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create treehouse %s: %w", name, err)
		}
		f.treehouses[name] = th
	}

	// Create Nims
	for name, nimCfg := range cfg.Nims {
		promptPath := cfg.ResolvePath(nimCfg.Prompt)
		nim, err := NewNim(nimCfg, nc, b, promptPath)
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
