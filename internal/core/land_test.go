package core

import "testing"

func TestLandType_String(t *testing.T) {
	tests := []struct {
		landType LandType
		expected string
	}{
		{LandTypeBase, "land"},
		{LandTypeNimland, "nimland"},
		{LandTypeManaland, "manaland"},
	}

	for _, tt := range tests {
		t.Run(string(tt.landType), func(t *testing.T) {
			result := tt.landType.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestLandInfo_CanRunAgents(t *testing.T) {
	tests := []struct {
		name     string
		info     *LandInfo
		expected bool
	}{
		{
			name:     "No Docker - cannot run agents",
			info:     &LandInfo{HasDocker: false},
			expected: false,
		},
		{
			name:     "Has Docker - can run agents",
			info:     &LandInfo{HasDocker: true},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.info.CanRunAgents()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestLandInfo_CanRunGPUWorkloads(t *testing.T) {
	tests := []struct {
		name     string
		info     *LandInfo
		expected bool
	}{
		{
			name: "No Docker, no GPU",
			info: &LandInfo{
				HasDocker: false,
				GPUVram:   0,
			},
			expected: false,
		},
		{
			name: "Docker but no GPU",
			info: &LandInfo{
				HasDocker: true,
				GPUVram:   0,
			},
			expected: false,
		},
		{
			name: "GPU but no Docker",
			info: &LandInfo{
				HasDocker: false,
				GPUVram:   8 * 1024 * 1024 * 1024,
			},
			expected: false,
		},
		{
			name: "Docker + GPU",
			info: &LandInfo{
				HasDocker: true,
				GPUVram:   8 * 1024 * 1024 * 1024,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.info.CanRunGPUWorkloads()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
