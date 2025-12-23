package leaves

import "time"

// PaymentCompleted represents a successful payment event.
// This leaf is typically emitted by payment trees when they parse
// successful payment provider webhooks.
type PaymentCompleted struct {
	CustomerID string  `json:"customer_id"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	ItemID     string  `json:"item_id"`
}

// PaymentFailed represents a failed payment event.
// This leaf is emitted when a payment attempt is unsuccessful.
type PaymentFailed struct {
	CustomerID string `json:"customer_id"`
	Amount     float64 `json:"amount"`
	Currency   string `json:"currency"`
	ItemID     string `json:"item_id"`
	Reason     string `json:"reason"`
}

// FollowupRequired represents a task that needs followup action.
// This leaf is typically created by nims to schedule future work.
type FollowupRequired struct {
	CustomerID string    `json:"customer_id"`
	Reason     string    `json:"reason"`
	DueDate    time.Time `json:"due_date"`
}

// EmailSend represents a request to send an email.
// This leaf would be caught by a communications nim.
type EmailSend struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	TemplateID string `json:"template_id,omitempty"`
}
