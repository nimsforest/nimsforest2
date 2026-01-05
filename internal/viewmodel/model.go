// Package viewmodel provides a view model for the NimsForest cluster state.
// It allows querying and displaying the current state of Land (nodes), Trees,
// Treehouses, and Nims deployed across the cluster.
//
// The ViewModel structs are named with a "ViewModel" suffix (e.g., LandViewModel,
// TreeViewModel) to avoid confusion with the core domain types in internal/core.
package viewmodel

import (
	"fmt"
	"time"
)

// ProcessType identifies the type of process running on a LandViewModel.
type ProcessType string

const (
	ProcessTypeTree      ProcessType = "tree"
	ProcessTypeTreehouse ProcessType = "treehouse"
	ProcessTypeNim       ProcessType = "nim"
)

// Process represents a running process (Tree, Treehouse, or Nim) on a LandViewModel.
type Process struct {
	ID           string      `json:"id"`            // Unique identifier
	Name         string      `json:"name"`          // Display name
	Type         ProcessType `json:"type"`          // tree, treehouse, or nim
	RAMAllocated uint64      `json:"ram_allocated"` // RAM in bytes
	LandID       string      `json:"land_id"`       // Which LandViewModel this process runs on
	Subjects     []string    `json:"subjects"`      // Subscribed subjects (for detection)
	StartedAt    time.Time   `json:"started_at"`    // When the process started
}

// TreeViewModel represents a tree process (parses river data into leaves).
type TreeViewModel struct {
	Process
}

// TreehouseViewModel represents a treehouse process (Lua script processor).
type TreehouseViewModel struct {
	Process
	ScriptPath string `json:"script_path"` // Path to the Lua script
}

// NimViewModel represents a nim process (business logic handler).
type NimViewModel struct {
	Process
	AIEnabled bool   `json:"ai_enabled"` // Whether AI-powered
	Model     string `json:"model"`      // AI model if enabled
}

// LandViewModel represents a node in the cluster.
// LandViewModel can have regular CPU resources or mana resources (Manaland).
type LandViewModel struct {
	ID         string  `json:"id"`           // Node identifier (from NATS server name)
	Hostname   string  `json:"hostname"`     // Node hostname
	RAMTotal   uint64  `json:"ram_total"`    // Total RAM in bytes
	CPUCores   int     `json:"cpu_cores"`    // Number of CPU cores
	CPUFreqGHz float64 `json:"cpu_freq_ghz"` // CPU frequency in GHz
	GPUVram    uint64  `json:"gpu_vram"`     // GPU VRAM in bytes (0 if no GPU)
	GPUTflops  float64 `json:"gpu_tflops"`   // GPU compute power in TFLOPS
	
	// Processes running on this LandViewModel
	Trees      []TreeViewModel      `json:"trees"`
	Treehouses []TreehouseViewModel `json:"treehouses"`
	Nims       []NimViewModel       `json:"nims"`
	
	// Metadata
	JoinedAt   time.Time `json:"joined_at"`   // When this node joined the cluster
	LastSeen   time.Time `json:"last_seen"`   // Last heartbeat/activity
	ClusterURL string    `json:"cluster_url"` // Cluster route URL
}

// HasMana returns true if this LandViewModel has mana (GPU) resources.
func (l *LandViewModel) HasMana() bool {
	return l.GPUVram > 0
}

// IsManaland returns true if this is a mana-enabled LandViewModel (Manaland).
func (l *LandViewModel) IsManaland() bool {
	return l.HasMana()
}

// RAMAllocated returns the total RAM allocated to all processes on this LandViewModel.
func (l *LandViewModel) RAMAllocated() uint64 {
	var total uint64
	for _, t := range l.Trees {
		total += t.RAMAllocated
	}
	for _, th := range l.Treehouses {
		total += th.RAMAllocated
	}
	for _, n := range l.Nims {
		total += n.RAMAllocated
	}
	return total
}

// RAMAvailable returns the available RAM on this LandViewModel.
func (l *LandViewModel) RAMAvailable() uint64 {
	allocated := l.RAMAllocated()
	if allocated >= l.RAMTotal {
		return 0
	}
	return l.RAMTotal - allocated
}

// Occupancy returns the RAM usage as a percentage (0-100).
func (l *LandViewModel) Occupancy() float64 {
	if l.RAMTotal == 0 {
		return 0
	}
	return float64(l.RAMAllocated()) / float64(l.RAMTotal) * 100
}

// ProcessCount returns the total number of processes running on this LandViewModel.
func (l *LandViewModel) ProcessCount() int {
	return len(l.Trees) + len(l.Treehouses) + len(l.Nims)
}

// AddTree adds a tree to this LandViewModel.
func (l *LandViewModel) AddTree(tree TreeViewModel) {
	tree.LandID = l.ID
	l.Trees = append(l.Trees, tree)
}

// AddTreehouse adds a treehouse to this LandViewModel.
func (l *LandViewModel) AddTreehouse(th TreehouseViewModel) {
	th.LandID = l.ID
	l.Treehouses = append(l.Treehouses, th)
}

// AddNim adds a nim to this LandViewModel.
func (l *LandViewModel) AddNim(nim NimViewModel) {
	nim.LandID = l.ID
	l.Nims = append(l.Nims, nim)
}

// RemoveProcess removes a process by ID from this LandViewModel.
func (l *LandViewModel) RemoveProcess(processID string) bool {
	// Try to remove from trees
	for i, t := range l.Trees {
		if t.ID == processID {
			l.Trees = append(l.Trees[:i], l.Trees[i+1:]...)
			return true
		}
	}
	// Try to remove from treehouses
	for i, th := range l.Treehouses {
		if th.ID == processID {
			l.Treehouses = append(l.Treehouses[:i], l.Treehouses[i+1:]...)
			return true
		}
	}
	// Try to remove from nims
	for i, n := range l.Nims {
		if n.ID == processID {
			l.Nims = append(l.Nims[:i], l.Nims[i+1:]...)
			return true
		}
	}
	return false
}

// FindProcess finds a process by ID on this LandViewModel.
func (l *LandViewModel) FindProcess(processID string) *Process {
	for i := range l.Trees {
		if l.Trees[i].ID == processID {
			return &l.Trees[i].Process
		}
	}
	for i := range l.Treehouses {
		if l.Treehouses[i].ID == processID {
			return &l.Treehouses[i].Process
		}
	}
	for i := range l.Nims {
		if l.Nims[i].ID == processID {
			return &l.Nims[i].Process
		}
	}
	return nil
}

// String returns a human-readable summary of the LandViewModel.
func (l *LandViewModel) String() string {
	landType := "Land"
	if l.IsManaland() {
		landType = "Manaland"
	}
	return fmt.Sprintf("%s: %s (ram: %s, cpu: %d, occupancy: %.0f%%)",
		landType, l.ID, FormatBytes(l.RAMTotal), l.CPUCores, l.Occupancy())
}

// FormatBytes formats bytes into a human-readable string.
func FormatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	
	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.1fTB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.0fGB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.0fMB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.0fKB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// NewLandViewModel creates a new LandViewModel with the given ID.
func NewLandViewModel(id string) *LandViewModel {
	return &LandViewModel{
		ID:         id,
		Trees:      make([]TreeViewModel, 0),
		Treehouses: make([]TreehouseViewModel, 0),
		Nims:       make([]NimViewModel, 0),
		JoinedAt:   time.Now(),
		LastSeen:   time.Now(),
	}
}

// NewProcess creates a new Process with the given parameters.
func NewProcess(id, name string, processType ProcessType, ramAllocated uint64) Process {
	return Process{
		ID:           id,
		Name:         name,
		Type:         processType,
		RAMAllocated: ramAllocated,
		StartedAt:    time.Now(),
	}
}

// NewTreeViewModel creates a new TreeViewModel process.
func NewTreeViewModel(id, name string, ramAllocated uint64, subjects []string) TreeViewModel {
	return TreeViewModel{
		Process: Process{
			ID:           id,
			Name:         name,
			Type:         ProcessTypeTree,
			RAMAllocated: ramAllocated,
			Subjects:     subjects,
			StartedAt:    time.Now(),
		},
	}
}

// NewTreehouseViewModel creates a new TreehouseViewModel process.
func NewTreehouseViewModel(id, name string, ramAllocated uint64, scriptPath string) TreehouseViewModel {
	return TreehouseViewModel{
		Process: Process{
			ID:           id,
			Name:         name,
			Type:         ProcessTypeTreehouse,
			RAMAllocated: ramAllocated,
			StartedAt:    time.Now(),
		},
		ScriptPath: scriptPath,
	}
}

// NewNimViewModel creates a new NimViewModel process.
func NewNimViewModel(id, name string, ramAllocated uint64, subjects []string, aiEnabled bool) NimViewModel {
	return NimViewModel{
		Process: Process{
			ID:           id,
			Name:         name,
			Type:         ProcessTypeNim,
			RAMAllocated: ramAllocated,
			Subjects:     subjects,
			StartedAt:    time.Now(),
		},
		AIEnabled: aiEnabled,
	}
}
