// Standalone Forest Runner - for testing without cluster config
// Run: go run ./cmd/forest-standalone
// Test: go run ./cmd/forest-standalone test
// Optional: ANTHROPIC_API_KEY or OPENAI_API_KEY for AI-powered evaluation
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/yourusername/nimsforest/internal/core"
	"github.com/yourusername/nimsforest/pkg/brain"
	aifactory "github.com/yourusername/nimsforest/pkg/integrations/aiservice"
	_ "github.com/yourusername/nimsforest/pkg/integrations/aiservice/thirdparty/claude"
	_ "github.com/yourusername/nimsforest/pkg/integrations/aiservice/thirdparty/openai"
	"github.com/yourusername/nimsforest/pkg/runtime"
)

func main() {
	// Check for test command
	if len(os.Args) > 1 && os.Args[1] == "test" {
		runTest()
		return
	}

	fmt.Println("🌲 NimsForest Standalone")
	fmt.Println("========================")
	fmt.Println()

	// 1. Start embedded NATS server
	fmt.Println("📡 Starting embedded NATS server...")
	opts := &server.Options{
		Host: "127.0.0.1",
		Port: 4222,
	}
	ns, err := server.NewServer(opts)
	if err != nil {
		log.Fatalf("❌ Failed to create NATS server: %v\n", err)
	}
	go ns.Start()
	if !ns.ReadyForConnections(5 * time.Second) {
		log.Fatal("❌ NATS server not ready")
	}
	defer ns.Shutdown()
	fmt.Printf("✅ NATS server at %s\n", ns.ClientURL())

	// 2. Connect to NATS
	nc, err := nats.Connect(ns.ClientURL())
	if err != nil {
		log.Fatalf("❌ Failed to connect: %v\n", err)
	}
	defer nc.Close()

	// 3. Create Wind
	wind := core.NewWind(nc)
	fmt.Println("✅ Wind initialized")

	// 4. Load config
	configPath := findConfig()
	if configPath == "" {
		log.Fatal("❌ No config/forest.yaml found")
	}
	fmt.Printf("📄 Config: %s\n", configPath)

	cfg, err := runtime.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v\n", err)
	}

	// 5. Create brain (AI or fallback)
	ctx := context.Background()
	nimBrain, brainType := createBrain(ctx)
	fmt.Printf("🧠 Brain: %s\n", brainType)

	// 6. Create and start forest
	forest, err := runtime.NewForestFromConfig(cfg, wind, nimBrain)
	if err != nil {
		log.Fatalf("❌ Failed to create forest: %v\n", err)
	}

	if err := forest.Start(ctx); err != nil {
		log.Fatalf("❌ Failed to start forest: %v\n", err)
	}
	defer forest.Stop()

	fmt.Println()
	fmt.Println("🌲 Forest running!")
	for name := range cfg.TreeHouses {
		fmt.Printf("   🏠 TreeHouse:%s\n", name)
	}
	for name := range cfg.Nims {
		fmt.Printf("   🧚 Nim:%s\n", name)
	}

	// 7. Subscribe to outputs to display results
	wind.Catch("lead.scored", func(leaf core.Leaf) {
		var data map[string]interface{}
		json.Unmarshal(leaf.Data, &data)
		fmt.Printf("\n📊 SCORED: %v → score=%v signals=%v\n", 
			data["contact_id"], data["score"], data["signals"])
	})

	wind.Catch("lead.qualified", func(leaf core.Leaf) {
		var data map[string]interface{}
		json.Unmarshal(leaf.Data, &data)
		pursue := data["pursue"]
		if pursue == true {
			fmt.Printf("🎯 QUALIFIED: ✅ PURSUE - %v\n", data["reason"])
		} else {
			fmt.Printf("🎯 QUALIFIED: ❌ PASS - %v\n", data["reason"])
		}
	})

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Send test leads with:")
	fmt.Println(`  nats pub contact.created '{"subject":"contact.created","data":{"id":"test","email":"cto@corp.com","title":"CTO","company_size":500,"industry":"technology"},"source":"cli","ts":"2026-01-01T00:00:00Z"}'`)
	fmt.Println()
	fmt.Println("Press Ctrl+C to exit...")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Wait for shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	fmt.Println("\n👋 Shutting down...")
}

func findConfig() string {
	paths := []string{
		"config/forest.yaml",
		"../config/forest.yaml",
		"../../config/forest.yaml",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func createBrain(ctx context.Context) (brain.Brain, string) {
	// Try Anthropic first
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		model := os.Getenv("ANTHROPIC_MODEL")
		if model == "" {
			model = "claude-3-haiku-20240307"
		}
		service, err := aifactory.NewService(aifactory.ServiceTypeClaude, apiKey, model)
		if err == nil {
			b := runtime.NewAIServiceBrain(service)
			if err := b.Initialize(ctx); err == nil {
				return b, fmt.Sprintf("Claude (%s)", model)
			}
		}
	}

	// Try OpenAI
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		model := os.Getenv("OPENAI_MODEL")
		if model == "" {
			model = "gpt-4o-mini"
		}
		service, err := aifactory.NewService(aifactory.ServiceTypeOpenAI, apiKey, model)
		if err == nil {
			b := runtime.NewAIServiceBrain(service)
			if err := b.Initialize(ctx); err == nil {
				return b, fmt.Sprintf("OpenAI (%s)", model)
			}
		}
	}

	// Fallback to simple brain
	fmt.Println("⚠️  No AI API key found - using rule-based SimpleBrain")
	fmt.Println("   Set ANTHROPIC_API_KEY or OPENAI_API_KEY for AI-powered evaluation")
	b := runtime.NewSimpleBrain()
	b.Initialize(ctx)
	return b, "SimpleBrain (rule-based fallback)"
}

// runTest runs an automated E2E test with sample leads
func runTest() {
	fmt.Println("🧪 NimsForest E2E Test")
	fmt.Println("======================")
	fmt.Println()

	// 1. Start embedded NATS server
	fmt.Println("📡 Starting embedded NATS server...")
	opts := &server.Options{
		Host: "127.0.0.1",
		Port: -1, // Random port for testing
	}
	ns, err := server.NewServer(opts)
	if err != nil {
		log.Fatalf("❌ Failed to create NATS server: %v\n", err)
	}
	go ns.Start()
	if !ns.ReadyForConnections(5 * time.Second) {
		log.Fatal("❌ NATS server not ready")
	}
	defer ns.Shutdown()
	fmt.Printf("✅ NATS server at %s\n", ns.ClientURL())

	// 2. Connect to NATS
	nc, err := nats.Connect(ns.ClientURL())
	if err != nil {
		log.Fatalf("❌ Failed to connect: %v\n", err)
	}
	defer nc.Close()

	// 3. Create Wind
	wind := core.NewWind(nc)

	// 4. Load config
	configPath := findConfig()
	if configPath == "" {
		log.Fatal("❌ No config/forest.yaml found")
	}

	cfg, err := runtime.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v\n", err)
	}

	// 5. Create brain
	ctx := context.Background()
	nimBrain, brainType := createBrain(ctx)
	fmt.Printf("🧠 Brain: %s\n", brainType)

	// 6. Create and start forest
	forest, err := runtime.NewForestFromConfig(cfg, wind, nimBrain)
	if err != nil {
		log.Fatalf("❌ Failed to create forest: %v\n", err)
	}

	if err := forest.Start(ctx); err != nil {
		log.Fatalf("❌ Failed to start forest: %v\n", err)
	}
	defer forest.Stop()

	fmt.Println("✅ Forest running")
	fmt.Println()

	// 7. Collect results
	results := make(chan map[string]interface{}, 10)
	
	wind.Catch("lead.qualified", func(leaf core.Leaf) {
		var data map[string]interface{}
		json.Unmarshal(leaf.Data, &data)
		results <- data
	})

	time.Sleep(200 * time.Millisecond)

	// 8. Define test cases
	testCases := []struct {
		name        string
		id          string
		email       string
		title       string
		companySize int
		industry    string
		expectScore string
		expectPursue bool
	}{
		{
			name:        "Enterprise CTO (should pursue)",
			id:          "test-001",
			email:       "cto@enterprise.com",
			title:       "CTO",
			companySize: 800,
			industry:    "technology",
			expectScore: "~105",
			expectPursue: true,
		},
		{
			name:        "Mid-market VP (should pursue)",
			id:          "test-002",
			email:       "vp@midsize.com",
			title:       "VP Engineering",
			companySize: 200,
			industry:    "finance",
			expectScore: "~85",
			expectPursue: true,
		},
		{
			name:        "Small startup engineer (should not pursue)",
			id:          "test-003",
			email:       "dev@tiny.io",
			title:       "Software Engineer",
			companySize: 10,
			industry:    "retail",
			expectScore: "~0",
			expectPursue: false,
		},
	}

	// 9. Run tests
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	passed := 0
	failed := 0

	for i, tc := range testCases {
		fmt.Printf("\n📤 Test %d: %s\n", i+1, tc.name)
		
		// Send test contact
		contact := map[string]interface{}{
			"id":           tc.id,
			"email":        tc.email,
			"title":        tc.title,
			"company_size": tc.companySize,
			"industry":     tc.industry,
		}
		data, _ := json.Marshal(contact)
		leaf := core.NewLeaf("contact.created", data, "test")
		wind.Drop(*leaf)

		// Wait for result
		select {
		case result := <-results:
			pursue, _ := result["pursue"].(bool)
			reason, _ := result["reason"].(string)
			priority, _ := result["priority"].(string)
			
			// Truncate reason for display
			if len(reason) > 80 {
				reason = reason[:80] + "..."
			}

			if pursue == tc.expectPursue {
				fmt.Printf("   ✅ PASS: pursue=%v priority=%v\n", pursue, priority)
				fmt.Printf("   📝 %s\n", reason)
				passed++
			} else {
				fmt.Printf("   ❌ FAIL: expected pursue=%v, got %v\n", tc.expectPursue, pursue)
				fmt.Printf("   📝 %s\n", reason)
				failed++
			}

		case <-time.After(10 * time.Second):
			fmt.Printf("   ❌ FAIL: timeout waiting for result\n")
			failed++
		}
	}

	// 10. Summary
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📊 Results: %d passed, %d failed\n", passed, failed)
	
	if failed == 0 {
		fmt.Println("✅ All tests passed!")
	} else {
		fmt.Println("❌ Some tests failed")
		os.Exit(1)
	}
}
