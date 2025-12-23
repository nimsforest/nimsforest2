package nims

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/yourusername/nimsforest/internal/core"
	"github.com/yourusername/nimsforest/internal/leaves"
)

// Task represents a followup task stored in soil.
type Task struct {
	ID          string    `json:"id"`
	CustomerID  string    `json:"customer_id"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	DueDate     time.Time `json:"due_date"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AfterSalesNim handles post-payment business logic.
// It catches payment events and creates followup tasks for customer success teams.
type AfterSalesNim struct {
	*core.BaseNim
	ctx    context.Context
	cancel context.CancelFunc
}

// NewAfterSalesNim creates a new AfterSalesNim.
func NewAfterSalesNim(wind *core.Wind, humus *core.Humus, soil *core.Soil) *AfterSalesNim {
	base := core.NewBaseNim("aftersales-nim", wind, humus, soil)
	return &AfterSalesNim{
		BaseNim: base,
	}
}

// Subjects returns the leaf patterns this nim listens to.
func (n *AfterSalesNim) Subjects() []string {
	return []string{"payment.completed", "payment.failed"}
}

// Handle processes a caught leaf.
func (n *AfterSalesNim) Handle(ctx context.Context, leaf core.Leaf) error {
	switch leaf.Subject {
	case "payment.completed":
		return n.handlePaymentCompleted(ctx, leaf)
	case "payment.failed":
		return n.handlePaymentFailed(ctx, leaf)
	default:
		return fmt.Errorf("unexpected leaf subject: %s", leaf.Subject)
	}
}

// handlePaymentCompleted processes successful payment events.
// Creates a followup task and optionally sends a thank you email.
func (n *AfterSalesNim) handlePaymentCompleted(ctx context.Context, leaf core.Leaf) error {
	// Parse payment data
	var payment leaves.PaymentCompleted
	if err := json.Unmarshal(leaf.Data, &payment); err != nil {
		return fmt.Errorf("failed to unmarshal payment: %w", err)
	}

	log.Printf("[AfterSalesNim] Processing completed payment: customer=%s, amount=%.2f %s",
		payment.CustomerID, payment.Amount, payment.Currency)

	// Create a followup task via compost
	taskID := fmt.Sprintf("task-%s-%d", payment.CustomerID, time.Now().Unix())
	task := Task{
		ID:          taskID,
		CustomerID:  payment.CustomerID,
		Type:        "followup",
		Description: fmt.Sprintf("Follow up on purchase of %s (%.2f %s)", 
			payment.ItemID, payment.Amount, payment.Currency),
		Status:      "pending",
		DueDate:     time.Now().Add(24 * time.Hour), // Follow up in 24 hours
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Send task state change to humus (decomposer will apply to soil)
	_, err := n.CompostStruct(taskID, "create", task)
	if err != nil {
		return fmt.Errorf("failed to compost task: %w", err)
	}

	log.Printf("[AfterSalesNim] Created followup task: %s", taskID)

	// Emit a followup required leaf for other systems
	followup := leaves.FollowupRequired{
		CustomerID: payment.CustomerID,
		Reason:     "post_purchase_followup",
		DueDate:    task.DueDate,
	}

	if err := n.LeafStruct("followup.required", followup); err != nil {
		return fmt.Errorf("failed to emit followup leaf: %w", err)
	}

	// Optionally send thank you email (emit leaf for comms nim)
	if payment.Amount >= 100.0 {
		email := leaves.EmailSend{
			To:         payment.CustomerID, // In reality, would look up email address
			Subject:    "Thank you for your purchase!",
			Body:       fmt.Sprintf("We appreciate your purchase of %.2f %s", payment.Amount, payment.Currency),
			TemplateID: "thank_you_template",
		}

		if err := n.LeafStruct("email.send", email); err != nil {
			log.Printf("[AfterSalesNim] Failed to emit email leaf: %v", err)
			// Don't fail the whole operation if email fails
		} else {
			log.Printf("[AfterSalesNim] Emitted thank you email leaf for high-value purchase")
		}
	}

	return nil
}

// handlePaymentFailed processes failed payment events.
// Creates a high-priority task to reach out to the customer.
func (n *AfterSalesNim) handlePaymentFailed(ctx context.Context, leaf core.Leaf) error {
	var payment leaves.PaymentFailed
	if err := json.Unmarshal(leaf.Data, &payment); err != nil {
		return fmt.Errorf("failed to unmarshal payment: %w", err)
	}

	log.Printf("[AfterSalesNim] Processing failed payment: customer=%s, reason=%s",
		payment.CustomerID, payment.Reason)

	// Create urgent followup task
	taskID := fmt.Sprintf("task-%s-failed-%d", payment.CustomerID, time.Now().Unix())
	task := Task{
		ID:          taskID,
		CustomerID:  payment.CustomerID,
		Type:        "payment_failure",
		Description: fmt.Sprintf("Reach out about failed payment (%.2f %s): %s",
			payment.Amount, payment.Currency, payment.Reason),
		Status:      "urgent",
		DueDate:     time.Now().Add(2 * time.Hour), // Urgent - follow up in 2 hours
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := n.CompostStruct(taskID, "create", task)
	if err != nil {
		return fmt.Errorf("failed to compost task: %w", err)
	}

	log.Printf("[AfterSalesNim] Created urgent task for failed payment: %s", taskID)

	// Emit followup leaf
	followup := leaves.FollowupRequired{
		CustomerID: payment.CustomerID,
		Reason:     fmt.Sprintf("payment_failed: %s", payment.Reason),
		DueDate:    task.DueDate,
	}

	if err := n.LeafStruct("followup.required", followup); err != nil {
		return fmt.Errorf("failed to emit followup leaf: %w", err)
	}

	return nil
}

// Start begins listening for payment leaves.
func (n *AfterSalesNim) Start(ctx context.Context) error {
	n.ctx, n.cancel = context.WithCancel(ctx)

	// Catch payment completed events
	if err := n.Catch("payment.completed", func(leaf core.Leaf) {
		if err := n.Handle(n.ctx, leaf); err != nil {
			log.Printf("[AfterSalesNim] Error handling payment.completed: %v", err)
		}
	}); err != nil {
		return fmt.Errorf("failed to catch payment.completed: %w", err)
	}

	// Catch payment failed events
	if err := n.Catch("payment.failed", func(leaf core.Leaf) {
		if err := n.Handle(n.ctx, leaf); err != nil {
			log.Printf("[AfterSalesNim] Error handling payment.failed: %v", err)
		}
	}); err != nil {
		return fmt.Errorf("failed to catch payment.failed: %w", err)
	}

	log.Printf("[AfterSalesNim] Started listening for payment events")
	return nil
}

// Stop gracefully shuts down the nim.
func (n *AfterSalesNim) Stop() error {
	if n.cancel != nil {
		n.cancel()
	}
	log.Printf("[AfterSalesNim] Stopped")
	return nil
}

// GetTask retrieves a task from soil by ID.
func (n *AfterSalesNim) GetTask(taskID string) (*Task, uint64, error) {
	var task Task
	revision, err := n.DigStruct(taskID, &task)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get task: %w", err)
	}
	return &task, revision, nil
}

// UpdateTask updates a task in soil using optimistic locking.
func (n *AfterSalesNim) UpdateTask(task Task, expectedRevision uint64) error {
	task.UpdatedAt = time.Now()
	_, err := n.CompostStruct(task.ID, "update", task)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}
	return nil
}

// CompleteTask marks a task as completed.
func (n *AfterSalesNim) CompleteTask(taskID string) error {
	task, revision, err := n.GetTask(taskID)
	if err != nil {
		return err
	}

	task.Status = "completed"
	return n.UpdateTask(*task, revision)
}
