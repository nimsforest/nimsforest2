package natsclusterconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFrom(t *testing.T) {
	// Create temp file with node info
	tmpDir, err := os.MkdirTemp("", "natsclusterconfig-test-*")
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
	tmpDir, err := os.MkdirTemp("", "natsclusterconfig-test-*")
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
	tmpDir, err := os.MkdirTemp("", "natsclusterconfig-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	registryPath := filepath.Join(tmpDir, "registry.json")
	registry := Registry{
		Nodes: map[string][]Node{
			"test-forest": {
				{ID: "node-1", IP: "2a01:4f8:1:1::1", IPv6: "2a01:4f8:1:1::1", IPv4: "192.168.1.1", ForestID: "test-forest"},
				{ID: "node-2", IP: "2a01:4f8:1:1::2", IPv6: "2a01:4f8:1:1::2", IPv4: "192.168.1.2", ForestID: "test-forest"},
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

	// Verify IPv4 and IPv6 fields were loaded
	if nodes[0].IPv6 != "2a01:4f8:1:1::1" {
		t.Errorf("Expected first node IPv6 '2a01:4f8:1:1::1', got '%s'", nodes[0].IPv6)
	}
	if nodes[0].IPv4 != "192.168.1.1" {
		t.Errorf("Expected first node IPv4 '192.168.1.1', got '%s'", nodes[0].IPv4)
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
	tmpDir, err := os.MkdirTemp("", "natsclusterconfig-test-*")
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
				{ID: "node-3", IP: "2a01:4f8:1:1::3", ForestID: "test-forest"},
			},
		},
	}

	data, _ := json.MarshalIndent(registry, "", "  ")
	os.WriteFile(registryPath, data, 0644)

	// Get peers excluding node-1 (with IPv6 connectivity)
	peers := GetPeersFrom(registryPath, "test-forest", "2a01:4f8:1:1::1", 6222, true)

	// Should get 2 peers (node-2 and node-3)
	if len(peers) != 2 {
		t.Errorf("Expected 2 peers, got %d", len(peers))
	}

	// Verify IPv6 format for node-2
	expected1 := "[2a01:4f8:1:1::2]:6222"
	found := false
	for _, peer := range peers {
		if peer == expected1 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected peer '%s' not found in %v", expected1, peers)
	}

	// Verify IPv6 format for node-3
	expected2 := "[2a01:4f8:1:1::3]:6222"
	found = false
	for _, peer := range peers {
		if peer == expected2 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected peer '%s' not found in %v", expected2, peers)
	}
}

func TestGetPeersFromWithIPv4IPv6(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "natsclusterconfig-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	registryPath := filepath.Join(tmpDir, "registry.json")
	registry := Registry{
		Nodes: map[string][]Node{
			"test-forest": {
				{ID: "node-1", IP: "2a01:4f8:1:1::1", IPv6: "2a01:4f8:1:1::1", IPv4: "192.168.1.1", ForestID: "test-forest"},
				{ID: "node-2", IP: "2a01:4f8:1:1::2", IPv6: "2a01:4f8:1:1::2", IPv4: "192.168.1.2", ForestID: "test-forest"},
				{ID: "node-3", IP: "2a01:4f8:1:1::3", IPv6: "2a01:4f8:1:1::3", IPv4: "192.168.1.3", ForestID: "test-forest"},
			},
		},
	}

	data, _ := json.MarshalIndent(registry, "", "  ")
	os.WriteFile(registryPath, data, 0644)

	// Test with IPv6 connectivity - should prefer IPv6
	peersIPv6 := GetPeersFrom(registryPath, "test-forest", "2a01:4f8:1:1::1", 6222, true)
	if len(peersIPv6) != 2 {
		t.Errorf("Expected 2 peers with IPv6, got %d", len(peersIPv6))
	}
	// Verify IPv6 addresses are used
	for _, peer := range peersIPv6 {
		if peer != "[2a01:4f8:1:1::2]:6222" && peer != "[2a01:4f8:1:1::3]:6222" {
			t.Errorf("Unexpected peer address with IPv6 connectivity: %s", peer)
		}
	}

	// Test with IPv4 only - should use IPv4
	peersIPv4 := GetPeersFrom(registryPath, "test-forest", "192.168.1.1", 6222, false)
	if len(peersIPv4) != 2 {
		t.Errorf("Expected 2 peers with IPv4, got %d", len(peersIPv4))
	}
	// Verify IPv4 addresses are used
	for _, peer := range peersIPv4 {
		if peer != "192.168.1.2:6222" && peer != "192.168.1.3:6222" {
			t.Errorf("Unexpected peer address with IPv4 connectivity: %s", peer)
		}
	}
}

func TestNodeGetPreferredIP(t *testing.T) {
	tests := []struct {
		name                string
		node                Node
		hasIPv6Connectivity bool
		expected            string
	}{
		{
			name:                "IPv6 preferred when available and has connectivity",
			node:                Node{IP: "2a01:4f8:1:1::1", IPv6: "2a01:4f8:1:1::1", IPv4: "192.168.1.1"},
			hasIPv6Connectivity: true,
			expected:            "2a01:4f8:1:1::1",
		},
		{
			name:                "IPv4 used when no IPv6 connectivity",
			node:                Node{IP: "2a01:4f8:1:1::1", IPv6: "2a01:4f8:1:1::1", IPv4: "192.168.1.1"},
			hasIPv6Connectivity: false,
			expected:            "192.168.1.1",
		},
		{
			name:                "IPv4 used when no IPv6 address available",
			node:                Node{IP: "192.168.1.1", IPv4: "192.168.1.1"},
			hasIPv6Connectivity: true,
			expected:            "192.168.1.1",
		},
		{
			name:                "Legacy IP used when no IPv4 or IPv6 set",
			node:                Node{IP: "10.0.0.1"},
			hasIPv6Connectivity: false,
			expected:            "10.0.0.1",
		},
		{
			name:                "Legacy IPv6 IP used when no separate fields",
			node:                Node{IP: "2a01:4f8:1:1::1"},
			hasIPv6Connectivity: true,
			expected:            "2a01:4f8:1:1::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.node.GetPreferredIP(tt.hasIPv6Connectivity)
			if result != tt.expected {
				t.Errorf("GetPreferredIP(%v) = %s, want %s", tt.hasIPv6Connectivity, result, tt.expected)
			}
		})
	}
}

func TestRegisterAndUnregisterNode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "natsclusterconfig-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	registryPath := filepath.Join(tmpDir, "registry.json")

	// Register first node with IPv4 and IPv6
	node1 := Node{
		ID:       "node-1",
		IP:       "2a01:4f8:1:1::1",
		IPv6:     "2a01:4f8:1:1::1",
		IPv4:     "192.168.1.1",
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

	// Verify IPv4 and IPv6 were saved
	savedNode := reg.Nodes["test-forest"][0]
	if savedNode.IPv6 != "2a01:4f8:1:1::1" {
		t.Errorf("Expected IPv6 '2a01:4f8:1:1::1', got '%s'", savedNode.IPv6)
	}
	if savedNode.IPv4 != "192.168.1.1" {
		t.Errorf("Expected IPv4 '192.168.1.1', got '%s'", savedNode.IPv4)
	}

	// Register second node
	node2 := Node{
		ID:       "node-2",
		IP:       "2a01:4f8:1:1::2",
		IPv6:     "2a01:4f8:1:1::2",
		IPv4:     "192.168.1.2",
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
		IPv6:     "2a01:4f8:1:1::10",
		IPv4:     "192.168.1.10",
		ForestID: "test-forest",
	}

	if err := RegisterNodeTo(registryPath, node1Updated); err != nil {
		t.Fatalf("Failed to update node: %v", err)
	}

	reg, _ = LoadRegistryFrom(registryPath)
	if len(reg.Nodes["test-forest"]) != 2 {
		t.Errorf("Expected still 2 nodes after update, got %d", len(reg.Nodes["test-forest"]))
	}

	// Verify IPs were updated
	for _, n := range reg.Nodes["test-forest"] {
		if n.ID == "node-1" {
			if n.IP != "2a01:4f8:1:1::10" {
				t.Errorf("Expected node-1 IP to be updated, got '%s'", n.IP)
			}
			if n.IPv4 != "192.168.1.10" {
				t.Errorf("Expected node-1 IPv4 to be updated, got '%s'", n.IPv4)
			}
			if n.IPv6 != "2a01:4f8:1:1::10" {
				t.Errorf("Expected node-1 IPv6 to be updated, got '%s'", n.IPv6)
			}
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

func TestMustLoadFrom(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "natsclusterconfig-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	nodeInfoPath := filepath.Join(tmpDir, "node-info.json")
	nodeInfo := NodeInfo{
		ForestID: "test-forest",
		NodeID:   "test-node-1",
	}

	data, _ := json.MarshalIndent(nodeInfo, "", "  ")
	os.WriteFile(nodeInfoPath, data, 0644)

	// Should succeed with valid config
	loaded := MustLoadFrom(nodeInfoPath)
	if loaded.ForestID != "test-forest" {
		t.Errorf("Expected ForestID 'test-forest', got '%s'", loaded.ForestID)
	}
	if loaded.NodeID != "test-node-1" {
		t.Errorf("Expected NodeID 'test-node-1', got '%s'", loaded.NodeID)
	}
}
