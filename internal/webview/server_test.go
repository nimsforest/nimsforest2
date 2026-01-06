package webview

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/yourusername/nimsforest/internal/viewmodel"
)

func TestHandleViewmodel(t *testing.T) {
	// Create a minimal NATS server for testing
	opts := &server.Options{
		ServerName: "test-server",
		Host:       "127.0.0.1",
		Port:       -1, // Random port
		JetStream:  true,
		StoreDir:   t.TempDir(),
	}

	ns, err := server.NewServer(opts)
	if err != nil {
		t.Fatalf("failed to create NATS server: %v", err)
	}

	go ns.Start()
	if !ns.ReadyForConnections(5 * 1e9) { // 5 seconds
		t.Fatal("NATS server failed to start")
	}
	defer ns.Shutdown()

	// Create viewmodel and server
	vm := viewmodel.New(ns)
	srv := New(vm, nil)

	// Test the /api/viewmodel endpoint
	req := httptest.NewRequest(http.MethodGet, "/api/viewmodel", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Check that response is valid JSON
	var result WorldJSON
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Errorf("failed to parse response as JSON: %v", err)
	}

	// Should have summary
	if result.Summary.LandCount < 0 {
		t.Errorf("unexpected land count: %d", result.Summary.LandCount)
	}
}

func TestHandleViewmodelCORS(t *testing.T) {
	opts := &server.Options{
		ServerName: "test-server",
		Host:       "127.0.0.1",
		Port:       -1,
		JetStream:  true,
		StoreDir:   t.TempDir(),
	}

	ns, err := server.NewServer(opts)
	if err != nil {
		t.Fatalf("failed to create NATS server: %v", err)
	}

	go ns.Start()
	if !ns.ReadyForConnections(5 * 1e9) {
		t.Fatal("NATS server failed to start")
	}
	defer ns.Shutdown()

	vm := viewmodel.New(ns)
	srv := New(vm, nil)

	// Test OPTIONS request for CORS
	req := httptest.NewRequest(http.MethodOptions, "/api/viewmodel", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for OPTIONS, got %d", w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected CORS header to be set")
	}
}

func TestHandleFallback(t *testing.T) {
	opts := &server.Options{
		ServerName: "test-server",
		Host:       "127.0.0.1",
		Port:       -1,
		JetStream:  true,
		StoreDir:   t.TempDir(),
	}

	ns, err := server.NewServer(opts)
	if err != nil {
		t.Fatalf("failed to create NATS server: %v", err)
	}

	go ns.Start()
	if !ns.ReadyForConnections(5 * 1e9) {
		t.Fatal("NATS server failed to start")
	}
	defer ns.Shutdown()

	vm := viewmodel.New(ns)
	srv := New(vm, nil) // No webDir, should serve fallback

	// Test the root endpoint
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Should be HTML content
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html" {
		t.Errorf("expected Content-Type text/html, got %s", contentType)
	}

	// Should contain NimsForest
	if !contains(w.Body.String(), "NimsForest") {
		t.Error("expected fallback page to contain 'NimsForest'")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
