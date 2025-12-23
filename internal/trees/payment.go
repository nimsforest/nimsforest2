package trees

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/yourusername/nimsforest/internal/core"
	"github.com/yourusername/nimsforest/internal/leaves"
)

// StripeWebhook represents a Stripe webhook payload.
// This is a simplified version - real Stripe webhooks have more fields.
type StripeWebhook struct {
	Type string          `json:"type"`
	Data StripeEventData `json:"data"`
}

// StripeEventData contains the event data from Stripe.
type StripeEventData struct {
	Object StripeCharge `json:"object"`
}

// StripeCharge represents a Stripe charge object.
type StripeCharge struct {
	ID         string  `json:"id"`
	Amount     int64   `json:"amount"`      // Amount in cents
	Currency   string  `json:"currency"`
	Customer   string  `json:"customer"`
	Status     string  `json:"status"`
	FailureMsg string  `json:"failure_message,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// PaymentTree parses payment provider webhooks and emits structured payment leaves.
// This example handles Stripe webhooks, but the pattern can be extended to any payment provider.
type PaymentTree struct {
	*core.BaseTree
	ctx    context.Context
	cancel context.CancelFunc
}

// NewPaymentTree creates a new PaymentTree that watches for payment webhooks.
func NewPaymentTree(wind *core.Wind, river *core.River) *PaymentTree {
	base := core.NewBaseTree("payment-tree", wind, river)
	return &PaymentTree{
		BaseTree: base,
	}
}

// Patterns returns the river patterns this tree watches.
// We're looking for Stripe webhook events.
func (t *PaymentTree) Patterns() []string {
	return []string{"river.stripe.webhook"}
}

// Parse attempts to parse river data as a payment webhook.
// Returns nil if the data doesn't match payment patterns.
func (t *PaymentTree) Parse(subject string, data []byte) *core.Leaf {
	// Try to parse as Stripe webhook
	var webhook StripeWebhook
	if err := json.Unmarshal(data, &webhook); err != nil {
		log.Printf("[PaymentTree] Failed to parse as Stripe webhook: %v", err)
		return nil
	}

	// Handle different event types
	switch webhook.Type {
	case "charge.succeeded":
		return t.parseChargeSucceeded(webhook)
	case "charge.failed":
		return t.parseChargeFailed(webhook)
	default:
		log.Printf("[PaymentTree] Ignoring unhandled event type: %s", webhook.Type)
		return nil
	}
}

// parseChargeSucceeded creates a PaymentCompleted leaf from a successful charge.
func (t *PaymentTree) parseChargeSucceeded(webhook StripeWebhook) *core.Leaf {
	charge := webhook.Data.Object

	// Extract item ID from metadata if available
	itemID := charge.Metadata["item_id"]
	if itemID == "" {
		itemID = "unknown"
	}

	// Convert amount from cents to dollars
	amount := float64(charge.Amount) / 100.0

	paymentData := leaves.PaymentCompleted{
		CustomerID: charge.Customer,
		Amount:     amount,
		Currency:   charge.Currency,
		ItemID:     itemID,
	}

	data, err := json.Marshal(paymentData)
	if err != nil {
		log.Printf("[PaymentTree] Failed to marshal payment data: %v", err)
		return nil
	}

	leaf := core.NewLeaf("payment.completed", data, t.Name())
	log.Printf("[PaymentTree] Parsed successful payment: customer=%s, amount=%.2f %s",
		charge.Customer, amount, charge.Currency)
	
	return leaf
}

// parseChargeFailed creates a PaymentFailed leaf from a failed charge.
func (t *PaymentTree) parseChargeFailed(webhook StripeWebhook) *core.Leaf {
	charge := webhook.Data.Object

	itemID := charge.Metadata["item_id"]
	if itemID == "" {
		itemID = "unknown"
	}

	amount := float64(charge.Amount) / 100.0

	paymentData := leaves.PaymentFailed{
		CustomerID: charge.Customer,
		Amount:     amount,
		Currency:   charge.Currency,
		ItemID:     itemID,
		Reason:     charge.FailureMsg,
	}

	data, err := json.Marshal(paymentData)
	if err != nil {
		log.Printf("[PaymentTree] Failed to marshal payment data: %v", err)
		return nil
	}

	leaf := core.NewLeaf("payment.failed", data, t.Name())
	log.Printf("[PaymentTree] Parsed failed payment: customer=%s, reason=%s",
		charge.Customer, charge.FailureMsg)
	
	return leaf
}

// Start begins watching the river for payment webhooks.
func (t *PaymentTree) Start(ctx context.Context) error {
	t.ctx, t.cancel = context.WithCancel(ctx)

	// Watch for Stripe webhooks
	err := t.Watch("river.stripe.webhook", func(data core.RiverData) {
		// Parse the webhook
		leaf := t.Parse(data.Subject, data.Data)
		if leaf == nil {
			return
		}

		// Drop the structured leaf onto the wind
		if err := t.Drop(*leaf); err != nil {
			log.Printf("[PaymentTree] Failed to drop leaf: %v", err)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to start payment tree: %w", err)
	}

	log.Printf("[PaymentTree] Started watching for payment webhooks")
	return nil
}

// Stop gracefully shuts down the payment tree.
func (t *PaymentTree) Stop() error {
	if t.cancel != nil {
		t.cancel()
	}
	log.Printf("[PaymentTree] Stopped")
	return nil
}
