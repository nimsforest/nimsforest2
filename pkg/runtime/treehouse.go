package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/yourusername/nimsforest/internal/core"
)

// TreeHouse is a runtime instance of a TreeHouse configuration.
// It uses Wind to subscribe to subjects, processes messages through a Lua script,
// and uses Wind to publish results.
type TreeHouse struct {
	config TreeHouseConfig
	wind   *core.Wind
	vm     *LuaVM
	sub    *nats.Subscription

	mu      sync.Mutex
	running bool
}

// NewTreeHouse creates a new TreeHouse instance using Wind for pub/sub.
func NewTreeHouse(cfg TreeHouseConfig, wind *core.Wind, scriptPath string) (*TreeHouse, error) {
	if wind == nil {
		return nil, fmt.Errorf("wind is required")
	}

	vm := NewLuaVM()

	if err := vm.LoadScript(scriptPath); err != nil {
		vm.Close()
		return nil, fmt.Errorf("failed to load script %s: %w", scriptPath, err)
	}

	return &TreeHouse{
		config: cfg,
		wind:   wind,
		vm:     vm,
	}, nil
}

// Start begins processing messages.
func (th *TreeHouse) Start(ctx context.Context) error {
	th.mu.Lock()
	defer th.mu.Unlock()

	if th.running {
		return fmt.Errorf("treehouse %s already running", th.config.Name)
	}

	// Use Wind for subscription (with Leaf type)
	sub, err := th.wind.Catch(th.config.Subscribes, func(leaf core.Leaf) {
		th.handleLeaf(ctx, leaf)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", th.config.Subscribes, err)
	}

	th.sub = sub
	th.running = true
	log.Printf("[TreeHouse:%s] Started - subscribes: %s, publishes: %s",
		th.config.Name, th.config.Subscribes, th.config.Publishes)
	return nil
}

// Stop stops processing messages.
func (th *TreeHouse) Stop() error {
	th.mu.Lock()
	defer th.mu.Unlock()

	if !th.running {
		return nil
	}

	if th.sub != nil {
		if err := th.sub.Unsubscribe(); err != nil {
			log.Printf("[TreeHouse:%s] Error unsubscribing: %v", th.config.Name, err)
		}
		th.sub = nil
	}

	if th.vm != nil {
		th.vm.Close()
		th.vm = nil
	}

	th.running = false
	log.Printf("[TreeHouse:%s] Stopped", th.config.Name)
	return nil
}

// handleLeaf processes a Leaf through the Lua script.
func (th *TreeHouse) handleLeaf(ctx context.Context, leaf core.Leaf) {
	// Decode input JSON from leaf data
	var input map[string]interface{}
	if err := json.Unmarshal(leaf.Data, &input); err != nil {
		log.Printf("[TreeHouse:%s] Error decoding leaf data: %v", th.config.Name, err)
		return
	}

	log.Printf("[TreeHouse:%s] Processing leaf from %s (source: %s)",
		th.config.Name, th.config.Subscribes, leaf.Source)

	// Call process(input) in Lua
	th.mu.Lock()
	output, err := th.vm.CallProcess(input)
	th.mu.Unlock()

	if err != nil {
		log.Printf("[TreeHouse:%s] Error in process(): %v", th.config.Name, err)
		return
	}

	// Encode output JSON
	outputData, err := json.Marshal(output)
	if err != nil {
		log.Printf("[TreeHouse:%s] Error encoding output: %v", th.config.Name, err)
		return
	}

	// Create and drop output leaf via Wind
	outputLeaf := core.NewLeaf(th.config.Publishes, outputData, "treehouse:"+th.config.Name)
	if err := th.wind.Drop(*outputLeaf); err != nil {
		log.Printf("[TreeHouse:%s] Error dropping leaf to %s: %v",
			th.config.Name, th.config.Publishes, err)
		return
	}

	log.Printf("[TreeHouse:%s] Dropped leaf to %s", th.config.Name, th.config.Publishes)
}

// Name returns the TreeHouse name.
func (th *TreeHouse) Name() string {
	return th.config.Name
}

// IsRunning returns whether the TreeHouse is currently running.
func (th *TreeHouse) IsRunning() bool {
	th.mu.Lock()
	defer th.mu.Unlock()
	return th.running
}
