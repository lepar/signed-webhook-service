package http

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"kii.com/internal/application/usecase"
	"kii.com/internal/domain/entity"
	"kii.com/internal/infrastructure/logger"
	"kii.com/internal/infrastructure/repository"
	"kii.com/internal/infrastructure/validator"
)

// mockValidator implements port.WebhookValidator
type mockValidator struct {
	validateFunc func(ctx context.Context, r *http.Request, body []byte) error
}

func (m *mockValidator) ValidateRequest(ctx context.Context, r *http.Request, body []byte) error {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, r, body)
	}
	return nil
}

// mockRepository implements port.LedgerRepository
type mockRepository struct {
	addEntryFunc   func(ctx context.Context, entry entity.LedgerEntry) error
	getBalanceFunc func(ctx context.Context, user string) (*entity.BalanceResponse, error)
}

func (m *mockRepository) AddEntry(ctx context.Context, entry entity.LedgerEntry) error {
	if m.addEntryFunc != nil {
		return m.addEntryFunc(ctx, entry)
	}
	return nil
}

func (m *mockRepository) GetBalance(ctx context.Context, user string) (*entity.BalanceResponse, error) {
	if m.getBalanceFunc != nil {
		return m.getBalanceFunc(ctx, user)
	}
	return &entity.BalanceResponse{User: user, Balances: make(map[string]string)}, nil
}

func TestHandler_HandleWebhook(t *testing.T) {
	logger := logger.NewLogger()

	tests := []struct {
		name           string
		method         string
		body           string
		headers        map[string]string
		validatorError error
		useCaseError   error
		wantStatus     int
		wantBody       string
	}{
		{
			name:   "valid webhook request",
			method: http.MethodPost,
			body:   `{"user":"user1","asset":"BTC","amount":"100.5"}`,
			headers: map[string]string{
				"X-Timestamp": strconv.FormatInt(time.Now().Unix(), 10),
				"X-Nonce":     "test-nonce-1",
				"X-Signature": "valid-signature",
			},
			wantStatus: http.StatusOK,
			wantBody:   `{"status":"ok"}`,
		},
		{
			name:       "wrong HTTP method",
			method:     http.MethodGet,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "invalid JSON body",
			method: http.MethodPost,
			body:   `invalid json`,
			headers: map[string]string{
				"X-Timestamp": strconv.FormatInt(time.Now().Unix(), 10),
				"X-Nonce":     "test-nonce-2",
				"X-Signature": "valid-signature",
			},
			wantStatus: http.StatusInternalServerError, // Use case validation returns error, handler returns 500
		},
		{
			name:   "validator error",
			method: http.MethodPost,
			body:   `{"user":"user1","asset":"BTC","amount":"100.5"}`,
			headers: map[string]string{
				"X-Timestamp": strconv.FormatInt(time.Now().Unix(), 10),
				"X-Nonce":     "test-nonce-3",
				"X-Signature": "invalid-signature",
			},
			validatorError: errors.New("invalid signature"),
			wantStatus:     http.StatusUnauthorized,
		},
		{
			name:   "missing user field",
			method: http.MethodPost,
			body:   `{"asset":"BTC","amount":"100.5"}`,
			headers: map[string]string{
				"X-Timestamp": strconv.FormatInt(time.Now().Unix(), 10),
				"X-Nonce":     "test-nonce-4",
				"X-Signature": "valid-signature",
			},
			wantStatus: http.StatusInternalServerError, // Use case validation returns error, handler returns 500
		},
		{
			name:   "use case error",
			method: http.MethodPost,
			body:   `{"user":"user1","asset":"BTC","amount":"100.5"}`,
			headers: map[string]string{
				"X-Timestamp": strconv.FormatInt(time.Now().Unix(), 10),
				"X-Nonce":     "test-nonce-5",
				"X-Signature": "valid-signature",
			},
			useCaseError: errors.New("repository error"),
			wantStatus:   http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := &mockValidator{
				validateFunc: func(ctx context.Context, r *http.Request, body []byte) error {
					return tt.validatorError
				},
			}

			// Create mock repository
			mockRepo := &mockRepository{
				addEntryFunc: func(ctx context.Context, entry entity.LedgerEntry) error {
					return tt.useCaseError
				},
			}

			// Create real use cases with mocked dependencies
			processUseCase := usecase.NewProcessWebhookUseCase(validator, mockRepo)
			getBalanceUseCase := usecase.NewGetBalanceUseCase(mockRepo)

			handler := NewHandler(
				processUseCase,
				getBalanceUseCase,
				validator,
				logger,
			)

			req := httptest.NewRequest(tt.method, "/webhook", bytes.NewBufferString(tt.body))
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			// Add logger to context
			ctx := context.WithValue(req.Context(), "logger", logger)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.HandleWebhook(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Handler.HandleWebhook() status = %v, want %v", w.Code, tt.wantStatus)
			}

			if tt.wantBody != "" {
				var gotBody map[string]string
				if err := json.Unmarshal(w.Body.Bytes(), &gotBody); err == nil {
					var wantBody map[string]string
					json.Unmarshal([]byte(tt.wantBody), &wantBody)
					if gotBody["status"] != wantBody["status"] {
						t.Errorf("Handler.HandleWebhook() body = %v, want %v", gotBody, wantBody)
					}
				}
			}
		})
	}
}

func TestHandler_HandleBalance(t *testing.T) {
	logger := logger.NewLogger()

	tests := []struct {
		name       string
		method     string
		path       string
		useCaseRes *entity.BalanceResponse
		useCaseErr error
		wantStatus int
		wantUser   string
	}{
		{
			name:   "valid balance request",
			method: http.MethodGet,
			path:   "/balance/user1",
			useCaseRes: &entity.BalanceResponse{
				User: "user1",
				Balances: map[string]string{
					"BTC": "100.5",
					"ETH": "50.25",
				},
			},
			wantStatus: http.StatusOK,
			wantUser:   "user1",
		},
		{
			name:       "wrong HTTP method",
			method:     http.MethodPost,
			path:       "/balance/user1",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "missing user parameter",
			method:     http.MethodGet,
			path:       "/balance/",
			wantStatus: http.StatusBadRequest, // Handler returns 400 for missing path parameter
		},
		{
			name:       "use case error",
			method:     http.MethodGet,
			path:       "/balance/user1",
			useCaseErr: errors.New("repository error"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repository
			mockRepo := &mockRepository{
				getBalanceFunc: func(ctx context.Context, user string) (*entity.BalanceResponse, error) {
					return tt.useCaseRes, tt.useCaseErr
				},
			}

			// Create real use cases with mocked dependencies
			processUseCase := usecase.NewProcessWebhookUseCase(&mockValidator{}, mockRepo)
			getBalanceUseCase := usecase.NewGetBalanceUseCase(mockRepo)

			handler := NewHandler(
				processUseCase,
				getBalanceUseCase,
				&mockValidator{},
				logger,
			)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			ctx := context.WithValue(req.Context(), "logger", logger)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.HandleBalance(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Handler.HandleBalance() status = %v, want %v", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK && tt.useCaseRes != nil {
				var gotBody entity.BalanceResponse
				if err := json.Unmarshal(w.Body.Bytes(), &gotBody); err != nil {
					t.Errorf("Handler.HandleBalance() failed to unmarshal response: %v", err)
				} else {
					if gotBody.User != tt.wantUser {
						t.Errorf("Handler.HandleBalance() user = %v, want %v", gotBody.User, tt.wantUser)
					}
				}
			}
		})
	}
}

func TestHandler_Integration_ValidWebhook(t *testing.T) {
	// Integration test with real validator
	secret := "test-secret-key"
	logger := logger.NewLogger()

	// Create real validator
	webhookValidator := validator.NewHMACValidator(secret, 5*time.Minute, logger)

	// Create real repository
	ledgerRepo := repository.NewInMemoryLedger(logger)

	// Create use cases
	processUseCase := usecase.NewProcessWebhookUseCase(webhookValidator, ledgerRepo)
	getBalanceUseCase := usecase.NewGetBalanceUseCase(ledgerRepo)

	// Create handler
	handler := NewHandler(processUseCase, getBalanceUseCase, webhookValidator, logger)

	// Prepare webhook request
	body := `{"user":"user1","asset":"BTC","amount":"100.5"}`
	timestamp := time.Now().Unix()
	nonce := "integration-test-nonce"

	// Compute signature
	message := strconv.FormatInt(timestamp, 10) + "\n" + nonce + "\n" + body
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	signature := hex.EncodeToString(mac.Sum(nil))

	// Create HTTP request
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString(body))
	req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature", signature)

	ctx := context.WithValue(req.Context(), "logger", logger)
	req = req.WithContext(ctx)

	// Execute webhook
	w := httptest.NewRecorder()
	handler.HandleWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Integration test: HandleWebhook() status = %v, want %v", w.Code, http.StatusOK)
	}

	// Verify balance was updated
	balanceReq := httptest.NewRequest(http.MethodGet, "/balance/user1", nil)
	balanceCtx := context.WithValue(balanceReq.Context(), "logger", logger)
	balanceReq = balanceReq.WithContext(balanceCtx)

	balanceW := httptest.NewRecorder()
	handler.HandleBalance(balanceW, balanceReq)

	if balanceW.Code != http.StatusOK {
		t.Errorf("Integration test: HandleBalance() status = %v, want %v", balanceW.Code, http.StatusOK)
	}

	var balance entity.BalanceResponse
	if err := json.Unmarshal(balanceW.Body.Bytes(), &balance); err != nil {
		t.Fatalf("Integration test: failed to unmarshal balance: %v", err)
	}

	if balance.Balances["BTC"] != "100.50000000" {
		t.Errorf("Integration test: balance = %v, want 100.50000000", balance.Balances["BTC"])
	}
}
