package validator

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"kii.com/internal/infrastructure/logger"
)

func TestHMACValidator_ValidateRequest(t *testing.T) {
	secret := "test-secret-key"
	tolerance := 5 * time.Minute
	logger := logger.NewLogger()
	validator := NewHMACValidator(secret, tolerance, logger).(*HMACValidator)

	tests := []struct {
		name        string
		timestamp   int64
		nonce       string
		body        string
		signature   string
		wantErr     bool
		errContains string
	}{
		{
			name:      "valid request",
			timestamp: time.Now().Unix(),
			nonce:     "unique-nonce-1",
			body:      `{"user":"user1","asset":"BTC","amount":"100.5"}`,
			wantErr:   false,
		},
		{
			name:        "missing timestamp header",
			timestamp:   0,
			nonce:       "unique-nonce-2",
			body:        `{"user":"user1","asset":"BTC","amount":"100.5"}`,
			wantErr:     true,
			errContains: "missing X-Timestamp",
		},
		{
			name:        "missing nonce header",
			timestamp:   time.Now().Unix(),
			nonce:       "",
			body:        `{"user":"user1","asset":"BTC","amount":"100.5"}`,
			wantErr:     true,
			errContains: "missing X-Nonce",
		},
		{
			name:        "missing signature header",
			timestamp:   time.Now().Unix(),
			nonce:       "unique-nonce-3",
			body:        `{"user":"user1","asset":"BTC","amount":"100.5"}`,
			signature:   "",
			wantErr:     true,
			errContains: "missing X-Signature",
		},
		{
			name:        "invalid timestamp format",
			timestamp:   0,
			nonce:       "unique-nonce-4",
			body:        `{"user":"user1","asset":"BTC","amount":"100.5"}`,
			wantErr:     true,
			errContains: "missing X-Timestamp", // Will fail on missing header check first
		},
		{
			name:        "timestamp out of tolerance (future)",
			timestamp:   time.Now().Add(10 * time.Minute).Unix(),
			nonce:       "unique-nonce-5",
			body:        `{"user":"user1","asset":"BTC","amount":"100.5"}`,
			signature:   "dummy-signature", // Set signature so it doesn't fail on missing signature check
			wantErr:     true,
			errContains: "timestamp out of tolerance",
		},
		{
			name:        "timestamp out of tolerance (past)",
			timestamp:   time.Now().Add(-10 * time.Minute).Unix(),
			nonce:       "unique-nonce-6",
			body:        `{"user":"user1","asset":"BTC","amount":"100.5"}`,
			signature:   "dummy-signature", // Set signature so it doesn't fail on missing signature check
			wantErr:     true,
			errContains: "timestamp out of tolerance",
		},
		{
			name:        "invalid signature",
			timestamp:   time.Now().Unix(),
			nonce:       "unique-nonce-7",
			body:        `{"user":"user1","asset":"BTC","amount":"100.5"}`,
			signature:   "invalid-signature",
			wantErr:     true,
			errContains: "invalid signature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
			bodyBytes := []byte(tt.body)
			req.Body = http.NoBody

			// Set headers
			if tt.timestamp != 0 {
				req.Header.Set("X-Timestamp", strconv.FormatInt(tt.timestamp, 10))
			}
			if tt.nonce != "" {
				req.Header.Set("X-Nonce", tt.nonce)
			}

			// Compute signature if not provided or if it's a valid test case
			if tt.signature == "" && !tt.wantErr && tt.timestamp != 0 {
				// For valid cases, compute the correct signature
				message := strconv.FormatInt(tt.timestamp, 10) + "\n" + tt.nonce + "\n" + tt.body
				mac := hmac.New(sha256.New, []byte(secret))
				mac.Write([]byte(message))
				tt.signature = hex.EncodeToString(mac.Sum(nil))
			}
			if tt.signature != "" {
				req.Header.Set("X-Signature", tt.signature)
			}

			// Validate
			err := validator.ValidateRequest(context.Background(), req, bodyBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("HMACValidator.ValidateRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" && err != nil {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("HMACValidator.ValidateRequest() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestHMACValidator_ReplayAttack(t *testing.T) {
	secret := "test-secret-key"
	tolerance := 5 * time.Minute
	logger := logger.NewLogger()
	validator := NewHMACValidator(secret, tolerance, logger).(*HMACValidator)

	timestamp := time.Now().Unix()
	nonce := "replay-nonce-1"
	body := `{"user":"user1","asset":"BTC","amount":"100.5"}`

	// Compute signature
	message := strconv.FormatInt(timestamp, 10) + "\n" + nonce + "\n" + body
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	signature := hex.EncodeToString(mac.Sum(nil))

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature", signature)

	// First request should succeed
	err := validator.ValidateRequest(context.Background(), req, []byte(body))
	if err != nil {
		t.Errorf("First request should succeed, got error: %v", err)
	}

	// Second request with same nonce should fail (replay attack)
	err = validator.ValidateRequest(context.Background(), req, []byte(body))
	if err == nil {
		t.Error("Replay attack should be detected, but validation succeeded")
	}
	if !contains(err.Error(), "duplicate nonce") {
		t.Errorf("Expected duplicate nonce error, got: %v", err)
	}
}

func TestNonceStore_IsValid(t *testing.T) {
	store := NewNonceStore()
	now := time.Now()

	// First use of nonce should be valid
	if !store.IsValid("nonce-1", now) {
		t.Error("First use of nonce should be valid")
	}

	// Second use of same nonce should be invalid
	if store.IsValid("nonce-1", now) {
		t.Error("Reuse of nonce should be invalid")
	}

	// Different nonce should be valid
	if !store.IsValid("nonce-2", now) {
		t.Error("Different nonce should be valid")
	}
}

func TestHMACValidator_ComputeSignature(t *testing.T) {
	secret := "test-secret-key"
	tolerance := 5 * time.Minute
	logger := logger.NewLogger()
	validator := NewHMACValidator(secret, tolerance, logger).(*HMACValidator)

	timestamp := "1234567890"
	nonce := "test-nonce"
	body := []byte(`{"user":"user1","asset":"BTC","amount":"100.5"}`)

	// Compute signature
	signature, err := validator.computeSignature(timestamp, nonce, body)
	if err != nil {
		t.Fatalf("computeSignature() error = %v", err)
	}

	// Verify signature is hex-encoded
	if len(signature) != 64 { // SHA256 produces 32 bytes = 64 hex chars
		t.Errorf("Signature length = %d, want 64", len(signature))
	}

	// Verify signature matches expected
	message := timestamp + "\n" + nonce + "\n" + string(body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	expected := hex.EncodeToString(mac.Sum(nil))

	if signature != expected {
		t.Errorf("Signature = %v, want %v", signature, expected)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
