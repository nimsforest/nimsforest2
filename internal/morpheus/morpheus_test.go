package morpheus

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFrom(t *testing.T) {
	// Create temp file with node info
	tmpDir, err := os.MkdirTemp("", "morpheus-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	nodeInfoPath := filepath.Join(tmpDir, "node-info.json")
	nodeInfo := NodeInfo{
		ForestID: "test-forest",
		NodeID:   "test-node-1",
	}

	data, err := json.MarshalIndent(nodeInfo, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal node info: %v", err)
	}

	if err := os.WriteFile(nodeInfoPath, data, 0644); err != nil {
		t.Fatalf("Failed to write node info file: %v", err)
	}

	// Load and verify
	loaded := LoadFrom(nodeInfoPath)
	if loaded == nil {
		t.Fatal("LoadFrom returned nil")
	}

	if loaded.ForestID != nodeInfo.ForestID {
		t.Errorf("Expected ForestID '%s', got '%s'", nodeInfo.ForestID, loaded.ForestID)
	}
	if loaded.NodeID != nodeInfo.NodeID {
		t.Errorf("Expected NodeID '%s', got '%s'", nodeInfo.NodeID, loaded.NodeID)
	}
}

func TestLoadFromMissingFile(t *testing.T) {
	info := LoadFrom("/nonexistent/path/node-info.json")
	if info != nil {
		t.Error("Expected nil for missing file")
	}
}

func TestLoadFromInvalidJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "morpheus-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	nodeInfoPath := filepath.Join(tmpDir, "node-info.json")
	if err := os.WriteFile(nodeInfoPath, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	info := LoadFrom(nodeInfoPath)
	if info != nil {
		t.Error("Expected nil for invalid JSON")
	}
}

func TestLoadRegistryFrom(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "morpheus-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	registryPath := filepath.Join(tmpDir, "registry.json")
	registry := Registry{
		Nodes: map[string][]Node{
			"test-forest": {
				{ID: "node-1", IP: "2a01:4f8:1:1::1", ForestID: "test-forest"},
				{ID: "node-2", IP: "2a01:4f8:1:1::2", ForestID: "test-forest"},
			},
		},
	}

	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal registry: %v", err)
	}

	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		t.Fatalf("Failed to write registry file: %v", err)
	}

	// Load and verify
	loaded, err := LoadRegistryFrom(registryPath)
	if err != nil {
		t.Fatalf("LoadRegistryFrom failed: %v", err)
	}

	nodes := loaded.Nodes["test-forest"]
	if len(nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(nodes))
	}

	if nodes[0].ID != "node-1" {
		t.Errorf("Expected first node ID 'node-1', got '%s'", nodes[0].ID)
	}
}

func TestLoadRegistryFromMissingFile(t *testing.T) {
	reg, err := LoadRegistryFrom("/nonexistent/path/registry.json")
	if err != nil {
		t.Fatalf("Expected no error for missing file, got: %v", err)
	}
	if reg == nil {
		t.Fatal("Expected empty registry, got nil")
	}
	if len(reg.Nodes) != 0 {
		t.Error("Expected empty nodes map")
	}
}

func TestGetPeersFrom(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "morpheus-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	registryPath := filepath.Join(tmpDir, "registry.json")
	registry := Registry{
		Nodes: map[string][]Node{
			"test-forest": {
				{ID: "node-1", IP: "2a01:4f8:1:1::1", ForestID: "test-forest"},
				{ID: "node-2", IP: "2a01:4f8:1:1::2", ForestID: "test-forest"},
				{ID: "node-3", IP: "192.168.1.100", ForestID: "test-forest"},
			},
		},
	}

	data, _ := json.MarshalIndent(registry, "", "  ")
	os.WriteFile(registryPath, data, 0644)

	// Get peers excluding node-1
	peers := GetPeersFrom(registryPath, "test-forest", "2a01:4f8:1:1::1", 6222)

	// Should get 2 peers (node-2 and node-3)
	if len(peers) != 2 {
		t.Errorf("Expected 2 peers, got %d", len(peers))
	}

	// Verify IPv6 format
	expectedIPv6 := "[2a01:4f8:1:1::2]:6222"
	found := false
	for _, peer := range peers {
		if peer == expectedIPv6 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected peer '%s' not found in %v", expectedIPv6, peers)
	}

	// Verify IPv4 format
	expectedIPv4 := "192.168.1.100:6222"
	found = false
	for _, peer := range peers {
		if peer == expectedIPv4 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected peer '%s' not found in %v", expectedIPv4, peers)
	}
}

func TestRegisterAndUnregisterNode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "morpheus-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	registryPath := filepath.Join(tmpDir, "registry.json")

	// Register first node
	node1 := Node{
		ID:       "node-1",
		IP:       "2a01:4f8:1:1::1",
		ForestID: "test-forest",
	}

	if err := RegisterNodeTo(registryPath, node1); err != nil {
		t.Fatalf("Failed to register node: %v", err)
	}

	// Verify registration
	reg, _ := LoadRegistryFrom(registryPath)
	if len(reg.Nodes["test-forest"]) != 1 {
		t.Errorf("Expected 1 node, got %d", len(reg.Nodes["test-forest"]))
	}

	// Register second node
	node2 := Node{
		ID:       "node-2",
		IP:       "2a01:4f8:1:1::2",
		ForestID: "test-forest",
	}

	if err := RegisterNodeTo(registryPath, node2); err != nil {
		t.Fatalf("Failed to register second node: %v", err)
	}

	reg, _ = LoadRegistryFrom(registryPath)
	if len(reg.Nodes["test-forest"]) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(reg.Nodes["test-forest"]))
	}

	// Update existing node
	node1Updated := Node{
		ID:       "node-1",
		IP:       "2a01:4f8:1:1::10", // Changed IP
		ForestID: "test-forest",
	}

	if err := RegisterNodeTo(registryPath, node1Updated); err != nil {
		t.Fatalf("Failed to update node: %v", err)
	}

	reg, _ = LoadRegistryFrom(registryPath)
	if len(reg.Nodes["test-forest"]) != 2 {
		t.Errorf("Expected still 2 nodes after update, got %d", len(reg.Nodes["test-forest"]))
	}

	// Verify IP was updated
	for _, n := range reg.Nodes["test-forest"] {
		if n.ID == "node-1" && n.IP != "2a01:4f8:1:1::10" {
			t.Errorf("Expected node-1 IP to be updated, got '%s'", n.IP)
		}
	}

	// Unregister node
	if err := UnregisterNodeFrom(registryPath, "test-forest", "node-1"); err != nil {
		t.Fatalf("Failed to unregister node: %v", err)
	}

	reg, _ = LoadRegistryFrom(registryPath)
	if len(reg.Nodes["test-forest"]) != 1 {
		t.Errorf("Expected 1 node after unregister, got %d", len(reg.Nodes["test-forest"]))
	}
}

func TestIsIPv6(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"2a01:4f8:1:1::1", true},
		{"::1", true},
		{"fe80::1", true},
		{"192.168.1.1", false},
		{"10.0.0.1", false},
		{"127.0.0.1", false},
	}

	for _, tt := range tests {
		result := isIPv6(tt.ip)
		if result != tt.expected {
			t.Errorf("isIPv6(%s) = %v, expected %v", tt.ip, result, tt.expected)
		}
	}
}

func TestGetForestID(t *testing.T) {
	// Without config, should return "standalone"
	id := GetForestID()
	if id != "standalone" {
		t.Errorf("Expected 'standalone' without config, got '%s'", id)
	}
}
