package nims

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/yourusername/nimsforest/internal/core"
	"github.com/yourusername/nimsforest/internal/leaves"
)

func TestAfterSalesNim_Subjects(t *testing.T) {
	nim := NewAfterSalesNim(nil, nil, nil)
	subjects := nim.Subjects()

	if len(subjects) != 2 {
		t.Errorf("Expected 2 subjects, got %d", len(subjects))
	}

	expected := map[string]bool{
		"payment.completed": true,
		"payment.failed":    true,
	}

	for _, subject := range subjects {
		if !expected[subject] {
			t.Errorf("Unexpected subject: %s", subject)
		}
	}
}

func TestAfterSalesNim_HandlePaymentCompleted(t *testing.T) {
	nc, js := core.SetupTestNATS(t)
	defer nc.Close()

	wind := core.NewWind(nc)
	humus, err := core.NewHumus(js)
	if err != nil {
		t.Fatalf("Failed to create humus: %v", err)
	}

	soil, err := core.NewSoil(js)
	if err != nil {
		t.Fatalf("Failed to create soil: %v", err)
	}

	nim := NewAfterSalesNim(wind, humus, soil)

	// Create a payment completed leaf
	payment := leaves.PaymentCompleted{
		CustomerID: "cus_test",
		Amount:     50.0,
		Currency:   "usd",
		ItemID:     "item_123",
	}

	paymentData, err := json.Marshal(payment)
	if err != nil {
		t.Fatalf("Failed to marshal payment: %v", err)
	}

	leaf := core.NewLeaf("payment.completed", paymentData, "test")

	// Handle the leaf
	ctx := context.Background()
	if err := nim.handlePaymentCompleted(ctx, *leaf); err != nil {
		t.Fatalf("Failed to handle payment completed: %v", err)
	}

	// Verify compost was created in humus
	// Note: In a real test, we'd verify the task was created
	// For now, just verify no error occurred
}

func TestAfterSalesNim_HandlePaymentFailed(t *testing.T) {
	nc, js := core.SetupTestNATS(t)
	defer nc.Close()

	wind := core.NewWind(nc)
	humus, err := core.NewHumus(js)
	if err != nil {
		t.Fatalf("Failed to create humus: %v", err)
	}

	soil, err := core.NewSoil(js)
	if err != nil {
		t.Fatalf("Failed to create soil: %v", err)
	}

	nim := NewAfterSalesNim(wind, humus, soil)

	// Create a payment failed leaf
	payment := leaves.PaymentFailed{
		CustomerID: "cus_test",
		Amount:     50.0,
		Currency:   "usd",
		ItemID:     "item_123",
		Reason:     "insufficient_funds",
	}

	paymentData, err := json.Marshal(payment)
	if err != nil {
		t.Fatalf("Failed to marshal payment: %v", err)
	}

	leaf := core.NewLeaf("payment.failed", paymentData, "test")

	// Handle the leaf
	ctx := context.Background()
	if err := nim.handlePaymentFailed(ctx, *leaf); err != nil {
		t.Fatalf("Failed to handle payment failed: %v", err)
	}
}

func TestAfterSalesNim_HandleInvalidSubject(t *testing.T) {
	nim := NewAfterSalesNim(nil, nil, nil)

	leaf := core.NewLeaf("unknown.subject", []byte("{}"), "test")

	ctx := context.Background()
	err := nim.Handle(ctx, *leaf)
	if err == nil {
		t.Error("Expected error for invalid subject, got nil")
	}
}

func TestAfterSalesNim_HandleInvalidJSON(t *testing.T) {
	nc, js := core.SetupTestNATS(t)
	defer nc.Close()

	wind := core.NewWind(nc)
	humus, err := core.NewHumus(js)
	if err != nil {
		t.Fatalf("Failed to create humus: %v", err)
	}

	soil, err := core.NewSoil(js)
	if err != nil {
		t.Fatalf("Failed to create soil: %v", err)
	}

	nim := NewAfterSalesNim(wind, humus, soil)

	leaf := core.NewLeaf("payment.completed", []byte("invalid json"), "test")

	ctx := context.Background()
	err = nim.handlePaymentCompleted(ctx, *leaf)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestAfterSalesNim_HighValuePurchaseEmail(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	nc, js := core.SetupTestNATS(t)
	defer nc.Close()

	wind := core.NewWind(nc)
	humus, err := core.NewHumus(js)
	if err != nil {
		t.Fatalf("Failed to create humus: %v", err)
	}

	soil, err := core.NewSoil(js)
	if err != nil {
		t.Fatalf("Failed to create soil: %v", err)
	}

	nim := NewAfterSalesNim(wind, humus, soil)

	// Subscribe to email leaves
	emailReceived := make(chan leaves.EmailSend, 1)
	_, err = wind.Catch("email.send", func(leaf core.Leaf) {
		var email leaves.EmailSend
		if err := json.Unmarshal(leaf.Data, &email); err != nil {
			t.Errorf("Failed to unmarshal email: %v", err)
			return
		}
		emailReceived <- email
	})
	if err != nil {
		t.Fatalf("Failed to catch email leaves: %v", err)
	}

	// Create a high-value payment (>= $100)
	payment := leaves.PaymentCompleted{
		CustomerID: "cus_vip",
		Amount:     150.0,
		Currency:   "usd",
		ItemID:     "premium_item",
	}

	paymentData, err := json.Marshal(payment)
	if err != nil {
		t.Fatalf("Failed to marshal payment: %v", err)
	}

	leaf := core.NewLeaf("payment.completed", paymentData, "test")

	// Handle the payment
	ctx := context.Background()
	if err := nim.handlePaymentCompleted(ctx, *leaf); err != nil {
		t.Fatalf("Failed to handle payment: %v", err)
	}

	// Wait for email leaf
	select {
	case email := <-emailReceived:
		if email.To != "cus_vip" {
			t.Errorf("Expected email to 'cus_vip', got '%s'", email.To)
		}
		if email.TemplateID != "thank_you_template" {
			t.Errorf("Expected template 'thank_you_template', got '%s'", email.TemplateID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for email leaf")
	}
}

func TestAfterSalesNim_LowValuePurchaseNoEmail(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	nc, js := core.SetupTestNATS(t)
	defer nc.Close()

	wind := core.NewWind(nc)
	humus, err := core.NewHumus(js)
	if err != nil {
		t.Fatalf("Failed to create humus: %v", err)
	}

	soil, err := core.NewSoil(js)
	if err != nil {
		t.Fatalf("Failed to create soil: %v", err)
	}

	nim := NewAfterSalesNim(wind, humus, soil)

	// Subscribe to email leaves
	emailReceived := make(chan leaves.EmailSend, 1)
	_, err = wind.Catch("email.send", func(leaf core.Leaf) {
		var email leaves.EmailSend
		if err := json.Unmarshal(leaf.Data, &email); err != nil {
			return
		}
		emailReceived <- email
	})
	if err != nil {
		t.Fatalf("Failed to catch email leaves: %v", err)
	}

	// Create a low-value payment (< $100)
	payment := leaves.PaymentCompleted{
		CustomerID: "cus_regular",
		Amount:     25.0,
		Currency:   "usd",
		ItemID:     "basic_item",
	}

	paymentData, err := json.Marshal(payment)
	if err != nil {
		t.Fatalf("Failed to marshal payment: %v", err)
	}

	leaf := core.NewLeaf("payment.completed", paymentData, "test")

	// Handle the payment
	ctx := context.Background()
	if err := nim.handlePaymentCompleted(ctx, *leaf); err != nil {
		t.Fatalf("Failed to handle payment: %v", err)
	}

	// Should NOT receive an email for low-value purchases
	select {
	case <-emailReceived:
		t.Error("Unexpected email leaf for low-value purchase")
	case <-time.After(500 * time.Millisecond):
		// Expected - no email sent
	}
}

// Integration test: Full flow from payment to task creation
func TestAfterSalesNim_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	nc, js := core.SetupTestNATS(t)
	defer nc.Close()

	wind := core.NewWind(nc)
	humus, err := core.NewHumus(js)
	if err != nil {
		t.Fatalf("Failed to create humus: %v", err)
	}

	soil, err := core.NewSoil(js)
	if err != nil {
		t.Fatalf("Failed to create soil: %v", err)
	}

	// Start decomposer to process compost with unique consumer name
	consumerName := fmt.Sprintf("decomposer-integration-%d", time.Now().UnixNano())
	decomposer, err := core.RunDecomposerWithConsumer(humus, soil, consumerName)
	if err != nil {
		t.Fatalf("Failed to start decomposer: %v", err)
	}
	defer decomposer.Stop()

	// Give decomposer time to start
	time.Sleep(100 * time.Millisecond)

	nim := NewAfterSalesNim(wind, humus, soil)

	// Subscribe to followup leaves
	followupReceived := make(chan leaves.FollowupRequired, 1)
	_, err = wind.Catch("followup.required", func(leaf core.Leaf) {
		var followup leaves.FollowupRequired
		if err := json.Unmarshal(leaf.Data, &followup); err != nil {
			t.Errorf("Failed to unmarshal followup: %v", err)
			return
		}
		followupReceived <- followup
	})
	if err != nil {
		t.Fatalf("Failed to catch followup leaves: %v", err)
	}

	// Start the nim
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := nim.Start(ctx); err != nil {
		t.Fatalf("Failed to start nim: %v", err)
	}
	defer nim.Stop()

	time.Sleep(100 * time.Millisecond)

	// Drop a payment completed leaf
	payment := leaves.PaymentCompleted{
		CustomerID: "cus_integration",
		Amount:     75.0,
		Currency:   "usd",
		ItemID:     "item_integration",
	}

	if err := wind.Drop(*core.NewLeaf("payment.completed",
		mustMarshal(t, payment), "integration-test")); err != nil {
		t.Fatalf("Failed to drop payment leaf: %v", err)
	}

	// Wait for followup leaf
	select {
	case followup := <-followupReceived:
		if followup.CustomerID != "cus_integration" {
			t.Errorf("Expected customer 'cus_integration', got '%s'", followup.CustomerID)
		}
		if followup.Reason != "post_purchase_followup" {
			t.Errorf("Expected reason 'post_purchase_followup', got '%s'", followup.Reason)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for followup leaf")
	}

	// Wait for decomposer to process compost
	time.Sleep(500 * time.Millisecond)

	// Verify task was created in soil
	// The task ID is generated with timestamp, so we can't predict it exactly
	// In a real test, we'd query soil for all tasks
}

func mustMarshal(t *testing.T, v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	return data
}

func TestTask_Marshaling(t *testing.T) {
	task := Task{
		ID:          "task_123",
		CustomerID:  "cus_test",
		Type:        "followup",
		Description: "Test task",
		Status:      "pending",
		DueDate:     time.Now().Add(24 * time.Hour),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Marshal and unmarshal
	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal task: %v", err)
	}

	var unmarshaled Task
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal task: %v", err)
	}

	if unmarshaled.ID != task.ID {
		t.Errorf("Expected ID %s, got %s", task.ID, unmarshaled.ID)
	}
	if unmarshaled.CustomerID != task.CustomerID {
		t.Errorf("Expected CustomerID %s, got %s", task.CustomerID, unmarshaled.CustomerID)
	}
}
