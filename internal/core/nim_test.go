package core

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// MockNim implements the Nim interface for testing
type MockNim struct {
	*BaseNim
	subjects []string
	handleFn func(ctx context.Context, leaf Leaf) error
}

func (m *MockNim) Subjects() []string {
	return m.subjects
}

func (m *MockNim) Handle(ctx context.Context, leaf Leaf) error {
	if m.handleFn != nil {
		return m.handleFn(ctx, leaf)
	}
	return nil
}

func (m *MockNim) Start(ctx context.Context) error {
	for _, subject := range m.subjects {
		err := m.Catch(subject, func(leaf Leaf) {
			m.Handle(ctx, leaf)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MockNim) Stop() error {
	return nil
}

func TestNewBaseNim(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	wind := NewWind(nc)
	js.DeleteStream("HUMUS")
	humus, _ := NewHumus(js)
	js.DeleteKeyValue("SOIL")
	soil, _ := NewSoil(js)

	base := NewBaseNim("test-nim", wind, humus, soil)
	if base == nil {
		t.Fatal("Expected non-nil base nim")
	}
	if base.Name() != "test-nim" {
		t.Errorf("Expected name 'test-nim', got '%s'", base.Name())
	}
}

func TestBaseNim_Leaf(t *testing.T) {
	nc := setupTestNATS(t)
	defer nc.Close()

	js, _ := nc.JetStream()
	wind := NewWind(nc)
	js.DeleteStream("HUMUS")
	humus, _ := NewHumus(js)
	js.DeleteKeyValue("SOIL")
	soil, _ := NewSoil(js)

	base := NewBaseNim("test-nim", wind, humus, soil)

	// Create channel to catch the leaf
	caughtLeaves := make(chan Leaf, 1)
	sub, err := wind.Catch("test.leaf", func(leaf Leaf) {
		caughtLeaves <- leaf
	})
	if err != nil {
		t.Fatalf("Failed to catch: %v", err)
	}
	defer sub.Unsubscribe()

	time.Sleep(100 * time.Millisecond)

	// Drop a leaf using the helper
	err = base.Leaf("test.leaf", []byte(`{"msg": "hello"}`))
	if err != nil {
		t.Fatalf("Failed to drop leaf: %v", err)
	}

	// Verify we caught it
	select {
	case leaf := <-caughtLeaves:
		if leaf.Subject != "test.leaf" {
			t.Errorf("Expected subject 'test.leaf', got '%s'", leaf.Subject)
		}
		if leaf.Source != "test-nim" {
			t.Errorf("Expected source 'test-nim', got '%s'", leaf.Source)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for leaf")
	}
}

func TestBaseNim_LeafStruct(t *testing.T) {
	nc := setupTestNATS(t)
	defer nc.Close()

	js, _ := nc.JetStream()
	wind := NewWind(nc)
	js.DeleteStream("HUMUS")
	humus, _ := NewHumus(js)
	js.DeleteKeyValue("SOIL")
	soil, _ := NewSoil(js)

	base := NewBaseNim("test-nim", wind, humus, soil)

	// Create channel to catch the leaf
	caughtLeaves := make(chan Leaf, 1)
	sub, err := wind.Catch("test.struct", func(leaf Leaf) {
		caughtLeaves <- leaf
	})
	if err != nil {
		t.Fatalf("Failed to catch: %v", err)
	}
	defer sub.Unsubscribe()

	time.Sleep(100 * time.Millisecond)

	// Drop a structured leaf
	type TestData struct {
		Message string `json:"message"`
		Count   int    `json:"count"`
	}
	err = base.LeafStruct("test.struct", TestData{Message: "test", Count: 42})
	if err != nil {
		t.Fatalf("Failed to drop struct leaf: %v", err)
	}

	// Verify we caught and can parse it
	select {
	case leaf := <-caughtLeaves:
		var data TestData
		if err := json.Unmarshal(leaf.Data, &data); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}
		if data.Message != "test" {
			t.Errorf("Expected message 'test', got '%s'", data.Message)
		}
		if data.Count != 42 {
			t.Errorf("Expected count 42, got %d", data.Count)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for leaf")
	}
}

func TestBaseNim_Compost(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	wind := NewWind(nc)
	js.DeleteStream("HUMUS")
	humus, _ := NewHumus(js)
	js.DeleteKeyValue("SOIL")
	soil, _ := NewSoil(js)

	base := NewBaseNim("test-nim", wind, humus, soil)

	// Compost a state change
	slot, err := base.Compost("tasks/task-1", "create", []byte(`{"status": "pending"}`))
	if err != nil {
		t.Fatalf("Failed to compost: %v", err)
	}
	if slot == 0 {
		t.Error("Expected non-zero slot")
	}
}

func TestBaseNim_CompostStruct(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	wind := NewWind(nc)
	js.DeleteStream("HUMUS")
	humus, _ := NewHumus(js)
	js.DeleteKeyValue("SOIL")
	soil, _ := NewSoil(js)

	base := NewBaseNim("test-nim", wind, humus, soil)

	type Task struct {
		Status string `json:"status"`
		Owner  string `json:"owner"`
	}

	slot, err := base.CompostStruct("tasks/task-2", "create", Task{Status: "active", Owner: "alice"})
	if err != nil {
		t.Fatalf("Failed to compost struct: %v", err)
	}
	if slot == 0 {
		t.Error("Expected non-zero slot")
	}
}

func TestBaseNim_BuryAndDig(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	wind := NewWind(nc)
	js.DeleteStream("HUMUS")
	humus, _ := NewHumus(js)
	js.DeleteKeyValue("SOIL")
	soil, _ := NewSoil(js)

	base := NewBaseNim("test-nim", wind, humus, soil)

	entity := "config/settings"
	data := []byte(`{"theme": "dark"}`)

	// Bury (create)
	err := base.Bury(entity, data, 0)
	if err != nil {
		t.Fatalf("Failed to bury: %v", err)
	}

	// Dig it back up
	retrieved, revision, err := base.Dig(entity)
	if err != nil {
		t.Fatalf("Failed to dig: %v", err)
	}
	if string(retrieved) != string(data) {
		t.Errorf("Data mismatch: expected %s, got %s", string(data), string(retrieved))
	}
	if revision == 0 {
		t.Error("Expected non-zero revision")
	}
}

func TestBaseNim_BuryStructAndDigStruct(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	wind := NewWind(nc)
	js.DeleteStream("HUMUS")
	humus, _ := NewHumus(js)
	js.DeleteKeyValue("SOIL")
	soil, _ := NewSoil(js)

	base := NewBaseNim("test-nim", wind, humus, soil)

	type Config struct {
		Theme    string `json:"theme"`
		Language string `json:"language"`
	}

	entity := "config/user-1"
	original := Config{Theme: "light", Language: "en"}

	// Bury struct
	err := base.BuryStruct(entity, original, 0)
	if err != nil {
		t.Fatalf("Failed to bury struct: %v", err)
	}

	// Dig struct
	var retrieved Config
	revision, err := base.DigStruct(entity, &retrieved)
	if err != nil {
		t.Fatalf("Failed to dig struct: %v", err)
	}
	if retrieved.Theme != original.Theme {
		t.Errorf("Theme mismatch: expected %s, got %s", original.Theme, retrieved.Theme)
	}
	if retrieved.Language != original.Language {
		t.Errorf("Language mismatch: expected %s, got %s", original.Language, retrieved.Language)
	}
	if revision == 0 {
		t.Error("Expected non-zero revision")
	}
}

func TestMockNim_Integration(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	wind := NewWind(nc)
	js.DeleteStream("HUMUS")
	humus, _ := NewHumus(js)
	js.DeleteKeyValue("SOIL")
	soil, _ := NewSoil(js)

	// Track what the nim does
	handledLeaves := make(chan Leaf, 1)
	
	// Create the base nim first
	baseNim := NewBaseNim("payment-nim", wind, humus, soil)
	
	// Create a mock nim that handles payment events
	mockNim := &MockNim{
		BaseNim:  baseNim,
		subjects: []string{"payment.completed"},
		handleFn: func(ctx context.Context, leaf Leaf) error {
			handledLeaves <- leaf
			// Business logic: create a followup task
			_, err := baseNim.Compost("tasks/followup-123", "create", []byte(`{"status": "pending"}`))
			return err
		},
	}

	// Start the nim
	ctx := context.Background()
	err := mockNim.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start nim: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Drop a payment leaf
	paymentLeaf := NewLeaf("payment.completed", []byte(`{"amount": 100}`), "payment-tree")
	wind.Drop(*paymentLeaf)

	// Verify the nim handled it
	select {
	case leaf := <-handledLeaves:
		if leaf.Subject != "payment.completed" {
			t.Errorf("Expected subject 'payment.completed', got '%s'", leaf.Subject)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for nim to handle leaf")
	}

	// Verify the nim composted a task (check humus)
	time.Sleep(100 * time.Millisecond)
	info, _ := humus.StreamInfo()
	if info.State.Msgs < 1 {
		t.Error("Expected at least 1 compost entry in humus")
	}
}

func TestBaseNim_Catch(t *testing.T) {
	nc := setupTestNATS(t)
	defer nc.Close()

	js, _ := nc.JetStream()
	wind := NewWind(nc)
	js.DeleteStream("HUMUS")
	humus, _ := NewHumus(js)
	js.DeleteKeyValue("SOIL")
	soil, _ := NewSoil(js)

	base := NewBaseNim("test-nim", wind, humus, soil)

	caughtLeaves := make(chan Leaf, 1)
	err := base.Catch("nim.test", func(leaf Leaf) {
		caughtLeaves <- leaf
	})
	if err != nil {
		t.Fatalf("Failed to catch: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Drop a leaf
	testLeaf := NewLeaf("nim.test", []byte(`{}`), "test")
	wind.Drop(*testLeaf)

	// Verify we caught it
	select {
	case <-caughtLeaves:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for caught leaf")
	}
}

func TestBaseNim_GetMethods(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	wind := NewWind(nc)
	js.DeleteStream("HUMUS")
	humus, _ := NewHumus(js)
	js.DeleteKeyValue("SOIL")
	soil, _ := NewSoil(js)

	base := NewBaseNim("test-nim", wind, humus, soil)

	if base.GetWind() != wind {
		t.Error("GetWind should return the wind instance")
	}
	if base.GetHumus() != humus {
		t.Error("GetHumus should return the humus instance")
	}
	if base.GetSoil() != soil {
		t.Error("GetSoil should return the soil instance")
	}
}
