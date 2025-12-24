package core

import (
	"encoding/json"
	"testing"
	"time"
)

// jsonEqual compares two JSON byte slices semantically
func jsonEqual(a, b []byte) bool {
	var j1, j2 interface{}
	if err := json.Unmarshal(a, &j1); err != nil {
		return false
	}
	if err := json.Unmarshal(b, &j2); err != nil {
		return false
	}

	// Convert back to JSON to normalize
	b1, _ := json.Marshal(j1)
	b2, _ := json.Marshal(j2)
	return string(b1) == string(b2)
}

func TestNewDecomposer(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	js.DeleteStream("HUMUS")
	humus, _ := NewHumus(js)
	js.DeleteKeyValue("SOIL")
	soil, _ := NewSoil(js)

	decomposer := NewDecomposer(humus, soil)
	if decomposer == nil {
		t.Fatal("Expected non-nil decomposer")
	}
	if decomposer.consumerName != "decomposer" {
		t.Errorf("Expected consumer name 'decomposer', got '%s'", decomposer.consumerName)
	}
}

func TestNewDecomposerWithConsumer(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	js.DeleteStream("HUMUS")
	humus, _ := NewHumus(js)
	js.DeleteKeyValue("SOIL")
	soil, _ := NewSoil(js)

	decomposer := NewDecomposerWithConsumer(humus, soil, "custom-decomposer")
	if decomposer.consumerName != "custom-decomposer" {
		t.Errorf("Expected consumer name 'custom-decomposer', got '%s'", decomposer.consumerName)
	}
}

func TestDecomposer_CreateAction(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	js.DeleteStream("HUMUS")
	humus, _ := NewHumus(js)
	js.DeleteKeyValue("SOIL")
	soil, _ := NewSoil(js)

	decomposer := NewDecomposer(humus, soil)
	err := decomposer.Start()
	if err != nil {
		t.Fatalf("Failed to start decomposer: %v", err)
	}
	defer decomposer.Stop()

	// Give decomposer time to start
	time.Sleep(200 * time.Millisecond)

	// Add a create compost
	entity := "users/user-1"
	data := []byte(`{"name": "Alice", "email": "alice@example.com"}`)
	_, err = humus.Add("test-nim", entity, "create", data)
	if err != nil {
		t.Fatalf("Failed to add compost: %v", err)
	}

	// Wait for decomposer to process and persist
	time.Sleep(800 * time.Millisecond)

	// Verify entity was created in soil
	retrieved, _, err := soil.Dig(entity)
	if err != nil {
		t.Fatalf("Entity not found in soil: %v", err)
	}
	// Compare JSON content semantically (Go removes whitespace)
	if !jsonEqual(data, retrieved) {
		t.Errorf("Data mismatch: expected %s, got %s", string(data), string(retrieved))
	}
}

func TestDecomposer_UpdateAction(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	js.DeleteStream("HUMUS")
	humus, _ := NewHumus(js)
	js.DeleteKeyValue("SOIL")
	soil, _ := NewSoil(js)

	// Pre-create an entity in soil
	entity := "users/user-2"
	originalData := []byte(`{"name": "Bob", "status": "active"}`)
	soil.Bury(entity, originalData, 0)

	decomposer := NewDecomposer(humus, soil)
	err := decomposer.Start()
	if err != nil {
		t.Fatalf("Failed to start decomposer: %v", err)
	}
	defer decomposer.Stop()

	time.Sleep(200 * time.Millisecond)

	// Add an update compost
	updatedData := []byte(`{"name": "Bob", "status": "inactive"}`)
	_, err = humus.Add("test-nim", entity, "update", updatedData)
	if err != nil {
		t.Fatalf("Failed to add compost: %v", err)
	}

	// Wait for decomposer to process and persist
	time.Sleep(800 * time.Millisecond)

	// Verify entity was updated in soil
	retrieved, _, err := soil.Dig(entity)
	if err != nil {
		t.Fatalf("Entity not found in soil: %v", err)
	}
	// Compare JSON content semantically
	if !jsonEqual(updatedData, retrieved) {
		t.Errorf("Data mismatch after update: expected %s, got %s", string(updatedData), string(retrieved))
	}
}

func TestDecomposer_UpdateNonExistent(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	js.DeleteStream("HUMUS")
	humus, _ := NewHumus(js)
	js.DeleteKeyValue("SOIL")
	soil, _ := NewSoil(js)

	decomposer := NewDecomposer(humus, soil)
	err := decomposer.Start()
	if err != nil {
		t.Fatalf("Failed to start decomposer: %v", err)
	}
	defer decomposer.Stop()

	time.Sleep(200 * time.Millisecond)

	// Try to update a non-existent entity (should create it)
	entity := "users/user-3"
	data := []byte(`{"name": "Charlie"}`)
	_, err = humus.Add("test-nim", entity, "update", data)
	if err != nil {
		t.Fatalf("Failed to add compost: %v", err)
	}

	// Wait for decomposer to process and persist
	time.Sleep(800 * time.Millisecond)

	// Verify entity was created (not errored)
	retrieved, _, err := soil.Dig(entity)
	if err != nil {
		t.Fatalf("Entity should have been created: %v", err)
	}
	// Compare JSON content semantically
	if !jsonEqual(data, retrieved) {
		t.Errorf("Data mismatch: expected %s, got %s", string(data), string(retrieved))
	}
}

func TestDecomposer_DeleteAction(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	js.DeleteStream("HUMUS")
	humus, _ := NewHumus(js)
	js.DeleteKeyValue("SOIL")
	soil, _ := NewSoil(js)

	// Pre-create an entity in soil
	entity := "users/user-4"
	soil.Bury(entity, []byte(`{"name": "Dave"}`), 0)

	decomposer := NewDecomposer(humus, soil)
	err := decomposer.Start()
	if err != nil {
		t.Fatalf("Failed to start decomposer: %v", err)
	}
	defer decomposer.Stop()

	time.Sleep(200 * time.Millisecond)

	// Add a delete compost
	_, err = humus.Add("test-nim", entity, "delete", []byte{})
	if err != nil {
		t.Fatalf("Failed to add compost: %v", err)
	}

	// Wait for decomposer to process and persist
	time.Sleep(800 * time.Millisecond)

	// Verify entity was deleted from soil
	_, _, err = soil.Dig(entity)
	if err == nil {
		t.Error("Entity should have been deleted")
	}
}

func TestDecomposer_DeleteNonExistent(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	js.DeleteStream("HUMUS")
	humus, _ := NewHumus(js)
	js.DeleteKeyValue("SOIL")
	soil, _ := NewSoil(js)

	decomposer := NewDecomposer(humus, soil)
	err := decomposer.Start()
	if err != nil {
		t.Fatalf("Failed to start decomposer: %v", err)
	}
	defer decomposer.Stop()

	time.Sleep(200 * time.Millisecond)

	// Try to delete a non-existent entity (should not error)
	entity := "users/user-5"
	_, err = humus.Add("test-nim", entity, "delete", []byte{})
	if err != nil {
		t.Fatalf("Failed to add compost: %v", err)
	}

	// Wait for decomposer to process
	time.Sleep(800 * time.Millisecond)

	// Should complete without error (idempotent)
}

func TestDecomposer_MultipleOperations(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	js.DeleteStream("HUMUS")
	humus, _ := NewHumus(js)
	js.DeleteKeyValue("SOIL")
	soil, _ := NewSoil(js)

	decomposer := NewDecomposer(humus, soil)
	err := decomposer.Start()
	if err != nil {
		t.Fatalf("Failed to start decomposer: %v", err)
	}
	defer decomposer.Stop()

	time.Sleep(200 * time.Millisecond)

	// Add multiple compost entries
	entity := "tasks/task-1"

	// Create
	humus.Add("test-nim", entity, "create", []byte(`{"status": "pending"}`))
	time.Sleep(300 * time.Millisecond)

	// Update
	humus.Add("test-nim", entity, "update", []byte(`{"status": "in_progress"}`))
	time.Sleep(300 * time.Millisecond)

	// Update again
	humus.Add("test-nim", entity, "update", []byte(`{"status": "complete"}`))
	time.Sleep(300 * time.Millisecond)

	// Verify final state
	retrieved, _, err := soil.Dig(entity)
	if err != nil {
		t.Fatalf("Entity not found: %v", err)
	}
	// JSON might have spacing differences, check the content
	expected := `complete`
	if !contains(string(retrieved), expected) {
		t.Errorf("Final state should contain 'complete', got %s", string(retrieved))
	}
}

func TestDecomposer_Stop(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	js.DeleteStream("HUMUS")
	humus, _ := NewHumus(js)
	js.DeleteKeyValue("SOIL")
	soil, _ := NewSoil(js)

	decomposer := NewDecomposer(humus, soil)
	err := decomposer.Start()
	if err != nil {
		t.Fatalf("Failed to start decomposer: %v", err)
	}

	// Stop the decomposer
	decomposer.Stop()

	// Should be able to call Stop multiple times
	decomposer.Stop()
}

func TestRunDecomposer(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	js.DeleteStream("HUMUS")
	humus, _ := NewHumus(js)
	js.DeleteKeyValue("SOIL")
	soil, _ := NewSoil(js)

	decomposer, err := RunDecomposer(humus, soil)
	if err != nil {
		t.Fatalf("Failed to run decomposer: %v", err)
	}
	if decomposer == nil {
		t.Fatal("Expected non-nil decomposer")
	}
	defer decomposer.Stop()

	time.Sleep(200 * time.Millisecond)

	// Verify it's working
	entity := "test/entity"
	humus.Add("test-nim", entity, "create", []byte(`{"test": true}`))
	time.Sleep(800 * time.Millisecond)

	_, _, err = soil.Dig(entity)
	if err != nil {
		t.Error("Decomposer should have processed the compost")
	}
}

func TestRunDecomposerWithConsumer(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	js.DeleteStream("HUMUS")
	humus, _ := NewHumus(js)
	js.DeleteKeyValue("SOIL")
	soil, _ := NewSoil(js)

	decomposer, err := RunDecomposerWithConsumer(humus, soil, "test-decomposer")
	if err != nil {
		t.Fatalf("Failed to run decomposer: %v", err)
	}
	if decomposer.consumerName != "test-decomposer" {
		t.Errorf("Expected consumer name 'test-decomposer', got '%s'", decomposer.consumerName)
	}
	defer decomposer.Stop()
}
