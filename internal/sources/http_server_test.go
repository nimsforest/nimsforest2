package sources

import (
	"testing"

	"github.com/yourusername/nimsforest/internal/core"
)

func TestWebhookServer_New(t *testing.T) {
	server := NewWebhookServer("")
	if server.Address() != "127.0.0.1:8081" {
		t.Errorf("Default address = %s, want 127.0.0.1:8081", server.Address())
	}

	server2 := NewWebhookServer("0.0.0.0:9000")
	if server2.Address() != "0.0.0.0:9000" {
		t.Errorf("Custom address = %s, want 0.0.0.0:9000", server2.Address())
	}
}

func TestWebhookServer_MountUnmount(t *testing.T) {
	server := NewWebhookServer("")

	// Create a webhook source
	cfg := WebhookSourceConfig{
		Name:      "test-webhook",
		Path:      "/webhooks/test",
		Publishes: "river.test.webhook",
	}
	ws := &WebhookSource{
		config: cfg,
	}
	ws.BaseSource = core.NewBaseSource(cfg.Name, nil, cfg.Publishes)

	// Mount
	if err := server.Mount(ws); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}

	// Verify it's mounted
	if server.Source("test-webhook") == nil {
		t.Error("Source should be mounted")
	}

	// Try to mount again (should fail)
	if err := server.Mount(ws); err == nil {
		t.Error("Mounting duplicate should fail")
	}

	// List sources
	sources := server.Sources()
	if len(sources) != 1 {
		t.Errorf("Sources count = %d, want 1", len(sources))
	}

	// Unmount
	if err := server.Unmount("test-webhook"); err != nil {
		t.Fatalf("Unmount failed: %v", err)
	}

	// Verify it's unmounted
	if server.Source("test-webhook") != nil {
		t.Error("Source should be unmounted")
	}

	// Try to unmount again (should fail)
	if err := server.Unmount("test-webhook"); err == nil {
		t.Error("Unmounting non-existent should fail")
	}
}

func TestWebhookServer_IsRunning(t *testing.T) {
	server := NewWebhookServer("")

	if server.IsRunning() {
		t.Error("Server should not be running initially")
	}
}
