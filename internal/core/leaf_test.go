package core

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewLeaf(t *testing.T) {
	subject := "payment.completed"
	data := []byte(`{"amount": 100}`)
	source := "payment-tree"

	leaf := NewLeaf(subject, data, source)

	if leaf.Subject != subject {
		t.Errorf("Expected subject %s, got %s", subject, leaf.Subject)
	}
	if string(leaf.Data) != string(data) {
		t.Errorf("Expected data %s, got %s", string(data), string(leaf.Data))
	}
	if leaf.Source != source {
		t.Errorf("Expected source %s, got %s", source, leaf.Source)
	}
	if leaf.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}
}

func TestLeaf_Validate(t *testing.T) {
	tests := []struct {
		name    string
		leaf    Leaf
		wantErr bool
	}{
		{
			name: "valid leaf",
			leaf: Leaf{
				Subject:   "test.event",
				Data:      json.RawMessage(`{"key": "value"}`),
				Source:    "test-source",
				Timestamp: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "missing subject",
			leaf: Leaf{
				Data:      json.RawMessage(`{"key": "value"}`),
				Source:    "test-source",
				Timestamp: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "missing source",
			leaf: Leaf{
				Subject:   "test.event",
				Data:      json.RawMessage(`{"key": "value"}`),
				Timestamp: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "missing data",
			leaf: Leaf{
				Subject:   "test.event",
				Source:    "test-source",
				Timestamp: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "zero timestamp",
			leaf: Leaf{
				Subject: "test.event",
				Data:    json.RawMessage(`{"key": "value"}`),
				Source:  "test-source",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.leaf.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLeaf_MarshalJSON(t *testing.T) {
	leaf := NewLeaf("test.event", []byte(`{"amount": 100}`), "test-source")

	data, err := json.Marshal(leaf)
	if err != nil {
		t.Fatalf("Failed to marshal leaf: %v", err)
	}

	// Verify we can unmarshal it back
	var unmarshaled Leaf
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal leaf: %v", err)
	}

	if unmarshaled.Subject != leaf.Subject {
		t.Errorf("Subject mismatch after marshal/unmarshal")
	}
	// Compare the JSON data semantically, not byte-for-byte
	var origData, unmarshaledData map[string]interface{}
	if err := json.Unmarshal(leaf.Data, &origData); err != nil {
		t.Fatalf("Failed to unmarshal original data: %v", err)
	}
	if err := json.Unmarshal(unmarshaled.Data, &unmarshaledData); err != nil {
		t.Fatalf("Failed to unmarshal unmarshaled data: %v", err)
	}
	if origData["amount"] != unmarshaledData["amount"] {
		t.Errorf("Data mismatch after marshal/unmarshal")
	}
	if unmarshaled.Source != leaf.Source {
		t.Errorf("Source mismatch after marshal/unmarshal")
	}
}

func TestLeaf_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"subject": "payment.completed",
		"data": {"customer_id": "123", "amount": 99.99},
		"source": "payment-tree",
		"ts": "2025-12-23T12:00:00Z"
	}`

	var leaf Leaf
	err := json.Unmarshal([]byte(jsonData), &leaf)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if leaf.Subject != "payment.completed" {
		t.Errorf("Expected subject 'payment.completed', got '%s'", leaf.Subject)
	}
	if leaf.Source != "payment-tree" {
		t.Errorf("Expected source 'payment-tree', got '%s'", leaf.Source)
	}
	if leaf.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}
}

func TestLeaf_String(t *testing.T) {
	leaf := NewLeaf("test.event", []byte(`{"key": "value"}`), "test-source")
	str := leaf.String()

	if str == "" {
		t.Error("Expected non-empty string representation")
	}

	// Check that the string contains key information
	if !contains(str, "test.event") {
		t.Error("String representation should contain subject")
	}
	if !contains(str, "test-source") {
		t.Error("String representation should contain source")
	}
}

func TestLeaf_RoundTrip(t *testing.T) {
	// Test complete round trip: create -> marshal -> unmarshal -> validate
	original := NewLeaf(
		"payment.completed",
		[]byte(`{"customer_id": "cust_123", "amount": 49.99}`),
		"payment-tree",
	)

	// Marshal to JSON
	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal back
	var roundtrip Leaf
	if err := json.Unmarshal(jsonData, &roundtrip); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Validate
	if err := roundtrip.Validate(); err != nil {
		t.Errorf("Round-tripped leaf failed validation: %v", err)
	}

	// Check fields match
	if roundtrip.Subject != original.Subject {
		t.Error("Subject mismatch after round trip")
	}
	if roundtrip.Source != original.Source {
		t.Error("Source mismatch after round trip")
	}
	
	// Compare JSON data semantically
	var origData, roundtripData map[string]interface{}
	if err := json.Unmarshal(original.Data, &origData); err != nil {
		t.Fatalf("Failed to unmarshal original data: %v", err)
	}
	if err := json.Unmarshal(roundtrip.Data, &roundtripData); err != nil {
		t.Fatalf("Failed to unmarshal roundtrip data: %v", err)
	}
	if origData["customer_id"] != roundtripData["customer_id"] {
		t.Error("Data mismatch after round trip")
	}
	if origData["amount"] != roundtripData["amount"] {
		t.Error("Data amount mismatch after round trip")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != "" && substr != "" && 
		   (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		   (len(s) > len(substr)*2 && findInString(s, substr))))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
