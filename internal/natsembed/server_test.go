package natsembed

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	// Create temp directory for JetStream data
	tmpDir, err := os.MkdirTemp("", "natsembed-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		NodeName:    "test-node",
		ClusterName: "test-cluster",
		DataDir:     filepath.Join(tmpDir, "jetstream"),
		ClientPort:  0,  // Random port
		MonitorPort: -1, // Disable monitoring for tests
	}

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if s == nil {
		t.Fatal("Server should not be nil")
	}
}

func TestNewServerRequiresNodeName(t *testing.T) {
	cfg := Config{
		ClusterName: "test-cluster",
	}

	_, err := New(cfg)
	if err == nil {
		t.Error("Expected error when NodeName is missing")
	}
}

func TestNewServerRequiresClusterName(t *testing.T) {
	cfg := Config{
		NodeName: "test-node",
	}

	_, err := New(cfg)
	if err == nil {
		t.Error("Expected error when ClusterName is missing")
	}
}

func TestServerStartAndShutdown(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "natsembed-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		NodeName:    "test-node",
		ClusterName: "test-cluster",
		DataDir:     filepath.Join(tmpDir, "jetstream"),
		ClientPort:  0,
		MonitorPort: -1, // Disable monitoring for tests
	}

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	if err := s.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Verify server is running
	if !s.IsRunning() {
		t.Error("Server should be running")
	}

	// Verify we can get client URL
	url := s.ClientURL()
	if url == "" {
		t.Error("Client URL should not be empty")
	}

	// Shutdown
	s.Shutdown()

	// Give time for shutdown
	time.Sleep(100 * time.Millisecond)

	// Verify server is not running
	if s.IsRunning() {
		t.Error("Server should not be running after shutdown")
	}
}

func TestClientConnection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "natsembed-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		NodeName:    "test-node",
		ClusterName: "test-cluster",
		DataDir:     filepath.Join(tmpDir, "jetstream"),
		ClientPort:  0,
		MonitorPort: -1, // Disable monitoring for tests
	}

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer s.Shutdown()

	if err := s.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Get client connection
	nc, err := s.ClientConn()
	if err != nil {
		t.Fatalf("Failed to get client connection: %v", err)
	}
	defer nc.Close()

	// Verify connection is valid
	if !nc.IsConnected() {
		t.Error("Client should be connected")
	}
}

func TestJetStreamContext(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "natsembed-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		NodeName:    "test-node",
		ClusterName: "test-cluster",
		DataDir:     filepath.Join(tmpDir, "jetstream"),
		ClientPort:  0,
		MonitorPort: -1, // Disable monitoring for tests
	}

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer s.Shutdown()

	if err := s.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Get JetStream context
	js, err := s.JetStream()
	if err != nil {
		t.Fatalf("Failed to get JetStream context: %v", err)
	}

	if js == nil {
		t.Error("JetStream context should not be nil")
	}
}

func TestGetLocalIPv6(t *testing.T) {
	ip := GetLocalIPv6()
	// IP might be empty in some test environments (no IPv6), so just verify it doesn't panic
	t.Logf("Local IPv6: %s", ip)
}

func TestHTTPMonitoring(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "natsembed-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use a specific port for monitoring (use high port to avoid conflicts)
	cfg := Config{
		NodeName:    "test-node",
		ClusterName: "test-cluster",
		DataDir:     filepath.Join(tmpDir, "jetstream"),
		ClientPort:  0,
		MonitorPort: 18222, // Enable monitoring on this port
	}

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer s.Shutdown()

	if err := s.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Verify MonitorURL returns correct URL
	monitorURL := s.MonitorURL()
	expectedURL := "http://127.0.0.1:18222"
	if monitorURL != expectedURL {
		t.Errorf("Expected monitor URL %q, got %q", expectedURL, monitorURL)
	}

	// Verify monitoring endpoint is accessible
	resp, err := http.Get(monitorURL + "/varz")
	if err != nil {
		t.Fatalf("Failed to access monitoring endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestMonitorURLWhenDisabled(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "natsembed-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		NodeName:    "test-node",
		ClusterName: "test-cluster",
		DataDir:     filepath.Join(tmpDir, "jetstream"),
		ClientPort:  0,
		MonitorPort: -1, // Explicitly disabled
	}

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer s.Shutdown()

	if err := s.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Verify MonitorURL returns empty when disabled
	monitorURL := s.MonitorURL()
	if monitorURL != "" {
		t.Errorf("Expected empty monitor URL when disabled, got %q", monitorURL)
	}
}
