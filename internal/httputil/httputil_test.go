package httputil

import (
	"runtime"
	"testing"
	"time"
)

func TestCreateHTTPClient(t *testing.T) {
	client := CreateHTTPClient(30 * time.Second)

	if client == nil {
		t.Fatal("CreateHTTPClient returned nil")
	}

	if client.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", client.Timeout)
	}
}

func TestGetPlatform(t *testing.T) {
	platform := GetPlatform()

	expected := runtime.GOOS + "/" + runtime.GOARCH
	if platform != expected {
		t.Errorf("Expected platform %s, got %s", expected, platform)
	}
}

func TestGetBinaryName(t *testing.T) {
	tests := []struct {
		baseName string
		expected string
	}{
		{"forest", "forest-" + runtime.GOOS + "-" + runtime.GOARCH},
		{"myapp", "myapp-" + runtime.GOOS + "-" + runtime.GOARCH},
	}

	for _, tt := range tests {
		result := GetBinaryName(tt.baseName)
		if result != tt.expected {
			t.Errorf("GetBinaryName(%q) = %q, expected %q", tt.baseName, result, tt.expected)
		}
	}
}

func TestIsRestrictedEnvironment(t *testing.T) {
	// Just verify it doesn't panic
	_ = IsRestrictedEnvironment()
}
