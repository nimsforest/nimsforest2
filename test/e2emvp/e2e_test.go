// E2E MVP Test
// Run: go test ./test/e2emvp -v
//
// Tests the full MVP flow:
// 1. Load config
// 2. Start TreeHouse (Lua) via Wind
// 3. Start Nim (mock brain) via Wind
// 4. Drop leaf → verify output

package e2emvp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/yourusername/nimsforest/internal/core"
	"github.com/yourusername/nimsforest/pkg/brain"
	"github.com/yourusername/nimsforest/pkg/runtime"
)

// TestMVPFlow tests: contact.created → scoring (Lua) → lead.scored → qualify (brain) → lead.qualified
func TestMVPFlow(t *testing.T) {
	// 1. Start embedded NATS and create Wind
	ns, nc := startNATS(t)
	defer ns.Shutdown()
	defer nc.Close()

	wind := core.NewWind(nc)

	// 2. Load config
	configPath := filepath.Join(testDataDir(), "forest.yaml")
	cfg, err := runtime.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// 3. Create mock brain that returns expected JSON
	mockBrain := brain.NewMockBrain()
	mockBrain.SetRawResponse(`{"pursue": true, "reason": "High score"}`)

	// 4. Create forest runtime using Wind
	forest, err := runtime.NewForestFromConfig(cfg, wind, mockBrain)
	if err != nil {
		t.Fatalf("failed to create forest: %v", err)
	}

	// 5. Start forest
	ctx := context.Background()
	if err := forest.Start(ctx); err != nil {
		t.Fatalf("failed to start forest: %v", err)
	}
	defer forest.Stop()

	// 6. Subscribe to output subjects using Wind
	scoredCh := catchLeaves(t, wind, "lead.scored")
	qualifiedCh := catchLeaves(t, wind, "lead.qualified")

	// Give subscriptions time to register
	time.Sleep(100 * time.Millisecond)

	// 7. Drop test contact leaf via Wind
	contact := map[string]interface{}{
		"id":           "test-123",
		"email":        "jane@acme.com",
		"title":        "VP Engineering",
		"company_size": float64(250),
		"industry":     "technology",
	}
	dropLeaf(t, wind, "contact.created", "test", contact)

	// 8. Verify lead.scored output
	scored := waitForLeaf(t, scoredCh, 2*time.Second)
	assertField(t, scored, "contact_id", "test-123")
	assertFieldExists(t, scored, "score")
	assertFieldExists(t, scored, "signals")

	// Score should be: mid_market(30) + executive(40) + target_industry(15) = 85
	if score, ok := scored["score"].(float64); ok {
		if score != 85 {
			t.Errorf("expected score 85, got %v", score)
		}
	} else {
		t.Errorf("score is not a number: %v (%T)", scored["score"], scored["score"])
	}

	// 9. Verify lead.qualified output
	qualified := waitForLeaf(t, qualifiedCh, 2*time.Second)
	assertFieldExists(t, qualified, "pursue")
	assertFieldExists(t, qualified, "reason")

	t.Log("MVP E2E test passed!")
}

// TestTreeHouseDeterminism verifies same input = same output
func TestTreeHouseDeterminism(t *testing.T) {
	ns, nc := startNATS(t)
	defer ns.Shutdown()
	defer nc.Close()

	wind := core.NewWind(nc)

	// Load config
	configPath := filepath.Join(testDataDir(), "forest.yaml")
	cfg, err := runtime.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Create just the scoring treehouse using Wind
	thCfg := cfg.TreeHouses["scoring"]
	scriptPath := cfg.ResolvePath(thCfg.Script)
	th, err := runtime.NewTreeHouse(thCfg, wind, scriptPath)
	if err != nil {
		t.Fatalf("failed to create treehouse: %v", err)
	}

	ctx := context.Background()
	if err := th.Start(ctx); err != nil {
		t.Fatalf("failed to start treehouse: %v", err)
	}
	defer th.Stop()

	scoredCh := catchLeaves(t, wind, "lead.scored")
	time.Sleep(100 * time.Millisecond)

	contact := map[string]interface{}{
		"id":           "det-test",
		"email":        "test@test.com",
		"title":        "CEO",
		"company_size": float64(600),
		"industry":     "finance",
	}

	// Drop same leaf twice
	dropLeaf(t, wind, "contact.created", "test", contact)
	result1 := waitForLeaf(t, scoredCh, 2*time.Second)

	dropLeaf(t, wind, "contact.created", "test", contact)
	result2 := waitForLeaf(t, scoredCh, 2*time.Second)

	// Results must be identical
	if result1["score"] != result2["score"] {
		t.Errorf("TreeHouse not deterministic: %v != %v", result1["score"], result2["score"])
	}

	// CEO(40) + enterprise(50) + target_industry(15) = 105
	if score, ok := result1["score"].(float64); ok {
		if score != 105 {
			t.Errorf("expected score 105, got %v", score)
		}
	}
}

// TestLuaHelpers verifies contains, json.encode, json.decode work
func TestLuaHelpers(t *testing.T) {
	vm := runtime.NewLuaVM()
	defer vm.Close()

	// Test contains helper
	script := `
function process(input)
    local result = {
        has_ceo = contains(input.title, "CEO"),
        has_vp = contains(input.title, "VP"),
        encoded = json.encode({key = "value"}),
    }
    
    local decoded = json.decode('{"foo": "bar"}')
    result.decoded_foo = decoded.foo
    
    return result
end
`
	if err := vm.LoadString(script); err != nil {
		t.Fatalf("failed to load script: %v", err)
	}

	input := map[string]interface{}{
		"title": "CEO of Things",
	}

	output, err := vm.CallProcess(input)
	if err != nil {
		t.Fatalf("failed to call process: %v", err)
	}

	if output["has_ceo"] != true {
		t.Errorf("expected has_ceo=true, got %v", output["has_ceo"])
	}
	if output["has_vp"] != false {
		t.Errorf("expected has_vp=false, got %v", output["has_vp"])
	}
	if output["decoded_foo"] != "bar" {
		t.Errorf("expected decoded_foo='bar', got %v", output["decoded_foo"])
	}
	if output["encoded"] != `{"key":"value"}` {
		t.Errorf("expected encoded json, got %v", output["encoded"])
	}
}

// TestConfigLoader tests config loading and validation
func TestConfigLoader(t *testing.T) {
	configPath := filepath.Join(testDataDir(), "forest.yaml")
	cfg, err := runtime.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Check treehouses
	if len(cfg.TreeHouses) != 1 {
		t.Errorf("expected 1 treehouse, got %d", len(cfg.TreeHouses))
	}
	scoring := cfg.TreeHouses["scoring"]
	if scoring.Subscribes != "contact.created" {
		t.Errorf("expected scoring.subscribes='contact.created', got %s", scoring.Subscribes)
	}

	// Check nims
	if len(cfg.Nims) != 1 {
		t.Errorf("expected 1 nim, got %d", len(cfg.Nims))
	}
	qualify := cfg.Nims["qualify"]
	if qualify.Subscribes != "lead.scored" {
		t.Errorf("expected qualify.subscribes='lead.scored', got %s", qualify.Subscribes)
	}
}

// --- Test Helpers ---

func testDataDir() string {
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

// catchLeaves subscribes to a subject via Wind and returns a channel for received leaf data
func catchLeaves(t *testing.T, wind *core.Wind, subject string) chan map[string]interface{} {
	ch := make(chan map[string]interface{}, 10)
	_, err := wind.Catch(subject, func(leaf core.Leaf) {
		var data map[string]interface{}
		if err := json.Unmarshal(leaf.Data, &data); err != nil {
			t.Errorf("failed to unmarshal leaf data: %v", err)
			return
		}
		ch <- data
	})
	if err != nil {
		t.Fatalf("failed to catch leaves on %s: %v", subject, err)
	}
	return ch
}

// dropLeaf creates and drops a leaf via Wind
func dropLeaf(t *testing.T, wind *core.Wind, subject, source string, data interface{}) {
	bytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal data: %v", err)
	}
	leaf := core.NewLeaf(subject, bytes, source)
	if err := wind.Drop(*leaf); err != nil {
		t.Fatalf("failed to drop leaf to %s: %v", subject, err)
	}
}

func waitForLeaf(t *testing.T, ch chan map[string]interface{}, timeout time.Duration) map[string]interface{} {
	select {
	case msg := <-ch:
		return msg
	case <-time.After(timeout):
		t.Fatal("timeout waiting for leaf")
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
		os.Chdir(filepath.Join("..", ".."))
	}
}
