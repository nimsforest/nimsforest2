// Package sources provides implementations of River sources.
package sources

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/yourusername/nimsforest/internal/core"
)

// WebhookSourceConfig configures an HTTP webhook source.
type WebhookSourceConfig struct {
	Name      string   // Source name (unique identifier)
	Path      string   // HTTP endpoint path (e.g., "/webhooks/stripe")
	Publishes string   // River subject to publish to
	Secret    string   // For signature verification (optional)
	Headers   []string // Headers to include in payload
}

// WebhookSource receives HTTP POST requests from external services and flows them into River.
type WebhookSource struct {
	*core.BaseSource
	config   WebhookSourceConfig
	verifier SignatureVerifier

	mu      sync.Mutex
	running bool
}

// WebhookPayload is the payload structure sent to the River.
type WebhookPayload struct {
	Headers   map[string]string `json:"headers,omitempty"`
	Body      json.RawMessage   `json:"body"`
	Timestamp time.Time         `json:"timestamp"`
	Source    string            `json:"source"`
}

// NewWebhookSource creates a new HTTP webhook source.
func NewWebhookSource(cfg WebhookSourceConfig, river *core.River) *WebhookSource {
	ws := &WebhookSource{
		BaseSource: core.NewBaseSource(cfg.Name, river, cfg.Publishes),
		config:     cfg,
	}

	// Set up signature verifier if secret is provided
	if cfg.Secret != "" {
		// Default to HMAC-SHA256 verification
		ws.verifier = NewHMACVerifier(cfg.Secret, "sha256")
	}

	return ws
}

// NewWebhookSourceWithVerifier creates a webhook source with a custom signature verifier.
func NewWebhookSourceWithVerifier(cfg WebhookSourceConfig, river *core.River, verifier SignatureVerifier) *WebhookSource {
	ws := &WebhookSource{
		BaseSource: core.NewBaseSource(cfg.Name, river, cfg.Publishes),
		config:     cfg,
		verifier:   verifier,
	}
	return ws
}

// Type returns the source type.
func (s *WebhookSource) Type() string {
	return "http_webhook"
}

// Config returns the source configuration.
func (s *WebhookSource) Config() WebhookSourceConfig {
	return s.config
}

// Path returns the HTTP endpoint path.
func (s *WebhookSource) Path() string {
	return s.config.Path
}

// Start starts the webhook source.
// Note: The actual HTTP server is managed by WebhookServer.
// This just marks the source as ready to receive requests.
func (s *WebhookSource) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	s.running = true
	s.SetRunning(true)
	log.Printf("[WebhookSource] Started: %s (path: %s, publishes: %s)",
		s.Name(), s.config.Path, s.Publishes())
	return nil
}

// Stop stops the webhook source.
func (s *WebhookSource) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false
	s.SetRunning(false)
	log.Printf("[WebhookSource] Stopped: %s", s.Name())
	return nil
}

// Handler returns the HTTP handler for this webhook endpoint.
func (s *WebhookSource) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST requests
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check if source is running
		s.mu.Lock()
		running := s.running
		s.mu.Unlock()

		if !running {
			http.Error(w, "Source not running", http.StatusServiceUnavailable)
			return
		}

		// Read body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("[WebhookSource] %s: Failed to read body: %v", s.Name(), err)
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Verify signature if verifier is configured
		if s.verifier != nil {
			signature := r.Header.Get("X-Signature-256")
			if signature == "" {
				signature = r.Header.Get("X-Hub-Signature-256")
			}
			if signature == "" {
				signature = r.Header.Get("Stripe-Signature")
			}
			if signature == "" {
				signature = r.Header.Get("X-Slack-Signature")
			}

			if err := s.verifier.Verify(body, signature); err != nil {
				log.Printf("[WebhookSource] %s: Signature verification failed: %v", s.Name(), err)
				http.Error(w, "Invalid signature", http.StatusUnauthorized)
				return
			}
		}

		// Extract configured headers
		headers := make(map[string]string)
		for _, h := range s.config.Headers {
			if v := r.Header.Get(h); v != "" {
				headers[h] = v
			}
		}

		// Build payload
		payload := WebhookPayload{
			Headers:   headers,
			Body:      body,
			Timestamp: time.Now(),
			Source:    s.Name(),
		}

		payloadData, err := json.Marshal(payload)
		if err != nil {
			log.Printf("[WebhookSource] %s: Failed to marshal payload: %v", s.Name(), err)
			http.Error(w, "Failed to process payload", http.StatusInternalServerError)
			return
		}

		// Flow to River
		if err := s.Flow(payloadData); err != nil {
			log.Printf("[WebhookSource] %s: Failed to flow data: %v", s.Name(), err)
			http.Error(w, "Failed to process webhook", http.StatusInternalServerError)
			return
		}

		log.Printf("[WebhookSource] %s: Received webhook, flowed to %s (%d bytes)",
			s.Name(), s.Publishes(), len(body))

		// Return success
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}
}

// SetVerifier sets a custom signature verifier.
func (s *WebhookSource) SetVerifier(v SignatureVerifier) {
	s.verifier = v
}
