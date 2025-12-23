package core

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// RiverData represents unstructured external data flowing into the system.
// This is raw data from external sources like webhooks, APIs, etc.
type RiverData struct {
	Subject   string    `json:"subject"`   // Raw source identifier (e.g., "stripe.webhook")
	Data      []byte    `json:"data"`      // Unstructured payload
	Timestamp time.Time `json:"ts"`        // When this data entered the river
}

// River represents a JetStream stream for external unstructured data.
// Trees watch the river and parse/structure the data into leaves.
type River struct {
	js     nats.JetStreamContext
	stream string
}

// NewRiver creates a new River backed by a JetStream stream.
// The stream is created if it doesn't exist, with the name "RIVER"
// and subjects matching "river.>".
func NewRiver(js nats.JetStreamContext) (*River, error) {
	streamName := "RIVER"

	// Create or update the stream
	streamInfo, err := js.StreamInfo(streamName)
	if err != nil {
		// Stream doesn't exist, create it
		_, err = js.AddStream(&nats.StreamConfig{
			Name:        streamName,
			Subjects:    []string{"river.>"},
			Storage:     nats.FileStorage,
			Retention:   nats.WorkQueuePolicy,
			MaxAge:      24 * time.Hour,     // Keep data for 24 hours
			Discard:     nats.DiscardOld,
			MaxMsgs:     100000,              // Maximum messages in stream
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create stream %s: %w", streamName, err)
		}
		log.Printf("[River] Created stream: %s", streamName)
	} else {
		log.Printf("[River] Using existing stream: %s (msgs: %d)", streamName, streamInfo.State.Msgs)
	}

	return &River{
		js:     js,
		stream: streamName,
	}, nil
}

// Flow adds unstructured external data to the river.
// The subject should identify the source (e.g., "river.stripe.webhook").
func (r *River) Flow(subject string, data []byte) error {
	if subject == "" {
		return fmt.Errorf("subject cannot be empty")
	}
	if len(data) == 0 {
		return fmt.Errorf("data cannot be empty")
	}

	// Ensure subject starts with "river."
	if len(subject) < 6 || subject[:6] != "river." {
		subject = "river." + subject
	}

	riverData := RiverData{
		Subject:   subject,
		Data:      data,
		Timestamp: time.Now(),
	}

	payload, err := json.Marshal(riverData)
	if err != nil {
		return fmt.Errorf("failed to marshal river data: %w", err)
	}

	// Publish to the stream
	_, err = r.js.Publish(subject, payload)
	if err != nil {
		return fmt.Errorf("failed to publish to river: %w", err)
	}

	log.Printf("[River] Data flowed: subject=%s, size=%d bytes", subject, len(data))
	return nil
}

// Observe watches for river data matching the given pattern.
// The pattern can include wildcards (* and >).
// The handler is called for each matching message.
func (r *River) Observe(pattern string, handler func(data RiverData)) error {
	if pattern == "" {
		pattern = "river.>"
	}

	// Ensure pattern starts with "river."
	if len(pattern) < 6 || pattern[:6] != "river." {
		pattern = "river." + pattern
	}

	// Create a durable consumer for this observer
	consumerName := fmt.Sprintf("observer-%d", time.Now().UnixNano())

	// Create consumer config
	consumerConfig := &nats.ConsumerConfig{
		Durable:       consumerName,
		FilterSubject: pattern,
		AckPolicy:     nats.AckExplicitPolicy,
		DeliverPolicy: nats.DeliverAllPolicy,
	}

	// Subscribe to the stream
	_, err := r.js.Subscribe(pattern, func(msg *nats.Msg) {
		// Deserialize the river data
		var data RiverData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Printf("[River] Failed to unmarshal data from subject %s: %v", msg.Subject, err)
			msg.Nak()
			return
		}

		// Call the handler
		handler(data)

		// Acknowledge the message
		msg.Ack()
	}, nats.Durable(consumerConfig.Durable), nats.ManualAck())

	if err != nil {
		return fmt.Errorf("failed to observe river pattern %s: %w", pattern, err)
	}

	log.Printf("[River] Observing pattern: %s", pattern)
	return nil
}

// ObserveWithConsumer is like Observe but allows specifying a custom consumer name.
// This enables multiple observers to share work (using the same name) or
// maintain separate positions in the stream (using different names).
func (r *River) ObserveWithConsumer(pattern, consumerName string, handler func(data RiverData)) error {
	if pattern == "" {
		pattern = "river.>"
	}
	if consumerName == "" {
		return fmt.Errorf("consumer name cannot be empty")
	}

	// Ensure pattern starts with "river."
	if len(pattern) < 6 || pattern[:6] != "river." {
		pattern = "river." + pattern
	}

	_, err := r.js.Subscribe(pattern, func(msg *nats.Msg) {
		var data RiverData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Printf("[River] Failed to unmarshal data: %v", err)
			msg.Nak()
			return
		}

		handler(data)
		msg.Ack()
	}, nats.Durable(consumerName), nats.ManualAck())

	if err != nil {
		return fmt.Errorf("failed to observe with consumer %s: %w", consumerName, err)
	}

	log.Printf("[River] Observing pattern %s with consumer %s", pattern, consumerName)
	return nil
}

// StreamInfo returns information about the river stream.
func (r *River) StreamInfo() (*nats.StreamInfo, error) {
	info, err := r.js.StreamInfo(r.stream)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}
	return info, nil
}
