package usecase

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"kii.com/internal/domain/entity"
)

// mockWebhookValidator is a mock implementation of WebhookValidator
type mockWebhookValidator struct {
	validateFunc func(ctx context.Context, r *http.Request, body []byte) error
}

func (m *mockWebhookValidator) ValidateRequest(ctx context.Context, r *http.Request, body []byte) error {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, r, body)
	}
	return nil
}

// mockWebhookRepository is a mock implementation of LedgerRepository
type mockWebhookRepository struct {
	addEntryFunc   func(ctx context.Context, entry entity.LedgerEntry) error
	getBalanceFunc func(ctx context.Context, user string) (*entity.BalanceResponse, error)
}

func (m *mockWebhookRepository) AddEntry(ctx context.Context, entry entity.LedgerEntry) error {
	if m.addEntryFunc != nil {
		return m.addEntryFunc(ctx, entry)
	}
	return nil
}

func (m *mockWebhookRepository) GetBalance(ctx context.Context, user string) (*entity.BalanceResponse, error) {
	if m.getBalanceFunc != nil {
		return m.getBalanceFunc(ctx, user)
	}
	return &entity.BalanceResponse{User: user, Balances: make(map[string]string)}, nil
}

func TestProcessWebhookUseCase_Execute(t *testing.T) {
	tests := []struct {
		name            string
		request         ProcessWebhookRequest
		validatorError  error
		repositoryError error
		wantErr         bool
		errContains     string
	}{
		{
			name: "valid webhook request",
			request: ProcessWebhookRequest{
				WebhookRequest: &entity.WebhookRequest{
					User:   "user1",
					Asset:  "BTC",
					Amount: "100.5",
				},
			},
			wantErr: false,
		},
		{
			name: "missing user",
			request: ProcessWebhookRequest{
				WebhookRequest: &entity.WebhookRequest{
					User:   "",
					Asset:  "BTC",
					Amount: "100.5",
				},
			},
			wantErr:     true,
			errContains: "missing required field: user",
		},
		{
			name: "missing asset",
			request: ProcessWebhookRequest{
				WebhookRequest: &entity.WebhookRequest{
					User:   "user1",
					Asset:  "",
					Amount: "100.5",
				},
			},
			wantErr:     true,
			errContains: "missing required field: asset",
		},
		{
			name: "missing amount",
			request: ProcessWebhookRequest{
				WebhookRequest: &entity.WebhookRequest{
					User:   "user1",
					Asset:  "BTC",
					Amount: "",
				},
			},
			wantErr:     true,
			errContains: "missing required field: amount",
		},
		{
			name: "repository error",
			request: ProcessWebhookRequest{
				WebhookRequest: &entity.WebhookRequest{
					User:   "user1",
					Asset:  "BTC",
					Amount: "100.5",
				},
			},
			repositoryError: errors.New("repository error"),
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := &mockWebhookValidator{
				validateFunc: func(ctx context.Context, r *http.Request, body []byte) error {
					return tt.validatorError
				},
			}

			repository := &mockWebhookRepository{
				addEntryFunc: func(ctx context.Context, entry entity.LedgerEntry) error {
					return tt.repositoryError
				},
			}

			useCase := NewProcessWebhookUseCase(validator, repository)
			err := useCase.Execute(context.Background(), tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessWebhookUseCase.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" && err != nil {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("ProcessWebhookUseCase.Execute() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
