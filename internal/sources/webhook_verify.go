package sources

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"strconv"
	"strings"
	"time"
)

// SignatureVerifier verifies webhook signatures.
type SignatureVerifier interface {
	Verify(payload []byte, signature string) error
}

// =============================================================================
// HMAC Verifier (Generic)
// =============================================================================

// HMACVerifier verifies HMAC signatures.
type HMACVerifier struct {
	secret []byte
	algo   string
}

// NewHMACVerifier creates a new HMAC signature verifier.
// Supported algorithms: sha256, sha512
func NewHMACVerifier(secret string, algo string) *HMACVerifier {
	return &HMACVerifier{
		secret: []byte(secret),
		algo:   algo,
	}
}

// Verify checks the HMAC signature.
func (v *HMACVerifier) Verify(payload []byte, signature string) error {
	if signature == "" {
		return errors.New("missing signature")
	}

	// Remove prefix if present (e.g., "sha256=")
	signature = strings.TrimPrefix(signature, "sha256=")
	signature = strings.TrimPrefix(signature, "sha512=")

	var h hash.Hash
	switch v.algo {
	case "sha256":
		h = hmac.New(sha256.New, v.secret)
	case "sha512":
		h = hmac.New(sha512.New, v.secret)
	default:
		return fmt.Errorf("unsupported algorithm: %s", v.algo)
	}

	h.Write(payload)
	expected := hex.EncodeToString(h.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return errors.New("signature mismatch")
	}

	return nil
}

// =============================================================================
// Stripe Verifier
// =============================================================================

// StripeVerifier verifies Stripe webhook signatures.
// Stripe uses a timestamp + HMAC-SHA256 scheme.
type StripeVerifier struct {
	secret    []byte
	tolerance time.Duration
}

// NewStripeVerifier creates a new Stripe signature verifier.
func NewStripeVerifier(secret string) *StripeVerifier {
	return &StripeVerifier{
		secret:    []byte(secret),
		tolerance: 5 * time.Minute, // Default tolerance
	}
}

// SetTolerance sets the timestamp tolerance for replay attack prevention.
func (v *StripeVerifier) SetTolerance(d time.Duration) {
	v.tolerance = d
}

// Verify checks the Stripe signature.
// Stripe-Signature format: t=<timestamp>,v1=<signature>
func (v *StripeVerifier) Verify(payload []byte, signature string) error {
	if signature == "" {
		return errors.New("missing Stripe-Signature header")
	}

	// Parse the signature header
	var timestamp string
	var sig string

	parts := strings.Split(signature, ",")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			timestamp = kv[1]
		case "v1":
			sig = kv[1]
		}
	}

	if timestamp == "" || sig == "" {
		return errors.New("invalid Stripe-Signature format")
	}

	// Verify timestamp is within tolerance
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}

	signedAt := time.Unix(ts, 0)
	if time.Since(signedAt) > v.tolerance {
		return errors.New("signature timestamp too old")
	}

	// Compute expected signature
	// Stripe's signed payload: timestamp.payload
	signedPayload := fmt.Sprintf("%s.%s", timestamp, string(payload))
	h := hmac.New(sha256.New, v.secret)
	h.Write([]byte(signedPayload))
	expected := hex.EncodeToString(h.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return errors.New("signature mismatch")
	}

	return nil
}

// =============================================================================
// GitHub Verifier
// =============================================================================

// GitHubVerifier verifies GitHub webhook signatures.
// GitHub uses HMAC-SHA256 with "sha256=" prefix.
type GitHubVerifier struct {
	secret []byte
}

// NewGitHubVerifier creates a new GitHub signature verifier.
func NewGitHubVerifier(secret string) *GitHubVerifier {
	return &GitHubVerifier{
		secret: []byte(secret),
	}
}

// Verify checks the GitHub signature.
// X-Hub-Signature-256 format: sha256=<signature>
func (v *GitHubVerifier) Verify(payload []byte, signature string) error {
	if signature == "" {
		return errors.New("missing X-Hub-Signature-256 header")
	}

	// Remove sha256= prefix
	sig := strings.TrimPrefix(signature, "sha256=")
	if sig == signature {
		return errors.New("signature must have sha256= prefix")
	}

	// Compute expected signature
	h := hmac.New(sha256.New, v.secret)
	h.Write(payload)
	expected := hex.EncodeToString(h.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return errors.New("signature mismatch")
	}

	return nil
}

// =============================================================================
// Slack Verifier
// =============================================================================

// SlackVerifier verifies Slack webhook signatures.
// Slack uses a timestamp + HMAC-SHA256 scheme with a specific format.
type SlackVerifier struct {
	secret    []byte
	tolerance time.Duration
}

// NewSlackVerifier creates a new Slack signature verifier.
func NewSlackVerifier(secret string) *SlackVerifier {
	return &SlackVerifier{
		secret:    []byte(secret),
		tolerance: 5 * time.Minute,
	}
}

// SetTolerance sets the timestamp tolerance.
func (v *SlackVerifier) SetTolerance(d time.Duration) {
	v.tolerance = d
}

// Verify checks the Slack signature.
// Requires X-Slack-Request-Timestamp and X-Slack-Signature headers.
// signature format: v0=<hex_signature>
func (v *SlackVerifier) Verify(payload []byte, signature string) error {
	// Note: This simplified version expects signature in format "timestamp:v0=signature"
	// In practice, you'd need to pass both headers
	if signature == "" {
		return errors.New("missing X-Slack-Signature header")
	}

	// Remove v0= prefix
	sig := strings.TrimPrefix(signature, "v0=")
	if sig == signature {
		return errors.New("signature must have v0= prefix")
	}

	// For simplified verification (full implementation would need timestamp header)
	h := hmac.New(sha256.New, v.secret)
	h.Write(payload)
	expected := hex.EncodeToString(h.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return errors.New("signature mismatch")
	}

	return nil
}

// =============================================================================
// No-Op Verifier (for testing or when verification is disabled)
// =============================================================================

// NoOpVerifier always passes verification.
type NoOpVerifier struct{}

// NewNoOpVerifier creates a verifier that always succeeds.
func NewNoOpVerifier() *NoOpVerifier {
	return &NoOpVerifier{}
}

// Verify always returns nil.
func (v *NoOpVerifier) Verify(payload []byte, signature string) error {
	return nil
}
