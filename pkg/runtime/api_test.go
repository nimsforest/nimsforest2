package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/yourusername/nimsforest/internal/core"
	"github.com/yourusername/nimsforest/pkg/brain"
)

// mockBrain implements brain.Brain for testing
type mockBrain struct{}

func (m *mockBrain) Initialize(ctx context.Context) error { return nil }
func (m *mockBrain) Ask(ctx context.Context, prompt string) (string, error) {
	return `{"result": "ok"}`, nil
}
func (m *mockBrain) Close(ctx context.Context) error                            { return nil }
func (m *mockBrain) Store(ctx context.Context, content string, tags []string) (*brain.Knowledge, error) {
	return nil, nil
}
func (m *mockBrain) Retrieve(ctx context.Context, id string) (*brain.Knowledge, error) {
	return nil, nil
}
func (m *mockBrain) Search(ctx context.Context, query string) ([]*brain.Knowledge, error) {
	return nil, nil
}
func (m *mockBrain) Update(ctx context.Context, id string, content string) error { return nil }
func (m *mockBrain) Delete(ctx context.Context, id string) error                 { return nil }
func (m *mockBrain) List(ctx context.Context) ([]*brain.Knowledge, error)        { return nil, nil }

func setupTestForest(t *testing.T) (*Forest, *core.Wind, func()) {
	t.Helper()

	// Start embedded NATS server
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

	// Connect to NATS
	nc, err := nats.Connect(ns.ClientURL())
	if err != nil {
		ns.Shutdown()
		t.Fatalf("Failed to connect to NATS: %v", err)
	}

	// Create Wind
	wind := core.NewWind(nc)

	// Create empty config
	cfg := &Config{
		TreeHouses: make(map[string]TreeHouseConfig),
		Nims:       make(map[string]NimConfig),
		BaseDir:    t.TempDir(),
	}

	// Create Forest
	forest, err := NewForestFromConfig(cfg, wind, &mockBrain{})
	if err != nil {
		nc.Close()
		ns.Shutdown()
		t.Fatalf("Failed to create forest: %v", err)
	}

	if err := forest.Start(context.Background()); err != nil {
		nc.Close()
		ns.Shutdown()
		t.Fatalf("Failed to start forest: %v", err)
	}

	cleanup := func() {
		forest.Stop()
		nc.Close()
		ns.Shutdown()
	}

	return forest, wind, cleanup
}

func TestForestStatus(t *testing.T) {
	forest, _, cleanup := setupTestForest(t)
	defer cleanup()

	status := forest.Status()

	if !status.Running {
		t.Error("Expected forest to be running")
	}
	if len(status.TreeHouses) != 0 {
		t.Errorf("Expected 0 treehouses, got %d", len(status.TreeHouses))
	}
	if len(status.Nims) != 0 {
		t.Errorf("Expected 0 nims, got %d", len(status.Nims))
	}
}

func TestAPIHealthEndpoint(t *testing.T) {
	forest, _, cleanup := setupTestForest(t)
	defer cleanup()

	api := NewAPI(APIConfig{
		Address: "127.0.0.1:0",
		Forest:  forest,
	})

	// Create test request
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Call handler directly
	api.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", resp["status"])
	}
}

func TestAPIStatusEndpoint(t *testing.T) {
	forest, _, cleanup := setupTestForest(t)
	defer cleanup()

	api := NewAPI(APIConfig{
		Address:    "127.0.0.1:0",
		Forest:     forest,
		ConfigPath: "/test/config.yaml",
	})

	req := httptest.NewRequest("GET", "/api/v1/status", nil)
	w := httptest.NewRecorder()

	api.handleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var status ForestStatus
	json.NewDecoder(w.Body).Decode(&status)

	if !status.Running {
		t.Error("Expected forest to be running")
	}
	if status.ConfigPath != "/test/config.yaml" {
		t.Errorf("Expected config path '/test/config.yaml', got '%s'", status.ConfigPath)
	}
}

func TestAPIAddRemoveTreeHouse(t *testing.T) {
	forest, _, cleanup := setupTestForest(t)
	defer cleanup()

	api := NewAPI(APIConfig{
		Address: "127.0.0.1:0",
		Forest:  forest,
	})

	// Test add with missing fields
	t.Run("add with missing fields", func(t *testing.T) {
		payload := map[string]string{
			"name": "test",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/treehouses", bytes.NewReader(body))
		w := httptest.NewRecorder()

		api.handleAddTreeHouse(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	// Note: Full add/remove test would require valid Lua script
	// This tests the API validation layer
}

func TestAPIAddRemoveNim(t *testing.T) {
	forest, _, cleanup := setupTestForest(t)
	defer cleanup()

	api := NewAPI(APIConfig{
		Address: "127.0.0.1:0",
		Forest:  forest,
	})

	// Test add with missing fields
	t.Run("add nim with missing fields", func(t *testing.T) {
		payload := map[string]string{
			"name": "test",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/nims", bytes.NewReader(body))
		w := httptest.NewRecorder()

		api.handleAddNim(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

func TestClientNewClient(t *testing.T) {
	client := NewClient("http://localhost:8080")

	if client.BaseURL() != "http://localhost:8080" {
		t.Errorf("Expected base URL 'http://localhost:8080', got '%s'", client.BaseURL())
	}
}

func TestClientFromEnv(t *testing.T) {
	client := NewClientFromEnv()

	// Should use default address
	expected := "http://" + DefaultAPIAddress
	if client.BaseURL() != expected {
		t.Errorf("Expected base URL '%s', got '%s'", expected, client.BaseURL())
	}
}

// Ensure brain.Brain interface is satisfied
var _ brain.Brain = (*mockBrain)(nil)
