//go:build viewer

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	smarttv "github.com/nimsforest/nimsforestsmarttv"
	viewer "github.com/nimsforest/nimsforestviewer"
	"github.com/yourusername/nimsforest/internal/core"
	"github.com/yourusername/nimsforest/internal/viewerdancer"
	"github.com/yourusername/nimsforest/internal/viewmodel"
	"github.com/yourusername/nimsforest/internal/windwaker"
	"github.com/yourusername/nimsforest/pkg/runtime"
)

// startViewer creates and starts the visualization viewer dancer.
func startViewer(ctx context.Context, ns *server.Server, cfg *runtime.ViewerConfig, wind *core.Wind) {
	// Create viewmodel
	vm := viewmodel.New(ns)
	if err := vm.Refresh(); err != nil {
		log.Printf("⚠️  Failed to refresh viewmodel: %v", err)
	}

	// Create viewer
	v := viewer.New()

	// Add web target if configured
	if cfg.WebAddr != "" {
		webTarget, err := viewer.NewWebTarget(cfg.WebAddr)
		if err != nil {
			log.Printf("⚠️  Failed to create web target: %v", err)
		} else {
			v.AddTarget(webTarget)
			fmt.Printf("   📡 Web API: http://localhost%s/api/viewmodel\n", cfg.WebAddr)
		}
	}

	// Discover and add Smart TV target if configured
	if cfg.SmartTV {
		fmt.Println("   📺 Discovering Smart TVs...")
		tvs, err := smarttv.Discover(ctx, 5*time.Second)
		if err != nil {
			log.Printf("⚠️  TV discovery error: %v", err)
		} else if len(tvs) == 0 {
			fmt.Println("   ⚠️  No Smart TVs found")
		} else {
			tv := &tvs[0]
			fmt.Printf("   📺 Found TV: %s\n", tv.String())

			tvTarget, err := viewer.NewSmartTVTarget(tv, viewer.WithJFIF(true))
			if err != nil {
				log.Printf("⚠️  Could not create TV target: %v", err)
			} else {
				v.AddTarget(tvTarget)
				fmt.Println("   📺 Smart TV target added!")
			}
		}
	}

	// Configure update interval
	updateInterval := uint64(90) // Default: once per second at 90Hz
	if cfg.UpdateInterval > 0 {
		updateInterval = uint64(cfg.UpdateInterval)
	}

	// Configure change detection
	onlyOnChange := true
	if cfg.OnlyOnChange != nil {
		onlyOnChange = *cfg.OnlyOnChange
	}

	// Create ViewerDancer
	dancer := viewerdancer.New(vm, v,
		viewerdancer.WithUpdateInterval(updateInterval),
		viewerdancer.WithOnlyOnChange(onlyOnChange),
	)

	// Register with windwaker - subscribe to dance beats
	wind.Catch("dance.beat", func(leaf core.Leaf) {
		var beat windwaker.Beat
		if err := json.Unmarshal(leaf.Data, &beat); err != nil {
			return
		}
		// Call Dance on each beat
		if err := dancer.Dance(beat); err != nil {
			log.Printf("⚠️  Viewer dance error: %v", err)
		}
	})

	// Initial update
	if err := dancer.ForceUpdate(); err != nil {
		log.Printf("⚠️  Initial viewer update failed: %v", err)
	}

	fmt.Println("   ✅ Viewer dancer registered with WindWaker")
}

// viewerEnabled returns true when built with viewer tag
func viewerEnabled() bool {
	return true
}
