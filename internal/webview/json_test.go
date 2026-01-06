package webview

import (
	"testing"

	"github.com/yourusername/nimsforest/internal/viewmodel"
)

func TestWorldToJSON(t *testing.T) {
	// Create a test world
	world := viewmodel.NewWorld()

	// Add some test lands
	land1 := viewmodel.NewLandViewModel("land-1")
	land1.Hostname = "node-1.local"
	land1.RAMTotal = 16 * 1024 * 1024 * 1024 // 16GB
	land1.CPUCores = 8
	land1.CPUFreqGHz = 3.5

	// Add a tree
	tree := viewmodel.NewTreeViewModel("tree-1", "webhook-parser", 256*1024*1024, []string{"river.webhook"})
	land1.AddTree(tree)

	// Add a treehouse
	th := viewmodel.NewTreehouseViewModel("th-1", "enricher", 128*1024*1024, "./scripts/enricher.lua")
	land1.AddTreehouse(th)

	world.AddLand(land1)

	// Add a manaland
	land2 := viewmodel.NewLandViewModel("land-2")
	land2.Hostname = "gpu-node.local"
	land2.RAMTotal = 64 * 1024 * 1024 * 1024 // 64GB
	land2.CPUCores = 16
	land2.GPUVram = 24 * 1024 * 1024 * 1024 // 24GB
	land2.GPUTflops = 40.0

	// Add a nim
	nim := viewmodel.NewNimViewModel("nim-1", "qualifier", 512*1024*1024, []string{"lead.scored"}, true)
	nim.Model = "gpt-4"
	land2.AddNim(nim)

	world.AddLand(land2)

	// Convert to JSON
	result := WorldToJSON(world)

	// Verify summary
	if result.Summary.LandCount != 2 {
		t.Errorf("expected 2 lands, got %d", result.Summary.LandCount)
	}
	if result.Summary.ManalandCount != 1 {
		t.Errorf("expected 1 manaland, got %d", result.Summary.ManalandCount)
	}
	if result.Summary.TreeCount != 1 {
		t.Errorf("expected 1 tree, got %d", result.Summary.TreeCount)
	}
	if result.Summary.TreehouseCount != 1 {
		t.Errorf("expected 1 treehouse, got %d", result.Summary.TreehouseCount)
	}
	if result.Summary.NimCount != 1 {
		t.Errorf("expected 1 nim, got %d", result.Summary.NimCount)
	}

	// Verify lands have grid positions
	if len(result.Lands) != 2 {
		t.Fatalf("expected 2 lands in result, got %d", len(result.Lands))
	}

	// Grid positions should be assigned
	for _, land := range result.Lands {
		if land.GridX < 0 || land.GridY < 0 {
			t.Errorf("land %s has invalid grid position (%d, %d)", land.ID, land.GridX, land.GridY)
		}
	}
}

func TestLandToJSON(t *testing.T) {
	land := viewmodel.NewLandViewModel("test-land")
	land.Hostname = "test.local"
	land.RAMTotal = 8 * 1024 * 1024 * 1024
	land.CPUCores = 4
	land.CPUFreqGHz = 2.5
	land.GPUVram = 8 * 1024 * 1024 * 1024
	land.GPUTflops = 10.5

	// Add processes
	tree := viewmodel.NewTreeViewModel("tree-1", "parser", 100*1024*1024, []string{"test.subject"})
	land.AddTree(tree)

	result := landToJSON(land, 2, 3)

	if result.ID != "test-land" {
		t.Errorf("expected ID 'test-land', got %s", result.ID)
	}
	if result.GridX != 2 {
		t.Errorf("expected GridX 2, got %d", result.GridX)
	}
	if result.GridY != 3 {
		t.Errorf("expected GridY 3, got %d", result.GridY)
	}
	if !result.IsManaland {
		t.Error("expected IsManaland to be true")
	}
	if len(result.Trees) != 1 {
		t.Errorf("expected 1 tree, got %d", len(result.Trees))
	}
	if result.Trees[0].Type != "tree" {
		t.Errorf("expected tree type 'tree', got %s", result.Trees[0].Type)
	}
}

func TestEmptyWorld(t *testing.T) {
	world := viewmodel.NewWorld()
	result := WorldToJSON(world)

	if len(result.Lands) != 0 {
		t.Errorf("expected 0 lands, got %d", len(result.Lands))
	}
	if result.Summary.LandCount != 0 {
		t.Errorf("expected 0 land count, got %d", result.Summary.LandCount)
	}
}
