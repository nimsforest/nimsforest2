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
// It subscribes to a NATS subject, processes messages through a Lua script,
// and publishes results to another subject.
type TreeHouse struct {
	config TreeHouseConfig
	wind   *core.Wind
	nc     *nats.Conn // Fallback if Wind not provided
	vm     *LuaVM
	sub    *nats.Subscription

	mu      sync.Mutex
	running bool
}

// NewTreeHouse creates a new TreeHouse instance using Wind for pub/sub.
func NewTreeHouse(cfg TreeHouseConfig, wind *core.Wind, scriptPath string) (*TreeHouse, error) {
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

// NewTreeHouseWithConn creates a new TreeHouse instance using raw NATS connection.
// This is useful for testing or when Wind is not available.
func NewTreeHouseWithConn(cfg TreeHouseConfig, nc *nats.Conn, scriptPath string) (*TreeHouse, error) {
	vm := NewLuaVM()

	if err := vm.LoadScript(scriptPath); err != nil {
		vm.Close()
		return nil, fmt.Errorf("failed to load script %s: %w", scriptPath, err)
	}

	return &TreeHouse{
		config: cfg,
		nc:     nc,
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

	var err error
	if th.wind != nil {
		// Use Wind for subscription (with Leaf type)
		th.sub, err = th.wind.Catch(th.config.Subscribes, func(leaf core.Leaf) {
			th.handleLeaf(ctx, leaf)
		})
	} else if th.nc != nil {
		// Fallback to raw NATS
		th.sub, err = th.nc.Subscribe(th.config.Subscribes, func(msg *nats.Msg) {
			th.handleMessage(ctx, msg)
		})
	} else {
		return fmt.Errorf("treehouse %s has no connection (need Wind or NATS conn)", th.config.Name)
	}

	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", th.config.Subscribes, err)
	}

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

	// Create and drop output leaf
	outputLeaf := core.NewLeaf(th.config.Publishes, outputData, "treehouse:"+th.config.Name)
	if err := th.wind.Drop(*outputLeaf); err != nil {
		log.Printf("[TreeHouse:%s] Error dropping leaf to %s: %v",
			th.config.Name, th.config.Publishes, err)
		return
	}

	log.Printf("[TreeHouse:%s] Dropped leaf to %s", th.config.Name, th.config.Publishes)
}

// handleMessage processes a raw NATS message through the Lua script.
func (th *TreeHouse) handleMessage(ctx context.Context, msg *nats.Msg) {
	// Decode input JSON
	var input map[string]interface{}
	if err := json.Unmarshal(msg.Data, &input); err != nil {
		log.Printf("[TreeHouse:%s] Error decoding message: %v", th.config.Name, err)
		return
	}

	log.Printf("[TreeHouse:%s] Processing message from %s", th.config.Name, th.config.Subscribes)

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

	// Publish result
	if err := th.nc.Publish(th.config.Publishes, outputData); err != nil {
		log.Printf("[TreeHouse:%s] Error publishing to %s: %v", th.config.Name, th.config.Publishes, err)
		return
	}

	log.Printf("[TreeHouse:%s] Published result to %s", th.config.Name, th.config.Publishes)
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
