package runtime

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/yourusername/nimsforest/internal/core"
	"github.com/yourusername/nimsforest/internal/sources"
	"github.com/yourusername/nimsforest/pkg/brain"
)

// Forest is the main runtime that manages all Sources, Trees, TreeHouses and Nims.
// It uses Wind for all pub/sub operations.
type Forest struct {
	config     *Config
	wind       *core.Wind
	river      *core.River // Optional: for Trees and Sources (requires JetStream)
	humus      *core.Humus // Optional: for state tracking
	brain      brain.Brain

	// Components
	sources       map[string]core.Source
	trees         map[string]*Tree
	treehouses    map[string]*TreeHouse
	nims          map[string]*Nim

	// HTTP server for webhook sources
	webhookServer *sources.WebhookServer
	sourceFactory *sources.Factory

	mu      sync.Mutex
	running bool
}

// NewForest creates a new Forest runtime from a configuration file.
func NewForest(configPath string, wind *core.Wind, b brain.Brain) (*Forest, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return NewForestFromConfig(cfg, wind, b)
}

// NewForestFromConfig creates a new Forest runtime from an existing config.
func NewForestFromConfig(cfg *Config, wind *core.Wind, b brain.Brain) (*Forest, error) {
	if wind == nil {
		return nil, fmt.Errorf("wind is required")
	}
	if b == nil {
		return nil, fmt.Errorf("brain is required")
	}

	f := &Forest{
		config:     cfg,
		wind:       wind,
		brain:      b,
		sources:    make(map[string]core.Source),
		trees:      make(map[string]*Tree),
		treehouses: make(map[string]*TreeHouse),
		nims:       make(map[string]*Nim),
	}

	// Note: Trees and Sources require River, which must be set via SetRiver() before Start()

	// Create TreeHouses
	for name, thCfg := range cfg.TreeHouses {
		scriptPath := cfg.ResolvePath(thCfg.Script)
		th, err := NewTreeHouse(thCfg, wind, scriptPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create treehouse %s: %w", name, err)
		}
		f.treehouses[name] = th
	}

	// Create Nims
	for name, nimCfg := range cfg.Nims {
		promptPath := cfg.ResolvePath(nimCfg.Prompt)
		nim, err := NewNim(nimCfg, wind, b, promptPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create nim %s: %w", name, err)
		}
		f.nims[name] = nim
	}

	return f, nil
}

// NewForestWithHumus creates a Forest with Humus for state change tracking.
func NewForestWithHumus(cfg *Config, wind *core.Wind, humus *core.Humus, b brain.Brain) (*Forest, error) {
	if wind == nil {
		return nil, fmt.Errorf("wind is required")
	}
	if b == nil {
		return nil, fmt.Errorf("brain is required")
	}

	f := &Forest{
		config:     cfg,
		wind:       wind,
		humus:      humus,
		brain:      b,
		sources:    make(map[string]core.Source),
		trees:      make(map[string]*Tree),
		treehouses: make(map[string]*TreeHouse),
		nims:       make(map[string]*Nim),
	}

	// Note: Trees and Sources require River, which must be set via SetRiver() before Start()

	// Create TreeHouses
	for name, thCfg := range cfg.TreeHouses {
		scriptPath := cfg.ResolvePath(thCfg.Script)
		th, err := NewTreeHouse(thCfg, wind, scriptPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create treehouse %s: %w", name, err)
		}
		f.treehouses[name] = th
	}

	// Create Nims (with Humus if provided)
	for name, nimCfg := range cfg.Nims {
		promptPath := cfg.ResolvePath(nimCfg.Prompt)
		var nim *Nim
		var err error

		if humus != nil {
			nim, err = NewNimWithHumus(nimCfg, wind, humus, b, promptPath)
		} else {
			nim, err = NewNim(nimCfg, wind, b, promptPath)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to create nim %s: %w", name, err)
		}
		f.nims[name] = nim
	}

	return f, nil
}

// Start starts all Sources, Trees, TreeHouses and Nims.
func (f *Forest) Start(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.running {
		return fmt.Errorf("forest already running")
	}

	// Create source factory if river is available
	if f.river != nil {
		f.sourceFactory = sources.NewFactory(f.river, f.wind)

		// Create Sources from config
		if f.config != nil {
			hasWebhooks := false
			for name, srcCfg := range f.config.Sources {
				if _, exists := f.sources[name]; exists {
					continue // Already created
				}

				// Convert config to sources.SourceConfig
				factoryCfg := sources.SourceConfig{
					Name:       name,
					Type:       srcCfg.Type,
					Publishes:  srcCfg.Publishes,
					Path:       srcCfg.Path,
					Secret:     srcCfg.Secret,
					Headers:    srcCfg.Headers,
					URL:        srcCfg.URL,
					Method:     srcCfg.Method,
					Interval:   srcCfg.Interval,
					ReqHeaders: srcCfg.ReqHeaders,
					Body:       srcCfg.Body,
					Timeout:    srcCfg.Timeout,
					Payload:    srcCfg.Payload,
					Script:     srcCfg.Script,
					Hz:         srcCfg.Hz,
				}
				if srcCfg.Cursor != nil {
					factoryCfg.Cursor = &sources.CursorConfig{
						Param:   srcCfg.Cursor.Param,
						Extract: srcCfg.Cursor.Extract,
						Store:   srcCfg.Cursor.Store,
					}
				}

				src, err := f.sourceFactory.Create(factoryCfg)
				if err != nil {
					log.Printf("[Forest] Warning: failed to create source %s: %v", name, err)
					continue
				}
				f.sources[name] = src

				// Track if we have webhook sources
				if srcCfg.Type == "http_webhook" {
					hasWebhooks = true
				}
			}

			// Start webhook server if we have webhook sources
			if hasWebhooks && f.webhookServer == nil {
				f.webhookServer = sources.NewWebhookServer(GetWebhookAddress())
			}
		}

		// Create Trees from config
		for name, treeCfg := range f.config.Trees {
			if _, exists := f.trees[name]; exists {
				continue // Already created
			}
			treeCfg.Name = name
			scriptPath := f.config.ResolvePath(treeCfg.Script)
			tree, err := NewTree(treeCfg, f.wind, f.river, scriptPath)
			if err != nil {
				log.Printf("[Forest] Warning: failed to create tree %s: %v", name, err)
				continue
			}
			f.trees[name] = tree
		}
	}

	// Mount and start webhook sources
	if f.webhookServer != nil {
		for _, src := range f.sources {
			if ws, ok := src.(*sources.WebhookSource); ok {
				if err := f.webhookServer.Mount(ws); err != nil {
					log.Printf("[Forest] Warning: failed to mount webhook source %s: %v", ws.Name(), err)
				}
			}
		}
		if err := f.webhookServer.Start(); err != nil {
			log.Printf("[Forest] Warning: failed to start webhook server: %v", err)
		}
	}

	// Start Sources
	for name, src := range f.sources {
		if err := src.Start(ctx); err != nil {
			log.Printf("[Forest] Warning: failed to start source %s: %v", name, err)
		}
	}

	// Start Trees
	for name, tree := range f.trees {
		if err := tree.Start(ctx); err != nil {
			f.stopAll()
			return fmt.Errorf("failed to start tree %s: %w", name, err)
		}
	}

	// Start TreeHouses
	for name, th := range f.treehouses {
		if err := th.Start(ctx); err != nil {
			f.stopAll()
			return fmt.Errorf("failed to start treehouse %s: %w", name, err)
		}
	}

	// Start Nims
	for name, nim := range f.nims {
		if err := nim.Start(ctx); err != nil {
			f.stopAll()
			return fmt.Errorf("failed to start nim %s: %w", name, err)
		}
	}

	f.running = true
	log.Printf("[Forest] Started with %d sources, %d trees, %d treehouses and %d nims",
		len(f.sources), len(f.trees), len(f.treehouses), len(f.nims))
	return nil
}

// Stop stops all TreeHouses and Nims.
func (f *Forest) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.running {
		return nil
	}

	f.stopAll()
	f.running = false
	log.Printf("[Forest] Stopped")
	return nil
}

// stopAll stops all components without locking (internal use).
func (f *Forest) stopAll() {
	// Stop sources first
	for _, src := range f.sources {
		src.Stop()
	}
	// Stop webhook server
	if f.webhookServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		f.webhookServer.Stop(ctx)
		cancel()
	}
	for _, tree := range f.trees {
		tree.Stop()
	}
	for _, th := range f.treehouses {
		th.Stop()
	}
	for _, nim := range f.nims {
		nim.Stop()
	}
}

// TreeHouse returns a TreeHouse by name.
func (f *Forest) TreeHouse(name string) *TreeHouse {
	return f.treehouses[name]
}

// Nim returns a Nim by name.
func (f *Forest) Nim(name string) *Nim {
	return f.nims[name]
}

// Config returns the forest configuration.
func (f *Forest) Config() *Config {
	return f.config
}

// IsRunning returns whether the forest is currently running.
func (f *Forest) IsRunning() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.running
}

// SetRiver sets the River connection for tree and source support.
// Must be called before adding trees or sources.
func (f *Forest) SetRiver(river *core.River) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.river = river
	// Create source factory when river is set
	if f.sourceFactory == nil && river != nil {
		f.sourceFactory = sources.NewFactory(river, f.wind)
	}
}

// GetWebhookAddress returns the webhook server address.
func GetWebhookAddress() string {
	if addr := getEnv("NIMSFOREST_WEBHOOK_ADDR", ""); addr != "" {
		return addr
	}
	return "127.0.0.1:8081"
}

// =============================================================================
// Runtime Component Management
// =============================================================================

// ComponentInfo provides information about a running component.
type ComponentInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"` // "treehouse" or "nim"
	Subscribes string `json:"subscribes"`
	Publishes  string `json:"publishes"`
	Script     string `json:"script,omitempty"` // TreeHouse only
	Prompt     string `json:"prompt,omitempty"` // Nim only
	Running    bool   `json:"running"`
}

// TreeInfo provides information about a running tree.
type TreeInfo struct {
	Name      string `json:"name"`
	Watches   string `json:"watches"`
	Publishes string `json:"publishes"`
	Script    string `json:"script,omitempty"`
	Running   bool   `json:"running"`
}

// SourceInfo provides information about a running source.
type SourceInfo struct {
	Name      string `json:"name"`
	Type      string `json:"type"` // "http_webhook", "http_poll", "ceremony"
	Publishes string `json:"publishes"`
	Running   bool   `json:"running"`

	// Type-specific fields
	Path     string `json:"path,omitempty"`     // http_webhook
	URL      string `json:"url,omitempty"`      // http_poll
	Interval string `json:"interval,omitempty"` // http_poll, ceremony
}

// ForestStatus provides a snapshot of the forest state.
type ForestStatus struct {
	Running    bool            `json:"running"`
	Sources    []SourceInfo    `json:"sources"`
	Trees      []TreeInfo      `json:"trees"`
	TreeHouses []ComponentInfo `json:"treehouses"`
	Nims       []ComponentInfo `json:"nims"`
	ConfigPath string          `json:"config_path,omitempty"`
}

// Status returns the current status of the forest.
func (f *Forest) Status() ForestStatus {
	f.mu.Lock()
	defer f.mu.Unlock()

	status := ForestStatus{
		Running:    f.running,
		Sources:    make([]SourceInfo, 0, len(f.sources)),
		Trees:      make([]TreeInfo, 0, len(f.trees)),
		TreeHouses: make([]ComponentInfo, 0, len(f.treehouses)),
		Nims:       make([]ComponentInfo, 0, len(f.nims)),
	}

	for name, src := range f.sources {
		info := sources.GetSourceInfo(src)
		status.Sources = append(status.Sources, SourceInfo{
			Name:      name,
			Type:      info.Type,
			Publishes: info.Publishes,
			Running:   info.Running,
			Path:      info.Path,
			URL:       info.URL,
			Interval:  info.Interval,
		})
	}

	for name, tree := range f.trees {
		cfg := f.config.Trees[name]
		status.Trees = append(status.Trees, TreeInfo{
			Name:      name,
			Watches:   cfg.Watches,
			Publishes: cfg.Publishes,
			Script:    cfg.Script,
			Running:   tree.IsRunning(),
		})
	}

	for name, th := range f.treehouses {
		cfg := f.config.TreeHouses[name]
		status.TreeHouses = append(status.TreeHouses, ComponentInfo{
			Name:       name,
			Type:       "treehouse",
			Subscribes: cfg.Subscribes,
			Publishes:  cfg.Publishes,
			Script:     cfg.Script,
			Running:    th.IsRunning(),
		})
	}

	for name, nim := range f.nims {
		cfg := f.config.Nims[name]
		status.Nims = append(status.Nims, ComponentInfo{
			Name:       name,
			Type:       "nim",
			Subscribes: cfg.Subscribes,
			Publishes:  cfg.Publishes,
			Prompt:     cfg.Prompt,
			Running:    nim.IsRunning(),
		})
	}

	return status
}

// AddSource adds a new Source at runtime.
// Requires River to be set via SetRiver() first.
func (f *Forest) AddSource(name string, cfg SourceConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.sources[name]; exists {
		return fmt.Errorf("source '%s' already exists", name)
	}

	if f.river == nil {
		return fmt.Errorf("river is required for sources (call SetRiver first)")
	}

	if f.sourceFactory == nil {
		f.sourceFactory = sources.NewFactory(f.river, f.wind)
	}

	// Ensure name is set
	cfg.Name = name

	// Convert to factory config
	factoryCfg := sources.SourceConfig{
		Name:       name,
		Type:       cfg.Type,
		Publishes:  cfg.Publishes,
		Path:       cfg.Path,
		Secret:     cfg.Secret,
		Headers:    cfg.Headers,
		URL:        cfg.URL,
		Method:     cfg.Method,
		Interval:   cfg.Interval,
		ReqHeaders: cfg.ReqHeaders,
		Body:       cfg.Body,
		Timeout:    cfg.Timeout,
		Payload:    cfg.Payload,
		Script:     cfg.Script,
		Hz:         cfg.Hz,
	}
	if cfg.Cursor != nil {
		factoryCfg.Cursor = &sources.CursorConfig{
			Param:   cfg.Cursor.Param,
			Extract: cfg.Cursor.Extract,
			Store:   cfg.Cursor.Store,
		}
	}

	// Create the source
	src, err := f.sourceFactory.Create(factoryCfg)
	if err != nil {
		return fmt.Errorf("failed to create source: %w", err)
	}

	// Handle webhook sources
	if ws, ok := src.(*sources.WebhookSource); ok {
		// Start webhook server if needed
		if f.webhookServer == nil {
			f.webhookServer = sources.NewWebhookServer(GetWebhookAddress())
			if f.running {
				if err := f.webhookServer.Start(); err != nil {
					return fmt.Errorf("failed to start webhook server: %w", err)
				}
			}
		}
		if err := f.webhookServer.Mount(ws); err != nil {
			return fmt.Errorf("failed to mount webhook source: %w", err)
		}
	}

	// Start it if the forest is running
	if f.running {
		if err := src.Start(context.Background()); err != nil {
			return fmt.Errorf("failed to start source: %w", err)
		}
	}

	// Add to maps
	f.sources[name] = src
	if f.config.Sources == nil {
		f.config.Sources = make(map[string]SourceConfig)
	}
	f.config.Sources[name] = cfg

	log.Printf("[Forest] Added source '%s' (type: %s, publishes: %s)",
		name, cfg.Type, cfg.Publishes)
	return nil
}

// RemoveSource removes a Source at runtime.
func (f *Forest) RemoveSource(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	src, exists := f.sources[name]
	if !exists {
		return fmt.Errorf("source '%s' not found", name)
	}

	// Stop it
	if err := src.Stop(); err != nil {
		log.Printf("[Forest] Warning: error stopping source '%s': %v", name, err)
	}

	// Unmount if webhook
	if _, ok := src.(*sources.WebhookSource); ok && f.webhookServer != nil {
		f.webhookServer.Unmount(name)
	}

	// Remove from maps
	delete(f.sources, name)
	delete(f.config.Sources, name)

	log.Printf("[Forest] Removed source '%s'", name)
	return nil
}

// PauseSource pauses a Source (stops it but keeps it in the config).
func (f *Forest) PauseSource(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	src, exists := f.sources[name]
	if !exists {
		return fmt.Errorf("source '%s' not found", name)
	}

	if err := src.Stop(); err != nil {
		return fmt.Errorf("failed to pause source: %w", err)
	}

	log.Printf("[Forest] Paused source '%s'", name)
	return nil
}

// ResumeSource resumes a paused Source.
func (f *Forest) ResumeSource(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	src, exists := f.sources[name]
	if !exists {
		return fmt.Errorf("source '%s' not found", name)
	}

	if err := src.Start(context.Background()); err != nil {
		return fmt.Errorf("failed to resume source: %w", err)
	}

	log.Printf("[Forest] Resumed source '%s'", name)
	return nil
}

// ListSources returns information about all sources.
func (f *Forest) ListSources() []SourceInfo {
	f.mu.Lock()
	defer f.mu.Unlock()

	infos := make([]SourceInfo, 0, len(f.sources))
	for name, src := range f.sources {
		info := sources.GetSourceInfo(src)
		infos = append(infos, SourceInfo{
			Name:      name,
			Type:      info.Type,
			Publishes: info.Publishes,
			Running:   info.Running,
			Path:      info.Path,
			URL:       info.URL,
			Interval:  info.Interval,
		})
	}
	return infos
}

// AddTree adds a new Tree at runtime.
// Requires River to be set via SetRiver() first.
func (f *Forest) AddTree(name string, cfg TreeConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.trees[name]; exists {
		return fmt.Errorf("tree '%s' already exists", name)
	}

	if f.river == nil {
		return fmt.Errorf("river is required for trees (call SetRiver first)")
	}

	// Ensure name is set
	cfg.Name = name

	// Resolve script path
	scriptPath := f.config.ResolvePath(cfg.Script)

	// Create the Tree
	tree, err := NewTree(cfg, f.wind, f.river, scriptPath)
	if err != nil {
		return fmt.Errorf("failed to create tree: %w", err)
	}

	// Start it if the forest is running
	if f.running {
		if err := tree.Start(context.Background()); err != nil {
			return fmt.Errorf("failed to start tree: %w", err)
		}
	}

	// Add to maps
	f.trees[name] = tree
	if f.config.Trees == nil {
		f.config.Trees = make(map[string]TreeConfig)
	}
	f.config.Trees[name] = cfg

	log.Printf("[Forest] Added tree '%s' (watches: %s, publishes: %s)",
		name, cfg.Watches, cfg.Publishes)
	return nil
}

// RemoveTree removes a Tree at runtime.
func (f *Forest) RemoveTree(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	tree, exists := f.trees[name]
	if !exists {
		return fmt.Errorf("tree '%s' not found", name)
	}

	// Stop it
	if err := tree.Stop(); err != nil {
		log.Printf("[Forest] Warning: error stopping tree '%s': %v", name, err)
	}

	// Remove from maps
	delete(f.trees, name)
	delete(f.config.Trees, name)

	log.Printf("[Forest] Removed tree '%s'", name)
	return nil
}

// AddTreeHouse adds a new TreeHouse at runtime.
func (f *Forest) AddTreeHouse(name string, cfg TreeHouseConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.treehouses[name]; exists {
		return fmt.Errorf("treehouse '%s' already exists", name)
	}

	// Ensure name is set
	cfg.Name = name

	// Resolve script path
	scriptPath := f.config.ResolvePath(cfg.Script)

	// Create the TreeHouse
	th, err := NewTreeHouse(cfg, f.wind, scriptPath)
	if err != nil {
		return fmt.Errorf("failed to create treehouse: %w", err)
	}

	// Start it if the forest is running
	if f.running {
		if err := th.Start(context.Background()); err != nil {
			return fmt.Errorf("failed to start treehouse: %w", err)
		}
	}

	// Add to maps
	f.treehouses[name] = th
	if f.config.TreeHouses == nil {
		f.config.TreeHouses = make(map[string]TreeHouseConfig)
	}
	f.config.TreeHouses[name] = cfg

	log.Printf("[Forest] Added treehouse '%s' (subscribes: %s, publishes: %s)",
		name, cfg.Subscribes, cfg.Publishes)
	return nil
}

// RemoveTreeHouse removes a TreeHouse at runtime.
func (f *Forest) RemoveTreeHouse(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	th, exists := f.treehouses[name]
	if !exists {
		return fmt.Errorf("treehouse '%s' not found", name)
	}

	// Stop it
	if err := th.Stop(); err != nil {
		log.Printf("[Forest] Warning: error stopping treehouse '%s': %v", name, err)
	}

	// Remove from maps
	delete(f.treehouses, name)
	delete(f.config.TreeHouses, name)

	log.Printf("[Forest] Removed treehouse '%s'", name)
	return nil
}

// AddNim adds a new Nim at runtime.
func (f *Forest) AddNim(name string, cfg NimConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.nims[name]; exists {
		return fmt.Errorf("nim '%s' already exists", name)
	}

	// Ensure name is set
	cfg.Name = name

	// Resolve prompt path
	promptPath := f.config.ResolvePath(cfg.Prompt)

	// Create the Nim
	var nim *Nim
	var err error

	if f.humus != nil {
		nim, err = NewNimWithHumus(cfg, f.wind, f.humus, f.brain, promptPath)
	} else {
		nim, err = NewNim(cfg, f.wind, f.brain, promptPath)
	}

	if err != nil {
		return fmt.Errorf("failed to create nim: %w", err)
	}

	// Start it if the forest is running
	if f.running {
		if err := nim.Start(context.Background()); err != nil {
			return fmt.Errorf("failed to start nim: %w", err)
		}
	}

	// Add to maps
	f.nims[name] = nim
	if f.config.Nims == nil {
		f.config.Nims = make(map[string]NimConfig)
	}
	f.config.Nims[name] = cfg

	log.Printf("[Forest] Added nim '%s' (subscribes: %s, publishes: %s)",
		name, cfg.Subscribes, cfg.Publishes)
	return nil
}

// RemoveNim removes a Nim at runtime.
func (f *Forest) RemoveNim(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	nim, exists := f.nims[name]
	if !exists {
		return fmt.Errorf("nim '%s' not found", name)
	}

	// Stop it
	if err := nim.Stop(); err != nil {
		log.Printf("[Forest] Warning: error stopping nim '%s': %v", name, err)
	}

	// Remove from maps
	delete(f.nims, name)
	delete(f.config.Nims, name)

	log.Printf("[Forest] Removed nim '%s'", name)
	return nil
}

// Reload reloads the forest configuration, adding/removing components as needed.
func (f *Forest) Reload(newCfg *Config) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Find TreeHouses to remove (in old config but not in new)
	for name := range f.treehouses {
		if _, exists := newCfg.TreeHouses[name]; !exists {
			if th := f.treehouses[name]; th != nil {
				th.Stop()
			}
			delete(f.treehouses, name)
			log.Printf("[Forest] Removed treehouse '%s' (not in new config)", name)
		}
	}

	// Find TreeHouses to add (in new config but not running)
	for name, cfg := range newCfg.TreeHouses {
		if _, exists := f.treehouses[name]; !exists {
			cfg.Name = name
			scriptPath := newCfg.ResolvePath(cfg.Script)
			th, err := NewTreeHouse(cfg, f.wind, scriptPath)
			if err != nil {
				log.Printf("[Forest] Warning: failed to create treehouse '%s': %v", name, err)
				continue
			}
			if f.running {
				if err := th.Start(context.Background()); err != nil {
					log.Printf("[Forest] Warning: failed to start treehouse '%s': %v", name, err)
					continue
				}
			}
			f.treehouses[name] = th
			log.Printf("[Forest] Added treehouse '%s' from new config", name)
		}
	}

	// Find Nims to remove
	for name := range f.nims {
		if _, exists := newCfg.Nims[name]; !exists {
			if nim := f.nims[name]; nim != nil {
				nim.Stop()
			}
			delete(f.nims, name)
			log.Printf("[Forest] Removed nim '%s' (not in new config)", name)
		}
	}

	// Find Nims to add
	for name, cfg := range newCfg.Nims {
		if _, exists := f.nims[name]; !exists {
			cfg.Name = name
			promptPath := newCfg.ResolvePath(cfg.Prompt)
			var nim *Nim
			var err error
			if f.humus != nil {
				nim, err = NewNimWithHumus(cfg, f.wind, f.humus, f.brain, promptPath)
			} else {
				nim, err = NewNim(cfg, f.wind, f.brain, promptPath)
			}
			if err != nil {
				log.Printf("[Forest] Warning: failed to create nim '%s': %v", name, err)
				continue
			}
			if f.running {
				if err := nim.Start(context.Background()); err != nil {
					log.Printf("[Forest] Warning: failed to start nim '%s': %v", name, err)
					continue
				}
			}
			f.nims[name] = nim
			log.Printf("[Forest] Added nim '%s' from new config", name)
		}
	}

	// Update config reference
	f.config = newCfg

	log.Printf("[Forest] Reloaded with %d treehouses and %d nims",
		len(f.treehouses), len(f.nims))
	return nil
}
