package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/yourusername/nimsforest/internal/core"
)

// PollSourceConfig configures an HTTP poll source.
type PollSourceConfig struct {
	Name      string            // Source name (unique identifier)
	URL       string            // URL to poll
	Publishes string            // River subject to publish to
	Interval  time.Duration     // Poll interval
	Method    string            // HTTP method (GET, POST, etc.)
	Headers   map[string]string // Request headers
	Body      []byte            // Request body (for POST)
	Cursor    *CursorConfig     // Cursor/pagination config (optional)
	Timeout   time.Duration     // Request timeout
}

// CursorConfig configures cursor-based pagination.
type CursorConfig struct {
	Param   string // Query param name for cursor
	Extract string // JSONPath to extract next cursor from response
	Store   string // Key to persist cursor (optional, for Soil integration)
}

// PollSource periodically fetches data from an HTTP API and flows it into River.
type PollSource struct {
	*core.BaseSource
	config PollSourceConfig
	client *http.Client
	cursor string // Current cursor value

	mu         sync.Mutex
	running    bool
	stopCh     chan struct{}
	wg         sync.WaitGroup
	lastPoll   time.Time
	pollCount  uint64
	errorCount uint64
}

// PollPayload is the payload structure sent to the River.
type PollPayload struct {
	URL        string            `json:"url"`
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       json.RawMessage   `json:"body"`
	Timestamp  time.Time         `json:"timestamp"`
	Source     string            `json:"source"`
	Cursor     string            `json:"cursor,omitempty"`
}

// NewPollSource creates a new HTTP poll source.
func NewPollSource(cfg PollSourceConfig, river *core.River) *PollSource {
	// Set defaults
	if cfg.Method == "" {
		cfg.Method = "GET"
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 5 * time.Minute
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &PollSource{
		BaseSource: core.NewBaseSource(cfg.Name, river, cfg.Publishes),
		config:     cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		stopCh: make(chan struct{}),
	}
}

// Type returns the source type.
func (s *PollSource) Type() string {
	return "http_poll"
}

// Config returns the source configuration.
func (s *PollSource) Config() PollSourceConfig {
	return s.config
}

// Start starts the poll source, beginning periodic polling.
func (s *PollSource) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	s.SetRunning(true)

	s.wg.Add(1)
	go s.pollLoop(ctx)

	log.Printf("[PollSource] Started: %s (url: %s, interval: %s, publishes: %s)",
		s.Name(), s.config.URL, s.config.Interval, s.Publishes())
	return nil
}

// Stop stops the poll source.
func (s *PollSource) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	close(s.stopCh)
	s.mu.Unlock()

	s.wg.Wait()
	s.SetRunning(false)

	log.Printf("[PollSource] Stopped: %s (polls: %d, errors: %d)",
		s.Name(), s.pollCount, s.errorCount)
	return nil
}

// pollLoop is the main polling loop.
func (s *PollSource) pollLoop(ctx context.Context) {
	defer s.wg.Done()

	// Do an initial poll immediately
	s.doPoll(ctx)

	ticker := time.NewTicker(s.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.doPoll(ctx)
		}
	}
}

// doPoll performs a single poll operation.
func (s *PollSource) doPoll(ctx context.Context) {
	s.mu.Lock()
	s.lastPoll = time.Now()
	s.pollCount++
	s.mu.Unlock()

	// Build URL with cursor if configured
	pollURL := s.config.URL
	if s.config.Cursor != nil && s.cursor != "" {
		u, err := url.Parse(pollURL)
		if err == nil {
			q := u.Query()
			q.Set(s.config.Cursor.Param, s.cursor)
			u.RawQuery = q.Encode()
			pollURL = u.String()
		}
	}

	// Create request
	var bodyReader io.Reader
	if len(s.config.Body) > 0 {
		bodyReader = &byteReader{data: s.config.Body}
	}

	req, err := http.NewRequestWithContext(ctx, s.config.Method, pollURL, bodyReader)
	if err != nil {
		log.Printf("[PollSource] %s: Failed to create request: %v", s.Name(), err)
		s.mu.Lock()
		s.errorCount++
		s.mu.Unlock()
		return
	}

	// Set headers
	for k, v := range s.config.Headers {
		req.Header.Set(k, v)
	}

	// Execute request
	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("[PollSource] %s: Request failed: %v", s.Name(), err)
		s.mu.Lock()
		s.errorCount++
		s.mu.Unlock()
		return
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[PollSource] %s: Failed to read response: %v", s.Name(), err)
		s.mu.Lock()
		s.errorCount++
		s.mu.Unlock()
		return
	}

	// Extract cursor if configured
	if s.config.Cursor != nil && s.config.Cursor.Extract != "" {
		newCursor := extractJSONPath(body, s.config.Cursor.Extract)
		if newCursor != "" {
			s.cursor = newCursor
		}
	}

	// Extract response headers
	headers := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	// Build payload
	payload := PollPayload{
		URL:        pollURL,
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       body,
		Timestamp:  time.Now(),
		Source:     s.Name(),
		Cursor:     s.cursor,
	}

	payloadData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[PollSource] %s: Failed to marshal payload: %v", s.Name(), err)
		s.mu.Lock()
		s.errorCount++
		s.mu.Unlock()
		return
	}

	// Flow to River
	if err := s.Flow(payloadData); err != nil {
		log.Printf("[PollSource] %s: Failed to flow data: %v", s.Name(), err)
		s.mu.Lock()
		s.errorCount++
		s.mu.Unlock()
		return
	}

	log.Printf("[PollSource] %s: Poll successful (status: %d, size: %d bytes)",
		s.Name(), resp.StatusCode, len(body))
}

// SetCursor sets the cursor value (useful for resuming from a specific point).
func (s *PollSource) SetCursor(cursor string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cursor = cursor
}

// Cursor returns the current cursor value.
func (s *PollSource) Cursor() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cursor
}

// Stats returns polling statistics.
func (s *PollSource) Stats() (pollCount, errorCount uint64, lastPoll time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pollCount, s.errorCount, s.lastPoll
}

// byteReader is a simple io.Reader for a byte slice.
type byteReader struct {
	data []byte
	pos  int
}

func (r *byteReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// extractJSONPath extracts a value from JSON using a simple dot-notation path.
// E.g., "$.meta.cursor" or "meta.cursor"
func extractJSONPath(data []byte, path string) string {
	// Remove $. prefix if present
	path = removePrefix(path, "$.")
	path = removePrefix(path, ".")

	if path == "" {
		return ""
	}

	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return ""
	}

	// Split path and traverse
	parts := splitPath(path)
	current := obj

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			current = v[part]
		default:
			return ""
		}
	}

	// Convert result to string
	switch v := current.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return fmt.Sprintf("%v", v)
	default:
		return ""
	}
}

func removePrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

func splitPath(path string) []string {
	var parts []string
	var current string

	for _, c := range path {
		if c == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}
