package core

import (
	"testing"

	"github.com/nats-io/nats.go"
)

// setupTestNATS connects to a local NATS server for testing.
// Tests will be skipped if NATS is not available.
func setupTestNATS(t *testing.T) *nats.Conn {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Skipf("NATS not available: %v (run 'make start' to start NATS)", err)
	}
	return nc
}

// setupTestJetStream connects to NATS and gets a JetStream context.
// Tests will be skipped if NATS or JetStream is not available.
func setupTestJetStream(t *testing.T) (nats.JetStreamContext, *nats.Conn) {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Skipf("NATS not available: %v", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		t.Skipf("JetStream not available: %v", err)
	}

	return js, nc
}
