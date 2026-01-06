package webview

import (
	"math"

	"github.com/yourusername/nimsforest/internal/viewmodel"
)

// WorldJSON is the JSON representation of World with grid positions assigned.
type WorldJSON struct {
	Lands   []LandJSON  `json:"lands"`
	Summary SummaryJSON `json:"summary"`
}

// LandJSON is the JSON representation of a Land tile.
type LandJSON struct {
	ID           string        `json:"id"`
	Hostname     string        `json:"hostname"`
	RAMTotal     uint64        `json:"ram_total"`
	RAMAllocated uint64        `json:"ram_allocated"`
	CPUCores     int           `json:"cpu_cores"`
	CPUFreqGHz   float64       `json:"cpu_freq_ghz"`
	GPUVram      uint64        `json:"gpu_vram,omitempty"`
	GPUTflops    float64       `json:"gpu_tflops,omitempty"`
	Occupancy    float64       `json:"occupancy"`
	IsManaland   bool          `json:"is_manaland"`
	GridX        int           `json:"grid_x"`
	GridY        int           `json:"grid_y"`
	Trees        []ProcessJSON `json:"trees"`
	Treehouses   []ProcessJSON `json:"treehouses"`
	Nims         []ProcessJSON `json:"nims"`
}

// ProcessJSON is the JSON representation of a process.
type ProcessJSON struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	RAMAllocated uint64   `json:"ram_allocated"`
	Type         string   `json:"type"`
	Subjects     []string `json:"subjects,omitempty"`
	ScriptPath   string   `json:"script_path,omitempty"`
	AIEnabled    bool     `json:"ai_enabled,omitempty"`
	Model        string   `json:"model,omitempty"`
}

// SummaryJSON is the JSON representation of the world summary.
type SummaryJSON struct {
	LandCount      int     `json:"land_count"`
	ManalandCount  int     `json:"manaland_count"`
	TreeCount      int     `json:"tree_count"`
	TreehouseCount int     `json:"treehouse_count"`
	NimCount       int     `json:"nim_count"`
	TotalRAM       uint64  `json:"total_ram"`
	RAMAllocated   uint64  `json:"ram_allocated"`
	Occupancy      float64 `json:"occupancy"`
}

// WorldToJSON converts a World to WorldJSON, assigning grid positions.
func WorldToJSON(w *viewmodel.World) WorldJSON {
	lands := w.Lands()

	// Calculate grid size for square-ish layout
	gridSize := int(math.Ceil(math.Sqrt(float64(len(lands)))))
	if gridSize < 1 {
		gridSize = 1
	}

	landsJSON := make([]LandJSON, len(lands))
	for i, land := range lands {
		landsJSON[i] = landToJSON(land, i%gridSize, i/gridSize)
	}

	summary := w.GetSummary()

	return WorldJSON{
		Lands: landsJSON,
		Summary: SummaryJSON{
			LandCount:      summary.LandCount + summary.ManalandCount,
			ManalandCount:  summary.ManalandCount,
			TreeCount:      summary.TreeCount,
			TreehouseCount: summary.TreehouseCount,
			NimCount:       summary.NimCount,
			TotalRAM:       summary.TotalRAM,
			RAMAllocated:   summary.TotalRAMAllocated,
			Occupancy:      summary.Occupancy,
		},
	}
}

// landToJSON converts a LandViewModel to LandJSON.
func landToJSON(land *viewmodel.LandViewModel, gridX, gridY int) LandJSON {
	trees := make([]ProcessJSON, len(land.Trees))
	for i, t := range land.Trees {
		trees[i] = ProcessJSON{
			ID:           t.ID,
			Name:         t.Name,
			RAMAllocated: t.RAMAllocated,
			Type:         "tree",
			Subjects:     t.Subjects,
		}
	}

	treehouses := make([]ProcessJSON, len(land.Treehouses))
	for i, th := range land.Treehouses {
		treehouses[i] = ProcessJSON{
			ID:           th.ID,
			Name:         th.Name,
			RAMAllocated: th.RAMAllocated,
			Type:         "treehouse",
			ScriptPath:   th.ScriptPath,
		}
	}

	nims := make([]ProcessJSON, len(land.Nims))
	for i, n := range land.Nims {
		nims[i] = ProcessJSON{
			ID:           n.ID,
			Name:         n.Name,
			RAMAllocated: n.RAMAllocated,
			Type:         "nim",
			Subjects:     n.Subjects,
			AIEnabled:    n.AIEnabled,
			Model:        n.Model,
		}
	}

	return LandJSON{
		ID:           land.ID,
		Hostname:     land.Hostname,
		RAMTotal:     land.RAMTotal,
		RAMAllocated: land.RAMAllocated(),
		CPUCores:     land.CPUCores,
		CPUFreqGHz:   land.CPUFreqGHz,
		GPUVram:      land.GPUVram,
		GPUTflops:    land.GPUTflops,
		Occupancy:    land.Occupancy(),
		IsManaland:   land.IsManaland(),
		GridX:        gridX,
		GridY:        gridY,
		Trees:        trees,
		Treehouses:   treehouses,
		Nims:         nims,
	}
}
