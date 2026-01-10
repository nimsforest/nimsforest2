package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	smarttv "github.com/nimsforest/nimsforestsmarttv"
	viewer "github.com/nimsforest/nimsforestviewer"
	"github.com/yourusername/nimsforest/internal/natsembed"
	"github.com/yourusername/nimsforest/internal/viewerdancer"
	"github.com/yourusername/nimsforest/internal/viewmodel"
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
	case "viewer":
		handleViewmodelViewer(args[1:])
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

// handleViewmodelViewer starts the viewer dancer to push updates to Smart TVs and/or web.
func handleViewmodelViewer(args []string) {
	// Parse flags
	webPort := ""
	discoverTV := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--web", "-w":
			if i+1 < len(args) {
				webPort = args[i+1]
				i++
			} else {
				webPort = ":8080"
			}
		case "--tv", "-t":
			discoverTV = true
		case "--help", "-h":
			printViewerHelp()
			return
		}
	}

	if webPort == "" && !discoverTV {
		fmt.Println("No targets specified. Use --web and/or --tv")
		fmt.Println()
		printViewerHelp()
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
	}()

	fmt.Println("🌲 NimsForest Viewer")
	fmt.Println()

	// Start NATS server
	ns, cleanup := getOrStartNATSServer()
	defer cleanup()

	// Create viewmodel
	vm := viewmodel.New(ns)
	if err := vm.Refresh(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to refresh viewmodel: %v\n", err)
		os.Exit(1)
	}

	// Create viewer
	v := viewer.New()

	// Add web target if requested
	if webPort != "" {
		webTarget, err := viewer.NewWebTarget(webPort)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to create web target: %v\n", err)
			os.Exit(1)
		}
		v.AddTarget(webTarget)
		fmt.Printf("📡 Web API: http://localhost%s/api/viewmodel\n", webPort)
	}

	// Discover and add Smart TV target if requested
	if discoverTV {
		fmt.Println("📺 Discovering Smart TVs...")
		tvs, err := smarttv.Discover(ctx, 5*time.Second)
		if err != nil {
			fmt.Printf("⚠️  TV discovery error: %v\n", err)
		} else if len(tvs) == 0 {
			fmt.Println("⚠️  No Smart TVs found")
		} else {
			tv := &tvs[0]
			fmt.Printf("📺 Found TV: %s\n", tv.String())

			tvTarget, err := viewer.NewSmartTVTarget(tv, viewer.WithJFIF(true))
			if err != nil {
				fmt.Printf("⚠️  Could not create TV target: %v\n", err)
			} else {
				v.AddTarget(tvTarget)
				fmt.Println("📺 Smart TV target added!")
			}
		}
	}

	// Create ViewerDancer
	dancer := viewerdancer.New(vm, v,
		viewerdancer.WithUpdateInterval(90), // Once per second at 90Hz
		viewerdancer.WithOnlyOnChange(true),
	)

	// Initial update
	if err := dancer.ForceUpdate(); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Initial update failed: %v\n", err)
	}

	fmt.Println()
	fmt.Println("Viewer running. Press Ctrl+C to stop.")
	fmt.Println()

	// Note: In a full implementation, we would register with windwaker via CatchBeat
	// For now, we'll use a simple ticker to simulate dance beats
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	beat := uint64(0)
	for {
		select {
		case <-ctx.Done():
			v.Close()
			return
		case <-ticker.C:
			beat++
			// Simulate dance beat - in real integration this would come from windwaker
			if err := vm.Refresh(); err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  Refresh error: %v\n", err)
				continue
			}
			world := vm.GetWorld()
			state := viewerdancer.ConvertToViewState(world)
			v.SetStateProvider(viewer.NewStaticStateProvider(state))
			if err := v.Update(); err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  Update error: %v\n", err)
			}
		}
	}
}

// printViewerHelp prints help for the viewer subcommand.
func printViewerHelp() {
	fmt.Println("Usage: forest viewmodel viewer [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --web, -w [PORT]   Start web API server (default :8080)")
	fmt.Println("  --tv, -t           Discover and connect to Smart TV")
	fmt.Println("  --help, -h         Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  forest viewmodel viewer --web :8080       # Web API only")
	fmt.Println("  forest viewmodel viewer --tv              # Smart TV only")
	fmt.Println("  forest viewmodel viewer --web :8080 --tv  # Both targets")
}

// handleViewmodelWebview informs user about external viewer packages.
func handleViewmodelWebview(args []string) {
	fmt.Println("🌲 NimsForest Web Viewer")
	fmt.Println()
	fmt.Println("The web viewer has been moved to separate packages:")
	fmt.Println()
	fmt.Println("  nimsforestviewer    - Go library for Smart TV + web JSON API")
	fmt.Println("                        github.com/nimsforest/nimsforestviewer")
	fmt.Println()
	fmt.Println("  nimsforestwebview   - Interactive Phaser 3 web frontend")
	fmt.Println("                        github.com/nimsforest/nimsforestwebview")
	fmt.Println()
	fmt.Println("To use:")
	fmt.Println("  1. Use nimsforestviewer to serve the JSON API")
	fmt.Println("  2. Build nimsforestwebview with 'npm run build'")
	fmt.Println("  3. Point nimsforestwebview at the API endpoint")
	fmt.Println()
	fmt.Println("For Smart TV display, see nimsforestviewer examples.")
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
	fmt.Println("  viewer         Start live viewer (web API and/or Smart TV)")
	fmt.Println("  webview        Info about external web viewer packages")
	fmt.Println("  help           Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  forest viewmodel print              # Show all land and processes")
	fmt.Println("  forest viewmodel summary            # Show capacity/usage summary")
	fmt.Println("  forest viewmodel viewer --web :8080 # Start web API server")
	fmt.Println("  forest viewmodel viewer --tv        # Display on Smart TV")
	fmt.Println()
	fmt.Println("The viewmodel shows:")
	fmt.Println("  • Land - Nodes in the NATS cluster (regular or GPU-enabled)")
	fmt.Println("  • Trees - Data parsers watching the river")
	fmt.Println("  • Treehouses - Lua script processors")
	fmt.Println("  • Nims - Business logic handlers")
	fmt.Println()
	fmt.Println("For web/TV visualization, see:")
	fmt.Println("  • nimsforestviewer  - Smart TV + JSON API (Go)")
	fmt.Println("  • nimsforestwebview - Interactive web UI (Phaser 3)")
}
