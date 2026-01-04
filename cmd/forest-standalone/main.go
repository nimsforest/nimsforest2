// Standalone Forest Runner - for testing without cluster config
// Run: go run ./cmd/forest-standalone
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
