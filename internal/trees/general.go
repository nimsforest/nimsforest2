package trees

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/yourusername/nimsforest/internal/core"
)

// GeneralTree is a template/example tree that demonstrates how to create
// custom trees for your domain. It watches for general river patterns and
// emits leaves based on the data it receives.
//
// TO EXTEND: Create your own tree by:
// 1. Copy this file and rename (e.g., crm_tree.go, webhook_tree.go)
// 2. Change Patterns() to match your data sources (e.g., "crm.salesforce.>")
// 3. Implement your parsing logic in the observer callback
// 4. Emit appropriate leaves for your domain events
type GeneralTree struct {
	*core.BaseTree
	ctx    context.Context
	cancel context.CancelFunc
}

// NewGeneralTree creates a new general tree that watches for any river data.
// This tree demonstrates the tree interface and can be used as a template.
func NewGeneralTree(wind *core.Wind, river *core.River) *GeneralTree {
	baseTree := core.NewBaseTree("general-tree", wind, river)
	return &GeneralTree{
		BaseTree: baseTree,
	}
}

// Patterns returns the river subjects this tree watches.
// The ">" wildcard means it catches ALL river subjects.
//
// TO CUSTOMIZE: Replace with specific patterns for your use case:
//   - "river.api.>" for all API webhooks
//   - "river.crm.>" for CRM data
//   - "river.iot.sensors.>" for IoT sensor data
//   - "river.database.changes.>" for database change streams
func (t *GeneralTree) Patterns() []string {
	return []string{"river.general.>"}
}

// Start begins watching the river for data matching this tree's patterns.
// This implementation shows the basic structure of a tree's observation logic.
func (t *GeneralTree) Start(ctx context.Context) error {
	t.ctx, t.cancel = context.WithCancel(ctx)

	// Watch the river for data matching our patterns
	err := t.Watch("river.general.>", func(data core.RiverData) {
		t.parseGeneralData(data)
	})

	if err != nil {
		return fmt.Errorf("failed to start general tree: %w", err)
	}

	log.Printf("[GeneralTree] Started watching for general events")
	return nil
}

// parseGeneralData demonstrates how to parse incoming river data and emit leaves.
// This is where your domain-specific parsing logic goes.
//
// TO CUSTOMIZE:
// 1. Parse the data according to your source format (JSON, XML, CSV, etc.)
// 2. Extract relevant fields and create strongly-typed leaf events
// 3. Emit leaves that other nims can catch and process
func (t *GeneralTree) parseGeneralData(data core.RiverData) {
	log.Printf("[GeneralTree] 📥 Received data on %s (%d bytes)", data.Subject, len(data.Data))

	// Try to parse as generic JSON
	var payload map[string]interface{}
	if err := json.Unmarshal(data.Data, &payload); err != nil {
		log.Printf("[GeneralTree] ⚠️  Not valid JSON: %v", err)
		return
	}

	// Example: Inspect the payload and decide what leaves to emit
	eventType, hasType := payload["type"].(string)
	if !hasType {
		log.Printf("[GeneralTree] ⚠️  No 'type' field in payload")
		return
	}

	log.Printf("[GeneralTree] 📋 Event type: %s", eventType)

	// Emit different leaves based on the event type
	// This demonstrates how one river event can fan out to multiple leaves
	switch eventType {
	case "data.received":
		t.emitDataReceivedLeaf(payload)
	case "status.update":
		t.emitStatusUpdateLeaf(payload)
	case "notification":
		t.emitNotificationLeaf(payload)
	default:
		log.Printf("[GeneralTree] 💡 Unknown type '%s' - you can add handler for this!", eventType)
		// Still emit a generic leaf so nims can process it
		t.emitGenericLeaf(eventType, payload)
	}
}

// emitDataReceivedLeaf shows how to emit a specific leaf type
func (t *GeneralTree) emitDataReceivedLeaf(payload map[string]interface{}) {
	leafData := map[string]interface{}{
		"event_type": "data.received",
		"timestamp":  payload["timestamp"],
		"source":     payload["source"],
		"data":       payload["data"],
	}

	data, _ := json.Marshal(leafData)
	leaf := *core.NewLeaf("data.received", data, t.Name())

	if err := t.Drop(leaf); err != nil {
		log.Printf("[GeneralTree] ❌ Failed to drop leaf: %v", err)
	} else {
		log.Printf("[GeneralTree] 🍃 Emitted leaf: data.received")
	}
}

// emitStatusUpdateLeaf shows emitting status updates
func (t *GeneralTree) emitStatusUpdateLeaf(payload map[string]interface{}) {
	leafData := map[string]interface{}{
		"event_type": "status.update",
		"entity_id":  payload["entity_id"],
		"status":     payload["status"],
		"message":    payload["message"],
	}

	data, _ := json.Marshal(leafData)
	leaf := *core.NewLeaf("status.update", data, t.Name())

	if err := t.Drop(leaf); err != nil {
		log.Printf("[GeneralTree] ❌ Failed to drop leaf: %v", err)
	} else {
		log.Printf("[GeneralTree] 🍃 Emitted leaf: status.update (entity: %v)", payload["entity_id"])
	}
}

// emitNotificationLeaf shows emitting notifications
func (t *GeneralTree) emitNotificationLeaf(payload map[string]interface{}) {
	leafData := map[string]interface{}{
		"event_type": "notification",
		"priority":   payload["priority"],
		"message":    payload["message"],
		"recipient":  payload["recipient"],
	}

	data, _ := json.Marshal(leafData)
	leaf := *core.NewLeaf("notification.required", data, t.Name())

	if err := t.Drop(leaf); err != nil {
		log.Printf("[GeneralTree] ❌ Failed to drop leaf: %v", err)
	} else {
		priority := payload["priority"]
		log.Printf("[GeneralTree] 🍃 Emitted leaf: notification.required (priority: %v)", priority)
	}
}

// emitGenericLeaf shows how to handle unknown event types
func (t *GeneralTree) emitGenericLeaf(eventType string, payload map[string]interface{}) {
	leafData := map[string]interface{}{
		"event_type": eventType,
		"raw_data":   payload,
	}

	data, _ := json.Marshal(leafData)
	leaf := *core.NewLeaf("general.event", data, t.Name())

	if err := t.Drop(leaf); err != nil {
		log.Printf("[GeneralTree] ❌ Failed to drop leaf: %v", err)
	} else {
		log.Printf("[GeneralTree] 🍃 Emitted generic leaf: general.event (type: %s)", eventType)
	}
}

// Stop stops the tree from processing river data
func (t *GeneralTree) Stop() error {
	if t.cancel != nil {
		t.cancel()
	}
	log.Printf("[GeneralTree] Stopped")
	return nil
}

/*
EXAMPLE USAGE:

1. Send test data to the river:
   nats pub river.general.test '{"type":"data.received","source":"api","data":"hello"}'

2. The tree will parse it and emit a "data.received" leaf

3. Any nim catching "data.received" will process it

CUSTOMIZATION GUIDE:

Want to process Salesforce CRM data? Create crm_tree.go:
  - Patterns(): return []string{"river.crm.salesforce.>"}
  - Parse Salesforce-specific JSON format
  - Emit leaves like: "contact.created", "opportunity.updated"

Want to process IoT sensor data? Create iot_tree.go:
  - Patterns(): return []string{"river.iot.sensors.>"}
  - Parse sensor data format (temp, humidity, etc.)
  - Emit leaves like: "sensor.reading", "sensor.alert"

Want to process database changes? Create db_tree.go:
  - Patterns(): return []string{"river.database.>"}
  - Parse database change events
  - Emit leaves like: "user.created", "order.updated"

The power of trees is they stand at the edge of your system,
converting messy external data into clean, typed events (leaves)
that the rest of your forest can process!
*/
