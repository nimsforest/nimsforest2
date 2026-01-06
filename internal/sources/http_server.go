package sources

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// WebhookServer manages HTTP endpoints for webhook sources.
type WebhookServer struct {
	server  *http.Server
	mux     *http.ServeMux
	sources map[string]*WebhookSource
	address string

	mu      sync.Mutex
	running bool
}

// NewWebhookServer creates a new webhook HTTP server.
func NewWebhookServer(address string) *WebhookServer {
	if address == "" {
		address = "127.0.0.1:8081"
	}

	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	return &WebhookServer{
		mux:     mux,
		sources: make(map[string]*WebhookSource),
		address: address,
	}
}

// Mount registers a webhook source's handler on the server.
func (s *WebhookServer) Mount(source *WebhookSource) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := source.Name()
	path := source.Path()

	if _, exists := s.sources[name]; exists {
		return fmt.Errorf("source '%s' already mounted", name)
	}

	// Register the handler
	pattern := fmt.Sprintf("POST %s", path)
	s.mux.HandleFunc(pattern, source.Handler())
	s.sources[name] = source

	log.Printf("[WebhookServer] Mounted source '%s' at POST %s", name, path)
	return nil
}

// Unmount removes a webhook source from the server.
// Note: Go's http.ServeMux doesn't support unregistering handlers,
// so this just removes from our tracking. The handler will return 404
// when the source is not running.
func (s *WebhookServer) Unmount(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sources[name]; !exists {
		return fmt.Errorf("source '%s' not found", name)
	}

	// Stop the source
	s.sources[name].Stop()
	delete(s.sources, name)

	log.Printf("[WebhookServer] Unmounted source '%s'", name)
	return nil
}

// Source returns a mounted webhook source by name.
func (s *WebhookServer) Source(name string) *WebhookSource {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sources[name]
}

// Sources returns all mounted webhook sources.
func (s *WebhookServer) Sources() []*WebhookSource {
	s.mu.Lock()
	defer s.mu.Unlock()

	sources := make([]*WebhookSource, 0, len(s.sources))
	for _, src := range s.sources {
		sources = append(sources, src)
	}
	return sources
}

// Start starts the webhook HTTP server (non-blocking).
func (s *WebhookServer) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("webhook server already running")
	}

	s.server = &http.Server{
		Addr:         s.address,
		Handler:      s.withLogging(s.mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	s.running = true
	s.mu.Unlock()

	go func() {
		log.Printf("[WebhookServer] Listening on %s", s.address)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[WebhookServer] Server error: %v", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the webhook server.
func (s *WebhookServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	server := s.server
	s.running = false
	s.mu.Unlock()

	// Stop all mounted sources
	for _, src := range s.Sources() {
		src.Stop()
	}

	if server != nil {
		if err := server.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown webhook server: %w", err)
		}
	}

	log.Printf("[WebhookServer] Stopped")
	return nil
}

// Address returns the server address.
func (s *WebhookServer) Address() string {
	return s.address
}

// IsRunning returns whether the server is running.
func (s *WebhookServer) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// withLogging wraps a handler with request logging.
func (s *WebhookServer) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[WebhookServer] %s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}
