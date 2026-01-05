package windwaker

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/yourusername/nimsforest/internal/core"
)

// setupTest creates an embedded NATS server and Wind for testing.
func setupTest(t *testing.T) (*core.Wind, func()) {
	t.Helper()

	opts := &server.Options{
		Host: "127.0.0.1",
		Port: -1, // Random port
	}
	ns, err := server.NewServer(opts)
	if err != nil {
		t.Fatalf("Failed to create NATS server: %v", err)
	}
	go ns.Start()
	if !ns.ReadyForConnections(5 * time.Second) {
		t.Fatal("NATS server not ready")
	}

	nc, err := nats.Connect(ns.ClientURL())
	if err != nil {
		ns.Shutdown()
		t.Fatalf("Failed to connect to NATS: %v", err)
	}

	wind := core.NewWind(nc)

	cleanup := func() {
		nc.Close()
		ns.Shutdown()
	}

	return wind, cleanup
}

func TestWindWakerStartStop(t *testing.T) {
	wind, cleanup := setupTest(t)
	defer cleanup()

	ww := New(wind, 100) // 100Hz for faster test

	if ww.IsRunning() {
		t.Error("WindWaker should not be running before Start()")
	}

	if err := ww.Start(); err != nil {
		t.Fatalf("Failed to start WindWaker: %v", err)
	}

	if !ww.IsRunning() {
		t.Error("WindWaker should be running after Start()")
	}

	// Let it run for a bit
	time.Sleep(50 * time.Millisecond)

	ww.Stop()

	if ww.IsRunning() {
		t.Error("WindWaker should not be running after Stop()")
	}

	beats := ww.BeatsSent()
	if beats == 0 {
		t.Error("WindWaker should have sent some beats")
	}
	t.Logf("Sent %d beats", beats)
}

func TestWindWakerBeatsReceived(t *testing.T) {
	wind, cleanup := setupTest(t)
	defer cleanup()

	var received atomic.Int32

	// Subscribe to beats before starting
	err := OnBeat(wind, "test-dancer", func(beat Beat) error {
		received.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	ww := New(wind, 100) // 100Hz
	if err := ww.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	// Let it run for ~100ms = ~10 beats at 100Hz
	time.Sleep(100 * time.Millisecond)
	ww.Stop()

	count := received.Load()
	if count < 5 {
		t.Errorf("Expected at least 5 beats, got %d", count)
	}
	t.Logf("Received %d beats", count)
}

func TestWindWakerBeatContent(t *testing.T) {
	wind, cleanup := setupTest(t)
	defer cleanup()

	var lastBeat Beat
	var beatReceived atomic.Bool

	err := OnBeat(wind, "test-dancer", func(beat Beat) error {
		lastBeat = beat
		beatReceived.Store(true)
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	ww := New(wind, 50) // 50Hz
	if err := ww.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	// Wait for at least one beat
	deadline := time.Now().Add(200 * time.Millisecond)
	for !beatReceived.Load() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	ww.Stop()

	if !beatReceived.Load() {
		t.Fatal("Did not receive any beat")
	}

	if lastBeat.Seq == 0 {
		t.Error("Beat sequence should be > 0")
	}
	if lastBeat.Ts == 0 {
		t.Error("Beat timestamp should be > 0")
	}
	if lastBeat.Hz != 50 {
		t.Errorf("Beat Hz should be 50, got %d", lastBeat.Hz)
	}

	t.Logf("Received beat: seq=%d ts=%d hz=%d", lastBeat.Seq, lastBeat.Ts, lastBeat.Hz)
}

func TestDancerInterface(t *testing.T) {
	wind, cleanup := setupTest(t)
	defer cleanup()

	var danceCount atomic.Int32

	dancer := NewSimpleDancer("test-dancer", func(beat Beat) error {
		danceCount.Add(1)
		return nil
	})

	if dancer.ID() != "test-dancer" {
		t.Errorf("Expected ID 'test-dancer', got '%s'", dancer.ID())
	}

	err := RegisterDancer(wind, dancer)
	if err != nil {
		t.Fatalf("Failed to register dancer: %v", err)
	}

	ww := New(wind, 100)
	if err := ww.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	ww.Stop()

	count := danceCount.Load()
	if count < 5 {
		t.Errorf("Dancer should have danced at least 5 times, got %d", count)
	}
	t.Logf("Dancer danced %d times", count)
}

func TestMultipleDancers(t *testing.T) {
	wind, cleanup := setupTest(t)
	defer cleanup()

	var count1, count2, count3 atomic.Int32

	OnBeat(wind, "dancer-1", func(beat Beat) error {
		count1.Add(1)
		return nil
	})
	OnBeat(wind, "dancer-2", func(beat Beat) error {
		count2.Add(1)
		return nil
	})
	OnBeat(wind, "dancer-3", func(beat Beat) error {
		count3.Add(1)
		return nil
	})

	ww := New(wind, 100)
	ww.Start()
	time.Sleep(100 * time.Millisecond)
	ww.Stop()

	c1, c2, c3 := count1.Load(), count2.Load(), count3.Load()

	// All dancers should receive roughly the same number of beats
	if c1 < 5 || c2 < 5 || c3 < 5 {
		t.Errorf("All dancers should have received at least 5 beats: %d, %d, %d", c1, c2, c3)
	}

	t.Logf("Dancer counts: %d, %d, %d", c1, c2, c3)
}

func TestWindWakerDoubleStart(t *testing.T) {
	wind, cleanup := setupTest(t)
	defer cleanup()

	ww := New(wind, 100)
	
	if err := ww.Start(); err != nil {
		t.Fatalf("First start failed: %v", err)
	}

	err := ww.Start()
	if err == nil {
		t.Error("Second start should return error")
	}

	ww.Stop()
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	
	if cfg.Hz != 90 {
		t.Errorf("Default Hz should be 90, got %d", cfg.Hz)
	}
	if cfg.Subject != "dance.beat" {
		t.Errorf("Default subject should be 'dance.beat', got '%s'", cfg.Subject)
	}
}
