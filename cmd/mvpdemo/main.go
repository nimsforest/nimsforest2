// MVP Demo - Demonstrates the runtime flow using Wind
// Run: go run ./cmd/mvpdemo
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/yourusername/nimsforest/internal/core"
	"github.com/yourusername/nimsforest/pkg/brain"
	"github.com/yourusername/nimsforest/pkg/runtime"
)

func main() {
	fmt.Println("🌲 NimsForest MVP Demo")
	fmt.Println("======================")
	fmt.Println()

	// 1. Start embedded NATS server
	fmt.Println("📡 Starting embedded NATS server...")
	opts := &server.Options{
		Host: "127.0.0.1",
		Port: 4222,
	}
	ns, err := server.NewServer(opts)
	if err != nil {
		fmt.Printf("❌ Failed to create NATS server: %v\n", err)
		os.Exit(1)
	}
	go ns.Start()
	if !ns.ReadyForConnections(5 * time.Second) {
		fmt.Println("❌ NATS server not ready")
		os.Exit(1)
	}
	defer ns.Shutdown()
	fmt.Printf("✅ NATS server running at %s\n", ns.ClientURL())

	// 2. Connect to NATS
	nc, err := nats.Connect(ns.ClientURL())
	if err != nil {
		fmt.Printf("❌ Failed to connect to NATS: %v\n", err)
		os.Exit(1)
	}
	defer nc.Close()
	fmt.Println("✅ Connected to NATS")

	// 3. Create Wind (the pub/sub layer)
	wind := core.NewWind(nc)
	fmt.Println("🌬️  Wind initialized for pub/sub")

	// 4. Find config path
	configPath := findConfig()
	fmt.Printf("📄 Using config: %s\n", configPath)

	// 5. Load config
	cfg, err := runtime.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("❌ Failed to load config: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Loaded config with %d treehouses and %d nims\n",
		len(cfg.TreeHouses), len(cfg.Nims))

	// 6. Create mock brain (simulates AI)
	mockBrain := brain.NewMockBrain()
	mockBrain.SetRawResponse(`{"pursue": true, "reason": "High score indicates strong fit"}`)
	fmt.Println("🧠 Mock brain ready (simulates AI responses)")

	// 7. Create and start forest using Wind
	forest, err := runtime.NewForestFromConfig(cfg, wind, mockBrain)
	if err != nil {
		fmt.Printf("❌ Failed to create forest: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := forest.Start(ctx); err != nil {
		fmt.Printf("❌ Failed to start forest: %v\n", err)
		os.Exit(1)
	}
	defer forest.Stop()

	fmt.Println()
	fmt.Println("🌲 Forest is running!")
	fmt.Println()

	// 8. Subscribe to output subjects via Wind to show results
	fmt.Println("📥 Catching leaves on output subjects via Wind...")

	wind.Catch("lead.scored", func(leaf core.Leaf) {
		var data map[string]interface{}
		json.Unmarshal(leaf.Data, &data)
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("📊 LEAD SCORED (from TreeHouse via Wind)")
		fmt.Printf("   Source: %s\n", leaf.Source)
		fmt.Printf("   Contact ID: %v\n", data["contact_id"])
		fmt.Printf("   Score: %v\n", data["score"])
		fmt.Printf("   Signals: %v\n", data["signals"])
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	})

	wind.Catch("lead.qualified", func(leaf core.Leaf) {
		var data map[string]interface{}
		json.Unmarshal(leaf.Data, &data)
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("🎯 LEAD QUALIFIED (from Nim/AI via Wind)")
		fmt.Printf("   Source: %s\n", leaf.Source)
		fmt.Printf("   Pursue: %v\n", data["pursue"])
		fmt.Printf("   Reason: %v\n", data["reason"])
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	})

	time.Sleep(100 * time.Millisecond)

	// 9. Send test contacts via Wind
	fmt.Println()
	fmt.Println("📤 Dropping test contact leaves via Wind...")
	fmt.Println()

	// Test contact 1: VP at mid-size tech company
	contact1 := map[string]interface{}{
		"id":           "contact-001",
		"email":        "jane@acme.com",
		"title":        "VP Engineering",
		"company_size": 250,
		"industry":     "technology",
	}
	dropContact(wind, contact1)
	time.Sleep(500 * time.Millisecond)

	// Test contact 2: CEO at enterprise finance company
	contact2 := map[string]interface{}{
		"id":           "contact-002",
		"email":        "bob@bigbank.com",
		"title":        "CEO",
		"company_size": 1000,
		"industry":     "finance",
	}
	dropContact(wind, contact2)
	time.Sleep(500 * time.Millisecond)

	// Test contact 3: Engineer at small startup
	contact3 := map[string]interface{}{
		"id":           "contact-003",
		"email":        "dev@startup.io",
		"title":        "Software Engineer",
		"company_size": 15,
		"industry":     "healthcare",
	}
	dropContact(wind, contact3)
	time.Sleep(500 * time.Millisecond)

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ Demo complete! The MVP flow is working:")
	fmt.Println()
	fmt.Println("   Wind.Drop(contact.created)")
	fmt.Println("           ↓")
	fmt.Println("   TreeHouse catches via Wind → Lua process")
	fmt.Println("           ↓")
	fmt.Println("   Wind.Drop(lead.scored)")
	fmt.Println("           ↓")
	fmt.Println("   Nim catches via Wind → AI process")
	fmt.Println("           ↓")
	fmt.Println("   Wind.Drop(lead.qualified)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("Press Ctrl+C to exit...")
	fmt.Println()

	// Wait for shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	fmt.Println("\n👋 Shutting down...")
}

func findConfig() string {
	// Try various locations
	paths := []string{
		"config/forest.yaml",
		"../config/forest.yaml",
		"../../config/forest.yaml",
	}

	// Also try from executable directory
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		paths = append(paths, filepath.Join(dir, "config/forest.yaml"))
		paths = append(paths, filepath.Join(dir, "../config/forest.yaml"))
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}

	// Default
	return "config/forest.yaml"
}

func dropContact(wind *core.Wind, contact map[string]interface{}) {
	data, _ := json.Marshal(contact)
	leaf := core.NewLeaf("contact.created", data, "demo")
	fmt.Printf("🍃 Dropping leaf: contact %s (%s at %v-person %s company)\n",
		contact["id"], contact["title"], contact["company_size"], contact["industry"])
	if err := wind.Drop(*leaf); err != nil {
		fmt.Printf("❌ Error dropping leaf: %v\n", err)
	}
}
