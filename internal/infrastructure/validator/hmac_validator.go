package validator

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"kii.com/internal/domain/port"
	"kii.com/internal/infrastructure/logger"
)

// NonceStore tracks used nonces to prevent replay attacks
type NonceStore struct {
	mu     sync.RWMutex
	nonces map[string]time.Time
}

// NewNonceStore creates a new nonce store
func NewNonceStore() *NonceStore {
	return &NonceStore{
		nonces: make(map[string]time.Time),
	}
}

// IsValid checks if a nonce is valid (not seen before) and records it
func (ns *NonceStore) IsValid(nonce string, timestamp time.Time) bool {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	// Check if nonce was already used
	if existingTime, exists := ns.nonces[nonce]; exists {
		// Allow cleanup of old nonces (older than 1 hour)
		if time.Since(existingTime) > time.Hour {
			delete(ns.nonces, nonce)
		} else {
			return false
		}
	}

	// Record the nonce
	ns.nonces[nonce] = timestamp

	// Cleanup old nonces periodically (simple approach - could be optimized)
	if len(ns.nonces) > 10000 {
		ns.cleanup()
	}

	return true
}

// cleanup removes nonces older than 1 hour
func (ns *NonceStore) cleanup() {
	now := time.Now()
	for nonce, timestamp := range ns.nonces {
		if now.Sub(timestamp) > time.Hour {
			delete(ns.nonces, nonce)
		}
	}
}

// HMACValidator implements the WebhookValidator port
type HMACValidator struct {
	secret             string
	nonceStore         *NonceStore
	timestampTolerance time.Duration
	logger             logger.Logger
}

// NewHMACValidator creates a new HMAC validator
func NewHMACValidator(
	secret string,
	timestampTolerance time.Duration,
	logger logger.Logger,
) port.WebhookValidator {
	return &HMACValidator{
		secret:             secret,
		nonceStore:         NewNonceStore(),
		timestampTolerance: timestampTolerance,
		logger:             logger,
	}
}

// ValidateRequest validates the incoming webhook request
func (v *HMACValidator) ValidateRequest(ctx context.Context, r *http.Request, body []byte) error {
	// Extract headers
	timestampStr := r.Header.Get("X-Timestamp")
	nonce := r.Header.Get("X-Nonce")
	signature := r.Header.Get("X-Signature")

	if timestampStr == "" {
		return fmt.Errorf("missing X-Timestamp header")
	}
	if nonce == "" {
		return fmt.Errorf("missing X-Nonce header")
	}
	if signature == "" {
		return fmt.Errorf("missing X-Signature header")
	}

	// Parse timestamp
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid X-Timestamp format: %w", err)
	}
	requestTime := time.Unix(timestamp, 0)

	// Validate timestamp is within tolerance
	now := time.Now()
	timeDiff := now.Sub(requestTime)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	if timeDiff > v.timestampTolerance {
		v.logger.LogWarning(ctx, "Request timestamp out of tolerance",
			"timestamp", timestamp,
			"current_time", now.Unix(),
			"difference_seconds", timeDiff.Seconds(),
			"tolerance_seconds", v.timestampTolerance.Seconds())
		return fmt.Errorf("timestamp out of tolerance: difference is %v, max allowed is %v", timeDiff, v.timestampTolerance)
	}

	// Validate nonce (prevent replay attacks)
	if !v.nonceStore.IsValid(nonce, requestTime) {
		v.logger.LogWarning(ctx, "Duplicate nonce detected (replay attack)",
			"nonce", nonce,
			"timestamp", timestamp)
		return fmt.Errorf("duplicate nonce detected: possible replay attack")
	}

	// Compute expected signature
	expectedSignature, err := v.computeSignature(timestampStr, nonce, body)
	if err != nil {
		return fmt.Errorf("failed to compute signature: %w", err)
	}

	// Compare signatures (constant-time comparison to prevent timing attacks)
	if !hmac.Equal([]byte(expectedSignature), []byte(signature)) {
		v.logger.LogWarning(ctx, "Invalid signature",
			"expected", expectedSignature,
			"received", signature)
		return fmt.Errorf("invalid signature")
	}

	return nil
}

// computeSignature computes the HMAC SHA256 signature
// Format: X-Timestamp + "\n" + X-Nonce + "\n" + <raw_request_body_bytes_as_string>
func (v *HMACValidator) computeSignature(timestamp, nonce string, body []byte) (string, error) {
	// Construct the message to sign
	message := timestamp + "\n" + nonce + "\n" + string(body)

	// Compute HMAC SHA256
	mac := hmac.New(sha256.New, []byte(v.secret))
	_, err := mac.Write([]byte(message))
	if err != nil {
		return "", err
	}

	// Return hex-encoded signature
	return hex.EncodeToString(mac.Sum(nil)), nil
}
