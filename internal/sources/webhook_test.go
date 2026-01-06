package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yourusername/nimsforest/internal/core"
)

func TestWebhookSource_Handler(t *testing.T) {
	// Create a mock river using a real River would require NATS
	// For unit testing, we'll test the handler directly

	cfg := WebhookSourceConfig{
		Name:      "test-webhook",
		Path:      "/webhooks/test",
		Publishes: "river.test.webhook",
		Headers:   []string{"X-Test-Header"},
	}

	// We can't easily mock core.River, so let's just test the HTTP handler aspects
	t.Run("rejects non-POST", func(t *testing.T) {
		ws := &WebhookSource{
			config:  cfg,
			running: true,
		}
		ws.BaseSource = core.NewBaseSource(cfg.Name, nil, cfg.Publishes)
		ws.SetRunning(true)

		handler := ws.Handler()
		req := httptest.NewRequest(http.MethodGet, "/webhooks/test", nil)
		rec := httptest.NewRecorder()

		handler(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
		}
	})

	t.Run("returns 503 when not running", func(t *testing.T) {
		ws := &WebhookSource{
			config:  cfg,
			running: false,
		}
		ws.BaseSource = core.NewBaseSource(cfg.Name, nil, cfg.Publishes)

		handler := ws.Handler()
		req := httptest.NewRequest(http.MethodPost, "/webhooks/test", bytes.NewReader([]byte(`{"test": true}`)))
		rec := httptest.NewRecorder()

		handler(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
		}
	})
}

func TestWebhookSource_StartStop(t *testing.T) {
	cfg := WebhookSourceConfig{
		Name:      "test-webhook",
		Path:      "/webhooks/test",
		Publishes: "river.test.webhook",
	}

	ws := &WebhookSource{
		config: cfg,
	}
	ws.BaseSource = core.NewBaseSource(cfg.Name, nil, cfg.Publishes)

	// Start
	if err := ws.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !ws.IsRunning() {
		t.Error("Expected source to be running after Start")
	}

	// Stop
	if err := ws.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if ws.IsRunning() {
		t.Error("Expected source to be stopped after Stop")
	}
}

func TestWebhookPayload_JSON(t *testing.T) {
	payload := WebhookPayload{
		Headers: map[string]string{
			"X-Test": "value",
		},
		Body:      []byte(`{"event": "test"}`),
		Timestamp: time.Now(),
		Source:    "test-source",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	var decoded WebhookPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if decoded.Source != payload.Source {
		t.Errorf("Source mismatch: got %s, want %s", decoded.Source, payload.Source)
	}
}
