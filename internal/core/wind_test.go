package core

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func TestNewWind(t *testing.T) {
	nc := setupTestNATS(t)
	defer nc.Close()

	wind := NewWind(nc)
	if wind == nil {
		t.Fatal("Expected non-nil wind")
	}
	if wind.nc != nc {
		t.Error("Wind should store the NATS connection")
	}
}

func TestWind_Drop(t *testing.T) {
	nc := setupTestNATS(t)
	defer nc.Close()

	wind := NewWind(nc)

	tests := []struct {
		name    string
		leaf    Leaf
		wantErr bool
	}{
		{
			name: "valid leaf",
			leaf: *NewLeaf(
				"test.event",
				[]byte(`{"key": "value"}`),
				"test-source",
			),
			wantErr: false,
		},
		{
			name: "invalid leaf - missing subject",
			leaf: Leaf{
				Data:      json.RawMessage(`{"key": "value"}`),
				Source:    "test-source",
				Timestamp: time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := wind.Drop(tt.leaf)
			if (err != nil) != tt.wantErr {
				t.Errorf("Drop() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWind_Catch(t *testing.T) {
	nc := setupTestNATS(t)
	defer nc.Close()

	wind := NewWind(nc)

	// Channel to receive caught leaves
	caughtLeaves := make(chan Leaf, 1)

	// Subscribe to a subject
	sub, err := wind.Catch("test.catch", func(leaf Leaf) {
		caughtLeaves <- leaf
	})
	if err != nil {
		t.Fatalf("Failed to catch: %v", err)
	}
	defer sub.Unsubscribe()

	// Give subscription time to be ready
	time.Sleep(100 * time.Millisecond)

	// Drop a leaf on the same subject
	leaf := NewLeaf("test.catch", []byte(`{"message": "hello"}`), "test-dropper")
	if err := wind.Drop(*leaf); err != nil {
		t.Fatalf("Failed to drop leaf: %v", err)
	}

	// Wait for the leaf to be caught
	select {
	case caught := <-caughtLeaves:
		if caught.Subject != leaf.Subject {
			t.Errorf("Expected subject %s, got %s", leaf.Subject, caught.Subject)
		}
		if caught.Source != leaf.Source {
			t.Errorf("Expected source %s, got %s", leaf.Source, caught.Source)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for leaf to be caught")
	}
}

func TestWind_CatchWithWildcard(t *testing.T) {
	nc := setupTestNATS(t)
	defer nc.Close()

	wind := NewWind(nc)

	caughtLeaves := make(chan Leaf, 2)

	// Subscribe with wildcard
	sub, err := wind.Catch("test.wildcard.*", func(leaf Leaf) {
		caughtLeaves <- leaf
	})
	if err != nil {
		t.Fatalf("Failed to catch with wildcard: %v", err)
	}
	defer sub.Unsubscribe()

	time.Sleep(100 * time.Millisecond)

	// Drop two leaves with different subjects that match the pattern
	leaf1 := NewLeaf("test.wildcard.one", []byte(`{"num": 1}`), "test")
	leaf2 := NewLeaf("test.wildcard.two", []byte(`{"num": 2}`), "test")

	if err := wind.Drop(*leaf1); err != nil {
		t.Fatalf("Failed to drop leaf1: %v", err)
	}
	if err := wind.Drop(*leaf2); err != nil {
		t.Fatalf("Failed to drop leaf2: %v", err)
	}

	// Catch both leaves
	count := 0
	timeout := time.After(2 * time.Second)

	for count < 2 {
		select {
		case <-caughtLeaves:
			count++
		case <-timeout:
			t.Fatalf("Only caught %d out of 2 leaves", count)
		}
	}
}

func TestWind_CatchWithGreaterThanWildcard(t *testing.T) {
	nc := setupTestNATS(t)
	defer nc.Close()

	wind := NewWind(nc)

	caughtLeaves := make(chan Leaf, 3)

	// Subscribe with > wildcard (matches one or more tokens)
	sub, err := wind.Catch("test.multi.>", func(leaf Leaf) {
		caughtLeaves <- leaf
	})
	if err != nil {
		t.Fatalf("Failed to catch with > wildcard: %v", err)
	}
	defer sub.Unsubscribe()

	time.Sleep(100 * time.Millisecond)

	// Drop leaves with varying depths
	leaves := []*Leaf{
		NewLeaf("test.multi.one", []byte(`{"depth": 1}`), "test"),
		NewLeaf("test.multi.one.two", []byte(`{"depth": 2}`), "test"),
		NewLeaf("test.multi.one.two.three", []byte(`{"depth": 3}`), "test"),
	}

	for _, leaf := range leaves {
		if err := wind.Drop(*leaf); err != nil {
			t.Fatalf("Failed to drop leaf: %v", err)
		}
	}

	// Catch all leaves
	count := 0
	timeout := time.After(2 * time.Second)

	for count < 3 {
		select {
		case <-caughtLeaves:
			count++
		case <-timeout:
			t.Fatalf("Only caught %d out of 3 leaves", count)
		}
	}
}

func TestWind_CatchWithQueue(t *testing.T) {
	nc := setupTestNATS(t)
	defer nc.Close()

	wind := NewWind(nc)

	var mu sync.Mutex
	received := make(map[string]int)
	wg := sync.WaitGroup{}

	// Create two subscribers in the same queue group
	for i := 1; i <= 2; i++ {
		wg.Add(1)
		sub, err := wind.CatchWithQueue("test.queue", "workers", func(leaf Leaf) {
			mu.Lock()
			received["worker"]++
			mu.Unlock()
			wg.Done()
		})
		if err != nil {
			t.Fatalf("Failed to create queue subscriber: %v", err)
		}
		defer sub.Unsubscribe()
	}

	time.Sleep(100 * time.Millisecond)

	// Drop 2 leaves - each should go to a different worker
	for i := 1; i <= 2; i++ {
		leaf := NewLeaf("test.queue", []byte(`{"msg": "test"}`), "test")
		if err := wind.Drop(*leaf); err != nil {
			t.Fatalf("Failed to drop leaf: %v", err)
		}
	}

	// Wait for both to be received
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		mu.Lock()
		count := received["worker"]
		mu.Unlock()
		if count != 2 {
			t.Errorf("Expected 2 leaves to be received, got %d", count)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for leaves to be distributed")
	}
}

func TestWind_DropInvalidLeaf(t *testing.T) {
	nc := setupTestNATS(t)
	defer nc.Close()

	wind := NewWind(nc)

	// Create an invalid leaf (missing subject)
	invalidLeaf := Leaf{
		Data:      json.RawMessage(`{"key": "value"}`),
		Source:    "test",
		Timestamp: time.Now(),
	}

	err := wind.Drop(invalidLeaf)
	if err == nil {
		t.Error("Expected error when dropping invalid leaf")
	}
}

func TestWind_Close(t *testing.T) {
	nc := setupTestNATS(t)

	wind := NewWind(nc)
	wind.Close()

	// Verify connection is closed
	if !nc.IsClosed() {
		t.Error("Expected NATS connection to be closed")
	}
}
