// EXAMPLE CODE - Shows how client/server CLI pattern works
// This is a reference implementation, not production code

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// ==============================================================
// PART 1: The running daemon exposes an HTTP API
// ==============================================================

// ManagementAPI runs alongside the Forest daemon
type ManagementAPI struct {
	forest *Forest // reference to the running forest
	server *http.Server
}

func NewManagementAPI(forest *Forest, port int) *ManagementAPI {
	api := &ManagementAPI{forest: forest}

	mux := http.NewServeMux()

	// List all components
	mux.HandleFunc("GET /api/v1/components", api.handleList)

	// Add treehouse
	mux.HandleFunc("POST /api/v1/treehouses", api.handleAddTreeHouse)

	// Remove treehouse
	mux.HandleFunc("DELETE /api/v1/treehouses/{name}", api.handleRemoveTreeHouse)

	// Add nim
	mux.HandleFunc("POST /api/v1/nims", api.handleAddNim)

	// Remove nim
	mux.HandleFunc("DELETE /api/v1/nims/{name}", api.handleRemoveNim)

	// Reload config
	mux.HandleFunc("POST /-/reload", api.handleReload)

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	api.server = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", port), // localhost only!
		Handler: mux,
	}

	return api
}

func (api *ManagementAPI) Start() error {
	go api.server.ListenAndServe()
	return nil
}

func (api *ManagementAPI) handleList(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"treehouses": api.forest.ListTreeHouses(),
		"nims":       api.forest.ListNims(),
		"running":    api.forest.IsRunning(),
	}
	json.NewEncoder(w).Encode(status)
}

func (api *ManagementAPI) handleAddTreeHouse(w http.ResponseWriter, r *http.Request) {
	var cfg TreeHouseConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := api.forest.AddTreeHouse(cfg.Name, cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "created", "name": cfg.Name})
}

func (api *ManagementAPI) handleRemoveTreeHouse(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := api.forest.RemoveTreeHouse(name); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *ManagementAPI) handleAddNim(w http.ResponseWriter, r *http.Request) {
	var cfg NimConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := api.forest.AddNim(cfg.Name, cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "created", "name": cfg.Name})
}

func (api *ManagementAPI) handleRemoveNim(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := api.forest.RemoveNim(name); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *ManagementAPI) handleReload(w http.ResponseWriter, r *http.Request) {
	// Reload from config file
	if err := api.forest.ReloadConfig(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "reloaded"})
}

// ==============================================================
// PART 2: The CLI client sends HTTP requests
// ==============================================================

// Client talks to the running daemon
type ForestClient struct {
	baseURL string
}

func NewForestClient() *ForestClient {
	// Default to localhost:8080, could also check env var or config
	return &ForestClient{baseURL: "http://127.0.0.1:8080"}
}

func (c *ForestClient) List() error {
	resp, err := http.Get(c.baseURL + "/api/v1/components")
	if err != nil {
		return fmt.Errorf("cannot connect to forest daemon: %w", err)
	}
	defer resp.Body.Close()

	var status map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&status)

	// Pretty print
	fmt.Println("TREEHOUSES:")
	if ths, ok := status["treehouses"].([]interface{}); ok {
		for _, th := range ths {
			fmt.Printf("  %v\n", th)
		}
	}

	fmt.Println("\nNIMS:")
	if nims, ok := status["nims"].([]interface{}); ok {
		for _, nim := range nims {
			fmt.Printf("  %v\n", nim)
		}
	}

	return nil
}

func (c *ForestClient) AddTreeHouse(name, subscribes, publishes, script string) error {
	cfg := map[string]string{
		"name":       name,
		"subscribes": subscribes,
		"publishes":  publishes,
		"script":     script,
	}

	data, _ := json.Marshal(cfg)
	resp, err := http.Post(c.baseURL+"/api/v1/treehouses", "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("cannot connect to forest daemon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to add treehouse: %s", resp.Status)
	}

	fmt.Printf("✅ Added treehouse '%s'\n", name)
	return nil
}

func (c *ForestClient) RemoveTreeHouse(name string) error {
	req, _ := http.NewRequest(http.MethodDelete, c.baseURL+"/api/v1/treehouses/"+name, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot connect to forest daemon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("treehouse '%s' not found", name)
	}

	fmt.Printf("✅ Removed treehouse '%s'\n", name)
	return nil
}

func (c *ForestClient) Reload() error {
	resp, err := http.Post(c.baseURL+"/-/reload", "application/json", nil)
	if err != nil {
		return fmt.Errorf("cannot connect to forest daemon: %w", err)
	}
	defer resp.Body.Close()

	fmt.Println("✅ Config reloaded")
	return nil
}

// ==============================================================
// PART 3: Main function routes to daemon or client
// ==============================================================

func exampleMain() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	// DAEMON COMMANDS (start the long-running process)
	case "run", "start", "standalone":
		runDaemon() // This blocks forever

	// CLIENT COMMANDS (talk to running daemon, then exit)
	case "list":
		client := NewForestClient()
		if err := client.List(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Is the forest daemon running? Try: forest run\n")
			os.Exit(1)
		}

	case "add":
		if len(os.Args) < 3 {
			fmt.Println("Usage: forest add <treehouse|nim> ...")
			os.Exit(1)
		}
		// Parse subcommand and flags...
		client := NewForestClient()
		// client.AddTreeHouse(...)

		_ = client // placeholder

	case "remove":
		if len(os.Args) < 4 {
			fmt.Println("Usage: forest remove <treehouse|nim> <name>")
			os.Exit(1)
		}
		client := NewForestClient()
		componentType := os.Args[2]
		name := os.Args[3]

		if componentType == "treehouse" {
			if err := client.RemoveTreeHouse(name); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}

	case "reload":
		client := NewForestClient()
		if err := client.Reload(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func runDaemon() {
	fmt.Println("🌲 Starting NimsForest daemon...")

	// Create and start forest (existing code)
	forest := &Forest{} // ... initialize ...

	// NEW: Start management API alongside forest
	api := NewManagementAPI(forest, 8080)
	api.Start()
	fmt.Println("🔧 Management API at http://127.0.0.1:8080")

	// Block waiting for shutdown (existing code)
	// sigChan := make(chan os.Signal, 1)
	// signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	// <-sigChan

	_ = forest
	_ = api
}

func printUsage() {
	fmt.Println(`Usage: forest <command>

Daemon commands (start long-running process):
  run, start       Start the forest daemon
  standalone       Start in standalone mode

Management commands (talk to running daemon):
  list             List all running components
  add treehouse    Add a new treehouse
  add nim          Add a new nim
  remove treehouse Remove a treehouse
  remove nim       Remove a nim
  reload           Reload configuration

Examples:
  forest run                                    # Start daemon
  forest list                                   # Show components
  forest add treehouse scoring2 --script=x.lua # Add component
  forest reload                                 # Reload config`)
}

// Placeholder types for the example
type Forest struct{}
type TreeHouseConfig struct {
	Name       string `json:"name"`
	Subscribes string `json:"subscribes"`
	Publishes  string `json:"publishes"`
	Script     string `json:"script"`
}
type NimConfig struct {
	Name       string `json:"name"`
	Subscribes string `json:"subscribes"`
	Publishes  string `json:"publishes"`
	Prompt     string `json:"prompt"`
}

func (f *Forest) IsRunning() bool                            { return true }
func (f *Forest) ListTreeHouses() []string                   { return []string{"scoring"} }
func (f *Forest) ListNims() []string                         { return []string{"qualify"} }
func (f *Forest) AddTreeHouse(name string, cfg TreeHouseConfig) error { return nil }
func (f *Forest) RemoveTreeHouse(name string) error          { return nil }
func (f *Forest) AddNim(name string, cfg NimConfig) error    { return nil }
func (f *Forest) RemoveNim(name string) error                { return nil }
func (f *Forest) ReloadConfig() error                        { return nil }
