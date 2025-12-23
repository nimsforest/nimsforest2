package core

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// Compost represents a state change entry in the humus.
// It records what entity changed, what action was performed, and the new data.
type Compost struct {
	Entity    string          `json:"entity"`    // Entity identifier (e.g., "tasks/followup-123")
	Action    string          `json:"action"`    // Action: create, update, delete
	Data      json.RawMessage `json:"data"`      // New state data
	NimName   string          `json:"nim"`       // Nim that created this compost
	Timestamp time.Time       `json:"ts"`        // When this compost was created
	Slot      uint64          `json:"slot"`      // Sequence number in the stream
}

// Humus represents a JetStream stream for persistent state changes.
// State changes flow from nims into humus, and the decomposer
// applies them to soil (the KV store).
type Humus struct {
	js     nats.JetStreamContext
	stream string
}

// NewHumus creates a new Humus backed by a JetStream stream.
// The stream is created if it doesn't exist, with the name "HUMUS".
func NewHumus(js nats.JetStreamContext) (*Humus, error) {
	streamName := "HUMUS"

	// Create or update the stream
	streamInfo, err := js.StreamInfo(streamName)
	if err != nil {
		// Stream doesn't exist, create it
		_, err = js.AddStream(&nats.StreamConfig{
			Name:        streamName,
			Subjects:    []string{"humus.>"},
			Storage:     nats.FileStorage,
			Retention:   nats.LimitsPolicy,  // Keep all messages up to limits
			MaxAge:      7 * 24 * time.Hour, // Keep for 7 days
			Discard:     nats.DiscardOld,
			MaxMsgs:     1000000,             // Maximum messages in stream
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create stream %s: %w", streamName, err)
		}
		log.Printf("[Humus] Created stream: %s", streamName)
	} else {
		log.Printf("[Humus] Using existing stream: %s (msgs: %d)", streamName, streamInfo.State.Msgs)
	}

	return &Humus{
		js:     js,
		stream: streamName,
	}, nil
}

// Add composts a state change into humus.
// Returns the sequence number (slot) assigned to this compost.
func (h *Humus) Add(nimName, entity, action string, data []byte) (uint64, error) {
	if nimName == "" {
		return 0, fmt.Errorf("nim name cannot be empty")
	}
	if entity == "" {
		return 0, fmt.Errorf("entity cannot be empty")
	}
	if action == "" {
		return 0, fmt.Errorf("action cannot be empty")
	}
	if action != "create" && action != "update" && action != "delete" {
		return 0, fmt.Errorf("invalid action: %s (must be create, update, or delete)", action)
	}
	// Data can be empty for delete operations
	if action != "delete" && len(data) == 0 {
		return 0, fmt.Errorf("data cannot be empty for %s action", action)
	}

	// Handle empty data for delete operations
	var rawData json.RawMessage
	if len(data) > 0 {
		rawData = json.RawMessage(data)
	} else {
		rawData = json.RawMessage([]byte("null"))
	}

	compost := Compost{
		Entity:    entity,
		Action:    action,
		Data:      rawData,
		NimName:   nimName,
		Timestamp: time.Now(),
		Slot:      0, // Will be set by the stream
	}

	payload, err := json.Marshal(compost)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal compost: %w", err)
	}

	// Publish to the stream
	subject := fmt.Sprintf("humus.%s.%s", nimName, action)
	ack, err := h.js.Publish(subject, payload)
	if err != nil {
		return 0, fmt.Errorf("failed to publish compost: %w", err)
	}

	log.Printf("[Humus] Composted: nim=%s, entity=%s, action=%s, slot=%d", 
		nimName, entity, action, ack.Sequence)
	return ack.Sequence, nil
}

// Decompose processes compost entries from humus.
// The handler is called for each compost entry in order.
// This is typically used by the decomposer worker to apply state changes to soil.
func (h *Humus) Decompose(handler func(compost Compost)) error {
	// Create a durable consumer
	consumerName := "decomposer"

	// Subscribe to all humus messages
	sub, err := h.js.Subscribe("humus.>", func(msg *nats.Msg) {
		// Deserialize the compost
		var compost Compost
		if err := json.Unmarshal(msg.Data, &compost); err != nil {
			log.Printf("[Humus] Failed to unmarshal compost: %v", err)
			msg.Nak()
			return
		}

		// Set the slot from the message metadata
		meta, err := msg.Metadata()
		if err == nil {
			compost.Slot = meta.Sequence.Stream
		}

		// Call the handler
		handler(compost)

		// Acknowledge the message
		msg.Ack()
	}, 
		nats.Durable(consumerName),
		nats.DeliverAll(),
		nats.AckExplicit(),
		nats.ManualAck(),
	)

	if err != nil {
		return fmt.Errorf("failed to decompose humus: %w", err)
	}

	log.Printf("[Humus] Decomposer started with consumer: %s", consumerName)
	_ = sub // Keep reference to prevent garbage collection
	return nil
}

// DecomposeWithConsumer is like Decompose but allows specifying a custom consumer name.
// This enables multiple decomposers with different consumption patterns.
func (h *Humus) DecomposeWithConsumer(consumerName string, handler func(compost Compost)) error {
	if consumerName == "" {
		return fmt.Errorf("consumer name cannot be empty")
	}

	sub, err := h.js.Subscribe("humus.>", func(msg *nats.Msg) {
		var compost Compost
		if err := json.Unmarshal(msg.Data, &compost); err != nil {
			log.Printf("[Humus] Failed to unmarshal compost: %v", err)
			msg.Nak()
			return
		}

		meta, err := msg.Metadata()
		if err == nil {
			compost.Slot = meta.Sequence.Stream
		}

		handler(compost)
		msg.Ack()
	},
		nats.Durable(consumerName),
		nats.DeliverAll(),
		nats.AckExplicit(),
		nats.ManualAck(),
	)

	if err != nil {
		return fmt.Errorf("failed to decompose with consumer %s: %w", consumerName, err)
	}

	log.Printf("[Humus] Decomposer started with consumer: %s", consumerName)
	_ = sub
	return nil
}

// StreamInfo returns information about the humus stream.
func (h *Humus) StreamInfo() (*nats.StreamInfo, error) {
	info, err := h.js.StreamInfo(h.stream)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}
	return info, nil
}
