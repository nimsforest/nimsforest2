package core

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
)

// Wind wraps NATS Core pub/sub functionality for carrying leaves.
// Wind is ephemeral - messages are not persisted and are delivered
// to active subscribers only.
type Wind struct {
	nc *nats.Conn
}

// NewWind creates a new Wind instance wrapping a NATS connection.
// The Wind provides a higher-level abstraction for leaf pub/sub operations.
func NewWind(nc *nats.Conn) *Wind {
	return &Wind{
		nc: nc,
	}
}

// Drop sends a leaf onto the wind (publishes to NATS).
// The leaf is serialized to JSON and published to its subject.
// Returns an error if the leaf is invalid or publishing fails.
func (w *Wind) Drop(leaf Leaf) error {
	// Validate the leaf before sending
	if err := leaf.Validate(); err != nil {
		return fmt.Errorf("invalid leaf: %w", err)
	}

	// Serialize leaf to JSON
	data, err := json.Marshal(leaf)
	if err != nil {
		return fmt.Errorf("failed to marshal leaf: %w", err)
	}

	// Publish to NATS
	if err := w.nc.Publish(leaf.Subject, data); err != nil {
		return fmt.Errorf("failed to publish leaf to subject %s: %w", leaf.Subject, err)
	}

	log.Printf("[Wind] Dropped leaf: subject=%s, source=%s", leaf.Subject, leaf.Source)
	return nil
}

// Catch listens for leaves matching a subject pattern and calls the handler
// for each received leaf. The subject pattern can include wildcards:
//   - "*" matches a single token (e.g., "payment.*" matches "payment.completed")
//   - ">" matches one or more tokens (e.g., "payment.>" matches "payment.completed.usd")
//
// Returns a subscription that can be used to unsubscribe later.
// Returns an error if subscription fails.
func (w *Wind) Catch(subject string, handler func(leaf Leaf)) (*nats.Subscription, error) {
	// Subscribe to the subject pattern
	sub, err := w.nc.Subscribe(subject, func(msg *nats.Msg) {
		// Deserialize the message data into a Leaf
		var leaf Leaf
		if err := json.Unmarshal(msg.Data, &leaf); err != nil {
			log.Printf("[Wind] Failed to unmarshal leaf from subject %s: %v", msg.Subject, err)
			return
		}

		// Validate the received leaf
		if err := leaf.Validate(); err != nil {
			log.Printf("[Wind] Received invalid leaf from subject %s: %v", msg.Subject, err)
			return
		}

		// Call the handler
		handler(leaf)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to subject %s: %w", subject, err)
	}

	log.Printf("[Wind] Catching leaves on subject: %s", subject)
	return sub, nil
}

// CatchWithQueue listens for leaves on a subject using a queue group.
// This enables load balancing across multiple subscribers - only one member
// of the queue group will receive each message.
//
// This is useful for scaling nims horizontally while ensuring each leaf
// is only processed once.
func (w *Wind) CatchWithQueue(subject, queue string, handler func(leaf Leaf)) (*nats.Subscription, error) {
	sub, err := w.nc.QueueSubscribe(subject, queue, func(msg *nats.Msg) {
		var leaf Leaf
		if err := json.Unmarshal(msg.Data, &leaf); err != nil {
			log.Printf("[Wind] Failed to unmarshal leaf from subject %s: %v", msg.Subject, err)
			return
		}

		if err := leaf.Validate(); err != nil {
			log.Printf("[Wind] Received invalid leaf from subject %s: %v", msg.Subject, err)
			return
		}

		handler(leaf)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to queue subscribe to subject %s with queue %s: %w", subject, queue, err)
	}

	log.Printf("[Wind] Catching leaves on subject %s with queue %s", subject, queue)
	return sub, nil
}

// Close closes the underlying NATS connection.
// This should be called when the Wind is no longer needed.
func (w *Wind) Close() {
	if w.nc != nil {
		w.nc.Close()
	}
}
