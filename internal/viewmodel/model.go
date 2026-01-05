// Package viewmodel provides a view model for the NimsForest cluster state.
// It allows querying and displaying the current state of Land (nodes), Trees,
// Treehouses, and Nims deployed across the cluster.
package viewmodel

import (
	"fmt"
	"time"
)

// ProcessType identifies the type of process running on a Land.
type ProcessType string

const (
	ProcessTypeTree      ProcessType = "tree"
	ProcessTypeTreehouse ProcessType = "treehouse"
	ProcessTypeNim       ProcessType = "nim"
)

// Process represents a running process (Tree, Treehouse, or Nim) on a Land.
type Process struct {
	ID           string      `json:"id"`            // Unique identifier
	Name         string      `json:"name"`          // Display name
	Type         ProcessType `json:"type"`          // tree, treehouse, or nim
	RAMAllocated uint64      `json:"ram_allocated"` // RAM in bytes
	LandID       string      `json:"land_id"`       // Which Land this process runs on
	Subjects     []string    `json:"subjects"`      // Subscribed subjects (for detection)
	StartedAt    time.Time   `json:"started_at"`    // When the process started
}

// Tree represents a tree process (parses river data into leaves).
type Tree struct {
	Process
}

// Treehouse represents a treehouse process (Lua script processor).
type Treehouse struct {
	Process
	ScriptPath string `json:"script_path"` // Path to the Lua script
}

// Nim represents a nim process (business logic handler).
type Nim struct {
	Process
	AIEnabled bool   `json:"ai_enabled"` // Whether AI-powered
	Model     string `json:"model"`      // AI model if enabled
}

// Land represents a node in the cluster.
// Land can have regular CPU resources or GPU resources (Manaland).
type Land struct {
	ID         string `json:"id"`          // Node identifier (from NATS server name)
	Hostname   string `json:"hostname"`    // Node hostname
	RAMTotal   uint64 `json:"ram_total"`   // Total RAM in bytes
	CPUCores   int    `json:"cpu_cores"`   // Number of CPU cores
	GPUVram    uint64 `json:"gpu_vram"`    // GPU VRAM in bytes (0 if no GPU)
	GPUTflops  float64 `json:"gpu_tflops"` // GPU compute power in TFLOPS
	
	// Processes running on this Land
	Trees      []Tree      `json:"trees"`
	Treehouses []Treehouse `json:"treehouses"`
	Nims       []Nim       `json:"nims"`
	
	// Metadata
	JoinedAt   time.Time `json:"joined_at"`   // When this node joined the cluster
	LastSeen   time.Time `json:"last_seen"`   // Last heartbeat/activity
	ClusterURL string    `json:"cluster_url"` // Cluster route URL
}

// HasGPU returns true if this Land has GPU resources.
func (l *Land) HasGPU() bool {
	return l.GPUVram > 0
}

// IsManaland returns true if this is a GPU-enabled Land (Manaland).
func (l *Land) IsManaland() bool {
	return l.HasGPU()
}

// RAMAllocated returns the total RAM allocated to all processes on this Land.
func (l *Land) RAMAllocated() uint64 {
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

// RAMAvailable returns the available RAM on this Land.
func (l *Land) RAMAvailable() uint64 {
	allocated := l.RAMAllocated()
	if allocated >= l.RAMTotal {
		return 0
	}
	return l.RAMTotal - allocated
}

// Occupancy returns the RAM usage as a percentage (0-100).
func (l *Land) Occupancy() float64 {
	if l.RAMTotal == 0 {
		return 0
	}
	return float64(l.RAMAllocated()) / float64(l.RAMTotal) * 100
}

// ProcessCount returns the total number of processes running on this Land.
func (l *Land) ProcessCount() int {
	return len(l.Trees) + len(l.Treehouses) + len(l.Nims)
}

// AddTree adds a tree to this Land.
func (l *Land) AddTree(tree Tree) {
	tree.LandID = l.ID
	l.Trees = append(l.Trees, tree)
}

// AddTreehouse adds a treehouse to this Land.
func (l *Land) AddTreehouse(th Treehouse) {
	th.LandID = l.ID
	l.Treehouses = append(l.Treehouses, th)
}

// AddNim adds a nim to this Land.
func (l *Land) AddNim(nim Nim) {
	nim.LandID = l.ID
	l.Nims = append(l.Nims, nim)
}

// RemoveProcess removes a process by ID from this Land.
func (l *Land) RemoveProcess(processID string) bool {
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

// FindProcess finds a process by ID on this Land.
func (l *Land) FindProcess(processID string) *Process {
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

// String returns a human-readable summary of the Land.
func (l *Land) String() string {
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

// NewLand creates a new Land with the given ID.
func NewLand(id string) *Land {
	return &Land{
		ID:         id,
		Trees:      make([]Tree, 0),
		Treehouses: make([]Treehouse, 0),
		Nims:       make([]Nim, 0),
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

// NewTree creates a new Tree process.
func NewTree(id, name string, ramAllocated uint64, subjects []string) Tree {
	return Tree{
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

// NewTreehouse creates a new Treehouse process.
func NewTreehouse(id, name string, ramAllocated uint64, scriptPath string) Treehouse {
	return Treehouse{
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

// NewNim creates a new Nim process.
func NewNim(id, name string, ramAllocated uint64, subjects []string, aiEnabled bool) Nim {
	return Nim{
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
