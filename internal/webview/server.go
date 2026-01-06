// Package webview provides an HTTP server for the isometric webview visualization.
package webview

import (
	"encoding/json"
	"io/fs"
	"net/http"

	"github.com/yourusername/nimsforest/internal/viewmodel"
)

// Server serves the webview frontend and API.
type Server struct {
	vm     *viewmodel.ViewModel
	mux    *http.ServeMux
	webDir fs.FS
}

// New creates a new webview Server.
func New(vm *viewmodel.ViewModel, webDir fs.FS) *Server {
	s := &Server{
		vm:     vm,
		mux:    http.NewServeMux(),
		webDir: webDir,
	}
	s.setupRoutes()
	return s
}

// setupRoutes configures the HTTP routes.
func (s *Server) setupRoutes() {
	// API endpoint
	s.mux.HandleFunc("/api/viewmodel", s.handleViewmodel)

	// Static files for the frontend
	if s.webDir != nil {
		s.mux.Handle("/", http.FileServer(http.FS(s.webDir)))
	} else {
		// Fallback to a basic HTML page if no webDir is provided
		s.mux.HandleFunc("/", s.handleFallback)
	}
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// handleViewmodel returns the World as JSON.
func (s *Server) handleViewmodel(w http.ResponseWriter, r *http.Request) {
	// Allow CORS for development
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Refresh the viewmodel
	if err := s.vm.Refresh(); err != nil {
		http.Error(w, "Failed to refresh viewmodel: "+err.Error(), http.StatusInternalServerError)
		return
	}

	world := s.vm.GetWorld()
	worldJSON := WorldToJSON(world)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(worldJSON); err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleFallback serves a basic HTML page when no web directory is provided.
func (s *Server) handleFallback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
    <title>NimsForest Webview</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #1a1a2e;
            color: #eee;
            margin: 0;
            padding: 20px;
        }
        h1 { color: #4ade80; }
        pre {
            background: #16213e;
            padding: 20px;
            border-radius: 8px;
            overflow: auto;
        }
        .info { color: #888; }
        button {
            background: #4ade80;
            color: #1a1a2e;
            border: none;
            padding: 10px 20px;
            border-radius: 4px;
            cursor: pointer;
            font-size: 16px;
            margin-bottom: 20px;
        }
        button:hover { background: #22c55e; }
    </style>
</head>
<body>
    <h1>🌲 NimsForest Webview</h1>
    <p class="info">The isometric view requires building the web frontend. For now, here's the raw API data:</p>
    <button onclick="refresh()">Refresh</button>
    <pre id="data">Loading...</pre>
    <script>
        async function refresh() {
            const res = await fetch('/api/viewmodel');
            const data = await res.json();
            document.getElementById('data').textContent = JSON.stringify(data, null, 2);
        }
        refresh();
    </script>
</body>
</html>`))
}
