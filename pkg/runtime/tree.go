package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/yourusername/nimsforest/internal/core"
)

// Tree is a runtime instance of a Tree configuration.
// It watches River (JetStream) for external data, processes it through a Lua script,
// and publishes structured Leaves to Wind.
//
// Trees are the "edge" - they convert unstructured external data into typed events.
// Think: webhooks, API data, sensor readings → structured domain events.
type Tree struct {
	config TreeConfig
	wind   *core.Wind
	river  *core.River
	vm     *LuaVM

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
}

// NewTree creates a new Tree instance.
func NewTree(cfg TreeConfig, wind *core.Wind, river *core.River, scriptPath string) (*Tree, error) {
	if wind == nil {
		return nil, fmt.Errorf("wind is required")
	}
	if river == nil {
		return nil, fmt.Errorf("river is required for trees")
	}

	vm := NewLuaVM()

	if err := vm.LoadScript(scriptPath); err != nil {
		vm.Close()
		return nil, fmt.Errorf("failed to load script %s: %w", scriptPath, err)
	}

	return &Tree{
		config: cfg,
		wind:   wind,
		river:  river,
		vm:     vm,
	}, nil
}

// Start begins watching the River and processing data.
func (t *Tree) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return fmt.Errorf("tree %s already running", t.config.Name)
	}

	childCtx, cancel := context.WithCancel(ctx)
	t.cancel = cancel

	// Watch River for data
	err := t.river.Observe(t.config.Watches, func(data core.RiverData) {
		t.handleRiverData(childCtx, data)
	})
	if err != nil {
		cancel()
		return fmt.Errorf("failed to observe %s: %w", t.config.Watches, err)
	}

	t.running = true
	log.Printf("[Tree:%s] Started - watches: %s, publishes: %s",
		t.config.Name, t.config.Watches, t.config.Publishes)
	return nil
}

// Stop stops the tree from processing.
func (t *Tree) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return nil
	}

	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
	}

	if t.vm != nil {
		t.vm.Close()
		t.vm = nil
	}

	t.running = false
	log.Printf("[Tree:%s] Stopped", t.config.Name)
	return nil
}

// handleRiverData processes incoming River data through the Lua script.
func (t *Tree) handleRiverData(ctx context.Context, data core.RiverData) {
	// Decode input JSON
	var input map[string]interface{}
	if err := json.Unmarshal(data.Data, &input); err != nil {
		log.Printf("[Tree:%s] Error decoding river data: %v", t.config.Name, err)
		return
	}

	// Add metadata
	input["_subject"] = data.Subject
	input["_source"] = "river"

	log.Printf("[Tree:%s] Processing data from %s", t.config.Name, data.Subject)

	// Call process(input) in Lua
	t.mu.Lock()
	output, err := t.vm.CallProcess(input)
	t.mu.Unlock()

	if err != nil {
		log.Printf("[Tree:%s] Error in process(): %v", t.config.Name, err)
		return
	}

	// If Lua returns nil, skip publishing (filtered out)
	if output == nil {
		log.Printf("[Tree:%s] Filtered out (process returned nil)", t.config.Name)
		return
	}

	// Encode output JSON
	outputData, err := json.Marshal(output)
	if err != nil {
		log.Printf("[Tree:%s] Error encoding output: %v", t.config.Name, err)
		return
	}

	// Create and drop Leaf via Wind
	leaf := core.NewLeaf(t.config.Publishes, outputData, "tree:"+t.config.Name)
	if err := t.wind.Drop(*leaf); err != nil {
		log.Printf("[Tree:%s] Error dropping leaf to %s: %v",
			t.config.Name, t.config.Publishes, err)
		return
	}

	log.Printf("[Tree:%s] Dropped leaf to %s", t.config.Name, t.config.Publishes)
}

// Name returns the Tree name.
func (t *Tree) Name() string {
	return t.config.Name
}

// IsRunning returns whether the Tree is currently running.
func (t *Tree) IsRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}
