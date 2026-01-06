package sources

import (
	"testing"
	"time"

	"github.com/yourusername/nimsforest/internal/core"
)

func TestSourceConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     SourceConfig
		wantErr bool
	}{
		{
			name: "valid http_webhook",
			cfg: SourceConfig{
				Name:      "test-webhook",
				Type:      TypeHTTPWebhook,
				Publishes: "river.test",
				Path:      "/webhooks/test",
			},
			wantErr: false,
		},
		{
			name: "http_webhook missing path",
			cfg: SourceConfig{
				Name:      "test-webhook",
				Type:      TypeHTTPWebhook,
				Publishes: "river.test",
			},
			wantErr: true,
		},
		{
			name: "valid http_poll",
			cfg: SourceConfig{
				Name:      "test-poll",
				Type:      TypeHTTPPoll,
				Publishes: "river.test",
				URL:       "https://example.com/api",
			},
			wantErr: false,
		},
		{
			name: "http_poll missing url",
			cfg: SourceConfig{
				Name:      "test-poll",
				Type:      TypeHTTPPoll,
				Publishes: "river.test",
			},
			wantErr: true,
		},
		{
			name: "valid ceremony",
			cfg: SourceConfig{
				Name:      "test-ceremony",
				Type:      TypeCeremony,
				Publishes: "river.test",
				Interval:  "30s",
			},
			wantErr: false,
		},
		{
			name: "ceremony missing interval",
			cfg: SourceConfig{
				Name:      "test-ceremony",
				Type:      TypeCeremony,
				Publishes: "river.test",
			},
			wantErr: true,
		},
		{
			name: "ceremony invalid interval",
			cfg: SourceConfig{
				Name:      "test-ceremony",
				Type:      TypeCeremony,
				Publishes: "river.test",
				Interval:  "invalid",
			},
			wantErr: true,
		},
		{
			name: "missing name",
			cfg: SourceConfig{
				Type:      TypeHTTPWebhook,
				Publishes: "river.test",
				Path:      "/webhooks/test",
			},
			wantErr: true,
		},
		{
			name: "missing publishes",
			cfg: SourceConfig{
				Name: "test",
				Type: TypeHTTPWebhook,
				Path: "/webhooks/test",
			},
			wantErr: true,
		},
		{
			name: "unknown type",
			cfg: SourceConfig{
				Name:      "test",
				Type:      "unknown",
				Publishes: "river.test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSourceTypeConstants(t *testing.T) {
	if TypeHTTPWebhook != "http_webhook" {
		t.Errorf("TypeHTTPWebhook = %s, want http_webhook", TypeHTTPWebhook)
	}
	if TypeHTTPPoll != "http_poll" {
		t.Errorf("TypeHTTPPoll = %s, want http_poll", TypeHTTPPoll)
	}
	if TypeCeremony != "ceremony" {
		t.Errorf("TypeCeremony = %s, want ceremony", TypeCeremony)
	}
}

func TestExpandEnv(t *testing.T) {
	// Test that strings without ${} are returned unchanged
	result := expandEnv("plain-string")
	if result != "plain-string" {
		t.Errorf("expandEnv(plain-string) = %s, want plain-string", result)
	}
}

func TestGetSourceInfo(t *testing.T) {
	// Test with a WebhookSource
	wsCfg := WebhookSourceConfig{
		Name:      "test-webhook",
		Path:      "/webhooks/test",
		Publishes: "river.test.webhook",
	}

	ws := &WebhookSource{
		config: wsCfg,
	}
	ws.BaseSource = core.NewBaseSource(wsCfg.Name, nil, wsCfg.Publishes)
	ws.SetRunning(true)

	info := GetSourceInfo(ws)

	if info.Name != "test-webhook" {
		t.Errorf("Name = %s, want test-webhook", info.Name)
	}
	if info.Type != "http_webhook" {
		t.Errorf("Type = %s, want http_webhook", info.Type)
	}
	if info.Path != "/webhooks/test" {
		t.Errorf("Path = %s, want /webhooks/test", info.Path)
	}
	if !info.Running {
		t.Error("Running should be true")
	}
}

func TestGetSourceInfo_Poll(t *testing.T) {
	psCfg := PollSourceConfig{
		Name:      "test-poll",
		URL:       "https://example.com/api",
		Publishes: "river.test.poll",
		Interval:  5 * time.Minute,
	}

	ps := &PollSource{
		config: psCfg,
	}
	ps.BaseSource = core.NewBaseSource(psCfg.Name, nil, psCfg.Publishes)

	info := GetSourceInfo(ps)

	if info.Name != "test-poll" {
		t.Errorf("Name = %s, want test-poll", info.Name)
	}
	if info.Type != "http_poll" {
		t.Errorf("Type = %s, want http_poll", info.Type)
	}
	if info.URL != "https://example.com/api" {
		t.Errorf("URL = %s, want https://example.com/api", info.URL)
	}
	if info.Interval != "5m0s" {
		t.Errorf("Interval = %s, want 5m0s", info.Interval)
	}
}

func TestGetSourceInfo_Ceremony(t *testing.T) {
	csCfg := CeremonySourceConfig{
		Name:      "test-ceremony",
		Publishes: "river.test.ceremony",
		Interval:  30 * time.Second,
	}

	cs := &CeremonySource{
		config: csCfg,
	}
	cs.BaseSource = core.NewBaseSource(csCfg.Name, nil, csCfg.Publishes)

	info := GetSourceInfo(cs)

	if info.Name != "test-ceremony" {
		t.Errorf("Name = %s, want test-ceremony", info.Name)
	}
	if info.Type != "ceremony" {
		t.Errorf("Type = %s, want ceremony", info.Type)
	}
	if info.Interval != "30s" {
		t.Errorf("Interval = %s, want 30s", info.Interval)
	}
}
