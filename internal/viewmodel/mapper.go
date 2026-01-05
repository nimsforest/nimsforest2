package viewmodel

import (
	"strings"
	"time"
)

// Mapper builds a TerritoryViewModel from a ClusterSnapshot.
type Mapper struct {
	// Default RAM allocation for processes when not specified
	DefaultProcessRAM uint64
}

// NewMapper creates a new Mapper with default settings.
func NewMapper() *Mapper {
	return &Mapper{
		DefaultProcessRAM: 256 * 1024 * 1024, // 256MB default
	}
}

// BuildTerritory builds a TerritoryViewModel from a ClusterSnapshot.
func (m *Mapper) BuildTerritory(snapshot *ClusterSnapshot) *TerritoryViewModel {
	territory := NewTerritoryViewModel()

	// Add local node as a LandViewModel
	localLand := m.nodeToLand(snapshot.LocalNode)
	territory.AddLand(localLand)

	// Add peer nodes as LandViewModel
	for _, peer := range snapshot.PeerNodes {
		peerLand := m.nodeToLand(peer)
		territory.AddLand(peerLand)
	}

	return territory
}

// nodeToLand converts a NodeInfo to a LandViewModel.
func (m *Mapper) nodeToLand(node NodeInfo) *LandViewModel {
	land := NewLandViewModel(node.ID)
	land.Hostname = node.Name
	land.RAMTotal = node.RAMTotal
	land.CPUCores = node.CPUCores
	land.GPUVram = node.GPUVram
	land.GPUTflops = node.GPUTflops
	land.JoinedAt = node.StartTime
	land.LastSeen = time.Now()
	land.ClusterURL = node.ClusterURL

	return land
}

// AttachProcesses attaches detected processes to the appropriate LandViewModel.
// This is called after building the initial territory from the cluster snapshot.
func (m *Mapper) AttachProcesses(territory *TerritoryViewModel, detectedProcesses []DetectedProcess) {
	for _, proc := range detectedProcesses {
		land := territory.GetLand(proc.LandID)
		if land == nil {
			// If we can't find the land, try to attach to the first available land
			lands := territory.Lands()
			if len(lands) > 0 {
				land = lands[0]
			} else {
				continue
			}
		}

		switch proc.Type {
		case ProcessTypeTree:
			tree := NewTreeViewModel(proc.ID, proc.Name, proc.RAMAllocated, proc.Subjects)
			land.AddTree(tree)
		case ProcessTypeTreehouse:
			th := NewTreehouseViewModel(proc.ID, proc.Name, proc.RAMAllocated, proc.ScriptPath)
			land.AddTreehouse(th)
		case ProcessTypeNim:
			nim := NewNimViewModel(proc.ID, proc.Name, proc.RAMAllocated, proc.Subjects, proc.AIEnabled)
			land.AddNim(nim)
		}
	}
}

// DetectedProcess represents a process detected from subscriptions.
type DetectedProcess struct {
	ID           string
	Name         string
	Type         ProcessType
	RAMAllocated uint64
	LandID       string
	Subjects     []string
	ScriptPath   string // For treehouses
	AIEnabled    bool   // For nims
}

// InferProcessType infers the process type from a subject pattern.
func InferProcessType(subject string) ProcessType {
	// Subject patterns typically follow conventions:
	// - Trees watch "river.>" patterns
	// - Treehouses process specific subjects
	// - Nims catch domain events like "payment.>", "lead.>"

	subject = strings.ToLower(subject)

	// Trees typically observe the river
	if strings.HasPrefix(subject, "river.") {
		return ProcessTypeTree
	}

	// Treehouses often have "treehouse" in the subject or watch specific events
	if strings.Contains(subject, "treehouse") ||
		strings.HasPrefix(subject, "contact.") ||
		strings.HasPrefix(subject, "lead.scored") {
		return ProcessTypeTreehouse
	}

	// Nims catch domain events
	if strings.HasPrefix(subject, "payment.") ||
		strings.HasPrefix(subject, "lead.") ||
		strings.HasPrefix(subject, "followup.") ||
		strings.HasPrefix(subject, "data.") ||
		strings.HasPrefix(subject, "status.") ||
		strings.HasPrefix(subject, "notification") {
		return ProcessTypeNim
	}

	// Default to nim for unknown patterns
	return ProcessTypeNim
}

// InferProcessName generates a name from a subject pattern.
func InferProcessName(subject string) string {
	// Clean up the subject to create a name
	name := strings.TrimPrefix(subject, "river.")
	name = strings.TrimSuffix(name, ".>")
	name = strings.TrimSuffix(name, ".*")
	name = strings.ReplaceAll(name, ".", "-")

	if name == "" {
		name = "unknown"
	}

	return name
}

// BuildTerritoryWithProcesses is a convenience method that builds a territory
// and attaches processes in one call.
func (m *Mapper) BuildTerritoryWithProcesses(snapshot *ClusterSnapshot, processes []DetectedProcess) *TerritoryViewModel {
	territory := m.BuildTerritory(snapshot)
	m.AttachProcesses(territory, processes)
	return territory
}
