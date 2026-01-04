// MVP Demo - Demonstrates the runtime flow
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

	// 3. Find config path
	configPath := findConfig()
	fmt.Printf("📄 Using config: %s\n", configPath)

	// 4. Load config
	cfg, err := runtime.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("❌ Failed to load config: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Loaded config with %d treehouses and %d nims\n",
		len(cfg.TreeHouses), len(cfg.Nims))

	// 5. Create mock brain (simulates AI)
	mockBrain := brain.NewMockBrain()
	mockBrain.SetRawResponse(`{"pursue": true, "reason": "High score indicates strong fit"}`)
	fmt.Println("🧠 Mock brain ready (simulates AI responses)")

	// 6. Create and start forest
	forest, err := runtime.NewForestFromConfig(cfg, nc, mockBrain)
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

	// 7. Subscribe to output subjects to show results
	fmt.Println("📥 Subscribing to output subjects...")
	
	nc.Subscribe("lead.scored", func(msg *nats.Msg) {
		var data map[string]interface{}
		json.Unmarshal(msg.Data, &data)
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("📊 LEAD SCORED (from TreeHouse)")
		fmt.Printf("   Contact ID: %v\n", data["contact_id"])
		fmt.Printf("   Score: %v\n", data["score"])
		fmt.Printf("   Signals: %v\n", data["signals"])
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	})

	nc.Subscribe("lead.qualified", func(msg *nats.Msg) {
		var data map[string]interface{}
		json.Unmarshal(msg.Data, &data)
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("🎯 LEAD QUALIFIED (from Nim/AI)")
		fmt.Printf("   Pursue: %v\n", data["pursue"])
		fmt.Printf("   Reason: %v\n", data["reason"])
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	})

	time.Sleep(100 * time.Millisecond)

	// 8. Send test contacts
	fmt.Println()
	fmt.Println("📤 Sending test contacts...")
	fmt.Println()

	// Test contact 1: VP at mid-size tech company
	contact1 := map[string]interface{}{
		"id":           "contact-001",
		"email":        "jane@acme.com",
		"title":        "VP Engineering",
		"company_size": 250,
		"industry":     "technology",
	}
	publishContact(nc, contact1)
	time.Sleep(500 * time.Millisecond)

	// Test contact 2: CEO at enterprise finance company
	contact2 := map[string]interface{}{
		"id":           "contact-002",
		"email":        "bob@bigbank.com",
		"title":        "CEO",
		"company_size": 1000,
		"industry":     "finance",
	}
	publishContact(nc, contact2)
	time.Sleep(500 * time.Millisecond)

	// Test contact 3: Engineer at small startup
	contact3 := map[string]interface{}{
		"id":           "contact-003",
		"email":        "dev@startup.io",
		"title":        "Software Engineer",
		"company_size": 15,
		"industry":     "healthcare",
	}
	publishContact(nc, contact3)
	time.Sleep(500 * time.Millisecond)

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ Demo complete! The MVP flow is working:")
	fmt.Println()
	fmt.Println("   contact.created → TreeHouse (Lua) → lead.scored")
	fmt.Println("                                          ↓")
	fmt.Println("   lead.qualified ← Nim (AI) ← ←  ← ← ← ← ←")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("Press Ctrl+C to exit, or send more contacts:")
	fmt.Println("  nats pub contact.created '{\"id\":\"test\",\"email\":\"x@y.com\",\"title\":\"CTO\",\"company_size\":500,\"industry\":\"technology\"}'")
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

func publishContact(nc *nats.Conn, contact map[string]interface{}) {
	data, _ := json.Marshal(contact)
	fmt.Printf("📤 Publishing contact: %s (%s at %v-person %s company)\n",
		contact["id"], contact["title"], contact["company_size"], contact["industry"])
	nc.Publish("contact.created", data)
	nc.Flush()
}
