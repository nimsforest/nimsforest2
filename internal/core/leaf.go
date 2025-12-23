package core

import (
	"encoding/json"
	"fmt"
	"time"
)

// Leaf represents a structured event in the NimsForest system.
// Leaves are the standardized data packets that flow through the wind,
// carrying typed information between trees and nims.
type Leaf struct {
	Subject   string          `json:"subject"`   // Event type identifier (e.g., "payment.completed")
	Data      json.RawMessage `json:"data"`      // Structured payload as JSON
	Source    string          `json:"source"`    // The tree or nim that created this leaf
	Timestamp time.Time       `json:"ts"`        // When this leaf was created
}

// NewLeaf creates a new leaf with the given parameters.
// It automatically sets the timestamp to the current time.
func NewLeaf(subject string, data []byte, source string) *Leaf {
	return &Leaf{
		Subject:   subject,
		Data:      json.RawMessage(data),
		Source:    source,
		Timestamp: time.Now(),
	}
}

// Validate checks if the leaf has all required fields.
func (l *Leaf) Validate() error {
	if l.Subject == "" {
		return fmt.Errorf("leaf subject cannot be empty")
	}
	if l.Source == "" {
		return fmt.Errorf("leaf source cannot be empty")
	}
	if len(l.Data) == 0 {
		return fmt.Errorf("leaf data cannot be empty")
	}
	if l.Timestamp.IsZero() {
		return fmt.Errorf("leaf timestamp cannot be zero")
	}
	return nil
}

// MarshalJSON implements json.Marshaler interface.
func (l *Leaf) MarshalJSON() ([]byte, error) {
	type Alias Leaf
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(l),
	})
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (l *Leaf) UnmarshalJSON(data []byte) error {
	type Alias Leaf
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(l),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("failed to unmarshal leaf: %w", err)
	}
	return nil
}

// String returns a string representation of the leaf for logging.
func (l *Leaf) String() string {
	return fmt.Sprintf("Leaf{subject=%s, source=%s, ts=%s}", 
		l.Subject, l.Source, l.Timestamp.Format(time.RFC3339))
}
