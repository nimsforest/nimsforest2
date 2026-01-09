package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"sync"
	"text/template"

	"github.com/nats-io/nats.go"
	"github.com/yourusername/nimsforest/internal/core"
	"github.com/yourusername/nimsforest/pkg/brain"
)

// Nim is a runtime instance of a Nim configuration.
// It uses Wind to subscribe to subjects, renders a prompt template with message data,
// calls a brain for AI processing, and uses Wind to publish results.
// Optionally uses Humus to record state changes.
type Nim struct {
	config   NimConfig
	wind     *core.Wind
	humus    *core.Humus // Optional: for recording state changes
	brain    brain.Brain
	template *template.Template
	sub      *nats.Subscription

	mu      sync.Mutex
	running bool
}

// NewNim creates a new Nim instance using Wind for pub/sub.
func NewNim(cfg NimConfig, wind *core.Wind, b brain.Brain, promptPath string) (*Nim, error) {
	if wind == nil {
		return nil, fmt.Errorf("wind is required")
	}
	if b == nil {
		return nil, fmt.Errorf("brain is required")
	}

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
		wind:     wind,
		brain:    b,
		template: tmpl,
	}, nil
}

// NewNimWithHumus creates a new Nim with Humus for state change recording.
func NewNimWithHumus(cfg NimConfig, wind *core.Wind, humus *core.Humus, b brain.Brain, promptPath string) (*Nim, error) {
	nim, err := NewNim(cfg, wind, b, promptPath)
	if err != nil {
		return nil, err
	}
	nim.humus = humus
	return nim, nil
}

// Start begins processing messages.
func (n *Nim) Start(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.running {
		return fmt.Errorf("nim %s already running", n.config.Name)
	}

	// Use Wind for subscription (with Leaf type)
	sub, err := n.wind.Catch(n.config.Subscribes, func(leaf core.Leaf) {
		n.handleLeaf(ctx, leaf)
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

// handleLeaf processes a Leaf through the AI brain.
func (n *Nim) handleLeaf(ctx context.Context, leaf core.Leaf) {
	// Decode input JSON from leaf data
	var input map[string]interface{}
	if err := json.Unmarshal(leaf.Data, &input); err != nil {
		log.Printf("[Nim:%s] Error decoding leaf data: %v", n.config.Name, err)
		return
	}

	log.Printf("[Nim:%s] Processing leaf from %s (source: %s)",
		n.config.Name, n.config.Subscribes, leaf.Source)

	// Process and get output
	output, err := n.processInput(ctx, input)
	if err != nil {
		log.Printf("[Nim:%s] Error processing: %v", n.config.Name, err)
		return
	}

	// Encode output JSON
	outputData, err := json.Marshal(output)
	if err != nil {
		log.Printf("[Nim:%s] Error encoding output: %v", n.config.Name, err)
		return
	}

	// Resolve dynamic subject (replace {field} placeholders with values from output)
	subject := resolveDynamicSubject(n.config.Publishes, output)

	// Create and drop output leaf via Wind
	outputLeaf := core.NewLeaf(subject, outputData, "nim:"+n.config.Name)
	if err := n.wind.Drop(*outputLeaf); err != nil {
		log.Printf("[Nim:%s] Error dropping leaf to %s: %v",
			n.config.Name, subject, err)
		return
	}

	log.Printf("[Nim:%s] Dropped leaf to %s", n.config.Name, subject)
}

// resolveDynamicSubject replaces {field} placeholders in a subject with values from data.
// For example: "song.telegram.{chat_id}" with data["chat_id"]="123" becomes "song.telegram.123"
// Also handles NATS wildcards: "song.telegram.>" stays as-is if no placeholder found.
func resolveDynamicSubject(subject string, data map[string]interface{}) string {
	// Match {field_name} patterns
	re := regexp.MustCompile(`\{(\w+)\}`)

	result := re.ReplaceAllStringFunc(subject, func(match string) string {
		// Extract field name (remove { and })
		fieldName := match[1 : len(match)-1]

		if value, ok := data[fieldName]; ok {
			return fmt.Sprintf("%v", value)
		}
		// If field not found, keep the placeholder
		return match
	})

	return result
}

// processInput is the common processing logic.
func (n *Nim) processInput(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	// Render prompt template
	var buf bytes.Buffer
	if err := n.template.Execute(&buf, input); err != nil {
		return nil, fmt.Errorf("error rendering prompt: %w", err)
	}
	prompt := buf.String()

	// Call brain
	response, err := n.brain.Ask(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("error calling brain: %w", err)
	}

	// Try to parse response as JSON
	var output map[string]interface{}
	if err := json.Unmarshal([]byte(response), &output); err != nil {
		// If not valid JSON, wrap in a response object
		output = map[string]interface{}{
			"response": response,
		}
	}

	// Optionally record to humus for state tracking
	if n.humus != nil {
		// Extract entity ID from input if available
		entityID := ""
		if id, ok := input["id"].(string); ok {
			entityID = id
		} else if id, ok := input["contact_id"].(string); ok {
			entityID = id
		} else if id, ok := input["entity_id"].(string); ok {
			entityID = id
		}

		if entityID != "" {
			outputData, _ := json.Marshal(output)
			_, err := n.humus.Add(n.config.Name, entityID, "update", outputData)
			if err != nil {
				log.Printf("[Nim:%s] Warning: failed to record to humus: %v", n.config.Name, err)
			}
		}
	}

	return output, nil
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
