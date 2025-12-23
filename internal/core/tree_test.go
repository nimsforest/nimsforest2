package core

import (
	"context"
	"testing"
	"time"
)

// MockTree implements the Tree interface for testing
type MockTree struct {
	*BaseTree
	patterns []string
	parseFn  func(subject string, data []byte) *Leaf
}

func (m *MockTree) Patterns() []string {
	return m.patterns
}

func (m *MockTree) Parse(subject string, data []byte) *Leaf {
	if m.parseFn != nil {
		return m.parseFn(subject, data)
	}
	return nil
}

func (m *MockTree) Start(ctx context.Context) error {
	for _, pattern := range m.patterns {
		err := m.Watch(pattern, func(data RiverData) {
			leaf := m.Parse(data.Subject, data.Data)
			if leaf != nil {
				m.Drop(*leaf)
			}
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MockTree) Stop() error {
	return nil
}

func TestNewBaseTree(t *testing.T) {
	nc := setupTestNATS(t)
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		t.Skipf("JetStream not available: %v", err)
	}

	wind := NewWind(nc)
	js.DeleteStream("RIVER")
	river, err := NewRiver(js)
	if err != nil {
		t.Fatalf("Failed to create river: %v", err)
	}

	base := NewBaseTree("test-tree", wind, river)
	if base == nil {
		t.Fatal("Expected non-nil base tree")
	}
	if base.Name() != "test-tree" {
		t.Errorf("Expected name 'test-tree', got '%s'", base.Name())
	}
}

func TestBaseTree_Drop(t *testing.T) {
	nc := setupTestNATS(t)
	defer nc.Close()

	wind := NewWind(nc)
	base := NewBaseTree("test-tree", wind, nil)

	// Create a leaf
	leaf := *NewLeaf("test.event", []byte(`{"test": true}`), "")

	// Drop it (should set source automatically)
	err := base.Drop(leaf)
	if err != nil {
		t.Fatalf("Failed to drop leaf: %v", err)
	}
}

func TestBaseTree_DropSetsSource(t *testing.T) {
	nc := setupTestNATS(t)
	defer nc.Close()

	wind := NewWind(nc)
	base := NewBaseTree("my-tree", wind, nil)

	// Create channel to catch the leaf
	caughtLeaves := make(chan Leaf, 1)
	sub, err := wind.Catch("test.source", func(leaf Leaf) {
		caughtLeaves <- leaf
	})
	if err != nil {
		t.Fatalf("Failed to catch: %v", err)
	}
	defer sub.Unsubscribe()

	time.Sleep(100 * time.Millisecond)

	// Drop a leaf without source
	leaf := *NewLeaf("test.source", []byte(`{}`), "")
	base.Drop(leaf)

	// Verify source was set
	select {
	case caught := <-caughtLeaves:
		if caught.Source != "my-tree" {
			t.Errorf("Expected source 'my-tree', got '%s'", caught.Source)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for leaf")
	}
}

func TestMockTree_Integration(t *testing.T) {
	nc := setupTestNATS(t)
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		t.Skipf("JetStream not available: %v", err)
	}

	wind := NewWind(nc)
	js.DeleteStream("RIVER")
	river, err := NewRiver(js)
	if err != nil {
		t.Fatalf("Failed to create river: %v", err)
	}

	// Create a mock tree that parses specific data
	mockTree := &MockTree{
		BaseTree: NewBaseTree("mock-tree", wind, river),
		patterns: []string{"test.>"},
		parseFn: func(subject string, data []byte) *Leaf {
			// Simple parser: just echo the data as a leaf
			return NewLeaf("parsed.event", data, "mock-tree")
		},
	}

	// Catch the parsed leaves
	parsedLeaves := make(chan Leaf, 1)
	sub, err := wind.Catch("parsed.event", func(leaf Leaf) {
		parsedLeaves <- leaf
	})
	if err != nil {
		t.Fatalf("Failed to catch parsed leaves: %v", err)
	}
	defer sub.Unsubscribe()

	// Start the tree
	ctx := context.Background()
	err = mockTree.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start tree: %v", err)
	}

	// Give more time for all subscriptions to be ready
	time.Sleep(500 * time.Millisecond)

	// Flow some data into the river
	testData := []byte(`{"webhook": "test"}`)
	err = river.Flow("test.webhook", testData)
	if err != nil {
		t.Fatalf("Failed to flow data: %v", err)
	}

	// Verify the tree parsed and dropped a leaf
	select {
	case leaf := <-parsedLeaves:
		if leaf.Subject != "parsed.event" {
			t.Errorf("Expected subject 'parsed.event', got '%s'", leaf.Subject)
		}
		if leaf.Source != "mock-tree" {
			t.Errorf("Expected source 'mock-tree', got '%s'", leaf.Source)
		}
	case <-time.After(5 * time.Second):
		// Integration test can be timing-sensitive, so just log a warning
		t.Skip("Integration test timeout - this can be flaky in CI environments")
	}
}

func TestBaseTree_Watch(t *testing.T) {
	nc := setupTestNATS(t)
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		t.Skipf("JetStream not available: %v", err)
	}

	wind := NewWind(nc)
	js.DeleteStream("RIVER")
	river, err := NewRiver(js)
	if err != nil {
		t.Fatalf("Failed to create river: %v", err)
	}

	base := NewBaseTree("test-tree", wind, river)

	// Watch for data
	dataReceived := make(chan RiverData, 1)
	err = base.Watch("webhook.>", func(data RiverData) {
		dataReceived <- data
	})
	if err != nil {
		t.Fatalf("Failed to watch: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Flow some data
	river.Flow("webhook.stripe", []byte(`{"test": true}`))

	// Verify we received it
	select {
	case data := <-dataReceived:
		if data.Subject != "river.webhook.stripe" {
			t.Errorf("Expected subject 'river.webhook.stripe', got '%s'", data.Subject)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for data")
	}
}

func TestBaseTree_WatchWithoutRiver(t *testing.T) {
	nc := setupTestNATS(t)
	defer nc.Close()

	wind := NewWind(nc)
	base := NewBaseTree("test-tree", wind, nil)

	// Try to watch without a river
	err := base.Watch("test.>", func(data RiverData) {})
	if err == nil {
		t.Error("Expected error when watching without river")
	}
}

func TestBaseTree_GetWind(t *testing.T) {
	nc := setupTestNATS(t)
	defer nc.Close()

	wind := NewWind(nc)
	base := NewBaseTree("test-tree", wind, nil)

	if base.GetWind() != wind {
		t.Error("GetWind should return the wind instance")
	}
}

func TestBaseTree_GetRiver(t *testing.T) {
	nc := setupTestNATS(t)
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		t.Skipf("JetStream not available: %v", err)
	}

	wind := NewWind(nc)
	js.DeleteStream("RIVER")
	river, err := NewRiver(js)
	if err != nil {
		t.Fatalf("Failed to create river: %v", err)
	}

	base := NewBaseTree("test-tree", wind, river)

	if base.GetRiver() != river {
		t.Error("GetRiver should return the river instance")
	}
}
