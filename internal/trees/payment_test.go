package trees

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/yourusername/nimsforest/internal/core"
	"github.com/yourusername/nimsforest/internal/leaves"
)

func TestPaymentTree_Patterns(t *testing.T) {
	tree := NewPaymentTree(nil, nil)
	patterns := tree.Patterns()

	if len(patterns) != 1 {
		t.Errorf("Expected 1 pattern, got %d", len(patterns))
	}

	if patterns[0] != "river.stripe.webhook" {
		t.Errorf("Expected pattern 'river.stripe.webhook', got '%s'", patterns[0])
	}
}

func TestPaymentTree_ParseChargeSucceeded(t *testing.T) {
	tree := NewPaymentTree(nil, nil)

	// Create a mock Stripe webhook for charge.succeeded
	webhook := StripeWebhook{
		Type: "charge.succeeded",
		Data: StripeEventData{
			Object: StripeCharge{
				ID:       "ch_123456",
				Amount:   5000, // $50.00
				Currency: "usd",
				Customer: "cus_ABC123",
				Status:   "succeeded",
				Metadata: map[string]string{
					"item_id": "item_789",
				},
			},
		},
	}

	webhookData, err := json.Marshal(webhook)
	if err != nil {
		t.Fatalf("Failed to marshal webhook: %v", err)
	}

	// Parse the webhook
	leaf := tree.Parse("river.stripe.webhook", webhookData)
	if leaf == nil {
		t.Fatal("Expected leaf to be parsed, got nil")
	}

	// Verify leaf properties
	if leaf.Subject != "payment.completed" {
		t.Errorf("Expected subject 'payment.completed', got '%s'", leaf.Subject)
	}

	if leaf.Source != "payment-tree" {
		t.Errorf("Expected source 'payment-tree', got '%s'", leaf.Source)
	}

	// Verify payment data
	var payment leaves.PaymentCompleted
	if err := json.Unmarshal(leaf.Data, &payment); err != nil {
		t.Fatalf("Failed to unmarshal payment data: %v", err)
	}

	if payment.CustomerID != "cus_ABC123" {
		t.Errorf("Expected customer 'cus_ABC123', got '%s'", payment.CustomerID)
	}

	if payment.Amount != 50.0 {
		t.Errorf("Expected amount 50.0, got %.2f", payment.Amount)
	}

	if payment.Currency != "usd" {
		t.Errorf("Expected currency 'usd', got '%s'", payment.Currency)
	}

	if payment.ItemID != "item_789" {
		t.Errorf("Expected item_id 'item_789', got '%s'", payment.ItemID)
	}
}

func TestPaymentTree_ParseChargeFailed(t *testing.T) {
	tree := NewPaymentTree(nil, nil)

	webhook := StripeWebhook{
		Type: "charge.failed",
		Data: StripeEventData{
			Object: StripeCharge{
				ID:         "ch_123456",
				Amount:     5000,
				Currency:   "usd",
				Customer:   "cus_ABC123",
				Status:     "failed",
				FailureMsg: "insufficient_funds",
				Metadata: map[string]string{
					"item_id": "item_789",
				},
			},
		},
	}

	webhookData, err := json.Marshal(webhook)
	if err != nil {
		t.Fatalf("Failed to marshal webhook: %v", err)
	}

	leaf := tree.Parse("river.stripe.webhook", webhookData)
	if leaf == nil {
		t.Fatal("Expected leaf to be parsed, got nil")
	}

	if leaf.Subject != "payment.failed" {
		t.Errorf("Expected subject 'payment.failed', got '%s'", leaf.Subject)
	}

	var payment leaves.PaymentFailed
	if err := json.Unmarshal(leaf.Data, &payment); err != nil {
		t.Fatalf("Failed to unmarshal payment data: %v", err)
	}

	if payment.Reason != "insufficient_funds" {
		t.Errorf("Expected reason 'insufficient_funds', got '%s'", payment.Reason)
	}
}

func TestPaymentTree_ParseUnknownEventType(t *testing.T) {
	tree := NewPaymentTree(nil, nil)

	webhook := StripeWebhook{
		Type: "charge.updated",
		Data: StripeEventData{
			Object: StripeCharge{
				ID:       "ch_123456",
				Amount:   5000,
				Currency: "usd",
				Customer: "cus_ABC123",
			},
		},
	}

	webhookData, err := json.Marshal(webhook)
	if err != nil {
		t.Fatalf("Failed to marshal webhook: %v", err)
	}

	// Should return nil for unhandled event types
	leaf := tree.Parse("river.stripe.webhook", webhookData)
	if leaf != nil {
		t.Error("Expected nil for unhandled event type, got a leaf")
	}
}

func TestPaymentTree_ParseInvalidJSON(t *testing.T) {
	tree := NewPaymentTree(nil, nil)

	// Invalid JSON should return nil
	leaf := tree.Parse("river.stripe.webhook", []byte("invalid json"))
	if leaf != nil {
		t.Error("Expected nil for invalid JSON, got a leaf")
	}
}

func TestPaymentTree_ParseMissingItemID(t *testing.T) {
	tree := NewPaymentTree(nil, nil)

	// Webhook without item_id in metadata
	webhook := StripeWebhook{
		Type: "charge.succeeded",
		Data: StripeEventData{
			Object: StripeCharge{
				ID:       "ch_123456",
				Amount:   5000,
				Currency: "usd",
				Customer: "cus_ABC123",
				Status:   "succeeded",
				Metadata: map[string]string{},
			},
		},
	}

	webhookData, err := json.Marshal(webhook)
	if err != nil {
		t.Fatalf("Failed to marshal webhook: %v", err)
	}

	leaf := tree.Parse("river.stripe.webhook", webhookData)
	if leaf == nil {
		t.Fatal("Expected leaf to be parsed, got nil")
	}

	var payment leaves.PaymentCompleted
	if err := json.Unmarshal(leaf.Data, &payment); err != nil {
		t.Fatalf("Failed to unmarshal payment data: %v", err)
	}

	// Should default to "unknown"
	if payment.ItemID != "unknown" {
		t.Errorf("Expected item_id 'unknown', got '%s'", payment.ItemID)
	}
}

// Integration test with real NATS (requires NATS server running)
func TestPaymentTree_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	nc, js := core.SetupTestNATS(t)
	defer nc.Close()

	// Create wind
	wind := core.NewWind(nc)
	
	// Create a fresh river with unique stream name for this test
	// We'll use the default river which is fine since NewRiver reuses existing streams
	river, err := core.NewRiver(js)
	if err != nil {
		t.Fatalf("Failed to create river: %v", err)
	}

	// Create payment tree
	tree := NewPaymentTree(wind, river)

	// Subscribe to payment leaves
	receivedLeaves := make(chan core.Leaf, 1)
	_, err = wind.Catch("payment.>", func(leaf core.Leaf) {
		receivedLeaves <- leaf
	})
	if err != nil {
		t.Fatalf("Failed to catch payment leaves: %v", err)
	}

	// Instead of using the Watch method which creates a consumer,
	// we'll manually observe the river for this test
	riverReceived := make(chan core.RiverData, 1)
	err = river.ObserveWithConsumer("river.stripe.webhook", 
		fmt.Sprintf("payment-tree-test-%d", time.Now().UnixNano()), 
		func(data core.RiverData) {
			riverReceived <- data
			// Parse and drop the leaf
			leaf := tree.Parse(data.Subject, data.Data)
			if leaf != nil {
				tree.Drop(*leaf)
			}
		})
	if err != nil {
		t.Fatalf("Failed to observe river: %v", err)
	}

	// Give the subscription a moment to set up
	time.Sleep(100 * time.Millisecond)

	// Send a Stripe webhook to the river
	webhook := StripeWebhook{
		Type: "charge.succeeded",
		Data: StripeEventData{
			Object: StripeCharge{
				ID:       "ch_integration_test",
				Amount:   10000,
				Currency: "usd",
				Customer: "cus_test",
				Status:   "succeeded",
				Metadata: map[string]string{
					"item_id": "integration_item",
				},
			},
		},
	}

	webhookData, err := json.Marshal(webhook)
	if err != nil {
		t.Fatalf("Failed to marshal webhook: %v", err)
	}

	if err := river.Flow("stripe.webhook", webhookData); err != nil {
		t.Fatalf("Failed to flow webhook: %v", err)
	}

	// Wait for the leaf to be processed
	select {
	case leaf := <-receivedLeaves:
		if leaf.Subject != "payment.completed" {
			t.Errorf("Expected subject 'payment.completed', got '%s'", leaf.Subject)
		}

		var payment leaves.PaymentCompleted
		if err := json.Unmarshal(leaf.Data, &payment); err != nil {
			t.Fatalf("Failed to unmarshal payment: %v", err)
		}

		if payment.CustomerID != "cus_test" {
			t.Errorf("Expected customer 'cus_test', got '%s'", payment.CustomerID)
		}

		if payment.Amount != 100.0 {
			t.Errorf("Expected amount 100.0, got %.2f", payment.Amount)
		}

	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for payment leaf")
	}
}
