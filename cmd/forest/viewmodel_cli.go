// Package main provides the forest CLI.
// Note: The viewer (with ebiten/graphics) runs as a separate process.
// This file handles viewmodel-related CLI commands without graphics deps.
package main

import (
	"fmt"
	"os"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/yourusername/nimsforest/internal/viewmodel"
)

// handleViewmodel handles the viewmodel CLI commands.
// The viewer itself runs as a separate process - this just prints state.
func handleViewmodel(args []string) {
	if len(args) == 0 {
		args = []string{"summary"}
	}

	subcommand := args[0]

	// For CLI commands, we need to connect to a running NATS server
	// This is just for printing state - the actual viewer is separate
	switch subcommand {
	case "print", "show":
		fmt.Println("📊 Viewmodel Print")
		fmt.Println("==================")
		fmt.Println()
		fmt.Println("To view cluster state, connect to a running NimsForest instance:")
		fmt.Println()
		fmt.Println("  # Subscribe to state updates")
		fmt.Println("  nats sub forest.viewmodel.state")
		fmt.Println()
		fmt.Println("  # Subscribe to events")
		fmt.Println("  nats sub forest.viewmodel.events")
		fmt.Println()
		fmt.Println("Or run the standalone viewer:")
		fmt.Println("  nimsforestviewer --nats-url nats://localhost:4222")
		fmt.Println()

	case "summary":
		fmt.Println("📊 Viewmodel Summary")
		fmt.Println("====================")
		fmt.Println()
		fmt.Println("The viewmodel is published to NATS subjects:")
		fmt.Println("  • forest.viewmodel.state  - Full state snapshot")
		fmt.Println("  • forest.viewmodel.events - Real-time events")
		fmt.Println()
		fmt.Println("External viewers subscribe to these subjects.")
		fmt.Println("This decouples visualization from the core.")
		fmt.Println()

	case "viewer":
		fmt.Println("📺 External Viewer")
		fmt.Println("==================")
		fmt.Println()
		fmt.Println("The viewer runs as a separate process.")
		fmt.Println("Install and run it with:")
		fmt.Println()
		fmt.Println("  go install github.com/nimsforest/nimsforestviewer@latest")
		fmt.Println("  nimsforestviewer --nats-url nats://localhost:4222")
		fmt.Println()
		fmt.Println("The viewer subscribes to forest.viewmodel.state")
		fmt.Println("and renders the cluster visualization.")
		fmt.Println()

	case "help", "--help", "-h":
		printViewmodelHelp()

	default:
		fmt.Fprintf(os.Stderr, "Unknown viewmodel subcommand: %s\n\n", subcommand)
		printViewmodelHelp()
		os.Exit(1)
	}
}

func printViewmodelHelp() {
	fmt.Println("Usage: forest viewmodel [subcommand]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  summary  Show viewmodel architecture summary (default)")
	fmt.Println("  print    Show how to view cluster state")
	fmt.Println("  viewer   Show how to run the external viewer")
	fmt.Println("  help     Show this help message")
	fmt.Println()
	fmt.Println("The viewmodel publishes cluster state to NATS:")
	fmt.Println("  forest.viewmodel.state  - Full state JSON")
	fmt.Println("  forest.viewmodel.events - Real-time change events")
	fmt.Println()
}

// printLocalViewmodel prints viewmodel state from a local NATS server.
// Used for quick CLI inspection without the full viewer.
func printLocalViewmodel(ns *server.Server) {
	vm := viewmodel.New(ns)
	if err := vm.Refresh(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read viewmodel: %v\n", err)
		os.Exit(1)
	}
	vm.Print(os.Stdout)
}
