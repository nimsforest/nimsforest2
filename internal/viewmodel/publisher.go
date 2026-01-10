// Package viewmodel provides a view model for the NimsForest cluster state.
package viewmodel

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// PublisherConfig configures the viewmodel publisher.
type PublisherConfig struct {
	// Subject is the NATS subject to publish state to.
	// Default: "forest.viewmodel.state"
	Subject string

	// EventSubject is the NATS subject to publish events to.
	// Default: "forest.viewmodel.events"
	EventSubject string

	// Interval is how often to publish state (if periodic publishing is enabled).
	// Set to 0 to only publish on changes.
	Interval time.Duration

	// OnlyOnChange when true only publishes when state has changed.
	OnlyOnChange bool
}

// Publisher publishes viewmodel state to NATS for external viewers.
// This enables the viewer to run as a separate process without compile-time coupling.
type Publisher struct {
	vm       *ViewModel
	nc       *nats.Conn
	config   PublisherConfig
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.Mutex
	lastHash string // For change detection
}

// PublishedState is the JSON structure published to NATS.
type PublishedState struct {
	Timestamp time.Time           `json:"timestamp"`
	Summary   Summary             `json:"summary"`
	Lands     []*LandViewModel    `json:"lands"`
	Trees     []TreeViewModel     `json:"trees"`
	Treehouses []TreehouseViewModel `json:"treehouses"`
	Nims      []NimViewModel      `json:"nims"`
}

// NewPublisher creates a new Publisher for the viewmodel.
func NewPublisher(vm *ViewModel, nc *nats.Conn, config PublisherConfig) *Publisher {
	// Set defaults
	if config.Subject == "" {
		config.Subject = "forest.viewmodel.state"
	}
	if config.EventSubject == "" {
		config.EventSubject = "forest.viewmodel.events"
	}

	p := &Publisher{
		vm:     vm,
		nc:     nc,
		config: config,
	}

	return p
}

// Start begins publishing viewmodel state.
func (p *Publisher) Start(ctx context.Context) error {
	ctx, p.cancel = context.WithCancel(ctx)

	// Subscribe to viewmodel changes for event publishing
	p.vm.OnChange(func(event Event) {
		p.publishEvent(event)
		if p.config.OnlyOnChange {
			p.publishState()
		}
	})

	// If interval is set, start periodic publishing
	if p.config.Interval > 0 {
		p.wg.Add(1)
		go p.periodicPublish(ctx)
	}

	// Publish initial state
	p.publishState()

	log.Printf("[ViewmodelPublisher] Started publishing to %s", p.config.Subject)
	return nil
}

// Stop stops the publisher.
func (p *Publisher) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
	log.Printf("[ViewmodelPublisher] Stopped")
}

// periodicPublish publishes state on a regular interval.
func (p *Publisher) periodicPublish(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !p.config.OnlyOnChange {
				p.publishState()
			}
		}
	}
}

// publishState publishes the current viewmodel state to NATS.
func (p *Publisher) publishState() {
	p.mu.Lock()
	defer p.mu.Unlock()

	world := p.vm.GetWorld()
	if world == nil {
		return
	}

	state := PublishedState{
		Timestamp:  time.Now(),
		Summary:    world.GetSummary(),
		Lands:      world.Lands(),
		Trees:      world.AllTrees(),
		Treehouses: world.AllTreehouses(),
		Nims:       world.AllNims(),
	}

	data, err := json.Marshal(state)
	if err != nil {
		log.Printf("[ViewmodelPublisher] Failed to marshal state: %v", err)
		return
	}

	// Check for changes if OnlyOnChange is enabled
	if p.config.OnlyOnChange {
		hash := string(data) // Simple comparison; could use actual hash
		if hash == p.lastHash {
			return // No change
		}
		p.lastHash = hash
	}

	if err := p.nc.Publish(p.config.Subject, data); err != nil {
		log.Printf("[ViewmodelPublisher] Failed to publish state: %v", err)
		return
	}
}

// publishEvent publishes a viewmodel event to NATS.
func (p *Publisher) publishEvent(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("[ViewmodelPublisher] Failed to marshal event: %v", err)
		return
	}

	if err := p.nc.Publish(p.config.EventSubject, data); err != nil {
		log.Printf("[ViewmodelPublisher] Failed to publish event: %v", err)
	}
}

// Publish forces an immediate state publication.
func (p *Publisher) Publish() {
	p.publishState()
}

// PublishStateSubject returns the subject used for state publishing.
func (p *Publisher) PublishStateSubject() string {
	return p.config.Subject
}

// PublishEventSubject returns the subject used for event publishing.
func (p *Publisher) PublishEventSubject() string {
	return p.config.EventSubject
}
