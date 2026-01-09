package sources

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yourusername/nimsforest/internal/core"
)

// SourceType constants for the supported source types.
const (
	TypeHTTPWebhook = "http_webhook"
	TypeHTTPPoll    = "http_poll"
	TypeCeremony    = "ceremony"
	TypeTelegram    = "telegram"
)

// SourceConfig is the generic configuration for any source type.
// This is used in forest.yaml configuration.
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

	// Telegram fields
	BotToken   string `yaml:"bot_token,omitempty"`
	SecretPath string `yaml:"secret_path,omitempty"`
}

// Validate checks that the source configuration is valid.
func (c *SourceConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("source name is required")
	}
	if c.Publishes == "" {
		return fmt.Errorf("source %q: publishes is required", c.Name)
	}

	switch c.Type {
	case TypeHTTPWebhook:
		if c.Path == "" {
			return fmt.Errorf("source %q: path is required for http_webhook", c.Name)
		}
	case TypeHTTPPoll:
		if c.URL == "" {
			return fmt.Errorf("source %q: url is required for http_poll", c.Name)
		}
	case TypeCeremony:
		if c.Interval == "" {
			return fmt.Errorf("source %q: interval is required for ceremony", c.Name)
		}
		if _, err := time.ParseDuration(c.Interval); err != nil {
			return fmt.Errorf("source %q: invalid interval %q: %w", c.Name, c.Interval, err)
		}
	case TypeTelegram:
		if c.Path == "" {
			return fmt.Errorf("source %q: path is required for telegram", c.Name)
		}
		if c.BotToken == "" {
			return fmt.Errorf("source %q: bot_token is required for telegram", c.Name)
		}
	default:
		return fmt.Errorf("source %q: unknown type %q", c.Name, c.Type)
	}

	return nil
}

// Factory creates sources from configuration.
type Factory struct {
	river *core.River
	wind  *core.Wind
}

// NewFactory creates a new source factory.
func NewFactory(river *core.River, wind *core.Wind) *Factory {
	return &Factory{
		river: river,
		wind:  wind,
	}
}

// Create creates a source from configuration.
func (f *Factory) Create(cfg SourceConfig) (core.Source, error) {
	// Expand environment variables in sensitive fields
	cfg.Secret = expandEnv(cfg.Secret)
	cfg.URL = expandEnv(cfg.URL)
	cfg.BotToken = expandEnv(cfg.BotToken)
	cfg.SecretPath = expandEnv(cfg.SecretPath)
	for k, v := range cfg.ReqHeaders {
		cfg.ReqHeaders[k] = expandEnv(v)
	}

	switch cfg.Type {
	case TypeHTTPWebhook:
		return f.createWebhookSource(cfg)
	case TypeHTTPPoll:
		return f.createPollSource(cfg)
	case TypeCeremony:
		return f.createCeremonySource(cfg)
	case TypeTelegram:
		return f.createTelegramSource(cfg)
	default:
		return nil, fmt.Errorf("unknown source type: %s", cfg.Type)
	}
}

// createWebhookSource creates an HTTP webhook source.
func (f *Factory) createWebhookSource(cfg SourceConfig) (*WebhookSource, error) {
	webhookCfg := WebhookSourceConfig{
		Name:      cfg.Name,
		Path:      cfg.Path,
		Publishes: cfg.Publishes,
		Secret:    cfg.Secret,
		Headers:   cfg.Headers,
	}

	ws := NewWebhookSource(webhookCfg, f.river)

	// Set up appropriate verifier based on path hints
	if cfg.Secret != "" {
		if strings.Contains(cfg.Path, "stripe") {
			ws.SetVerifier(NewStripeVerifier(cfg.Secret))
		} else if strings.Contains(cfg.Path, "github") {
			ws.SetVerifier(NewGitHubVerifier(cfg.Secret))
		} else if strings.Contains(cfg.Path, "slack") {
			ws.SetVerifier(NewSlackVerifier(cfg.Secret))
		}
		// Otherwise, default HMAC verifier is already set
	}

	return ws, nil
}

// createPollSource creates an HTTP poll source.
func (f *Factory) createPollSource(cfg SourceConfig) (*PollSource, error) {
	interval := 5 * time.Minute // Default
	if cfg.Interval != "" {
		var err error
		interval, err = time.ParseDuration(cfg.Interval)
		if err != nil {
			return nil, fmt.Errorf("invalid interval %q: %w", cfg.Interval, err)
		}
	}

	timeout := 30 * time.Second // Default
	if cfg.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(cfg.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout %q: %w", cfg.Timeout, err)
		}
	}

	pollCfg := PollSourceConfig{
		Name:      cfg.Name,
		URL:       cfg.URL,
		Publishes: cfg.Publishes,
		Interval:  interval,
		Method:    cfg.Method,
		Headers:   cfg.ReqHeaders,
		Body:      []byte(cfg.Body),
		Cursor:    cfg.Cursor,
		Timeout:   timeout,
	}

	return NewPollSource(pollCfg, f.river), nil
}

// createCeremonySource creates a ceremony source.
func (f *Factory) createCeremonySource(cfg SourceConfig) (*CeremonySource, error) {
	if f.wind == nil {
		return nil, fmt.Errorf("wind is required for ceremony sources")
	}

	interval, err := time.ParseDuration(cfg.Interval)
	if err != nil {
		return nil, fmt.Errorf("invalid interval %q: %w", cfg.Interval, err)
	}

	ceremonyCfg := CeremonySourceConfig{
		Name:      cfg.Name,
		Interval:  interval,
		Publishes: cfg.Publishes,
		Payload:   cfg.Payload,
		Script:    cfg.Script,
		Hz:        cfg.Hz,
	}

	return NewCeremonySource(ceremonyCfg, f.wind, f.river), nil
}

// createTelegramSource creates a Telegram webhook source.
func (f *Factory) createTelegramSource(cfg SourceConfig) (*TelegramSource, error) {
	telegramCfg := TelegramSourceConfig{
		Name:       cfg.Name,
		Path:       cfg.Path,
		Publishes:  cfg.Publishes,
		BotToken:   cfg.BotToken,
		SecretPath: cfg.SecretPath,
	}

	return NewTelegramSource(telegramCfg, f.river), nil
}

// expandEnv expands environment variables in a string.
// Supports ${VAR_NAME} syntax.
func expandEnv(s string) string {
	if !strings.Contains(s, "${") {
		return s
	}
	return os.ExpandEnv(s)
}

// SourceInfo provides information about a source.
type SourceInfo struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Publishes string `json:"publishes"`
	Running   bool   `json:"running"`

	// Type-specific fields
	Path     string `json:"path,omitempty"`     // http_webhook
	URL      string `json:"url,omitempty"`      // http_poll
	Interval string `json:"interval,omitempty"` // http_poll, ceremony
}

// GetSourceInfo extracts info from a source.
func GetSourceInfo(s core.Source) SourceInfo {
	info := SourceInfo{
		Name:    s.Name(),
		Type:    s.Type(),
		Running: s.IsRunning(),
	}

	switch src := s.(type) {
	case *WebhookSource:
		cfg := src.Config()
		info.Publishes = cfg.Publishes
		info.Path = cfg.Path
	case *PollSource:
		cfg := src.Config()
		info.Publishes = cfg.Publishes
		info.URL = cfg.URL
		info.Interval = cfg.Interval.String()
	case *CeremonySource:
		cfg := src.Config()
		info.Publishes = cfg.Publishes
		info.Interval = cfg.Interval.String()
	case *TelegramSource:
		cfg := src.Config()
		info.Publishes = cfg.Publishes
		info.Path = cfg.Path
	}

	return info
}
