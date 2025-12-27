package nims

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/yourusername/nimsforest/internal/core"
)

// GeneralNim is a template/example nim that demonstrates how to create
// custom nims for your domain. It catches general leaf patterns and
// implements business logic in response to events.
//
// TO EXTEND: Create your own nim by:
// 1. Copy this file and rename (e.g., inventory_nim.go, billing_nim.go)
// 2. Change Subjects() to match the leaves you care about
// 3. Implement your business logic in Handle()
// 4. Use Compost() to persist state changes
// 5. Emit new leaves to trigger downstream processes
type GeneralNim struct {
	*core.BaseNim
	ctx    context.Context
	cancel context.CancelFunc
}

// NewGeneralNim creates a new general nim that catches various leaf types.
// This nim demonstrates the nim interface and can be used as a template.
func NewGeneralNim(wind *core.Wind, humus *core.Humus, soil *core.Soil) *GeneralNim {
	baseNim := core.NewBaseNim("general-nim", wind, humus, soil)
	return &GeneralNim{
		BaseNim: baseNim,
	}
}

// Subjects returns the leaf subjects this nim catches.
// The ">" wildcard means it catches ALL leaf subjects.
//
// TO CUSTOMIZE: Replace with specific subjects for your use case:
//   - []string{"payment.completed", "payment.failed"} for payment processing
//   - []string{"user.created", "user.updated"} for user management
//   - []string{"order.placed", "order.shipped"} for order fulfillment
//   - []string{"sensor.reading"} for IoT data processing
//   - []string{"notification.>"} for all notification types
func (n *GeneralNim) Subjects() []string {
	// Catch specific patterns to demonstrate selectivity
	return []string{
		"data.received",
		"status.update",
		"notification.required",
		"general.>", // Also catch any general events
	}
}

// Start begins listening for leaves matching this nim's subjects.
func (n *GeneralNim) Start(ctx context.Context) error {
	n.ctx, n.cancel = context.WithCancel(ctx)
	
	log.Printf("[GeneralNim] 🧚 Starting general nim")
	log.Printf("[GeneralNim] 💡 TIP: Create your own nim by copying this file!")
	log.Printf("[GeneralNim]     Catching: %v", n.Subjects())
	
	// Register handlers for each subject
	for _, subject := range n.Subjects() {
		if err := n.Catch(subject, func(leaf core.Leaf) {
			if err := n.Handle(n.ctx, leaf); err != nil {
				log.Printf("[GeneralNim] Error handling %s: %v", leaf.Subject, err)
			}
		}); err != nil {
			return fmt.Errorf("failed to catch %s: %w", subject, err)
		}
	}
	
	log.Printf("[GeneralNim] Started listening for general events")
	return nil
}

// Handle processes caught leaves and implements business logic.
// This is the heart of your nim - where decisions happen!
//
// TO CUSTOMIZE:
// 1. Switch on leaf.Subject to handle different event types
// 2. Parse leaf.Data into your domain types
// 3. Make business decisions based on the data
// 4. Read/write state via Soil if needed
// 5. Emit new leaves to trigger downstream processes
// 6. Compost state changes to Humus for persistence
func (n *GeneralNim) Handle(ctx context.Context, leaf core.Leaf) error {
	log.Printf("[GeneralNim] 🍃 Caught leaf: %s from %s", leaf.Subject, leaf.Source)

	// Route to specific handlers based on the leaf subject
	switch leaf.Subject {
	case "data.received":
		return n.handleDataReceived(ctx, leaf)
	case "status.update":
		return n.handleStatusUpdate(ctx, leaf)
	case "notification.required":
		return n.handleNotification(ctx, leaf)
	default:
		// Handle any other general events
		if len(leaf.Subject) > 8 && leaf.Subject[:8] == "general." {
			return n.handleGeneralEvent(ctx, leaf)
		}
		log.Printf("[GeneralNim] ⚠️  Unhandled subject: %s", leaf.Subject)
	}

	return nil
}

// handleDataReceived demonstrates processing a data.received event
func (n *GeneralNim) handleDataReceived(ctx context.Context, leaf core.Leaf) error {
	var event map[string]interface{}
	if err := json.Unmarshal(leaf.Data, &event); err != nil {
		return fmt.Errorf("failed to parse data.received: %w", err)
	}

	source := event["source"]
	dataValue := event["data"]
	log.Printf("[GeneralNim] 📦 Processing data from: %v, value: %v", source, dataValue)

	// BUSINESS LOGIC EXAMPLE: Create a record of this data
	recordID := fmt.Sprintf("data-record-%d", time.Now().Unix())
	record := map[string]interface{}{
		"id":         recordID,
		"source":     source,
		"data":       dataValue,
		"received_at": time.Now().Format(time.RFC3339),
		"processed":  true,
	}

	recordData, _ := json.Marshal(record)

	// Persist the record to Humus (which will be applied to Soil by decomposer)
	slot, err := n.Compost(recordID, "create", recordData)
	if err != nil {
		return fmt.Errorf("failed to compost record: %w", err)
	}
	log.Printf("[GeneralNim] 💾 Created data record: %s (slot: %d)", recordID, slot)

	// Emit a confirmation leaf
	confirmData := map[string]interface{}{
		"record_id": recordID,
		"status":    "processed",
	}
	confirmJSON, _ := json.Marshal(confirmData)
	if err := n.Leaf("data.processed", confirmJSON); err != nil {
		log.Printf("[GeneralNim] ⚠️  Failed to emit confirmation: %v", err)
	} else {
		log.Printf("[GeneralNim] 🍃 Emitted: data.processed")
	}

	return nil
}

// handleStatusUpdate demonstrates processing a status.update event
func (n *GeneralNim) handleStatusUpdate(ctx context.Context, leaf core.Leaf) error {
	var event map[string]interface{}
	if err := json.Unmarshal(leaf.Data, &event); err != nil {
		return fmt.Errorf("failed to parse status.update: %w", err)
	}

	entityID := event["entity_id"]
	status := event["status"]
	message := event["message"]
	log.Printf("[GeneralNim] 📊 Status update for %v: %v - %v", entityID, status, message)

	// BUSINESS LOGIC EXAMPLE: Update entity status in soil
	if entityID != nil {
		entityKey := fmt.Sprintf("entity-%v", entityID)
		
		// Try to read existing entity
		existingData, revision, err := n.Dig(entityKey)
		var entity map[string]interface{}
		
		if err == nil {
			// Entity exists, update it
			json.Unmarshal(existingData, &entity)
			entity["status"] = status
			entity["last_update"] = time.Now().Format(time.RFC3339)
			entity["message"] = message
		} else {
			// Entity doesn't exist, create it
			entity = map[string]interface{}{
				"id":          entityID,
				"status":      status,
				"created_at":  time.Now().Format(time.RFC3339),
				"last_update": time.Now().Format(time.RFC3339),
				"message":     message,
			}
		}

		entityData, _ := json.Marshal(entity)
		
		// Compost the change
		action := "update"
		if revision == 0 {
			action = "create"
		}
		
		slot, err := n.Compost(entityKey, action, entityData)
		if err != nil {
			return fmt.Errorf("failed to compost status update: %w", err)
		}
		log.Printf("[GeneralNim] 💾 Updated entity %s status to: %v (slot: %d)", entityKey, status, slot)
	}

	return nil
}

// handleNotification demonstrates processing a notification.required event
func (n *GeneralNim) handleNotification(ctx context.Context, leaf core.Leaf) error {
	var event map[string]interface{}
	if err := json.Unmarshal(leaf.Data, &event); err != nil {
		return fmt.Errorf("failed to parse notification: %w", err)
	}

	priority := event["priority"]
	message := event["message"]
	recipient := event["recipient"]
	
	log.Printf("[GeneralNim] 📧 Notification needed: [%v] to %v: %v", priority, recipient, message)

	// BUSINESS LOGIC EXAMPLE: Route based on priority
	if priority == "high" || priority == "urgent" {
		// High priority - send immediately
		log.Printf("[GeneralNim] 🚨 HIGH PRIORITY - sending immediately!")
		
		// Emit a leaf for immediate sending
		if err := n.Leaf("notification.send.immediate", leaf.Data); err != nil {
			log.Printf("[GeneralNim] ⚠️  Failed to emit immediate send: %v", err)
		}
	} else {
		// Normal priority - queue for batch sending
		log.Printf("[GeneralNim] 📝 Normal priority - queuing for batch")
		
		// Create a queued notification record
		notifID := fmt.Sprintf("notification-%d", time.Now().UnixNano())
		queuedNotif := map[string]interface{}{
			"id":         notifID,
			"priority":   priority,
			"message":    message,
			"recipient":  recipient,
			"queued_at":  time.Now().Format(time.RFC3339),
			"sent":       false,
		}
		
		notifData, _ := json.Marshal(queuedNotif)
		slot, err := n.Compost(notifID, "create", notifData)
		if err != nil {
			return fmt.Errorf("failed to compost notification: %w", err)
		}
		log.Printf("[GeneralNim] 💾 Queued notification: %s (slot: %d)", notifID, slot)
	}

	return nil
}

// handleGeneralEvent demonstrates handling catch-all events
func (n *GeneralNim) handleGeneralEvent(ctx context.Context, leaf core.Leaf) error {
	log.Printf("[GeneralNim] 📋 Processing general event: %s", leaf.Subject)
	
	var event map[string]interface{}
	if err := json.Unmarshal(leaf.Data, &event); err != nil {
		log.Printf("[GeneralNim] ⚠️  Could not parse event data")
		return nil
	}

	// Log the event details
	if eventType, ok := event["event_type"]; ok {
		log.Printf("[GeneralNim] 💡 Event type: %v", eventType)
		log.Printf("[GeneralNim] 💡 TIP: Add a specific handler for '%v' events!", eventType)
	}

	// Could archive general events for analytics
	eventID := fmt.Sprintf("event-%d", time.Now().UnixNano())
	archiveData := map[string]interface{}{
		"id":         eventID,
		"subject":    leaf.Subject,
		"source":     leaf.Source,
		"timestamp":  time.Now().Format(time.RFC3339),
		"data":       event,
	}
	
	archiveJSON, _ := json.Marshal(archiveData)
	slot, err := n.Compost(eventID, "create", archiveJSON)
	if err != nil {
		log.Printf("[GeneralNim] ⚠️  Failed to archive event: %v", err)
	} else {
		log.Printf("[GeneralNim] 💾 Archived general event: %s (slot: %d)", eventID, slot)
	}

	return nil
}

// Stop stops the nim from processing leaves
func (n *GeneralNim) Stop() error {
	if n.cancel != nil {
		n.cancel()
	}
	log.Printf("[GeneralNim] Stopped")
	return nil
}

/*
CUSTOMIZATION GUIDE:

Want to process inventory updates? Create inventory_nim.go:
  - Subjects(): return []string{"payment.completed", "order.shipped"}
  - Decrement stock levels when payments complete
  - Check reorder thresholds
  - Emit "inventory.low" leaves when stock is low

Want to process billing? Create billing_nim.go:
  - Subjects(): return []string{"subscription.created", "subscription.renewed"}
  - Calculate charges based on usage
  - Create invoice records
  - Emit "invoice.created" leaves

Want to process user lifecycle? Create user_nim.go:
  - Subjects(): return []string{"user.created", "user.activated"}
  - Send welcome emails
  - Set up initial data
  - Emit "onboarding.started" leaves

Want to aggregate analytics? Create analytics_nim.go:
  - Subjects(): return []string{"*.completed", "*.failed"}
  - Count successes and failures
  - Calculate metrics
  - Store aggregated data in Soil

The power of nims is they contain your business logic in isolated,
testable units that react to events. Each nim can:
- Read state from Soil
- Make decisions based on data
- Persist changes via Humus
- Emit new leaves to trigger other processes

Chain multiple nims together to create complex workflows!
*/
