package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"text/template"

	"github.com/nats-io/nats.go"
	"github.com/yourusername/nimsforest/pkg/brain"
)

// Nim is a runtime instance of a Nim configuration.
// It subscribes to a NATS subject, renders a prompt template with message data,
// calls a brain for AI processing, and publishes the result.
type Nim struct {
	config   NimConfig
	nc       *nats.Conn
	brain    brain.Brain
	template *template.Template
	sub      *nats.Subscription

	mu      sync.Mutex
	running bool
}

// NewNim creates a new Nim instance.
func NewNim(cfg NimConfig, nc *nats.Conn, b brain.Brain, promptPath string) (*Nim, error) {
	// Load prompt template
	tmplData, err := os.ReadFile(promptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt %s: %w", promptPath, err)
	}

	tmpl, err := template.New(cfg.Name).Parse(string(tmplData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse prompt template: %w", err)
	}

	return &Nim{
		config:   cfg,
		nc:       nc,
		brain:    b,
		template: tmpl,
	}, nil
}

// Start begins processing messages.
func (n *Nim) Start(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.running {
		return fmt.Errorf("nim %s already running", n.config.Name)
	}

	sub, err := n.nc.Subscribe(n.config.Subscribes, func(msg *nats.Msg) {
		n.handleMessage(ctx, msg)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", n.config.Subscribes, err)
	}

	n.sub = sub
	n.running = true
	log.Printf("[Nim:%s] Started - subscribes: %s, publishes: %s",
		n.config.Name, n.config.Subscribes, n.config.Publishes)
	return nil
}

// Stop stops processing messages.
func (n *Nim) Stop() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.running {
		return nil
	}

	if n.sub != nil {
		if err := n.sub.Unsubscribe(); err != nil {
			log.Printf("[Nim:%s] Error unsubscribing: %v", n.config.Name, err)
		}
		n.sub = nil
	}

	n.running = false
	log.Printf("[Nim:%s] Stopped", n.config.Name)
	return nil
}

// handleMessage processes a single message through the AI brain.
func (n *Nim) handleMessage(ctx context.Context, msg *nats.Msg) {
	// Decode input JSON
	var input map[string]interface{}
	if err := json.Unmarshal(msg.Data, &input); err != nil {
		log.Printf("[Nim:%s] Error decoding message: %v", n.config.Name, err)
		return
	}

	log.Printf("[Nim:%s] Processing message from %s", n.config.Name, n.config.Subscribes)

	// Render prompt template
	var buf bytes.Buffer
	if err := n.template.Execute(&buf, input); err != nil {
		log.Printf("[Nim:%s] Error rendering prompt: %v", n.config.Name, err)
		return
	}
	prompt := buf.String()

	// Call brain
	response, err := n.brain.Ask(ctx, prompt)
	if err != nil {
		log.Printf("[Nim:%s] Error calling brain: %v", n.config.Name, err)
		return
	}

	// Try to parse response as JSON
	var output map[string]interface{}
	if err := json.Unmarshal([]byte(response), &output); err != nil {
		// If not valid JSON, wrap in a response object
		output = map[string]interface{}{
			"response": response,
		}
	}

	// Encode output JSON
	outputData, err := json.Marshal(output)
	if err != nil {
		log.Printf("[Nim:%s] Error encoding output: %v", n.config.Name, err)
		return
	}

	// Publish result
	if err := n.nc.Publish(n.config.Publishes, outputData); err != nil {
		log.Printf("[Nim:%s] Error publishing to %s: %v", n.config.Name, n.config.Publishes, err)
		return
	}

	log.Printf("[Nim:%s] Published result to %s", n.config.Name, n.config.Publishes)
}

// Name returns the Nim name.
func (n *Nim) Name() string {
	return n.config.Name
}

// IsRunning returns whether the Nim is currently running.
func (n *Nim) IsRunning() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.running
}
