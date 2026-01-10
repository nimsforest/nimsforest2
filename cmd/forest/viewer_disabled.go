//go:build !viewer

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/yourusername/nimsforest/internal/core"
	"github.com/yourusername/nimsforest/pkg/runtime"
)

// startViewer is a no-op when built without viewer tag
func startViewer(ctx context.Context, ns *server.Server, cfg *runtime.ViewerConfig, wind *core.Wind) {
	// Viewer disabled at build time
}

// viewerEnabled returns false when built without viewer tag
func viewerEnabled() bool {
	return false
}

// handleViewmodel prints a message that viewer is not enabled
func handleViewmodel(args []string) {
	fmt.Println("❌ Viewer functionality not available.")
	fmt.Println("   Rebuild with: go build -tags viewer ./...")
	os.Exit(1)
}
