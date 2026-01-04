// E2E MVP Test
// Run: go test ./test/e2emvp -v
//
// Tests the full MVP flow:
// 1. Load config
// 2. Start TreeHouse (Lua)
// 3. Start Nim (mock brain)
// 4. Publish event → verify output

package e2emvp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

// TestMVPFlow tests: contact.created → scoring (Lua) → lead.scored → qualify (brain) → lead.qualified
func TestMVPFlow(t *testing.T) {
	// 1. Start embedded NATS
	ns, nc := startNATS(t)
	defer ns.Shutdown()
	defer nc.Close()

	// 2. Load config
	configPath := filepath.Join(testDataDir(), "forest.yaml")
	// TODO: cfg, err := runtime.LoadConfig(configPath)
	// if err != nil {
	// 	t.Fatalf("failed to load config: %v", err)
	// }
	_ = configPath

	// 3. Create mock brain
	// TODO: mockBrain := brain.NewMockBrain()
	// mockBrain.SetResponse(`{"pursue": true, "reason": "High score"}`)

	// 4. Start TreeHouses
	// TODO: for _, th := range cfg.TreeHouses {
	// 	treehouse := runtime.NewTreeHouse(th, nc)
	// 	go treehouse.Start()
	// 	defer treehouse.Stop()
	// }

	// 5. Start Nims
	// TODO: for _, n := range cfg.Nims {
	// 	nim := runtime.NewNim(n, nc, mockBrain)
	// 	go nim.Start()
	// 	defer nim.Stop()
	// }

	// 6. Subscribe to output subjects
	scoredCh := subscribe(t, nc, "lead.scored")
	qualifiedCh := subscribe(t, nc, "lead.qualified")

	// 7. Publish test contact
	contact := map[string]interface{}{
		"id":           "test-123",
		"email":        "jane@acme.com",
		"title":        "VP Engineering",
		"company_size": 250,
		"industry":     "technology",
	}
	publish(t, nc, "contact.created", contact)

	// 8. Verify lead.scored output
	scored := waitForMessage(t, scoredCh, 2*time.Second)
	assertField(t, scored, "contact_id", "test-123")
	assertFieldExists(t, scored, "score")
	assertFieldExists(t, scored, "signals")

	// Score should be: mid_market(30) + executive(40) + target_industry(15) = 85
	if score, ok := scored["score"].(float64); ok {
		if score != 85 {
			t.Errorf("expected score 85, got %v", score)
		}
	}

	// 9. Verify lead.qualified output
	qualified := waitForMessage(t, qualifiedCh, 2*time.Second)
	assertFieldExists(t, qualified, "pursue")
	assertFieldExists(t, qualified, "reason")

	t.Log("MVP E2E test passed!")
}

// TestTreeHouseDeterminism verifies same input = same output
func TestTreeHouseDeterminism(t *testing.T) {
	ns, nc := startNATS(t)
	defer ns.Shutdown()
	defer nc.Close()

	// TODO: Load config and start scoring TreeHouse

	scoredCh := subscribe(t, nc, "lead.scored")

	contact := map[string]interface{}{
		"id":           "det-test",
		"email":        "test@test.com",
		"title":        "CEO",
		"company_size": 600,
		"industry":     "finance",
	}

	// Publish same event twice
	publish(t, nc, "contact.created", contact)
	result1 := waitForMessage(t, scoredCh, 2*time.Second)

	publish(t, nc, "contact.created", contact)
	result2 := waitForMessage(t, scoredCh, 2*time.Second)

	// Results must be identical
	if result1["score"] != result2["score"] {
		t.Errorf("TreeHouse not deterministic: %v != %v", result1["score"], result2["score"])
	}
}

// TestLuaHelpers verifies contains, json.encode, json.decode work
func TestLuaHelpers(t *testing.T) {
	// TODO: Test Lua helpers in isolation
	// lua := runtime.NewLuaVM()
	// lua.LoadScript(testDataDir() + "/test_helpers.lua")
	// result, err := lua.Call("test_contains")
	// ...
	t.Skip("TODO: implement Lua helper tests")
}

// --- Test Helpers ---

func testDataDir() string {
	// Returns path to test/e2emvp/testdata
	return filepath.Join("testdata")
}

func startNATS(t *testing.T) (*server.Server, *nats.Conn) {
	opts := &server.Options{
		Host: "127.0.0.1",
		Port: -1, // Random available port
	}
	ns, err := server.NewServer(opts)
	if err != nil {
		t.Fatalf("failed to create NATS server: %v", err)
	}
	go ns.Start()
	if !ns.ReadyForConnections(2 * time.Second) {
		t.Fatal("NATS server not ready")
	}

	nc, err := nats.Connect(ns.ClientURL())
	if err != nil {
		t.Fatalf("failed to connect to NATS: %v", err)
	}
	return ns, nc
}

func subscribe(t *testing.T, nc *nats.Conn, subject string) chan map[string]interface{} {
	ch := make(chan map[string]interface{}, 10)
	_, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		var data map[string]interface{}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			t.Errorf("failed to unmarshal message: %v", err)
			return
		}
		ch <- data
	})
	if err != nil {
		t.Fatalf("failed to subscribe to %s: %v", subject, err)
	}
	return ch
}

func publish(t *testing.T, nc *nats.Conn, subject string, data interface{}) {
	bytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal data: %v", err)
	}
	if err := nc.Publish(subject, bytes); err != nil {
		t.Fatalf("failed to publish to %s: %v", subject, err)
	}
	nc.Flush()
}

func waitForMessage(t *testing.T, ch chan map[string]interface{}, timeout time.Duration) map[string]interface{} {
	select {
	case msg := <-ch:
		return msg
	case <-time.After(timeout):
		t.Fatal("timeout waiting for message")
		return nil
	}
}

func assertField(t *testing.T, data map[string]interface{}, field string, expected interface{}) {
	if data[field] != expected {
		t.Errorf("expected %s=%v, got %v", field, expected, data[field])
	}
}

func assertFieldExists(t *testing.T, data map[string]interface{}, field string) {
	if _, ok := data[field]; !ok {
		t.Errorf("expected field %s to exist", field)
	}
}

func init() {
	// Ensure we're in the right directory for testdata
	if _, err := os.Stat("testdata"); os.IsNotExist(err) {
		// Try from repo root
		os.Chdir(filepath.Join("..", ".."))
	}
}
