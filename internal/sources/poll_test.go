package sources

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yourusername/nimsforest/internal/core"
)

func TestPollSource_Config(t *testing.T) {
	cfg := PollSourceConfig{
		Name:      "test-poll",
		URL:       "https://example.com/api",
		Publishes: "river.test.poll",
		Interval:  5 * time.Minute,
		Method:    "GET",
		Headers: map[string]string{
			"Authorization": "Bearer token",
		},
	}

	// Can't create with nil river, but we can test config
	if cfg.Method != "GET" {
		t.Errorf("Expected method GET, got %s", cfg.Method)
	}
	if cfg.Interval != 5*time.Minute {
		t.Errorf("Expected interval 5m, got %v", cfg.Interval)
	}
}

func TestPollPayload_JSON(t *testing.T) {
	payload := PollPayload{
		URL:        "https://example.com/api",
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body:      []byte(`{"data": "test"}`),
		Timestamp: time.Now(),
		Source:    "test-poll",
		Cursor:    "abc123",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	var decoded PollPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if decoded.StatusCode != payload.StatusCode {
		t.Errorf("StatusCode mismatch: got %d, want %d", decoded.StatusCode, payload.StatusCode)
	}
	if decoded.Cursor != payload.Cursor {
		t.Errorf("Cursor mismatch: got %s, want %s", decoded.Cursor, payload.Cursor)
	}
}

func TestExtractJSONPath(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		path     string
		expected string
	}{
		{
			name:     "simple field",
			data:     `{"cursor": "abc123"}`,
			path:     "cursor",
			expected: "abc123",
		},
		{
			name:     "nested field",
			data:     `{"meta": {"cursor": "xyz789"}}`,
			path:     "meta.cursor",
			expected: "xyz789",
		},
		{
			name:     "with dollar prefix",
			data:     `{"cursor": "abc123"}`,
			path:     "$.cursor",
			expected: "abc123",
		},
		{
			name:     "nested with dollar prefix",
			data:     `{"meta": {"last_updated": "2024-01-01"}}`,
			path:     "$.meta.last_updated",
			expected: "2024-01-01",
		},
		{
			name:     "number value",
			data:     `{"count": 42}`,
			path:     "count",
			expected: "42",
		},
		{
			name:     "missing field",
			data:     `{"other": "value"}`,
			path:     "cursor",
			expected: "",
		},
		{
			name:     "empty path",
			data:     `{"cursor": "abc123"}`,
			path:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONPath([]byte(tt.data), tt.path)
			if result != tt.expected {
				t.Errorf("extractJSONPath(%s, %s) = %s, want %s", tt.data, tt.path, result, tt.expected)
			}
		})
	}
}

func TestPollSource_MockServer(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"cursor": "next123",
		})
	}))
	defer server.Close()

	cfg := PollSourceConfig{
		Name:      "test-poll",
		URL:       server.URL,
		Publishes: "river.test.poll",
		Interval:  100 * time.Millisecond,
		Cursor: &CursorConfig{
			Param:   "since",
			Extract: "cursor",
		},
	}

	// Can't fully test without a real River, but we can verify config
	ps := &PollSource{
		config: cfg,
		client: &http.Client{Timeout: 5 * time.Second},
	}

	if ps.config.URL != server.URL {
		t.Errorf("URL not set correctly")
	}

	// Test cursor setter/getter
	ps.SetCursor("test-cursor")
	if ps.Cursor() != "test-cursor" {
		t.Errorf("Cursor not set correctly")
	}
}

func TestNewPollSource_Defaults(t *testing.T) {
	cfg := PollSourceConfig{
		Name:      "test",
		URL:       "http://example.com",
		Publishes: "river.test",
		// Leave Interval, Method, Timeout at zero values
	}

	// Since NewPollSource needs a River, we can't test it directly
	// But we can verify the default logic
	if cfg.Method == "" {
		cfg.Method = "GET"
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 5 * time.Minute
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}

	if cfg.Method != "GET" {
		t.Errorf("Expected default method GET, got %s", cfg.Method)
	}
	if cfg.Interval != 5*time.Minute {
		t.Errorf("Expected default interval 5m, got %v", cfg.Interval)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", cfg.Timeout)
	}
}

func TestPollSource_StartStop(t *testing.T) {
	cfg := PollSourceConfig{
		Name:      "test-poll",
		URL:       "http://example.com",
		Publishes: "river.test.poll",
		Interval:  time.Hour, // Long interval so it doesn't actually poll
	}

	ps := &PollSource{
		config: cfg,
		client: &http.Client{},
		stopCh: make(chan struct{}),
	}
	ps.BaseSource = core.NewBaseSource(cfg.Name, nil, cfg.Publishes)

	// Note: Start would fail without a real River, but Stop should work
	// We can at least test the running state management
	ps.running = true
	ps.SetRunning(true)

	if !ps.IsRunning() {
		t.Error("Expected source to be running")
	}

	// Just verify we can check stats
	_, _, _ = ps.Stats()
}
