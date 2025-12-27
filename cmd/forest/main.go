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

	"github.com/nats-io/nats.go"
	"github.com/yourusername/nimsforest/internal/core"
	"github.com/yourusername/nimsforest/internal/nims"
	"github.com/yourusername/nimsforest/internal/trees"
)

func main() {
	printBanner()

	// Get NATS URL from environment or use default
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	fmt.Printf("🌲 Starting NimsForest...\n")
	fmt.Printf("Connecting to NATS at %s...\n", natsURL)

	// Connect to NATS
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to NATS: %v\n", err)
	}
	defer nc.Close()
	fmt.Println("✅ Connected to NATS")

	// Get JetStream context
	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("❌ Failed to get JetStream context: %v\n", err)
	}
	fmt.Println("✅ JetStream context created")

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
	paymentTree := trees.NewPaymentTree(wind, river)
	if err := paymentTree.Start(ctx); err != nil {
		log.Fatalf("❌ Failed to start payment tree: %v\n", err)
	}
	defer paymentTree.Stop()
	fmt.Println("  🌳 PaymentTree planted and watching river")

	// Awaken nims
	fmt.Println("Awakening nims...")
	afterSalesNim := nims.NewAfterSalesNim(wind, humus, soil)
	if err := afterSalesNim.Start(ctx); err != nil {
		log.Fatalf("❌ Failed to start aftersales nim: %v\n", err)
	}
	defer afterSalesNim.Stop()
	fmt.Println("  🧚 AfterSalesNim awake and catching leaves")

	// Give components time to initialize
	time.Sleep(500 * time.Millisecond)

	fmt.Println("\n🌲 NimsForest is fully operational!")
	fmt.Println()

	// Check if demo mode is enabled
	if os.Getenv("DEMO") == "true" {
		fmt.Println("📢 Demo mode enabled - sending test data...")
		go sendDemoData(river)
	}

	// Display instructions
	fmt.Println("Send test data using:")
	fmt.Println("  nats pub river.stripe.webhook '{\"type\": \"charge.succeeded\", \"data\": {\"object\": {...}}}'")
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

	fmt.Println("\n📨 Sending demo payment webhook...")

	// Create a successful payment webhook (matches Stripe format)
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

	fmt.Println("✅ Demo payment sent! Watch the logs above for processing...")
	fmt.Println()

	// Send a failed payment after a delay
	time.Sleep(3 * time.Second)
	fmt.Println("\n📨 Sending demo failed payment webhook...")

	failedWebhook := map[string]interface{}{
		"type": "charge.failed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":              "ch_demo_456",
				"amount":          5000, // $50.00
				"currency":        "usd",
				"customer":        "cus_demo_bob",
				"failure_message": "insufficient_funds",
				"metadata": map[string]string{
					"item_id": "basic-tee",
				},
			},
		},
	}

	failedData, err := json.Marshal(failedWebhook)
	if err != nil {
		log.Printf("❌ Failed to marshal webhook: %v\n", err)
		return
	}

	if err := river.Flow("river.stripe.webhook", failedData); err != nil {
		log.Printf("❌ Failed to send to river: %v\n", err)
		return
	}

	fmt.Println("✅ Demo failed payment sent!")
	fmt.Println()
}
