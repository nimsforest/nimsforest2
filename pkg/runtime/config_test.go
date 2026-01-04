package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	configContent := `
treehouses:
  scoring:
    subscribes: contact.created
    publishes: lead.scored
    script: scripts/scoring.lua

nims:
  qualify:
    subscribes: lead.scored
    publishes: lead.qualified
    prompt: scripts/qualify.md
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Check TreeHouses
	if len(cfg.TreeHouses) != 1 {
		t.Errorf("expected 1 treehouse, got %d", len(cfg.TreeHouses))
	}
	scoring := cfg.TreeHouses["scoring"]
	if scoring.Name != "scoring" {
		t.Errorf("expected name 'scoring', got %s", scoring.Name)
	}
	if scoring.Subscribes != "contact.created" {
		t.Errorf("expected subscribes 'contact.created', got %s", scoring.Subscribes)
	}
	if scoring.Publishes != "lead.scored" {
		t.Errorf("expected publishes 'lead.scored', got %s", scoring.Publishes)
	}

	// Check Nims
	if len(cfg.Nims) != 1 {
		t.Errorf("expected 1 nim, got %d", len(cfg.Nims))
	}
	qualify := cfg.Nims["qualify"]
	if qualify.Name != "qualify" {
		t.Errorf("expected name 'qualify', got %s", qualify.Name)
	}

	// Check path resolution
	expectedScript := filepath.Join(tmpDir, "scripts/scoring.lua")
	if cfg.ResolvePath("scripts/scoring.lua") != expectedScript {
		t.Errorf("unexpected resolved path: %s", cfg.ResolvePath("scripts/scoring.lua"))
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing treehouse subscribes",
			config: `
treehouses:
  test:
    publishes: output
    script: test.lua
`,
			expectError: true,
			errorMsg:    "missing subscribes",
		},
		{
			name: "missing treehouse publishes",
			config: `
treehouses:
  test:
    subscribes: input
    script: test.lua
`,
			expectError: true,
			errorMsg:    "missing publishes",
		},
		{
			name: "missing treehouse script",
			config: `
treehouses:
  test:
    subscribes: input
    publishes: output
`,
			expectError: true,
			errorMsg:    "missing script",
		},
		{
			name: "missing nim subscribes",
			config: `
nims:
  test:
    publishes: output
    prompt: test.md
`,
			expectError: true,
			errorMsg:    "missing subscribes",
		},
		{
			name: "missing nim publishes",
			config: `
nims:
  test:
    subscribes: input
    prompt: test.md
`,
			expectError: true,
			errorMsg:    "missing publishes",
		},
		{
			name: "missing nim prompt",
			config: `
nims:
  test:
    subscribes: input
    publishes: output
`,
			expectError: true,
			errorMsg:    "missing prompt",
		},
		{
			name: "valid empty config",
			config: `
treehouses: {}
nims: {}
`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "test.yaml")
			if err := os.WriteFile(configPath, []byte(tt.config), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			_, err := LoadConfig(configPath)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestResolveAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	configContent := `
treehouses: {}
nims: {}
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Absolute paths should remain unchanged
	absPath := "/absolute/path/to/file.lua"
	if cfg.ResolvePath(absPath) != absPath {
		t.Errorf("absolute path should remain unchanged: got %s", cfg.ResolvePath(absPath))
	}
}
