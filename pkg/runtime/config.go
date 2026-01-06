// Package runtime provides the MVP runtime for NimsForest.
// It handles loading YAML configuration and running TreeHouses (Lua) and Nims (AI).
package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the forest configuration loaded from YAML.
type Config struct {
	Sources    map[string]SourceConfig    `yaml:"sources"`
	Trees      map[string]TreeConfig      `yaml:"trees"`
	TreeHouses map[string]TreeHouseConfig `yaml:"treehouses"`
	Nims       map[string]NimConfig       `yaml:"nims"`

	// BaseDir is the directory from which the config was loaded.
	// Used to resolve relative script/prompt paths.
	BaseDir string `yaml:"-"`
}

// SourceConfig defines a Source - an entry point for external data.
type SourceConfig struct {
	Name string `yaml:"-"` // Set from map key

	// Type of source: http_webhook, http_poll, ceremony
	Type string `yaml:"type"`

	// Common fields
	Publishes string `yaml:"publishes"`

	// HTTP Webhook fields
	Path    string   `yaml:"path,omitempty"`
	Secret  string   `yaml:"secret,omitempty"`
	Headers []string `yaml:"headers,omitempty"`

	// HTTP Poll fields
	URL        string            `yaml:"url,omitempty"`
	Method     string            `yaml:"method,omitempty"`
	Interval   string            `yaml:"interval,omitempty"` // Duration string (e.g., "5m", "1h")
	ReqHeaders map[string]string `yaml:"request_headers,omitempty"`
	Body       string            `yaml:"body,omitempty"`
	Cursor     *CursorConfig     `yaml:"cursor,omitempty"`
	Timeout    string            `yaml:"timeout,omitempty"`

	// Ceremony fields
	Payload map[string]any `yaml:"payload,omitempty"`
	Script  string         `yaml:"script,omitempty"`
	Hz      int            `yaml:"hz,omitempty"`
}

// CursorConfig configures cursor-based pagination for poll sources.
type CursorConfig struct {
	Param   string `yaml:"param"`   // Query param name for cursor
	Extract string `yaml:"extract"` // JSONPath to extract next cursor from response
	Store   string `yaml:"store"`   // Key to persist cursor (optional)
}

// TreeConfig defines a Tree - a River-to-Wind adapter that parses external data.
type TreeConfig struct {
	Name      string `yaml:"-"`         // Set from map key
	Watches   string `yaml:"watches"`   // River subject to observe (JetStream)
	Publishes string `yaml:"publishes"` // Wind subject to publish Leaves to
	Script    string `yaml:"script"`    // Path to Lua script
}

// TreeHouseConfig defines a TreeHouse - a Lua-based data transformer.
type TreeHouseConfig struct {
	Name       string `yaml:"-"`          // Set from map key
	Subscribes string `yaml:"subscribes"` // NATS subject to listen on
	Publishes  string `yaml:"publishes"`  // NATS subject to publish to
	Script     string `yaml:"script"`     // Path to Lua script
}

// NimConfig defines a Nim - an AI-powered processor.
type NimConfig struct {
	Name       string `yaml:"-"`          // Set from map key
	Subscribes string `yaml:"subscribes"` // NATS subject to listen on
	Publishes  string `yaml:"publishes"`  // NATS subject to publish to
	Prompt     string `yaml:"prompt"`     // Path to prompt template (.md file)
}

// LoadConfig loads a forest configuration from a YAML file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config YAML: %w", err)
	}

	// Set BaseDir for resolving relative paths
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}
	cfg.BaseDir = filepath.Dir(absPath)

	// Set names from map keys
	for name := range cfg.Sources {
		s := cfg.Sources[name]
		s.Name = name
		cfg.Sources[name] = s
	}
	for name := range cfg.Trees {
		t := cfg.Trees[name]
		t.Name = name
		cfg.Trees[name] = t
	}
	for name := range cfg.TreeHouses {
		th := cfg.TreeHouses[name]
		th.Name = name
		cfg.TreeHouses[name] = th
	}
	for name := range cfg.Nims {
		n := cfg.Nims[name]
		n.Name = name
		cfg.Nims[name] = n
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
	// Validate sources
	for name, s := range c.Sources {
		if s.Type == "" {
			return fmt.Errorf("source %q: missing type", name)
		}
		if s.Publishes == "" {
			return fmt.Errorf("source %q: missing publishes", name)
		}
		switch s.Type {
		case "http_webhook":
			if s.Path == "" {
				return fmt.Errorf("source %q: http_webhook requires path", name)
			}
		case "http_poll":
			if s.URL == "" {
				return fmt.Errorf("source %q: http_poll requires url", name)
			}
		case "ceremony":
			if s.Interval == "" {
				return fmt.Errorf("source %q: ceremony requires interval", name)
			}
			if _, err := time.ParseDuration(s.Interval); err != nil {
				return fmt.Errorf("source %q: invalid interval %q: %w", name, s.Interval, err)
			}
		default:
			return fmt.Errorf("source %q: unknown type %q (use http_webhook, http_poll, or ceremony)", name, s.Type)
		}
	}

	for name, t := range c.Trees {
		if t.Watches == "" {
			return fmt.Errorf("tree %q: missing watches", name)
		}
		if t.Publishes == "" {
			return fmt.Errorf("tree %q: missing publishes", name)
		}
		if t.Script == "" {
			return fmt.Errorf("tree %q: missing script", name)
		}
	}

	for name, th := range c.TreeHouses {
		if th.Subscribes == "" {
			return fmt.Errorf("treehouse %q: missing subscribes", name)
		}
		if th.Publishes == "" {
			return fmt.Errorf("treehouse %q: missing publishes", name)
		}
		if th.Script == "" {
			return fmt.Errorf("treehouse %q: missing script", name)
		}
	}

	for name, n := range c.Nims {
		if n.Subscribes == "" {
			return fmt.Errorf("nim %q: missing subscribes", name)
		}
		if n.Publishes == "" {
			return fmt.Errorf("nim %q: missing publishes", name)
		}
		if n.Prompt == "" {
			return fmt.Errorf("nim %q: missing prompt", name)
		}
	}

	return nil
}

// ResolvePath resolves a relative path to an absolute path using the config's BaseDir.
func (c *Config) ResolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(c.BaseDir, path)
}

// GetTreeHouseScript returns the absolute path to a TreeHouse's Lua script.
func (c *Config) GetTreeHouseScript(name string) (string, error) {
	th, ok := c.TreeHouses[name]
	if !ok {
		return "", fmt.Errorf("treehouse %q not found", name)
	}
	return c.ResolvePath(th.Script), nil
}

// GetNimPrompt returns the absolute path to a Nim's prompt template.
func (c *Config) GetNimPrompt(name string) (string, error) {
	n, ok := c.Nims[name]
	if !ok {
		return "", fmt.Errorf("nim %q not found", name)
	}
	return c.ResolvePath(n.Prompt), nil
}
