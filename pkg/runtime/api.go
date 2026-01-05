// Package runtime provides the management API for NimsForest.
// This HTTP API allows runtime management of TreeHouses and Nims.
package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// APIConfig configures the management API server.
type APIConfig struct {
	// Address to listen on (default: "127.0.0.1:8080")
	Address string

	// Forest instance to manage
	Forest *Forest

	// ConfigPath for reload functionality
	ConfigPath string
}

// API is the HTTP management API server for NimsForest.
type API struct {
	config APIConfig
	server *http.Server
}

// NewAPI creates a new management API server.
func NewAPI(cfg APIConfig) *API {
	if cfg.Address == "" {
		cfg.Address = "127.0.0.1:8080"
	}

	api := &API{config: cfg}

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", api.handleHealth)

	// Status
	mux.HandleFunc("GET /api/v1/status", api.handleStatus)

	// TreeHouses
	mux.HandleFunc("GET /api/v1/treehouses", api.handleListTreeHouses)
	mux.HandleFunc("POST /api/v1/treehouses", api.handleAddTreeHouse)
	mux.HandleFunc("DELETE /api/v1/treehouses/{name}", api.handleRemoveTreeHouse)

	// Nims
	mux.HandleFunc("GET /api/v1/nims", api.handleListNims)
	mux.HandleFunc("POST /api/v1/nims", api.handleAddNim)
	mux.HandleFunc("DELETE /api/v1/nims/{name}", api.handleRemoveNim)

	// Reload
	mux.HandleFunc("POST /-/reload", api.handleReload)

	api.server = &http.Server{
		Addr:         cfg.Address,
		Handler:      api.withLogging(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return api
}

// Start starts the API server (non-blocking).
func (api *API) Start() error {
	go func() {
		log.Printf("[API] Management API listening on %s", api.config.Address)
		if err := api.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[API] Server error: %v", err)
		}
	}()
	return nil
}

// Stop gracefully shuts down the API server.
func (api *API) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return api.server.Shutdown(ctx)
}

// Address returns the API server address.
func (api *API) Address() string {
	return api.config.Address
}

// Middleware for request logging
func (api *API) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[API] %s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// Helper to write JSON responses
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Helper to write error responses
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// =============================================================================
// Handlers
// =============================================================================

func (api *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (api *API) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := api.config.Forest.Status()
	status.ConfigPath = api.config.ConfigPath
	writeJSON(w, http.StatusOK, status)
}

func (api *API) handleListTreeHouses(w http.ResponseWriter, r *http.Request) {
	status := api.config.Forest.Status()
	writeJSON(w, http.StatusOK, status.TreeHouses)
}

func (api *API) handleAddTreeHouse(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string `json:"name"`
		Subscribes string `json:"subscribes"`
		Publishes  string `json:"publishes"`
		Script     string `json:"script"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Validate required fields
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Subscribes == "" {
		writeError(w, http.StatusBadRequest, "subscribes is required")
		return
	}
	if req.Publishes == "" {
		writeError(w, http.StatusBadRequest, "publishes is required")
		return
	}
	if req.Script == "" {
		writeError(w, http.StatusBadRequest, "script is required")
		return
	}

	cfg := TreeHouseConfig{
		Name:       req.Name,
		Subscribes: req.Subscribes,
		Publishes:  req.Publishes,
		Script:     req.Script,
	}

	if err := api.config.Forest.AddTreeHouse(req.Name, cfg); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			writeError(w, http.StatusConflict, err.Error())
		} else {
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"status": "created",
		"name":   req.Name,
	})
}

func (api *API) handleRemoveTreeHouse(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if err := api.config.Forest.RemoveTreeHouse(name); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (api *API) handleListNims(w http.ResponseWriter, r *http.Request) {
	status := api.config.Forest.Status()
	writeJSON(w, http.StatusOK, status.Nims)
}

func (api *API) handleAddNim(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string `json:"name"`
		Subscribes string `json:"subscribes"`
		Publishes  string `json:"publishes"`
		Prompt     string `json:"prompt"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Validate required fields
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Subscribes == "" {
		writeError(w, http.StatusBadRequest, "subscribes is required")
		return
	}
	if req.Publishes == "" {
		writeError(w, http.StatusBadRequest, "publishes is required")
		return
	}
	if req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	cfg := NimConfig{
		Name:       req.Name,
		Subscribes: req.Subscribes,
		Publishes:  req.Publishes,
		Prompt:     req.Prompt,
	}

	if err := api.config.Forest.AddNim(req.Name, cfg); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			writeError(w, http.StatusConflict, err.Error())
		} else {
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"status": "created",
		"name":   req.Name,
	})
}

func (api *API) handleRemoveNim(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if err := api.config.Forest.RemoveNim(name); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (api *API) handleReload(w http.ResponseWriter, r *http.Request) {
	if api.config.ConfigPath == "" {
		writeError(w, http.StatusBadRequest, "no config path configured")
		return
	}

	newCfg, err := LoadConfig(api.config.ConfigPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to load config: "+err.Error())
		return
	}

	if err := api.config.Forest.Reload(newCfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reload: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "reloaded",
	})
}

// =============================================================================
// API URL helper (for client to find the API)
// =============================================================================

const (
	// DefaultAPIAddress is the default address for the management API.
	DefaultAPIAddress = "127.0.0.1:8080"

	// APIAddressEnvVar is the environment variable to override the API address.
	APIAddressEnvVar = "NIMSFOREST_API"
)

// GetAPIAddress returns the API address from environment or default.
func GetAPIAddress() string {
	if addr := getEnv(APIAddressEnvVar, ""); addr != "" {
		return addr
	}
	return DefaultAPIAddress
}

// GetAPIURL returns the full URL for the API.
func GetAPIURL() string {
	return fmt.Sprintf("http://%s", GetAPIAddress())
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
