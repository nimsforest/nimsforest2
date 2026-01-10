package sources

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/yourusername/nimsforest/internal/core"
)

// TelegramSourceConfig configures a Telegram webhook source.
type TelegramSourceConfig struct {
	Name       string // Source name (unique identifier)
	Path       string // HTTP endpoint path (e.g., "/webhooks/telegram")
	Publishes  string // River subject to publish to
	BotToken   string // Bot token (for URL verification and API calls)
	SecretPath string // Secret path token in webhook URL (optional extra security)
}

// TelegramSource receives Telegram webhook updates and flows them into River.
type TelegramSource struct {
	*core.BaseSource
	config TelegramSourceConfig

	mu      sync.Mutex
	running bool
}

// TelegramUpdate represents a Telegram webhook update.
type TelegramUpdate struct {
	UpdateID int64            `json:"update_id"`
	Message  *TelegramMessage `json:"message,omitempty"`
}

// TelegramMessage represents a Telegram message.
type TelegramMessage struct {
	MessageID int64            `json:"message_id"`
	From      *TelegramUser    `json:"from,omitempty"`
	Chat      *TelegramChat    `json:"chat"`
	Date      int64            `json:"date"`
	Text      string           `json:"text,omitempty"`
	ReplyTo   *TelegramMessage `json:"reply_to_message,omitempty"`
	Entities  []TelegramEntity `json:"entities,omitempty"`
}

// TelegramUser represents a Telegram user.
type TelegramUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// TelegramChat represents a Telegram chat.
type TelegramChat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"` // "private", "group", "supergroup", "channel"
	Title string `json:"title,omitempty"`
}

// TelegramEntity represents a message entity (mentions, commands, etc.)
type TelegramEntity struct {
	Type   string `json:"type"` // "mention", "bot_command", etc.
	Offset int    `json:"offset"`
	Length int    `json:"length"`
}

// TelegramPayload wraps the raw Telegram update for River flow.
type TelegramPayload struct {
	Body      json.RawMessage `json:"body"`
	Timestamp time.Time       `json:"timestamp"`
	Source    string          `json:"source"`
}

// NewTelegramSource creates a new Telegram webhook source.
func NewTelegramSource(cfg TelegramSourceConfig, river *core.River) *TelegramSource {
	return &TelegramSource{
		BaseSource: core.NewBaseSource(cfg.Name, river, cfg.Publishes),
		config:     cfg,
	}
}

// Type returns the source type.
func (s *TelegramSource) Type() string {
	return "telegram"
}

// Config returns the source configuration.
func (s *TelegramSource) Config() TelegramSourceConfig {
	return s.config
}

// Path returns the HTTP endpoint path.
func (s *TelegramSource) Path() string {
	return s.config.Path
}

// BotToken returns the bot token (for use by TelegramSongbird).
func (s *TelegramSource) BotToken() string {
	return s.config.BotToken
}

// Start starts the Telegram source.
// Note: The actual HTTP server is managed by WebhookServer.
func (s *TelegramSource) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	s.running = true
	s.SetRunning(true)
	log.Printf("[TelegramSource] Started: %s (path: %s, publishes: %s)",
		s.Name(), s.config.Path, s.Publishes())
	return nil
}

// Stop stops the Telegram source.
func (s *TelegramSource) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false
	s.SetRunning(false)
	log.Printf("[TelegramSource] Stopped: %s", s.Name())
	return nil
}

// Handler returns the HTTP handler for this Telegram webhook endpoint.
func (s *TelegramSource) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST requests
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check if source is running
		s.mu.Lock()
		running := s.running
		s.mu.Unlock()

		if !running {
			http.Error(w, "Source not running", http.StatusServiceUnavailable)
			return
		}

		// Read body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("[TelegramSource] %s: Failed to read body: %v", s.Name(), err)
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Parse to validate it's a Telegram update
		var update TelegramUpdate
		if err := json.Unmarshal(body, &update); err != nil {
			log.Printf("[TelegramSource] %s: Invalid Telegram update: %v", s.Name(), err)
			http.Error(w, "Invalid update", http.StatusBadRequest)
			return
		}

		// Only process messages (skip edited_message, channel_post, etc. for MVP)
		if update.Message == nil {
			// Acknowledge non-message updates silently
			w.WriteHeader(http.StatusOK)
			return
		}

		// Build payload
		payload := TelegramPayload{
			Body:      body,
			Timestamp: time.Now(),
			Source:    s.Name(),
		}

		payloadData, err := json.Marshal(payload)
		if err != nil {
			log.Printf("[TelegramSource] %s: Failed to marshal payload: %v", s.Name(), err)
			http.Error(w, "Failed to process payload", http.StatusInternalServerError)
			return
		}

		// Flow to River
		if err := s.Flow(payloadData); err != nil {
			log.Printf("[TelegramSource] %s: Failed to flow data: %v", s.Name(), err)
			http.Error(w, "Failed to process webhook", http.StatusInternalServerError)
			return
		}

		log.Printf("[TelegramSource] %s: Received message from chat %d, flowed to %s",
			s.Name(), update.Message.Chat.ID, s.Publishes())

		// Return success (Telegram expects 200 OK)
		w.WriteHeader(http.StatusOK)
	}
}
