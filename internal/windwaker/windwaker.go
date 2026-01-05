// Package windwaker provides the forest conductor that orchestrates synchronized
// dance cycles across all forest components.
//
// The WindWaker publishes a "dance.beat" event through the Wind at a fixed rate
// (default 90Hz). Any component that wants to participate in the synchronized
// ceremony simply catches the beat via wind.Catch("dance.beat", ...).
//
// This design leverages NATS pub/sub for natural fan-out, eliminating the need
// for a centralized dancer registry. Components self-register by subscribing,
// and health is implicit - stop subscribing and you're gone.
package windwaker

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yourusername/nimsforest/internal/core"
)

// Beat represents a single dance beat event.
type Beat struct {
	Seq uint64 `json:"seq"` // Sequence number (monotonic)
	Ts  int64  `json:"ts"`  // Timestamp in nanoseconds
	Hz  int    `json:"hz"`  // Current beat rate
}

// WindWaker is the forest conductor that publishes dance beats.
// It ensures all dancers move in harmony through regular cycles.
type WindWaker struct {
	wind    *core.Wind
	hz      int
	running atomic.Bool
	stopCh  chan struct{}
	wg      sync.WaitGroup

	// Stats
	seq       uint64
	started   time.Time
	beatsSent atomic.Uint64
}

// Config holds WindWaker configuration.
type Config struct {
	// Hz is the beat frequency (default: 90)
	Hz int

	// Subject is the NATS subject for dance beats (default: "dance.beat")
	Subject string
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Hz:      90,
		Subject: "dance.beat",
	}
}

// New creates a new WindWaker with the given Wind and frequency.
// Use hz=90 for standard 90Hz dance rate, or hz=0 for default.
func New(wind *core.Wind, hz int) *WindWaker {
	if hz <= 0 {
		hz = 90
	}
	return &WindWaker{
		wind:   wind,
		hz:     hz,
		stopCh: make(chan struct{}),
	}
}

// NewWithConfig creates a WindWaker from a Config.
func NewWithConfig(wind *core.Wind, cfg Config) *WindWaker {
	if cfg.Hz <= 0 {
		cfg.Hz = 90
	}
	return &WindWaker{
		wind:   wind,
		hz:     cfg.Hz,
		stopCh: make(chan struct{}),
	}
}

// Start begins conducting the dance.
// The WindWaker will publish dance.beat events at the configured frequency.
func (w *WindWaker) Start() error {
	if w.running.Load() {
		return fmt.Errorf("windwaker already running")
	}

	w.running.Store(true)
	w.started = time.Now()
	w.stopCh = make(chan struct{})

	w.wg.Add(1)
	go w.conduct()

	log.Printf("[WindWaker] 🎵 Started conducting at %dHz", w.hz)
	return nil
}

// Stop ends the dance gracefully.
func (w *WindWaker) Stop() {
	if !w.running.Load() {
		return
	}

	w.running.Store(false)
	close(w.stopCh)
	w.wg.Wait()

	log.Printf("[WindWaker] 🎵 Stopped after %d beats (uptime: %v)",
		w.beatsSent.Load(), time.Since(w.started).Round(time.Second))
}

// IsRunning returns whether the WindWaker is currently conducting.
func (w *WindWaker) IsRunning() bool {
	return w.running.Load()
}

// Hz returns the current beat frequency.
func (w *WindWaker) Hz() int {
	return w.hz
}

// BeatsSent returns the total number of beats published.
func (w *WindWaker) BeatsSent() uint64 {
	return w.beatsSent.Load()
}

// Uptime returns how long the WindWaker has been running.
func (w *WindWaker) Uptime() time.Duration {
	if !w.running.Load() {
		return 0
	}
	return time.Since(w.started)
}

// conduct is the main loop that publishes dance beats.
func (w *WindWaker) conduct() {
	defer w.wg.Done()

	interval := time.Second / time.Duration(w.hz)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case now := <-ticker.C:
			w.seq++
			w.publishBeat(now)
		}
	}
}

// publishBeat sends a single dance beat through the wind.
func (w *WindWaker) publishBeat(t time.Time) {
	beat := Beat{
		Seq: w.seq,
		Ts:  t.UnixNano(),
		Hz:  w.hz,
	}

	data, err := json.Marshal(beat)
	if err != nil {
		log.Printf("[WindWaker] Failed to marshal beat: %v", err)
		return
	}

	leaf := core.NewLeaf("dance.beat", data, "windwaker")
	w.wind.Drop(*leaf)
	w.beatsSent.Add(1)
}

// Subject constants for dance events.
const (
	SubjectDanceBeat = "dance.beat"
)
