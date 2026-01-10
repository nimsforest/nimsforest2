package core

// LandType identifies the capabilities of this node.
type LandType string

const (
	// LandTypeBase is a basic Land with no Docker - backbone only
	LandTypeBase LandType = "land"

	// LandTypeNimland is a Land with Docker capabilities
	LandTypeNimland LandType = "nimland"

	// LandTypeManaland is a Land with Docker and GPU capabilities
	LandTypeManaland LandType = "manaland"
)

// LandInfo holds information about a compute node.
// This is detected at startup and stored in Forest.
type LandInfo struct {
	ID       string   `json:"id"`       // From NATS server ID
	Name     string   `json:"name"`     // From NATS server name (config)
	Type     LandType `json:"type"`     // Detected: land/nimland/manaland
	Hostname string   `json:"hostname"` // OS hostname

	// Capacity (detected from system)
	RAMTotal   uint64  `json:"ram_total"`    // Bytes
	CPUCores   int     `json:"cpu_cores"`    // Number of CPU cores
	CPUModel   string  `json:"cpu_model"`    // CPU model name
	CPUFreqMHz float64 `json:"cpu_freq_mhz"` // CPU frequency in MHz

	// Capabilities (probed)
	HasDocker bool `json:"has_docker"` // Docker available and running

	// GPU (if available)
	GPUVendor string  `json:"gpu_vendor,omitempty"` // "nvidia", "amd", etc.
	GPUModel  string  `json:"gpu_model,omitempty"`  // GPU model name
	GPUVram   uint64  `json:"gpu_vram,omitempty"`   // GPU VRAM in bytes
	GPUTflops float64 `json:"gpu_tflops,omitempty"` // GPU compute power
}

// String returns a human-readable representation of LandType
func (lt LandType) String() string {
	return string(lt)
}

// CanRunAgents returns true if this Land can run agent containers
func (li *LandInfo) CanRunAgents() bool {
	return li.HasDocker
}

// CanRunGPUWorkloads returns true if this Land has GPU capabilities
func (li *LandInfo) CanRunGPUWorkloads() bool {
	return li.GPUVram > 0 && li.HasDocker
}
