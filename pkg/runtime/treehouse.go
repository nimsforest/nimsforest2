package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/nats-io/nats.go"
)

// TreeHouse is a runtime instance of a TreeHouse configuration.
// It subscribes to a NATS subject, processes messages through a Lua script,
// and publishes results to another subject.
type TreeHouse struct {
	config TreeHouseConfig
	nc     *nats.Conn
	vm     *LuaVM
	sub    *nats.Subscription

	mu      sync.Mutex
	running bool
}

// NewTreeHouse creates a new TreeHouse instance.
func NewTreeHouse(cfg TreeHouseConfig, nc *nats.Conn, scriptPath string) (*TreeHouse, error) {
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

	sub, err := th.nc.Subscribe(th.config.Subscribes, func(msg *nats.Msg) {
		th.handleMessage(ctx, msg)
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

// handleMessage processes a single message through the Lua script.
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
