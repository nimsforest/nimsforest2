package viewmodel

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// World represents the collection of all LandViewModel in the cluster.
// It provides methods for querying and managing the cluster state.
type World struct {
	mu        sync.RWMutex
	lands     map[string]*LandViewModel // Map of LandViewModel ID to LandViewModel
	updatedAt time.Time                 // Last update time
}

// NewWorld creates a new empty World.
func NewWorld() *World {
	return &World{
		lands:     make(map[string]*LandViewModel),
		updatedAt: time.Now(),
	}
}

// AddLand adds a LandViewModel to the territory.
func (t *World) AddLand(land *LandViewModel) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lands[land.ID] = land
	t.updatedAt = time.Now()
}

// RemoveLand removes a LandViewModel from the territory.
func (t *World) RemoveLand(landID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, exists := t.lands[landID]; exists {
		delete(t.lands, landID)
		t.updatedAt = time.Now()
		return true
	}
	return false
}

// GetLand returns a LandViewModel by ID.
func (t *World) GetLand(landID string) *LandViewModel {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.lands[landID]
}

// Lands returns all LandViewModel in the territory, sorted by ID.
func (t *World) Lands() []*LandViewModel {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	lands := make([]*LandViewModel, 0, len(t.lands))
	for _, land := range t.lands {
		lands = append(lands, land)
	}
	
	sort.Slice(lands, func(i, j int) bool {
		return lands[i].ID < lands[j].ID
	})
	
	return lands
}

// LandCount returns the total number of LandViewModel.
func (t *World) LandCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.lands)
}

// ManalandCount returns the number of GPU-enabled LandViewModel (Manaland).
func (t *World) ManalandCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	count := 0
	for _, land := range t.lands {
		if land.IsManaland() {
			count++
		}
	}
	return count
}

// RegularLandCount returns the number of non-GPU LandViewModel.
func (t *World) RegularLandCount() int {
	return t.LandCount() - t.ManalandCount()
}

// TotalRAM returns the total RAM across all LandViewModel.
func (t *World) TotalRAM() uint64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	var total uint64
	for _, land := range t.lands {
		total += land.RAMTotal
	}
	return total
}

// TotalRAMAllocated returns the total allocated RAM across all LandViewModel.
func (t *World) TotalRAMAllocated() uint64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	var total uint64
	for _, land := range t.lands {
		total += land.RAMAllocated()
	}
	return total
}

// TotalCPUCores returns the total CPU cores across all LandViewModel.
func (t *World) TotalCPUCores() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	total := 0
	for _, land := range t.lands {
		total += land.CPUCores
	}
	return total
}

// TotalGPUVram returns the total GPU VRAM across all Manaland.
func (t *World) TotalGPUVram() uint64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	var total uint64
	for _, land := range t.lands {
		total += land.GPUVram
	}
	return total
}

// TreeCount returns the total number of Trees across all LandViewModel.
func (t *World) TreeCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	count := 0
	for _, land := range t.lands {
		count += len(land.Trees)
	}
	return count
}

// TreehouseCount returns the total number of Treehouses across all LandViewModel.
func (t *World) TreehouseCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	count := 0
	for _, land := range t.lands {
		count += len(land.Treehouses)
	}
	return count
}

// NimCount returns the total number of Nims across all LandViewModel.
func (t *World) NimCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	count := 0
	for _, land := range t.lands {
		count += len(land.Nims)
	}
	return count
}

// TotalProcessCount returns the total number of all processes.
func (t *World) TotalProcessCount() int {
	return t.TreeCount() + t.TreehouseCount() + t.NimCount()
}

// TreeRAM returns the total RAM used by Trees.
func (t *World) TreeRAM() uint64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	var total uint64
	for _, land := range t.lands {
		for _, tree := range land.Trees {
			total += tree.RAMAllocated
		}
	}
	return total
}

// TreehouseRAM returns the total RAM used by Treehouses.
func (t *World) TreehouseRAM() uint64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	var total uint64
	for _, land := range t.lands {
		for _, th := range land.Treehouses {
			total += th.RAMAllocated
		}
	}
	return total
}

// NimRAM returns the total RAM used by Nims.
func (t *World) NimRAM() uint64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	var total uint64
	for _, land := range t.lands {
		for _, nim := range land.Nims {
			total += nim.RAMAllocated
		}
	}
	return total
}

// Occupancy returns the overall RAM occupancy percentage.
func (t *World) Occupancy() float64 {
	totalRAM := t.TotalRAM()
	if totalRAM == 0 {
		return 0
	}
	return float64(t.TotalRAMAllocated()) / float64(totalRAM) * 100
}

// FindProcess finds a process by ID across all LandViewModel.
func (t *World) FindProcess(processID string) (*Process, *LandViewModel) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	for _, land := range t.lands {
		if proc := land.FindProcess(processID); proc != nil {
			return proc, land
		}
	}
	return nil, nil
}

// AllTrees returns all Trees across all LandViewModel.
func (t *World) AllTrees() []TreeViewModel {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	var trees []TreeViewModel
	for _, land := range t.lands {
		trees = append(trees, land.Trees...)
	}
	return trees
}

// AllTreehouses returns all Treehouses across all LandViewModel.
func (t *World) AllTreehouses() []TreehouseViewModel {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	var treehouses []TreehouseViewModel
	for _, land := range t.lands {
		treehouses = append(treehouses, land.Treehouses...)
	}
	return treehouses
}

// AllNims returns all Nims across all LandViewModel.
func (t *World) AllNims() []NimViewModel {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	var nims []NimViewModel
	for _, land := range t.lands {
		nims = append(nims, land.Nims...)
	}
	return nims
}

// UpdatedAt returns the last update time.
func (t *World) UpdatedAt() time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.updatedAt
}

// String returns a summary string of the territory.
func (t *World) String() string {
	return fmt.Sprintf("World: %d land (%d regular, %d manaland), %d processes",
		t.LandCount(), t.RegularLandCount(), t.ManalandCount(), t.TotalProcessCount())
}

// Summary holds summary statistics for the territory.
type Summary struct {
	// Capacity
	LandCount         int
	ManalandCount     int
	TotalRAM          uint64
	TotalCPUCores     int
	TotalGPUVram      uint64
	
	// Usage
	TreeCount         int
	TreehouseCount    int
	NimCount          int
	TreeRAM           uint64
	TreehouseRAM      uint64
	NimRAM            uint64
	TotalRAMAllocated uint64
	Occupancy         float64
}

// GetSummary returns a Summary of the territory.
func (t *World) GetSummary() Summary {
	return Summary{
		LandCount:         t.RegularLandCount(),
		ManalandCount:     t.ManalandCount(),
		TotalRAM:          t.TotalRAM(),
		TotalCPUCores:     t.TotalCPUCores(),
		TotalGPUVram:      t.TotalGPUVram(),
		TreeCount:         t.TreeCount(),
		TreehouseCount:    t.TreehouseCount(),
		NimCount:          t.NimCount(),
		TreeRAM:           t.TreeRAM(),
		TreehouseRAM:      t.TreehouseRAM(),
		NimRAM:            t.NimRAM(),
		TotalRAMAllocated: t.TotalRAMAllocated(),
		Occupancy:         t.Occupancy(),
	}
}
