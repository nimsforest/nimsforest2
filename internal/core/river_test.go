package core

import (
	"testing"
	"time"
)

func TestNewRiver(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	// Clean up any existing stream
	js.DeleteStream("RIVER")

	river, err := NewRiver(js)
	if err != nil {
		t.Fatalf("Failed to create river: %v", err)
	}
	if river == nil {
		t.Fatal("Expected non-nil river")
	}

	// Verify stream was created
	info, err := river.StreamInfo()
	if err != nil {
		t.Fatalf("Failed to get stream info: %v", err)
	}
	if info.Config.Name != "RIVER" {
		t.Errorf("Expected stream name RIVER, got %s", info.Config.Name)
	}
}

func TestRiver_Flow(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	js.DeleteStream("RIVER")
	river, err := NewRiver(js)
	if err != nil {
		t.Fatalf("Failed to create river: %v", err)
	}

	tests := []struct {
		name    string
		subject string
		data    []byte
		wantErr bool
	}{
		{
			name:    "valid data",
			subject: "stripe.webhook",
			data:    []byte(`{"event": "payment"}`),
			wantErr: false,
		},
		{
			name:    "valid with river prefix",
			subject: "river.paypal.webhook",
			data:    []byte(`{"event": "payment"}`),
			wantErr: false,
		},
		{
			name:    "empty subject",
			subject: "",
			data:    []byte(`{"event": "test"}`),
			wantErr: true,
		},
		{
			name:    "empty data",
			subject: "test.event",
			data:    []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := river.Flow(tt.subject, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Flow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRiver_Observe(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	js.DeleteStream("RIVER")
	river, err := NewRiver(js)
	if err != nil {
		t.Fatalf("Failed to create river: %v", err)
	}

	// Channel to receive observed data
	observed := make(chan RiverData, 1)

	// Start observing
	err = river.Observe("stripe.>", func(data RiverData) {
		observed <- data
	})
	if err != nil {
		t.Fatalf("Failed to observe: %v", err)
	}

	// Give observer time to be ready
	time.Sleep(200 * time.Millisecond)

	// Flow data
	testData := []byte(`{"type": "charge.succeeded"}`)
	if err := river.Flow("stripe.webhook", testData); err != nil {
		t.Fatalf("Failed to flow data: %v", err)
	}

	// Wait for observation
	select {
	case data := <-observed:
		if data.Subject != "river.stripe.webhook" {
			t.Errorf("Expected subject river.stripe.webhook, got %s", data.Subject)
		}
		if string(data.Data) != string(testData) {
			t.Errorf("Data mismatch: expected %s, got %s", string(testData), string(data.Data))
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for observation")
	}
}

func TestRiver_ObserveWithWildcard(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	js.DeleteStream("RIVER")
	river, err := NewRiver(js)
	if err != nil {
		t.Fatalf("Failed to create river: %v", err)
	}

	observed := make(chan RiverData, 3)

	// Observe all payment-related data
	err = river.Observe("payment.>", func(data RiverData) {
		observed <- data
	})
	if err != nil {
		t.Fatalf("Failed to observe: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Flow multiple data items
	testSubjects := []string{
		"payment.stripe",
		"payment.paypal",
		"payment.square",
	}

	for _, subj := range testSubjects {
		if err := river.Flow(subj, []byte(`{"test": "data"}`)); err != nil {
			t.Fatalf("Failed to flow data: %v", err)
		}
	}

	// Verify we received all three
	count := 0
	timeout := time.After(3 * time.Second)

	for count < 3 {
		select {
		case <-observed:
			count++
		case <-timeout:
			t.Fatalf("Only received %d out of 3 observations", count)
		}
	}
}

func TestRiver_ObserveWithConsumer(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	js.DeleteStream("RIVER")
	river, err := NewRiver(js)
	if err != nil {
		t.Fatalf("Failed to create river: %v", err)
	}

	observed := make(chan RiverData, 1)

	// Observe with a named consumer
	err = river.ObserveWithConsumer("webhook.>", "test-consumer", func(data RiverData) {
		observed <- data
	})
	if err != nil {
		t.Fatalf("Failed to observe with consumer: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Flow data
	if err := river.Flow("webhook.test", []byte(`{"msg": "hello"}`)); err != nil {
		t.Fatalf("Failed to flow data: %v", err)
	}

	// Verify observation
	select {
	case data := <-observed:
		if data.Subject != "river.webhook.test" {
			t.Errorf("Expected subject river.webhook.test, got %s", data.Subject)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for observation")
	}
}

func TestRiver_StreamInfo(t *testing.T) {
	js, nc := setupTestJetStream(t)
	defer nc.Close()

	js.DeleteStream("RIVER")
	river, err := NewRiver(js)
	if err != nil {
		t.Fatalf("Failed to create river: %v", err)
	}

	info, err := river.StreamInfo()
	if err != nil {
		t.Fatalf("Failed to get stream info: %v", err)
	}

	if info.Config.Name != "RIVER" {
		t.Errorf("Expected stream name RIVER, got %s", info.Config.Name)
	}

	// Flow some data and verify count increases
	river.Flow("test.data", []byte(`{"test": true}`))

	info, err = river.StreamInfo()
	if err != nil {
		t.Fatalf("Failed to get updated stream info: %v", err)
	}

	if info.State.Msgs < 1 {
		t.Errorf("Expected at least 1 message in stream, got %d", info.State.Msgs)
	}
}
