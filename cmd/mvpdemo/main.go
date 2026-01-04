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
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/yourusername/nimsforest/internal/core"
	"github.com/yourusername/nimsforest/pkg/brain"
	"github.com/yourusername/nimsforest/pkg/runtime"
)

// SmartMockBrain evaluates leads based on the score in the prompt
type SmartMockBrain struct {
	*brain.MockBrain
}

func NewSmartMockBrain() *SmartMockBrain {
	return &SmartMockBrain{
		MockBrain: brain.NewMockBrain(),
	}
}

func (s *SmartMockBrain) Ask(ctx context.Context, prompt string) (string, error) {
	// Extract score from prompt using regex
	scoreRegex := regexp.MustCompile(`Score:\s*(\d+)`)
	matches := scoreRegex.FindStringSubmatch(prompt)
	
	score := 0
	if len(matches) > 1 {
		score, _ = strconv.Atoi(matches[1])
	}

	// Extract signals
	signalsRegex := regexp.MustCompile(`Signals:\s*\[([^\]]*)\]`)
	signalMatches := signalsRegex.FindStringSubmatch(prompt)
	signals := ""
	if len(signalMatches) > 1 {
		signals = signalMatches[1]
	}

	// Make decision based on score
	var response map[string]interface{}
	
	if score >= 100 {
		response = map[string]interface{}{
			"pursue":   true,
			"reason":   fmt.Sprintf("Excellent lead! Score of %d with signals [%s] indicates high-value enterprise prospect with decision-making authority.", score, signals),
			"priority": "high",
			"action":   "Schedule executive demo within 24 hours",
		}
	} else if score >= 70 {
		response = map[string]interface{}{
			"pursue":   true,
			"reason":   fmt.Sprintf("Good lead. Score of %d meets threshold. Signals [%s] suggest potential fit.", score, signals),
			"priority": "medium",
			"action":   "Add to nurture sequence and schedule discovery call",
		}
	} else if score >= 40 {
		response = map[string]interface{}{
			"pursue":   false,
			"reason":   fmt.Sprintf("Below threshold. Score of %d with signals [%s] - may revisit if company grows or role changes.", score, signals),
			"priority": "low",
			"action":   "Add to long-term nurture list",
		}
	} else {
		response = map[string]interface{}{
			"pursue":   false,
			"reason":   fmt.Sprintf("Not a fit. Score of %d indicates small company or non-decision-maker role.", score),
			"priority": "none",
			"action":   "No action required",
		}
	}

	jsonBytes, _ := json.Marshal(response)
	return string(jsonBytes), nil
}

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

	// 6. Create smart mock brain that actually evaluates leads
	smartBrain := NewSmartMockBrain()
	fmt.Println("🧠 Smart brain ready (evaluates leads based on score)")

	// 7. Create and start forest using Wind
	forest, err := runtime.NewForestFromConfig(cfg, wind, smartBrain)
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
		fmt.Println("📊 LEAD SCORED (TreeHouse/Lua)")
		fmt.Printf("   Contact: %v\n", data["contact_id"])
		fmt.Printf("   Score:   %v\n", data["score"])
		fmt.Printf("   Signals: %v\n", data["signals"])
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	})

	wind.Catch("lead.qualified", func(leaf core.Leaf) {
		var data map[string]interface{}
		json.Unmarshal(leaf.Data, &data)
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		pursue := data["pursue"].(bool)
		if pursue {
			fmt.Println("🎯 LEAD QUALIFIED (Nim/AI) - ✅ PURSUE")
		} else {
			fmt.Println("🎯 LEAD QUALIFIED (Nim/AI) - ❌ PASS")
		}
		fmt.Printf("   Pursue:   %v\n", data["pursue"])
		fmt.Printf("   Priority: %v\n", data["priority"])
		fmt.Printf("   Reason:   %v\n", data["reason"])
		fmt.Printf("   Action:   %v\n", data["action"])
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	})

	time.Sleep(100 * time.Millisecond)

	// 9. Send test contacts via Wind
	fmt.Println()
	fmt.Println("📤 Testing lead qualification with different profiles...")
	fmt.Println()

	// Test contact 1: VP at mid-size tech company (score ~85)
	contact1 := map[string]interface{}{
		"id":           "lead-001",
		"email":        "vp@midtech.com",
		"title":        "VP Engineering",
		"company_size": 250,
		"industry":     "technology",
	}
	fmt.Println("👤 Test 1: VP Engineering at 250-person tech company")
	dropContact(wind, contact1)
	time.Sleep(300 * time.Millisecond)

	// Test contact 2: CEO at enterprise finance company (score ~105)
	contact2 := map[string]interface{}{
		"id":           "lead-002",
		"email":        "ceo@bigbank.com",
		"title":        "CEO",
		"company_size": 1000,
		"industry":     "finance",
	}
	fmt.Println("\n👤 Test 2: CEO at 1000-person finance company")
	dropContact(wind, contact2)
	time.Sleep(300 * time.Millisecond)

	// Test contact 3: Manager at small company (score ~50)
	contact3 := map[string]interface{}{
		"id":           "lead-003",
		"email":        "manager@smallco.io",
		"title":        "Product Manager",
		"company_size": 80,
		"industry":     "retail",
	}
	fmt.Println("\n👤 Test 3: Product Manager at 80-person retail company")
	dropContact(wind, contact3)
	time.Sleep(300 * time.Millisecond)

	// Test contact 4: Engineer at tiny startup (score ~0)
	contact4 := map[string]interface{}{
		"id":           "lead-004",
		"email":        "dev@tinystartup.io",
		"title":        "Software Engineer",
		"company_size": 8,
		"industry":     "healthcare",
	}
	fmt.Println("\n👤 Test 4: Software Engineer at 8-person startup")
	dropContact(wind, contact4)
	time.Sleep(300 * time.Millisecond)

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ Demo complete!")
	fmt.Println()
	fmt.Println("The Nim evaluated each lead based on:")
	fmt.Println("  • Score >= 100: HIGH priority, schedule demo")
	fmt.Println("  • Score >= 70:  MEDIUM priority, discovery call")
	fmt.Println("  • Score >= 40:  LOW priority, nurture list")
	fmt.Println("  • Score < 40:   NO ACTION")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("Try sending your own leads via NATS CLI:")
	fmt.Println(`  nats pub contact.created '{"subject":"contact.created","data":{"id":"test","email":"x@y.com","title":"CTO","company_size":500,"industry":"technology"},"source":"cli","ts":"2026-01-01T00:00:00Z"}'`)
	fmt.Println()
	fmt.Println("Press Ctrl+C to exit...")

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

	return "config/forest.yaml"
}

func dropContact(wind *core.Wind, contact map[string]interface{}) {
	data, _ := json.Marshal(contact)
	leaf := core.NewLeaf("contact.created", data, "demo")
	if err := wind.Drop(*leaf); err != nil {
		fmt.Printf("❌ Error dropping leaf: %v\n", err)
	}
}
