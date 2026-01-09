package songbirds

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/yourusername/nimsforest/internal/core"
	"github.com/yourusername/nimsforest/internal/leaves"
)

// TelegramSongbirdConfig configures a Telegram songbird.
type TelegramSongbirdConfig struct {
	Name     string // Songbird name (unique identifier)
	Pattern  string // Wind subject pattern to listen for (e.g., "song.telegram.>")
	BotToken string // Telegram Bot API token
}

// TelegramSongbird listens for song.telegram.* patterns on the wind
// and chirps messages to Telegram.
type TelegramSongbird struct {
	*BaseSongbird
	config     TelegramSongbirdConfig
	httpClient *http.Client
	sub        *nats.Subscription

	mu      sync.Mutex
	running bool
}

// NewTelegramSongbird creates a new Telegram songbird.
func NewTelegramSongbird(cfg TelegramSongbirdConfig, wind *core.Wind) *TelegramSongbird {
	return &TelegramSongbird{
		BaseSongbird: NewBaseSongbird(cfg.Name, cfg.Pattern, wind),
		config:       cfg,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// Type returns the songbird type.
func (s *TelegramSongbird) Type() string {
	return "telegram"
}

// Config returns the songbird configuration.
func (s *TelegramSongbird) Config() TelegramSongbirdConfig {
	return s.config
}

// Start begins listening for chirps on the wind.
func (s *TelegramSongbird) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	// Subscribe to the pattern (e.g., "song.telegram.>")
	sub, err := s.GetWind().Catch(s.config.Pattern, func(leaf core.Leaf) {
		s.handleLeaf(ctx, leaf)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", s.config.Pattern, err)
	}

	s.sub = sub
	s.running = true
	s.SetRunning(true)
	log.Printf("[TelegramSongbird] Started: %s (listening: %s)", s.Name(), s.config.Pattern)
	return nil
}

// Stop gracefully shuts down the songbird.
func (s *TelegramSongbird) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	if s.sub != nil {
		if err := s.sub.Unsubscribe(); err != nil {
			log.Printf("[TelegramSongbird] %s: Error unsubscribing: %v", s.Name(), err)
		}
		s.sub = nil
	}

	s.running = false
	s.SetRunning(false)
	log.Printf("[TelegramSongbird] Stopped: %s", s.Name())
	return nil
}

// handleLeaf processes an incoming leaf and chirps to Telegram.
func (s *TelegramSongbird) handleLeaf(ctx context.Context, leaf core.Leaf) {
	// Parse the chirp from the leaf data
	var chirp leaves.Chirp
	if err := json.Unmarshal(leaf.Data, &chirp); err != nil {
		log.Printf("[TelegramSongbird] %s: Failed to parse chirp: %v", s.Name(), err)
		return
	}

	// Validate this is for Telegram
	if chirp.Platform != "telegram" {
		log.Printf("[TelegramSongbird] %s: Ignoring non-telegram chirp (platform: %s)", s.Name(), chirp.Platform)
		return
	}

	// Send to Telegram
	if err := s.sendMessage(ctx, chirp.ChatID, chirp.Text); err != nil {
		log.Printf("[TelegramSongbird] %s: Failed to send message: %v", s.Name(), err)
		return
	}

	log.Printf("[TelegramSongbird] %s: Chirped to chat %s", s.Name(), chirp.ChatID)
}

// sendMessage sends a message to Telegram via the Bot API.
func (s *TelegramSongbird) sendMessage(ctx context.Context, chatID, text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.config.BotToken)

	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}

// SendMessage exposes the sendMessage functionality for direct use.
func (s *TelegramSongbird) SendMessage(ctx context.Context, chatID, text string) error {
	return s.sendMessage(ctx, chatID, text)
}
