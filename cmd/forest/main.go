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

	"github.com/yourusername/nimsforest/internal/core"
	"github.com/yourusername/nimsforest/internal/natsclusterconfig"
	"github.com/yourusername/nimsforest/internal/natsembed"
	"github.com/yourusername/nimsforest/internal/nims"
	"github.com/yourusername/nimsforest/internal/trees"
	"github.com/yourusername/nimsforest/internal/updater"
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
	fmt.Println("  (no command)    Start the NimsForest event processor")
	fmt.Println("  run, start      Start the NimsForest event processor")
	fmt.Println("  version         Show version information")
	fmt.Println("  update          Check for updates and install if available")
	fmt.Println("  check-update    Check for updates without installing")
	fmt.Println("  help            Show this help message")
	fmt.Println()
	fmt.Println("Required Configuration:")
	fmt.Println("  /etc/nimsforest/node-info.json  Node identity (forest_id, node_id)")
	fmt.Println("  /mnt/forest/registry.json       Cluster registry (shared storage)")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  NATS_CLUSTER_NODE_INFO  Override node info path")
	fmt.Println("  NATS_CLUSTER_REGISTRY   Override registry path")
	fmt.Println("  JETSTREAM_DIR           JetStream data directory (default: /var/lib/nimsforest/jetstream)")
	fmt.Println("  DEMO                    Set to 'true' to run demo mode")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  forest                          # Start the forest")
	fmt.Println("  forest version                  # Show version")
	fmt.Println("  forest check-update             # Check for new versions")
	fmt.Println("  forest update                   # Update to latest version")
	fmt.Println("  DEMO=true forest                # Start with demo mode")
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
