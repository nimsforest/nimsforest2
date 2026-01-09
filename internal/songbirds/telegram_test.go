package songbirds

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yourusername/nimsforest/internal/leaves"
)

func TestTelegramSongbird_Type(t *testing.T) {
	cfg := TelegramSongbirdConfig{
		Name:     "test-telegram",
		Pattern:  "song.telegram.>",
		BotToken: "test-token",
	}

	sb := NewTelegramSongbird(cfg, nil)

	if sb.Type() != "telegram" {
		t.Errorf("Type() = %s, want telegram", sb.Type())
	}
}

func TestTelegramSongbird_Config(t *testing.T) {
	cfg := TelegramSongbirdConfig{
		Name:     "test-telegram",
		Pattern:  "song.telegram.>",
		BotToken: "test-token-123",
	}

	sb := NewTelegramSongbird(cfg, nil)
	gotCfg := sb.Config()

	if gotCfg.Name != cfg.Name {
		t.Errorf("Config().Name = %s, want %s", gotCfg.Name, cfg.Name)
	}
	if gotCfg.Pattern != cfg.Pattern {
		t.Errorf("Config().Pattern = %s, want %s", gotCfg.Pattern, cfg.Pattern)
	}
	if gotCfg.BotToken != cfg.BotToken {
		t.Errorf("Config().BotToken = %s, want %s", gotCfg.BotToken, cfg.BotToken)
	}
}

func TestTelegramSongbird_StartStop(t *testing.T) {
	cfg := TelegramSongbirdConfig{
		Name:     "test-telegram",
		Pattern:  "song.telegram.>",
		BotToken: "test-token",
	}

	// Without wind, Start will fail - test that Stop works on non-started songbird
	sb := NewTelegramSongbird(cfg, nil)

	// Stop on non-running should succeed
	if err := sb.Stop(); err != nil {
		t.Errorf("Stop() on non-running songbird failed: %v", err)
	}

	if sb.IsRunning() {
		t.Error("IsRunning() should be false after Stop")
	}
}

func TestTelegramSongbird_SendMessage(t *testing.T) {
	// Create a mock Telegram API server
	var receivedChatID, receivedText string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		receivedChatID = payload["chat_id"].(string)
		receivedText = payload["text"].(string)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	cfg := TelegramSongbirdConfig{
		Name:     "test-telegram",
		Pattern:  "song.telegram.>",
		BotToken: "test-token",
	}

	sb := NewTelegramSongbird(cfg, nil)

	// Verify songbird was created with correct config
	if sb.Name() != cfg.Name {
		t.Errorf("Name() = %s, want %s", sb.Name(), cfg.Name)
	}

	// Note: Full SendMessage testing would require dependency injection
	// to mock the Telegram API. For now, we verify types are correct.
	_ = server
	_ = receivedChatID
	_ = receivedText
}

func TestChirp_JSON(t *testing.T) {
	chirp := leaves.Chirp{
		Platform: "telegram",
		ChatID:   "123456789",
		Text:     "Hello, world!",
		ReplyTo:  "99",
	}

	data, err := json.Marshal(chirp)
	if err != nil {
		t.Fatalf("Failed to marshal chirp: %v", err)
	}

	var decoded leaves.Chirp
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal chirp: %v", err)
	}

	if decoded.Platform != chirp.Platform {
		t.Errorf("Platform mismatch: got %s, want %s", decoded.Platform, chirp.Platform)
	}
	if decoded.ChatID != chirp.ChatID {
		t.Errorf("ChatID mismatch: got %s, want %s", decoded.ChatID, chirp.ChatID)
	}
	if decoded.Text != chirp.Text {
		t.Errorf("Text mismatch: got %s, want %s", decoded.Text, chirp.Text)
	}
}
