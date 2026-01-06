package sources

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"
)

func TestHMACVerifier(t *testing.T) {
	secret := "test-secret"
	payload := []byte(`{"event": "test"}`)

	// Compute expected signature
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	expected := hex.EncodeToString(h.Sum(nil))

	verifier := NewHMACVerifier(secret, "sha256")

	t.Run("valid signature", func(t *testing.T) {
		err := verifier.Verify(payload, expected)
		if err != nil {
			t.Errorf("Verify failed with valid signature: %v", err)
		}
	})

	t.Run("valid signature with prefix", func(t *testing.T) {
		err := verifier.Verify(payload, "sha256="+expected)
		if err != nil {
			t.Errorf("Verify failed with prefixed signature: %v", err)
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		err := verifier.Verify(payload, "invalid")
		if err == nil {
			t.Error("Expected error for invalid signature")
		}
	})

	t.Run("empty signature", func(t *testing.T) {
		err := verifier.Verify(payload, "")
		if err == nil {
			t.Error("Expected error for empty signature")
		}
	})
}

func TestGitHubVerifier(t *testing.T) {
	secret := "github-secret"
	payload := []byte(`{"action": "opened"}`)

	// Compute expected GitHub signature
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	sig := "sha256=" + hex.EncodeToString(h.Sum(nil))

	verifier := NewGitHubVerifier(secret)

	t.Run("valid signature", func(t *testing.T) {
		err := verifier.Verify(payload, sig)
		if err != nil {
			t.Errorf("Verify failed: %v", err)
		}
	})

	t.Run("missing sha256 prefix", func(t *testing.T) {
		// Remove prefix
		err := verifier.Verify(payload, hex.EncodeToString(h.Sum(nil)))
		if err == nil {
			t.Error("Expected error for missing sha256= prefix")
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		err := verifier.Verify(payload, "sha256=invalid")
		if err == nil {
			t.Error("Expected error for invalid signature")
		}
	})
}

func TestStripeVerifier(t *testing.T) {
	secret := "whsec_test"
	payload := []byte(`{"type": "charge.succeeded"}`)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	// Compute Stripe signature
	signedPayload := fmt.Sprintf("%s.%s", timestamp, string(payload))
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(signedPayload))
	sig := hex.EncodeToString(h.Sum(nil))

	stripeHeader := fmt.Sprintf("t=%s,v1=%s", timestamp, sig)

	verifier := NewStripeVerifier(secret)

	t.Run("valid signature", func(t *testing.T) {
		err := verifier.Verify(payload, stripeHeader)
		if err != nil {
			t.Errorf("Verify failed: %v", err)
		}
	})

	t.Run("old timestamp", func(t *testing.T) {
		oldTimestamp := fmt.Sprintf("%d", time.Now().Add(-10*time.Minute).Unix())
		oldSignedPayload := fmt.Sprintf("%s.%s", oldTimestamp, string(payload))
		h := hmac.New(sha256.New, []byte(secret))
		h.Write([]byte(oldSignedPayload))
		oldSig := hex.EncodeToString(h.Sum(nil))
		oldHeader := fmt.Sprintf("t=%s,v1=%s", oldTimestamp, oldSig)

		err := verifier.Verify(payload, oldHeader)
		if err == nil {
			t.Error("Expected error for old timestamp")
		}
	})

	t.Run("invalid format", func(t *testing.T) {
		err := verifier.Verify(payload, "invalid")
		if err == nil {
			t.Error("Expected error for invalid format")
		}
	})
}

func TestNoOpVerifier(t *testing.T) {
	verifier := NewNoOpVerifier()

	// Should always pass
	err := verifier.Verify([]byte("anything"), "")
	if err != nil {
		t.Errorf("NoOpVerifier should always pass: %v", err)
	}
}
