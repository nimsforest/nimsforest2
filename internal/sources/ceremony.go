package sources

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/yourusername/nimsforest/internal/core"
)

// CeremonySourceConfig configures a ceremony source.
type CeremonySourceConfig struct {
	Name      string            // Source name (unique identifier)
	Interval  time.Duration     // How often to trigger (e.g., 30s, 5m, 1h)
	Publishes string            // River subject to publish to
	Payload   map[string]any    // Static payload (optional)
	Script    string            // Lua script path for dynamic payload (optional)
	Hz        int               // WindWaker frequency (default: 90)
}

// CeremonySource triggers events at intervals by counting WindWaker beats.
// This keeps timing synchronized with the forest's conductor.
type CeremonySource struct {
	*core.BaseSource
	config CeremonySourceConfig
	wind   *core.Wind

	// Beat counting
	beatsPerTrigger uint64
	beatCount       atomic.Uint64
	hz              int

	// State
	mu         sync.Mutex
	running    bool
	sub        *nats.Subscription
	triggerCnt atomic.Uint64
	started    time.Time
}

// CeremonyPayload is the payload structure sent to the River.
type CeremonyPayload struct {
	Source      string         `json:"source"`
	Trigger     uint64         `json:"trigger"`     // Trigger count
	Interval    string         `json:"interval"`    // Configured interval
	Timestamp   time.Time      `json:"timestamp"`
	Payload     map[string]any `json:"payload,omitempty"` // Static payload data
	BeatCount   uint64         `json:"beat_count"`        // Beats since last trigger
	Uptime      string         `json:"uptime"`            // Source uptime
}

// Beat matches the WindWaker's beat structure.
type Beat struct {
	Seq uint64 `json:"seq"`
	Ts  int64  `json:"ts"`
	Hz  int    `json:"hz"`
}

// NewCeremonySource creates a new ceremony source.
func NewCeremonySource(cfg CeremonySourceConfig, wind *core.Wind, river *core.River) *CeremonySource {
	// Set defaults
	if cfg.Hz <= 0 {
		cfg.Hz = 90
	}
	if cfg.Interval <= 0 {
		cfg.Interval = time.Minute
	}

	cs := &CeremonySource{
		BaseSource: core.NewBaseSource(cfg.Name, river, cfg.Publishes),
		config:     cfg,
		wind:       wind,
		hz:         cfg.Hz,
	}

	// Calculate beats per trigger: interval_seconds * hz
	cs.beatsPerTrigger = uint64(cfg.Interval.Seconds()) * uint64(cfg.Hz)
	if cs.beatsPerTrigger == 0 {
		cs.beatsPerTrigger = 1 // Minimum 1 beat
	}

	return cs
}

// Type returns the source type.
func (s *CeremonySource) Type() string {
	return "ceremony"
}

// Config returns the source configuration.
func (s *CeremonySource) Config() CeremonySourceConfig {
	return s.config
}

// Start starts the ceremony source, subscribing to dance beats.
func (s *CeremonySource) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.started = time.Now()
	s.mu.Unlock()

	s.SetRunning(true)

	// Subscribe to dance.beat from WindWaker
	sub, err := s.wind.Catch("dance.beat", s.onBeat)
	if err != nil {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		s.SetRunning(false)
		return err
	}

	s.mu.Lock()
	s.sub = sub
	s.mu.Unlock()

	log.Printf("[CeremonySource] Started: %s (interval: %s, beats_per_trigger: %d, publishes: %s)",
		s.Name(), s.config.Interval, s.beatsPerTrigger, s.Publishes())
	return nil
}

// Stop stops the ceremony source.
func (s *CeremonySource) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	sub := s.sub
	s.sub = nil
	s.mu.Unlock()

	// Unsubscribe from dance.beat
	if sub != nil {
		sub.Unsubscribe()
	}

	s.SetRunning(false)

	log.Printf("[CeremonySource] Stopped: %s (triggers: %d, uptime: %v)",
		s.Name(), s.triggerCnt.Load(), time.Since(s.started).Round(time.Second))
	return nil
}

// onBeat handles incoming dance beats.
func (s *CeremonySource) onBeat(leaf core.Leaf) {
	s.mu.Lock()
	running := s.running
	s.mu.Unlock()

	if !running {
		return
	}

	// Increment beat count
	count := s.beatCount.Add(1)

	// Check if we should trigger
	if count >= s.beatsPerTrigger {
		s.trigger()
		s.beatCount.Store(0)
	}
}

// trigger flows an event into River.
func (s *CeremonySource) trigger() {
	triggerNum := s.triggerCnt.Add(1)

	payload := CeremonyPayload{
		Source:    s.Name(),
		Trigger:   triggerNum,
		Interval:  s.config.Interval.String(),
		Timestamp: time.Now(),
		Payload:   s.config.Payload,
		BeatCount: s.beatsPerTrigger,
		Uptime:    time.Since(s.started).Round(time.Second).String(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[CeremonySource] %s: Failed to marshal payload: %v", s.Name(), err)
		return
	}

	if err := s.Flow(data); err != nil {
		log.Printf("[CeremonySource] %s: Failed to flow data: %v", s.Name(), err)
		return
	}

	log.Printf("[CeremonySource] %s: Triggered #%d (interval: %s)",
		s.Name(), triggerNum, s.config.Interval)
}

// Stats returns ceremony statistics.
func (s *CeremonySource) Stats() (triggers uint64, beatCount uint64, uptime time.Duration) {
	s.mu.Lock()
	started := s.started
	s.mu.Unlock()

	return s.triggerCnt.Load(), s.beatCount.Load(), time.Since(started)
}

// BeatsPerTrigger returns how many beats are needed for each trigger.
func (s *CeremonySource) BeatsPerTrigger() uint64 {
	return s.beatsPerTrigger
}

// CurrentBeatCount returns the current beat count.
func (s *CeremonySource) CurrentBeatCount() uint64 {
	return s.beatCount.Load()
}

// TriggerCount returns the total number of triggers.
func (s *CeremonySource) TriggerCount() uint64 {
	return s.triggerCnt.Load()
}

// ForceTrigger manually triggers the ceremony (useful for testing).
func (s *CeremonySource) ForceTrigger() {
	s.trigger()
	s.beatCount.Store(0)
}
