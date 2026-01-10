package claude

import (
	"testing"
)

func TestResolveModel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "opus alias",
			input:    "opus",
			expected: ClaudeOpusModel,
		},
		{
			name:     "sonnet alias",
			input:    "sonnet",
			expected: ClaudeSonnetModel,
		},
		{
			name:     "haiku alias",
			input:    "haiku",
			expected: ClaudeHaikuModel,
		},
		{
			name:     "full model name unchanged",
			input:    "claude-3-opus-20240229",
			expected: "claude-3-opus-20240229",
		},
		{
			name:     "unknown model unchanged",
			input:    "some-custom-model",
			expected: "some-custom-model",
		},
		{
			name:     "empty string unchanged",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveModel(tt.input)
			if result != tt.expected {
				t.Errorf("ResolveModel(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestModelConstants(t *testing.T) {
	// Verify model constants have expected values
	if ClaudeOpusModel != "claude-3-opus-20240229" {
		t.Errorf("ClaudeOpusModel = %q, want %q", ClaudeOpusModel, "claude-3-opus-20240229")
	}
	if ClaudeSonnetModel != "claude-3-5-sonnet-20241022" {
		t.Errorf("ClaudeSonnetModel = %q, want %q", ClaudeSonnetModel, "claude-3-5-sonnet-20241022")
	}
	if ClaudeHaikuModel != "claude-3-haiku-20240307" {
		t.Errorf("ClaudeHaikuModel = %q, want %q", ClaudeHaikuModel, "claude-3-haiku-20240307")
	}
	if DefaultClaudeModel != ClaudeHaikuModel {
		t.Errorf("DefaultClaudeModel = %q, want %q (same as ClaudeHaikuModel)", DefaultClaudeModel, ClaudeHaikuModel)
	}
}

func TestNewClaudeService_WithModelAlias(t *testing.T) {
	tests := []struct {
		name          string
		modelInput    string
		expectedModel string
	}{
		{
			name:          "opus alias resolves",
			modelInput:    "opus",
			expectedModel: ClaudeOpusModel,
		},
		{
			name:          "sonnet alias resolves",
			modelInput:    "sonnet",
			expectedModel: ClaudeSonnetModel,
		},
		{
			name:          "haiku alias resolves",
			modelInput:    "haiku",
			expectedModel: ClaudeHaikuModel,
		},
		{
			name:          "full name unchanged",
			modelInput:    "claude-3-opus-20240229",
			expectedModel: "claude-3-opus-20240229",
		},
		{
			name:          "empty uses default",
			modelInput:    "",
			expectedModel: DefaultClaudeModel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewClaudeService("test-api-key", tt.modelInput)
			if err != nil {
				t.Fatalf("NewClaudeService() error = %v", err)
			}

			claudeService, ok := service.(*ClaudeService)
			if !ok {
				t.Fatal("service is not *ClaudeService")
			}

			if claudeService.model != tt.expectedModel {
				t.Errorf("service.model = %q, want %q", claudeService.model, tt.expectedModel)
			}
		})
	}
}

func TestNewClaudeService_RequiresAPIKey(t *testing.T) {
	_, err := NewClaudeService("", "opus")
	if err == nil {
		t.Error("NewClaudeService() with empty API key should return error")
	}
}
