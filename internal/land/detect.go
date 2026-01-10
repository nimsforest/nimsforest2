package land

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/yourusername/nimsforest/internal/core"
)

// Detect probes the local system and returns LandInfo.
// Called once during Forest startup.
func Detect(natsID, natsName string) *core.LandInfo {
	info := &core.LandInfo{
		ID:   natsID,
		Name: natsName,
	}

	// Hostname
	info.Hostname, _ = os.Hostname()

	// RAM
	if vmStat, err := mem.VirtualMemory(); err == nil {
		info.RAMTotal = vmStat.Total
	}

	// CPU
	info.CPUCores = runtime.NumCPU()
	if cpuInfo, err := cpu.Info(); err == nil && len(cpuInfo) > 0 {
		info.CPUModel = cpuInfo[0].ModelName
		info.CPUFreqMHz = cpuInfo[0].Mhz
	}

	// Docker
	info.HasDocker = detectDocker()

	// GPU
	detectGPU(info)

	// Determine type based on capabilities
	info.Type = determineType(info)

	return info
}

// detectDocker checks if Docker is available and running
func detectDocker() bool {
	// Check if docker command exists
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}

	// Check if docker daemon is running
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		return false
	}

	return true
}

// detectGPU attempts to detect GPU information
func detectGPU(info *core.LandInfo) {
	// Try NVIDIA first (most common for compute)
	if detectNvidiaGPU(info) {
		return
	}

	// Could add AMD ROCm detection here in the future
	// if detectAMDGPU(info) {
	//     return
	// }
}

// detectNvidiaGPU uses nvidia-smi to detect NVIDIA GPUs
func detectNvidiaGPU(info *core.LandInfo) bool {
	// Check if nvidia-smi exists
	if _, err := exec.LookPath("nvidia-smi"); err != nil {
		return false
	}

	cmd := exec.Command("nvidia-smi",
		"--query-gpu=name,memory.total",
		"--format=csv,noheader,nounits")

	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Parse: "NVIDIA GeForce RTX 4090, 24564"
	line := strings.TrimSpace(string(output))
	parts := strings.Split(line, ", ")
	if len(parts) < 2 {
		return false
	}

	info.GPUVendor = "nvidia"
	info.GPUModel = strings.TrimSpace(parts[0])

	// Parse VRAM (nvidia-smi reports in MiB)
	if vramMiB, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64); err == nil {
		info.GPUVram = vramMiB * 1024 * 1024 // Convert to bytes
	}

	return true
}

// determineType determines the LandType based on detected capabilities
func determineType(info *core.LandInfo) core.LandType {
	if info.GPUVram > 0 && info.HasDocker {
		return core.LandTypeManaland
	}
	if info.HasDocker {
		return core.LandTypeNimland
	}
	return core.LandTypeBase
}
