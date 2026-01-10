package land

import (
	"testing"

	"github.com/yourusername/nimsforest/internal/core"
)

func TestDetect(t *testing.T) {
	// Test basic detection - should not panic
	land := Detect("test-id", "test-node")

	if land == nil {
		t.Fatal("Detect returned nil")
	}

	// Verify ID and Name are set
	if land.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got '%s'", land.ID)
	}

	if land.Name != "test-node" {
		t.Errorf("Expected Name 'test-node', got '%s'", land.Name)
	}

	// Verify basic system info is detected
	if land.CPUCores <= 0 {
		t.Error("Expected positive CPU cores")
	}

	if land.RAMTotal == 0 {
		t.Error("Expected non-zero RAM")
	}

	// Verify Land type is valid
	validTypes := map[core.LandType]bool{
		core.LandTypeBase:     true,
		core.LandTypeNimland:  true,
		core.LandTypeManaland: true,
	}

	if !validTypes[land.Type] {
		t.Errorf("Invalid Land type: %s", land.Type)
	}

	t.Logf("Detected Land: %s", land.Type)
	t.Logf("  RAM: %d bytes", land.RAMTotal)
	t.Logf("  CPU: %d cores", land.CPUCores)
	t.Logf("  Docker: %v", land.HasDocker)
	if land.GPUVram > 0 {
		t.Logf("  GPU: %s (%d bytes VRAM)", land.GPUModel, land.GPUVram)
	}
}

func TestDetermineType(t *testing.T) {
	tests := []struct {
		name     string
		info     *core.LandInfo
		expected core.LandType
	}{
		{
			name: "Base Land - no Docker, no GPU",
			info: &core.LandInfo{
				HasDocker: false,
				GPUVram:   0,
			},
			expected: core.LandTypeBase,
		},
		{
			name: "Nimland - Docker, no GPU",
			info: &core.LandInfo{
				HasDocker: true,
				GPUVram:   0,
			},
			expected: core.LandTypeNimland,
		},
		{
			name: "Manaland - Docker + GPU",
			info: &core.LandInfo{
				HasDocker: true,
				GPUVram:   8 * 1024 * 1024 * 1024, // 8GB
			},
			expected: core.LandTypeManaland,
		},
		{
			name: "Base Land - GPU without Docker",
			info: &core.LandInfo{
				HasDocker: false,
				GPUVram:   8 * 1024 * 1024 * 1024,
			},
			expected: core.LandTypeBase,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineType(tt.info)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestDetectDocker(t *testing.T) {
	// Just verify it doesn't panic
	hasDocker := detectDocker()
	t.Logf("Docker detected: %v", hasDocker)
}

func TestDetectGPU(t *testing.T) {
	// Just verify it doesn't panic
	info := &core.LandInfo{}
	detectGPU(info)
	t.Logf("GPU detected: %v (VRAM: %d)", info.GPUVendor, info.GPUVram)
}
