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
	"github.com/yourusername/nimsforest/internal/natsclusterconfig"
	"github.com/yourusername/nimsforest/internal/natsembed"
	"github.com/yourusername/nimsforest/internal/nims"
	"github.com/yourusername/nimsforest/internal/trees"
	"github.com/yourusername/nimsforest/internal/updater"
	"github.com/yourusername/nimsforest/pkg/brain"
	aifactory "github.com/yourusername/nimsforest/pkg/integrations/aiservice"
	_ "github.com/yourusername/nimsforest/pkg/integrations/aiservice/thirdparty/claude"
	_ "github.com/yourusername/nimsforest/pkg/integrations/aiservice/thirdparty/openai"
	"github.com/yourusername/nimsforest/pkg/runtime"
)

// version is set at build time via -ldflags
var version = "dev"

func main() {
	// Handle command-line arguments
	if len(os.Args) > 1 {
		command := os.Args[1]
		switch command {
		case "version", "--version", "-v":
			fmt.Printf("forest version %s\n", version)
			return
		case "update":
			handleUpdate()
			return
		case "check-update":
			handleCheckUpdate()
			return
		case "help", "--help", "-h":
			printHelp()
			return
		case "run", "start":
			// Continue to run the forest
		case "standalone":
			runStandalone()
			return
		case "test":
			runTest()
			return
		case "viewmodel":
			handleViewmodel(os.Args[2:])
			return
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
			printHelp()
			os.Exit(1)
		}
	}

	// Run the forest
	runForest()
}

func handleUpdate() {
	fmt.Println("🔄 Checking for updates...")

	u := updater.NewUpdater(version)

	info, err := u.CheckForUpdate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to check for updates: %v\n", err)
		os.Exit(1)
	}

	if !info.Available {
		fmt.Printf("✅ You're already running the latest version (%s)\n", version)
		return
	}

	fmt.Printf("\n🆕 New version available: %s → %s\n", info.CurrentVersion, info.LatestVersion)
	fmt.Printf("   Release URL: %s\n\n", info.UpdateURL)

	if info.ReleaseNotes != "" {
		fmt.Println("📝 Release Notes:")
		fmt.Println(info.ReleaseNotes)
		fmt.Println()
	}

	fmt.Println("📦 Starting update...")
	if err := u.PerformUpdate(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Update failed: %v\n", err)
		os.Exit(1)
	}
}

func handleCheckUpdate() {
	fmt.Println("🔍 Checking for updates...")

	u := updater.NewUpdater(version)

	info, err := u.CheckForUpdate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to check for updates: %v\n", err)
		os.Exit(1)
	}

	updater.PrintUpdateInfo(info)
}

func printHelp() {
	fmt.Println("🌲 NimsForest - Event-Driven Organizational Orchestration")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  forest [command]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  (no command)    Start the NimsForest event processor (requires cluster config)")
	fmt.Println("  run, start      Start the NimsForest event processor (requires cluster config)")
	fmt.Println("  standalone      Start standalone mode (embedded NATS, no cluster config needed)")
	fmt.Println("  test            Run E2E tests with sample leads")
	fmt.Println("  viewmodel       View cluster state (print, summary)")
	fmt.Println("  version         Show version information")
	fmt.Println("  update          Check for updates and install if available")
	fmt.Println("  check-update    Check for updates without installing")
	fmt.Println("  help            Show this help message")
	fmt.Println()
	fmt.Println("Standalone Mode (for development/testing):")
	fmt.Println("  forest standalone               # Start with embedded NATS server")
	fmt.Println("  forest test                     # Run E2E tests")
	fmt.Println()
	fmt.Println("Required for cluster mode:")
	fmt.Println("  /etc/nimsforest/node-info.json  Node identity (forest_id, node_id)")
	fmt.Println("  /mnt/forest/registry.json       Cluster registry (shared storage)")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  ANTHROPIC_API_KEY       Claude API key for AI-powered Nims")
	fmt.Println("  OPENAI_API_KEY          OpenAI API key (alternative to Claude)")
	fmt.Println("  FOREST_CONFIG           Path to forest.yaml config file")
	fmt.Println("  NATS_CLUSTER_NODE_INFO  Override node info path")
	fmt.Println("  NATS_CLUSTER_REGISTRY   Override registry path")
	fmt.Println("  JETSTREAM_DIR           JetStream data directory")
	fmt.Println("  DEMO                    Set to 'true' to run demo mode")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  forest standalone                          # Development mode")
	fmt.Println("  forest test                                # Run E2E tests")
	fmt.Println("  ANTHROPIC_API_KEY=sk-... forest standalone # With Claude AI")
	fmt.Println("  forest                                     # Production cluster mode")
	fmt.Println()
	fmt.Println("More info: https://github.com/yourusername/nimsforest")
}

func runForest() {
	printBanner()

	fmt.Printf("🌲 Starting NimsForest...\n")

	// Load cluster configuration (required)
	fmt.Println("Loading cluster configuration...")
	nodeInfo := natsclusterconfig.MustLoad()
	fmt.Printf("  forest_id: %s\n", nodeInfo.ForestID)
	fmt.Printf("  node_id: %s\n", nodeInfo.NodeID)

	// Get local IPv6 and cluster peers
	localIP := natsembed.GetLocalIPv6()
	if localIP == "" {
		log.Fatalf("❌ No IPv6 address found. NimsForest requires IPv6.\n")
	}
	fmt.Printf("  local_ip: %s\n", localIP)

	peers := natsclusterconfig.GetPeers(nodeInfo.ForestID, localIP)
	if len(peers) > 0 {
		fmt.Printf("  peers: %v\n", peers)
	} else {
		fmt.Println("  peers: none (first node in cluster)")
	}

	// Create embedded NATS server
	fmt.Println("Starting embedded NATS server...")
	cfg := natsembed.Config{
		NodeName:    nodeInfo.NodeID,
		ClusterName: nodeInfo.ForestID,
		DataDir:     getDataDir(),
		Peers:       peers,
		// MonitorPort defaults to 8222 for HTTP monitoring
	}

	ns, err := natsembed.New(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to create embedded NATS server: %v\n", err)
	}

	// Start the embedded server
	if err := ns.Start(); err != nil {
		log.Fatalf("❌ Failed to start embedded NATS server: %v\n", err)
	}
	fmt.Printf("✅ Embedded NATS server started at %s\n", ns.ClientURL())
	if monitorURL := ns.MonitorURL(); monitorURL != "" {
		fmt.Printf("📊 HTTP monitoring available at %s\n", monitorURL)
	}

	// Get client connection to embedded server
	nc, err := ns.ClientConn()
	if err != nil {
		log.Fatalf("❌ Failed to connect to embedded NATS: %v\n", err)
	}
	fmt.Println("✅ Connected to embedded NATS")

	// Get JetStream context
	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("❌ Failed to get JetStream context: %v\n", err)
	}
	fmt.Println("✅ JetStream context created")

	defer func() {
		nc.Close()
		ns.Shutdown()
	}()

	// Initialize core components
	fmt.Println("Initializing core components...")

	wind := core.NewWind(nc)
	fmt.Println("  ✅ Wind (NATS Pub/Sub) ready")

	river, err := core.NewRiver(js)
	if err != nil {
		log.Fatalf("❌ Failed to create river: %v\n", err)
	}
	fmt.Println("  ✅ River (External Data Stream) ready")

	humus, err := core.NewHumus(js)
	if err != nil {
		log.Fatalf("❌ Failed to create humus: %v\n", err)
	}
	fmt.Println("  ✅ Humus (State Change Stream) ready")

	soil, err := core.NewSoil(js)
	if err != nil {
		log.Fatalf("❌ Failed to create soil: %v\n", err)
	}
	fmt.Println("  ✅ Soil (KV Store) ready")

	// Start decomposer worker
	fmt.Println("Starting decomposer worker...")
	decomposer, err := core.RunDecomposer(humus, soil)
	if err != nil {
		log.Fatalf("❌ Failed to start decomposer: %v\n", err)
	}
	defer decomposer.Stop()
	fmt.Println("  ✅ Decomposer worker running")

	// Create context for lifecycle management
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Plant trees
	fmt.Println("Planting trees...")

	// Specific payment tree for Stripe webhooks
	paymentTree := trees.NewPaymentTree(wind, river)
	if err := paymentTree.Start(ctx); err != nil {
		log.Fatalf("❌ Failed to start payment tree: %v\n", err)
	}
	defer paymentTree.Stop()
	fmt.Println("  🌳 PaymentTree planted (watches: river.stripe.webhook)")

	// General tree that demonstrates extensibility
	generalTree := trees.NewGeneralTree(wind, river)
	if err := generalTree.Start(ctx); err != nil {
		log.Fatalf("❌ Failed to start general tree: %v\n", err)
	}
	defer generalTree.Stop()
	fmt.Println("  🌳 GeneralTree planted (watches: river.general.>)")

	// Awaken nims
	fmt.Println("Awakening nims...")

	// Specific aftersales nim for payment events
	afterSalesNim := nims.NewAfterSalesNim(wind, humus, soil)
	if err := afterSalesNim.Start(ctx); err != nil {
		log.Fatalf("❌ Failed to start aftersales nim: %v\n", err)
	}
	defer afterSalesNim.Stop()
	fmt.Println("  🧚 AfterSalesNim awake (catches: payment.completed, payment.failed)")

	// General nim that demonstrates extensibility
	generalNim := nims.NewGeneralNim(wind, humus, soil)
	if err := generalNim.Start(ctx); err != nil {
		log.Fatalf("❌ Failed to start general nim: %v\n", err)
	}
	defer generalNim.Stop()
	fmt.Println("  🧚 GeneralNim awake (catches: data.received, status.update, etc.)")

	// Start YAML-configured runtime (TreeHouses + AI Nims)
	var runtimeForest *runtime.Forest
	configPath := getConfigPath()
	if configPath != "" {
		fmt.Println("\n🌳 Loading YAML-configured runtime...")
		fmt.Printf("   Config: %s\n", configPath)

		cfg, err := runtime.LoadConfig(configPath)
		if err != nil {
			log.Printf("⚠️  Failed to load runtime config: %v\n", err)
		} else {
			// Create brain from AI service
			aiBrain, err := createBrain()
			if err != nil {
				log.Printf("⚠️  Failed to create AI brain: %v\n", err)
				log.Println("   Using simple fallback brain (set ANTHROPIC_API_KEY for AI)")
				simpleBrain := runtime.NewSimpleBrain()
				if err := simpleBrain.Initialize(ctx); err != nil {
					log.Printf("⚠️  Failed to initialize simple brain: %v\n", err)
				} else {
					runtimeForest, err = runtime.NewForestWithHumus(cfg, wind, humus, simpleBrain)
					if err != nil {
						log.Printf("⚠️  Failed to create runtime forest: %v\n", err)
					}
				}
			} else {
				if err := aiBrain.Initialize(ctx); err != nil {
					log.Printf("⚠️  Failed to initialize AI brain: %v\n", err)
				} else {
					runtimeForest, err = runtime.NewForestWithHumus(cfg, wind, humus, aiBrain)
					if err != nil {
						log.Printf("⚠️  Failed to create runtime forest: %v\n", err)
					}
				}
			}

			if runtimeForest != nil {
				if err := runtimeForest.Start(ctx); err != nil {
					log.Printf("⚠️  Failed to start runtime forest: %v\n", err)
				} else {
					defer runtimeForest.Stop()
					fmt.Printf("   ✅ Runtime started with %d TreeHouses and %d Nims\n",
						len(cfg.TreeHouses), len(cfg.Nims))

					// List what's running
					for name := range cfg.TreeHouses {
						fmt.Printf("   🏠 TreeHouse:%s (Lua script)\n", name)
					}
					for name := range cfg.Nims {
						fmt.Printf("   🧚 Nim:%s (AI-powered)\n", name)
					}
				}
			}
		}
	}

	// Give components time to initialize
	time.Sleep(500 * time.Millisecond)

	fmt.Println("\n🌲 NimsForest is fully operational!")
	fmt.Printf("   Version: %s\n", version)
	fmt.Println()

	// Check if demo mode is enabled
	if os.Getenv("DEMO") == "true" {
		fmt.Println("📢 Demo mode enabled - sending test data...")
		go sendDemoData(river)
	}

	// Display instructions
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📖 HOW TO EXTEND NIMSFOREST:")
	fmt.Println()
	fmt.Println("1️⃣  ADD YOUR OWN TREE (Data Parser):")
	fmt.Println("   • Copy internal/trees/general.go → your_tree.go")
	fmt.Println("   • Change Patterns() to match your data source")
	fmt.Println("   • Parse data and emit domain-specific leaves")
	fmt.Println("   • Example: CRM webhooks, IoT sensors, API events")
	fmt.Println()
	fmt.Println("2️⃣  ADD YOUR OWN NIM (Business Logic):")
	fmt.Println("   • Copy internal/nims/general.go → your_nim.go")
	fmt.Println("   • Change Subjects() to catch relevant leaves")
	fmt.Println("   • Implement business logic in Handle()")
	fmt.Println("   • Example: Inventory, billing, notifications")
	fmt.Println()
	fmt.Println("3️⃣  TEST YOUR COMPONENTS:")
	fmt.Println("   • Send data: nats pub river.your.subject '{...}'")
	fmt.Println("   • Watch logs to see processing")
	fmt.Println("   • Check soil: Data persisted in KV store")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("💡 TRY THESE EXAMPLES:")
	fmt.Println()
	fmt.Println("Payment webhook:")
	fmt.Println("  nats pub river.stripe.webhook '{\"type\":\"charge.succeeded\",\"data\":{\"object\":{\"id\":\"ch_123\",\"amount\":10000,\"currency\":\"usd\",\"customer\":\"cus_alice\",\"metadata\":{\"item_id\":\"jacket\"}}}}'")
	fmt.Println()
	fmt.Println("General data event:")
	fmt.Println("  nats pub river.general.api '{\"type\":\"data.received\",\"source\":\"api\",\"data\":\"hello world\",\"timestamp\":\"2024-01-01T12:00:00Z\"}'")
	fmt.Println()
	fmt.Println("Status update:")
	fmt.Println("  nats pub river.general.status '{\"type\":\"status.update\",\"entity_id\":\"user-123\",\"status\":\"active\",\"message\":\"User activated\"}'")
	fmt.Println()
	fmt.Println("Monitor logs to see the forest in action!")
	fmt.Println("Press Ctrl+C to stop...")
	fmt.Println()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n🛑 Shutting down gracefully...")
	cancel()
	time.Sleep(500 * time.Millisecond)
	fmt.Println("✅ Shutdown complete")
}

// getDataDir returns the JetStream data directory.
func getDataDir() string {
	dir := os.Getenv("JETSTREAM_DIR")
	if dir != "" {
		return dir
	}
	return "/var/lib/nimsforest/jetstream"
}

// getConfigPath finds the forest.yaml config file.
func getConfigPath() string {
	// Check environment variable first
	if path := os.Getenv("FOREST_CONFIG"); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Check common locations
	paths := []string{
		"config/forest.yaml",
		"/etc/nimsforest/forest.yaml",
		"forest.yaml",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

// createBrain creates an AI service brain from environment configuration.
func createBrain() (*runtime.AIServiceBrain, error) {
	// Check for API keys in order of preference
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	serviceType := aifactory.ServiceTypeClaude
	model := os.Getenv("ANTHROPIC_MODEL")
	if model == "" {
		model = "claude-3-haiku-20240307"
	}

	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
		serviceType = aifactory.ServiceTypeOpenAI
		model = os.Getenv("OPENAI_MODEL")
		if model == "" {
			model = "gpt-4o-mini"
		}
	}

	if apiKey == "" {
		return nil, fmt.Errorf("no AI API key found (set ANTHROPIC_API_KEY or OPENAI_API_KEY)")
	}

	// Create AI service
	service, err := aifactory.NewService(serviceType, apiKey, model)
	if err != nil {
		return nil, fmt.Errorf("failed to create AI service: %w", err)
	}

	// Wrap in brain adapter
	return runtime.NewAIServiceBrain(service), nil
}

func printBanner() {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════╗")
	fmt.Println("║                                                   ║")
	fmt.Println("║           🌲  N I M S F O R E S T  🌲           ║")
	fmt.Println("║                                                   ║")
	fmt.Println("║    Event-Driven Organizational Orchestration      ║")
	fmt.Println("║                                                   ║")
	fmt.Println("╚═══════════════════════════════════════════════════╝")
	fmt.Println()
}

func sendDemoData(river *core.River) {
	time.Sleep(2 * time.Second)

	// Demo 1: Payment webhook
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📨 DEMO 1: Sending payment webhook...")
	fmt.Println("   Tree: PaymentTree will parse this")
	fmt.Println("   Nim: AfterSalesNim will process it")

	webhook := map[string]interface{}{
		"type": "charge.succeeded",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":       "ch_demo_123",
				"amount":   15000, // $150.00 in cents
				"currency": "usd",
				"customer": "cus_demo_alice",
				"metadata": map[string]string{
					"item_id": "premium-jacket",
				},
			},
		},
	}

	webhookData, err := json.Marshal(webhook)
	if err != nil {
		log.Printf("❌ Failed to marshal webhook: %v\n", err)
		return
	}

	if err := river.Flow("river.stripe.webhook", webhookData); err != nil {
		log.Printf("❌ Failed to send to river: %v\n", err)
		return
	}

	fmt.Println("✅ Payment webhook sent!")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Demo 2: General data event
	time.Sleep(3 * time.Second)
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📨 DEMO 2: Sending general data event...")
	fmt.Println("   Tree: GeneralTree will parse this")
	fmt.Println("   Nim: GeneralNim will process it")

	dataEvent := map[string]interface{}{
		"type":      "data.received",
		"source":    "demo-api",
		"data":      "Hello from the forest!",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	dataJSON, err := json.Marshal(dataEvent)
	if err != nil {
		log.Printf("❌ Failed to marshal data event: %v\n", err)
		return
	}

	if err := river.Flow("river.general.api", dataJSON); err != nil {
		log.Printf("❌ Failed to send to river: %v\n", err)
		return
	}

	fmt.Println("✅ Data event sent!")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Demo 3: Status update
	time.Sleep(3 * time.Second)
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📨 DEMO 3: Sending status update...")
	fmt.Println("   Tree: GeneralTree will parse this")
	fmt.Println("   Nim: GeneralNim will update entity state")

	statusEvent := map[string]interface{}{
		"type":      "status.update",
		"entity_id": "user-42",
		"status":    "premium",
		"message":   "User upgraded to premium",
	}

	statusJSON, err := json.Marshal(statusEvent)
	if err != nil {
		log.Printf("❌ Failed to marshal status event: %v\n", err)
		return
	}

	if err := river.Flow("river.general.system", statusJSON); err != nil {
		log.Printf("❌ Failed to send to river: %v\n", err)
		return
	}

	fmt.Println("✅ Status update sent!")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Demo 4: Notification
	time.Sleep(3 * time.Second)
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📨 DEMO 4: Sending high-priority notification...")
	fmt.Println("   Tree: GeneralTree will parse this")
	fmt.Println("   Nim: GeneralNim will route based on priority")

	notifEvent := map[string]interface{}{
		"type":      "notification",
		"priority":  "high",
		"recipient": "admin@example.com",
		"message":   "System alert: High memory usage detected",
	}

	notifJSON, err := json.Marshal(notifEvent)
	if err != nil {
		log.Printf("❌ Failed to marshal notification: %v\n", err)
		return
	}

	if err := river.Flow("river.general.monitoring", notifJSON); err != nil {
		log.Printf("❌ Failed to send to river: %v\n", err)
		return
	}

	fmt.Println("✅ Notification sent!")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("✨ All demo events sent! See processing above.")
	fmt.Println("💡 Now YOU can add your own trees and nims!")
	fmt.Println()
}

// runStandalone runs the forest in standalone mode with embedded NATS
func runStandalone() {
	fmt.Println("🌲 NimsForest Standalone Mode")
	fmt.Println("=============================")
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
	configPath := getConfigPath()
	if configPath == "" {
		log.Fatal("❌ No config/forest.yaml found")
	}
	fmt.Printf("📄 Config: %s\n", configPath)

	cfg, err := runtime.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v\n", err)
	}

	// 5. Create brain
	ctx := context.Background()
	nimBrain, brainType := createBrainWithFallback(ctx)
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

// runTest runs E2E tests with sample leads
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
	configPath := getConfigPath()
	if configPath == "" {
		log.Fatal("❌ No config/forest.yaml found")
	}

	cfg, err := runtime.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v\n", err)
	}

	// 5. Create brain
	ctx := context.Background()
	nimBrain, brainType := createBrainWithFallback(ctx)
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
		name         string
		id           string
		email        string
		title        string
		companySize  int
		industry     string
		expectPursue bool
	}{
		{
			name:         "Enterprise CTO (should pursue)",
			id:           "test-001",
			email:        "cto@enterprise.com",
			title:        "CTO",
			companySize:  800,
			industry:     "technology",
			expectPursue: true,
		},
		{
			name:         "Mid-market VP (should pursue)",
			id:           "test-002",
			email:        "vp@midsize.com",
			title:        "VP Engineering",
			companySize:  200,
			industry:     "finance",
			expectPursue: true,
		},
		{
			name:         "Small startup engineer (should not pursue)",
			id:           "test-003",
			email:        "dev@tiny.io",
			title:        "Software Engineer",
			companySize:  10,
			industry:     "retail",
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

// createBrainWithFallback creates a brain with fallback to SimpleBrain
func createBrainWithFallback(ctx context.Context) (brain.Brain, string) {
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
