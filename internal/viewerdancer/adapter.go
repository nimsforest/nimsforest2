// Package viewerdancer provides integration between nimsforest2's viewmodel
// and nimsforestviewer for visualization on Smart TVs and web browsers.
package viewerdancer

import (
	"math"

	viewer "github.com/nimsforest/nimsforestviewer"
	"github.com/yourusername/nimsforest/internal/viewmodel"
)

// ConvertToViewState converts a viewmodel.World to nimsforestviewer.ViewState.
func ConvertToViewState(world *viewmodel.World) *viewer.ViewState {
	if world == nil {
		return &viewer.ViewState{}
	}

	lands := world.Lands()

	// Calculate grid size for square-ish layout
	gridSize := int(math.Ceil(math.Sqrt(float64(len(lands)))))
	if gridSize < 1 {
		gridSize = 1
	}

	landViews := make([]viewer.LandView, len(lands))
	for i, land := range lands {
		landViews[i] = convertLand(land, i%gridSize, i/gridSize)
	}

	summary := world.GetSummary()

	return &viewer.ViewState{
		Lands: landViews,
		Summary: viewer.SummaryView{
			TotalLands:      summary.LandCount + summary.ManalandCount,
			TotalManalands:  summary.ManalandCount,
			TotalTrees:      summary.TreeCount,
			TotalTreehouses: summary.TreehouseCount,
			TotalNims:       summary.NimCount,
			TotalRAM:        summary.TotalRAM,
			AllocatedRAM:    summary.TotalRAMAllocated,
		},
	}
}

func convertLand(land *viewmodel.LandViewModel, gridX, gridY int) viewer.LandView {
	trees := make([]viewer.ProcessView, len(land.Trees))
	for i, t := range land.Trees {
		trees[i] = viewer.ProcessView{
			ID:           t.ID,
			Name:         t.Name,
			Type:         "tree",
			RAMAllocated: t.RAMAllocated,
		}
	}

	treehouses := make([]viewer.ProcessView, len(land.Treehouses))
	for i, th := range land.Treehouses {
		treehouses[i] = viewer.ProcessView{
			ID:           th.ID,
			Name:         th.Name,
			Type:         "treehouse",
			RAMAllocated: th.RAMAllocated,
		}
	}

	nims := make([]viewer.ProcessView, len(land.Nims))
	for i, n := range land.Nims {
		nims[i] = viewer.ProcessView{
			ID:           n.ID,
			Name:         n.Name,
			Type:         "nim",
			RAMAllocated: n.RAMAllocated,
		}
	}

	return viewer.LandView{
		ID:           land.ID,
		Hostname:     land.Hostname,
		GridX:        gridX,
		GridY:        gridY,
		IsManaland:   land.IsManaland(),
		Occupancy:    land.Occupancy() / 100, // Convert from percentage to 0-1
		RAMTotal:     land.RAMTotal,
		RAMAllocated: land.RAMAllocated(),
		Trees:        trees,
		Treehouses:   treehouses,
		Nims:         nims,
	}
}
