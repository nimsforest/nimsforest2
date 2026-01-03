package natsembed

import (
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
		ClientPort:  0, // Random port
		MonitorPort: 0, // Disable monitoring
	}

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if s == nil {
		t.Fatal("Server should not be nil")
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
		MonitorPort: 0,
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
		MonitorPort: 0,
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
		MonitorPort: 0,
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

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.NodeName != "standalone" {
		t.Errorf("Expected NodeName 'standalone', got '%s'", cfg.NodeName)
	}
	if cfg.ClusterName != "nimsforest" {
		t.Errorf("Expected ClusterName 'nimsforest', got '%s'", cfg.ClusterName)
	}
	if cfg.DataDir != "/var/lib/nimsforest/jetstream" {
		t.Errorf("Expected DataDir '/var/lib/nimsforest/jetstream', got '%s'", cfg.DataDir)
	}
	if cfg.ClientPort != 4222 {
		t.Errorf("Expected ClientPort 4222, got %d", cfg.ClientPort)
	}
	if cfg.ClusterPort != 6222 {
		t.Errorf("Expected ClusterPort 6222, got %d", cfg.ClusterPort)
	}
	if cfg.MonitorPort != 8222 {
		t.Errorf("Expected MonitorPort 8222, got %d", cfg.MonitorPort)
	}
}

func TestGetLocalIP(t *testing.T) {
	ip := GetLocalIP()
	// IP might be empty in some test environments, so just verify it doesn't panic
	t.Logf("Local IP: %s", ip)
}
