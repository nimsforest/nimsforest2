package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/yourusername/nimsforest/internal/natsembed"
	"github.com/yourusername/nimsforest/internal/viewmodel"
	"github.com/yourusername/nimsforest/internal/webview"
)

// handleViewmodel handles the 'viewmodel' command and its subcommands.
func handleViewmodel(args []string) {
	if len(args) < 1 {
		printViewmodelHelp()
		os.Exit(1)
	}

	subcommand := args[0]
	switch subcommand {
	case "print":
		handleViewmodelPrint()
	case "summary":
		handleViewmodelSummary()
	case "webview":
		handleViewmodelWebview(args[1:])
	case "help", "--help", "-h":
		printViewmodelHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown viewmodel subcommand: %s\n\n", subcommand)
		printViewmodelHelp()
		os.Exit(1)
	}
}

// handleViewmodelPrint prints the full territory view.
func handleViewmodelPrint() {
	ns, cleanup := getOrStartNATSServer()
	defer cleanup()

	vm := viewmodel.New(ns)
	if err := vm.Refresh(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to refresh viewmodel: %v\n", err)
		os.Exit(1)
	}

	vm.Print(os.Stdout)
}

// handleViewmodelSummary prints a summary of the territory.
func handleViewmodelSummary() {
	ns, cleanup := getOrStartNATSServer()
	defer cleanup()

	vm := viewmodel.New(ns)
	if err := vm.Refresh(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to refresh viewmodel: %v\n", err)
		os.Exit(1)
	}

	vm.PrintSummary(os.Stdout)
}

// handleViewmodelWebview starts the webview HTTP server.
func handleViewmodelWebview(args []string) {
	port := "8080"

	// Parse --port flag
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--port=") {
			port = strings.TrimPrefix(arg, "--port=")
		} else if arg == "--port" && i+1 < len(args) {
			port = args[i+1]
			i++
		}
	}

	ns, cleanup := getOrStartNATSServer()
	defer cleanup()

	vm := viewmodel.New(ns)

	// Try to find external web/out directory for development
	// If not found, the server will use embedded files
	var webDir string
	possiblePaths := []string{
		"web/out",
		"./web/out",
	}
	for _, p := range possiblePaths {
		if _, err := os.Stat(p); err == nil {
			webDir = p
			break
		}
	}

	var server *webview.Server
	if webDir != "" {
		// Use external files (development mode)
		server = webview.New(vm, os.DirFS(webDir))
		fmt.Printf("🌲 Starting NimsForest webview at http://localhost:%s\n", port)
		fmt.Printf("   Serving static files from: %s\n", webDir)
	} else {
		// Use embedded files (production mode)
		server = webview.New(vm, nil)
		fmt.Printf("🌲 Starting NimsForest webview at http://localhost:%s\n", port)
		fmt.Println("   Serving embedded web frontend")
	}

	fmt.Println("   Press Ctrl+C to stop.")
	fmt.Println()

	if err := http.ListenAndServe(":"+port, server); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Server error: %v\n", err)
		os.Exit(1)
	}
}

// getOrStartNATSServer connects to an existing NATS server or starts an embedded one.
func getOrStartNATSServer() (*server.Server, func()) {
	// First, try to connect to an existing server
	// Check if there's a running instance by looking for the NATS URL in env
	natsURL := os.Getenv("NATS_URL")
	if natsURL != "" {
		// For now, we can't get the *server.Server from an external connection
		// Fall through to embedded server
		fmt.Println("⚠️  External NATS connection not yet supported for viewmodel")
		fmt.Println("   Starting temporary embedded server for inspection...")
	}

	// Start a temporary embedded server
	tmpDir, err := os.MkdirTemp("", "nimsforest-viewmodel-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create temp dir: %v\n", err)
		os.Exit(1)
	}

	cfg := natsembed.Config{
		NodeName:    "viewmodel-inspector",
		ClusterName: "nimsforest",
		DataDir:     filepath.Join(tmpDir, "jetstream"),
		ClientPort:  0,  // Random port
		MonitorPort: -1, // No monitoring
	}

	ns, err := natsembed.New(cfg)
	if err != nil {
		os.RemoveAll(tmpDir)
		fmt.Fprintf(os.Stderr, "❌ Failed to create embedded NATS: %v\n", err)
		os.Exit(1)
	}

	if err := ns.Start(); err != nil {
		os.RemoveAll(tmpDir)
		fmt.Fprintf(os.Stderr, "❌ Failed to start embedded NATS: %v\n", err)
		os.Exit(1)
	}

	// Give server time to fully initialize
	time.Sleep(100 * time.Millisecond)

	// Return the server's internal *server.Server
	// We need to get the internal server from natsembed.Server
	// For now, we'll need to expose it or use a different approach

	// Actually, natsembed.Server wraps *server.Server but doesn't expose it
	// We need to either:
	// 1. Add a method to natsembed.Server to expose the internal server
	// 2. Create the server.Server directly here

	// Let's create it directly for the viewmodel command
	opts := &server.Options{
		ServerName: cfg.NodeName,
		Host:       "127.0.0.1",
		Port:       -1, // Random port
		JetStream:  true,
		StoreDir:   cfg.DataDir,
	}

	// Shutdown the natsembed server since we'll create our own
	ns.Shutdown()

	internalServer, err := server.NewServer(opts)
	if err != nil {
		os.RemoveAll(tmpDir)
		fmt.Fprintf(os.Stderr, "❌ Failed to create NATS server: %v\n", err)
		os.Exit(1)
	}

	go internalServer.Start()
	if !internalServer.ReadyForConnections(5 * time.Second) {
		os.RemoveAll(tmpDir)
		fmt.Fprintf(os.Stderr, "❌ NATS server failed to start\n")
		os.Exit(1)
	}

	cleanup := func() {
		internalServer.Shutdown()
		internalServer.WaitForShutdown()
		os.RemoveAll(tmpDir)
	}

	return internalServer, cleanup
}

// printViewmodelHelp prints help for the viewmodel command.
func printViewmodelHelp() {
	fmt.Println("🌲 NimsForest Viewmodel - Cluster State Visualization")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  forest viewmodel <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  print          Print full territory view with all Land and processes")
	fmt.Println("  summary        Print capacity and usage summary")
	fmt.Println("  webview        Launch interactive isometric webview")
	fmt.Println("  help           Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  forest viewmodel print              # Show all land and processes")
	fmt.Println("  forest viewmodel summary            # Show capacity/usage summary")
	fmt.Println("  forest viewmodel webview            # Start webview on :8080")
	fmt.Println("  forest viewmodel webview --port=3000  # Use custom port")
	fmt.Println()
	fmt.Println("The viewmodel shows:")
	fmt.Println("  • Land - Nodes in the NATS cluster (regular or GPU-enabled)")
	fmt.Println("  • Trees - Data parsers watching the river")
	fmt.Println("  • Treehouses - Lua script processors")
	fmt.Println("  • Nims - Business logic handlers")
	fmt.Println()
	fmt.Println("Webview provides an isometric visualization in your browser where:")
	fmt.Println("  • Green tiles represent regular Land")
	fmt.Println("  • Purple tiles represent Manaland (GPU-enabled)")
	fmt.Println("  • Tree/Treehouse/Nim sprites stack on their Land")
	fmt.Println("  • Click entities to see details in the sidebar")
}
