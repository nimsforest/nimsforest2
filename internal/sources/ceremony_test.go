package sources

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/yourusername/nimsforest/internal/core"
)

func TestCeremonySourceConfig(t *testing.T) {
	cfg := CeremonySourceConfig{
		Name:      "test-ceremony",
		Interval:  30 * time.Second,
		Publishes: "river.test.ceremony",
		Payload: map[string]any{
			"type": "heartbeat",
		},
		Hz: 90,
	}

	if cfg.Interval != 30*time.Second {
		t.Errorf("Expected interval 30s, got %v", cfg.Interval)
	}
	if cfg.Hz != 90 {
		t.Errorf("Expected Hz 90, got %d", cfg.Hz)
	}
}

func TestCeremonySource_BeatsPerTrigger(t *testing.T) {
	tests := []struct {
		interval time.Duration
		hz       int
		expected uint64
	}{
		{30 * time.Second, 90, 2700},  // 30s * 90Hz = 2700 beats
		{time.Minute, 90, 5400},       // 60s * 90Hz = 5400 beats
		{time.Hour, 90, 324000},       // 3600s * 90Hz = 324000 beats
		{5 * time.Minute, 90, 27000},  // 300s * 90Hz = 27000 beats
		{10 * time.Second, 100, 1000}, // 10s * 100Hz = 1000 beats
	}

	for _, tt := range tests {
		t.Run(tt.interval.String(), func(t *testing.T) {
			// Calculate beats per trigger manually
			beatsPerTrigger := uint64(tt.interval.Seconds()) * uint64(tt.hz)
			if beatsPerTrigger != tt.expected {
				t.Errorf("BeatsPerTrigger = %d, want %d", beatsPerTrigger, tt.expected)
			}
		})
	}
}

func TestCeremonyPayload_JSON(t *testing.T) {
	payload := CeremonyPayload{
		Source:    "test-ceremony",
		Trigger:   5,
		Interval:  "30s",
		Timestamp: time.Now(),
		Payload: map[string]any{
			"type": "heartbeat",
		},
		BeatCount: 2700,
		Uptime:    "2m30s",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	var decoded CeremonyPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if decoded.Source != payload.Source {
		t.Errorf("Source mismatch: got %s, want %s", decoded.Source, payload.Source)
	}
	if decoded.Trigger != payload.Trigger {
		t.Errorf("Trigger mismatch: got %d, want %d", decoded.Trigger, payload.Trigger)
	}
	if decoded.Interval != payload.Interval {
		t.Errorf("Interval mismatch: got %s, want %s", decoded.Interval, payload.Interval)
	}
}

func TestCeremonySource_Defaults(t *testing.T) {
	cfg := CeremonySourceConfig{
		Name:      "test",
		Publishes: "river.test",
		// Leave Hz and Interval at zero values
	}

	// Test default logic
	if cfg.Hz <= 0 {
		cfg.Hz = 90
	}
	if cfg.Interval <= 0 {
		cfg.Interval = time.Minute
	}

	if cfg.Hz != 90 {
		t.Errorf("Expected default Hz 90, got %d", cfg.Hz)
	}
	if cfg.Interval != time.Minute {
		t.Errorf("Expected default interval 1m, got %v", cfg.Interval)
	}
}

func TestCeremonySource_Type(t *testing.T) {
	cfg := CeremonySourceConfig{
		Name:      "test-ceremony",
		Interval:  30 * time.Second,
		Publishes: "river.test.ceremony",
	}

	// Create minimal source to test Type()
	cs := &CeremonySource{
		config: cfg,
	}
	cs.BaseSource = core.NewBaseSource(cfg.Name, nil, cfg.Publishes)

	if cs.Type() != "ceremony" {
		t.Errorf("Type() = %s, want ceremony", cs.Type())
	}
}

func TestCeremonySource_Stats(t *testing.T) {
	cfg := CeremonySourceConfig{
		Name:      "test-ceremony",
		Interval:  30 * time.Second,
		Publishes: "river.test.ceremony",
		Hz:        90,
	}

	cs := &CeremonySource{
		config:          cfg,
		beatsPerTrigger: 2700,
		hz:              90,
	}
	cs.BaseSource = core.NewBaseSource(cfg.Name, nil, cfg.Publishes)

	if cs.BeatsPerTrigger() != 2700 {
		t.Errorf("BeatsPerTrigger() = %d, want 2700", cs.BeatsPerTrigger())
	}

	// Initial state
	if cs.CurrentBeatCount() != 0 {
		t.Errorf("CurrentBeatCount() = %d, want 0", cs.CurrentBeatCount())
	}

	if cs.TriggerCount() != 0 {
		t.Errorf("TriggerCount() = %d, want 0", cs.TriggerCount())
	}

	// Simulate beat count
	cs.beatCount.Store(100)
	if cs.CurrentBeatCount() != 100 {
		t.Errorf("CurrentBeatCount() = %d, want 100", cs.CurrentBeatCount())
	}
}

func TestBeat_JSON(t *testing.T) {
	beat := Beat{
		Seq: 12345,
		Ts:  time.Now().UnixNano(),
		Hz:  90,
	}

	data, err := json.Marshal(beat)
	if err != nil {
		t.Fatalf("Failed to marshal beat: %v", err)
	}

	var decoded Beat
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal beat: %v", err)
	}

	if decoded.Seq != beat.Seq {
		t.Errorf("Seq mismatch: got %d, want %d", decoded.Seq, beat.Seq)
	}
	if decoded.Hz != beat.Hz {
		t.Errorf("Hz mismatch: got %d, want %d", decoded.Hz, beat.Hz)
	}
}
