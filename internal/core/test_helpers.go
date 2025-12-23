package core

import (
	"testing"

	"github.com/nats-io/nats.go"
)

// SetupTestNATS connects to a local NATS server for testing.
// Tests will be skipped if NATS is not available.
func SetupTestNATS(t *testing.T) (*nats.Conn, nats.JetStreamContext) {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Skipf("NATS not available: %v (run 'make start' to start NATS)", err)
	}
	
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		t.Skipf("JetStream not available: %v", err)
	}
	
	return nc, js
}

// setupTestNATS is kept for backward compatibility with existing tests.
func setupTestNATS(t *testing.T) *nats.Conn {
	nc, js := SetupTestNATS(t)
	_ = js // Ignore JetStream for legacy tests
	return nc
}

// setupTestJetStream is kept for backward compatibility with existing tests.
func setupTestJetStream(t *testing.T) (nats.JetStreamContext, *nats.Conn) {
	nc, js := SetupTestNATS(t)
	return js, nc
}
