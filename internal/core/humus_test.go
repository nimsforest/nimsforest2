package core

import (
	"sync"
	"testing"
	"time"
)

func TestNewHumus(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	// Clean up any existing stream
	js.DeleteStream("HUMUS")

	humus, err := NewHumus(js)
	if err != nil {
		t.Fatalf("Failed to create humus: %v", err)
	}
	if humus == nil {
		t.Fatal("Expected non-nil humus")
	}

	// Verify stream was created
	info, err := humus.StreamInfo()
	if err != nil {
		t.Fatalf("Failed to get stream info: %v", err)
	}
	if info.Config.Name != "HUMUS" {
		t.Errorf("Expected stream name HUMUS, got %s", info.Config.Name)
	}
}

func TestHumus_Add(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	js.DeleteStream("HUMUS")
	humus, err := NewHumus(js)
	if err != nil {
		t.Fatalf("Failed to create humus: %v", err)
	}

	tests := []struct {
		name    string
		nimName string
		entity  string
		action  string
		data    []byte
		wantErr bool
	}{
		{
			name:    "valid create",
			nimName: "test-nim",
			entity:  "tasks/test-1",
			action:  "create",
			data:    []byte(`{"status": "pending"}`),
			wantErr: false,
		},
		{
			name:    "valid update",
			nimName: "test-nim",
			entity:  "tasks/test-1",
			action:  "update",
			data:    []byte(`{"status": "complete"}`),
			wantErr: false,
		},
		{
			name:    "valid delete",
			nimName: "test-nim",
			entity:  "tasks/test-1",
			action:  "delete",
			data:    []byte{}, // Empty data is ok for delete
			wantErr: false,
		},
		{
			name:    "empty nim name",
			nimName: "",
			entity:  "tasks/test",
			action:  "create",
			data:    []byte(`{}`),
			wantErr: true,
		},
		{
			name:    "empty entity",
			nimName: "test-nim",
			entity:  "",
			action:  "create",
			data:    []byte(`{}`),
			wantErr: true,
		},
		{
			name:    "invalid action",
			nimName: "test-nim",
			entity:  "tasks/test",
			action:  "invalid",
			data:    []byte(`{}`),
			wantErr: true,
		},
		{
			name:    "empty data for create",
			nimName: "test-nim",
			entity:  "tasks/test",
			action:  "create",
			data:    []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slot, err := humus.Add(tt.nimName, tt.entity, tt.action, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && slot == 0 {
				t.Error("Expected non-zero slot for successful Add")
			}
		})
	}
}

func TestHumus_Decompose(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	js.DeleteStream("HUMUS")
	humus, err := NewHumus(js)
	if err != nil {
		t.Fatalf("Failed to create humus: %v", err)
	}

	// Channel to receive decomposed compost
	composts := make(chan Compost, 3)
	var wg sync.WaitGroup

	// Start decomposer
	err = humus.Decompose(func(compost Compost) {
		composts <- compost
		wg.Done()
	})
	if err != nil {
		t.Fatalf("Failed to start decomposer: %v", err)
	}

	// Give decomposer time to be ready
	time.Sleep(200 * time.Millisecond)

	// Add some compost
	testData := []struct {
		entity string
		action string
		data   string
	}{
		{"tasks/task-1", "create", `{"status": "pending"}`},
		{"tasks/task-2", "create", `{"status": "pending"}`},
		{"tasks/task-1", "update", `{"status": "complete"}`},
	}

	wg.Add(len(testData))
	for _, td := range testData {
		_, err := humus.Add("test-nim", td.entity, td.action, []byte(td.data))
		if err != nil {
			t.Fatalf("Failed to add compost: %v", err)
		}
	}

	// Wait for all to be decomposed
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success - check we received the right number
		if len(composts) != len(testData) {
			t.Errorf("Expected %d composts, got %d", len(testData), len(composts))
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for decomposition")
	}
}

func TestHumus_DecomposeWithConsumer(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	js.DeleteStream("HUMUS")
	humus, err := NewHumus(js)
	if err != nil {
		t.Fatalf("Failed to create humus: %v", err)
	}

	composts := make(chan Compost, 1)

	// Start decomposer with custom consumer name
	err = humus.DecomposeWithConsumer("test-consumer", func(compost Compost) {
		composts <- compost
	})
	if err != nil {
		t.Fatalf("Failed to start decomposer: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Add compost
	_, err = humus.Add("test-nim", "tasks/test", "create", []byte(`{"test": true}`))
	if err != nil {
		t.Fatalf("Failed to add compost: %v", err)
	}

	// Verify decomposition
	select {
	case compost := <-composts:
		if compost.Entity != "tasks/test" {
			t.Errorf("Expected entity tasks/test, got %s", compost.Entity)
		}
		if compost.Action != "create" {
			t.Errorf("Expected action create, got %s", compost.Action)
		}
		if compost.NimName != "test-nim" {
			t.Errorf("Expected nim test-nim, got %s", compost.NimName)
		}
		if compost.Slot == 0 {
			t.Error("Expected non-zero slot")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for decomposition")
	}
}

func TestHumus_StreamInfo(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	js.DeleteStream("HUMUS")
	humus, err := NewHumus(js)
	if err != nil {
		t.Fatalf("Failed to create humus: %v", err)
	}

	info, err := humus.StreamInfo()
	if err != nil {
		t.Fatalf("Failed to get stream info: %v", err)
	}

	if info.Config.Name != "HUMUS" {
		t.Errorf("Expected stream name HUMUS, got %s", info.Config.Name)
	}

	// Add compost and verify count increases
	humus.Add("test-nim", "test/entity", "create", []byte(`{"test": true}`))

	info, err = humus.StreamInfo()
	if err != nil {
		t.Fatalf("Failed to get updated stream info: %v", err)
	}

	if info.State.Msgs < 1 {
		t.Errorf("Expected at least 1 message in stream, got %d", info.State.Msgs)
	}
}

func TestHumus_OrderingGuarantee(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	js.DeleteStream("HUMUS")
	humus, err := NewHumus(js)
	if err != nil {
		t.Fatalf("Failed to create humus: %v", err)
	}

	var mu sync.Mutex
	var slots []uint64
	var wg sync.WaitGroup

	// Start decomposer
	err = humus.Decompose(func(compost Compost) {
		mu.Lock()
		slots = append(slots, compost.Slot)
		mu.Unlock()
		wg.Done()
	})
	if err != nil {
		t.Fatalf("Failed to start decomposer: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Add multiple composts
	numComposts := 5
	wg.Add(numComposts)
	for i := 0; i < numComposts; i++ {
		_, err := humus.Add("test-nim", "tasks/test", "update", []byte(`{"count": 1}`))
		if err != nil {
			t.Fatalf("Failed to add compost: %v", err)
		}
	}

	// Wait for all to be processed
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Verify slots are in order
		mu.Lock()
		defer mu.Unlock()

		if len(slots) != numComposts {
			t.Errorf("Expected %d slots, got %d", numComposts, len(slots))
		}

		for i := 1; i < len(slots); i++ {
			if slots[i] <= slots[i-1] {
				t.Errorf("Slots not in order: %v", slots)
				break
			}
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for decomposition")
	}
}
