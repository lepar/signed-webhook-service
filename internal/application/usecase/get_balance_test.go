package usecase

import (
	"context"
	"errors"
	"testing"

	"kii.com/internal/domain/entity"
)

// mockBalanceRepository is a mock implementation of LedgerRepository
type mockBalanceRepository struct {
	getBalanceFunc func(ctx context.Context, user string) (*entity.BalanceResponse, error)
}

func (m *mockBalanceRepository) AddEntry(ctx context.Context, entry entity.LedgerEntry) error {
	return nil
}

func (m *mockBalanceRepository) GetBalance(ctx context.Context, user string) (*entity.BalanceResponse, error) {
	if m.getBalanceFunc != nil {
		return m.getBalanceFunc(ctx, user)
	}
	return &entity.BalanceResponse{User: user, Balances: make(map[string]string)}, nil
}

func TestGetBalanceUseCase_Execute(t *testing.T) {
	tests := []struct {
		name          string
		user          string
		repositoryRes *entity.BalanceResponse
		repositoryErr error
		wantErr       bool
		wantUser      string
		wantBalances  map[string]string
	}{
		{
			name: "successful balance retrieval",
			user: "user1",
			repositoryRes: &entity.BalanceResponse{
				User: "user1",
				Balances: map[string]string{
					"BTC": "100.5",
					"ETH": "50.25",
				},
			},
			wantErr:  false,
			wantUser: "user1",
			wantBalances: map[string]string{
				"BTC": "100.5",
				"ETH": "50.25",
			},
		},
		{
			name: "user with no balances",
			user: "user2",
			repositoryRes: &entity.BalanceResponse{
				User:     "user2",
				Balances: make(map[string]string),
			},
			wantErr:      false,
			wantUser:     "user2",
			wantBalances: make(map[string]string),
		},
		{
			name:          "repository error",
			user:          "user3",
			repositoryErr: errors.New("repository error"),
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &mockBalanceRepository{
				getBalanceFunc: func(ctx context.Context, user string) (*entity.BalanceResponse, error) {
					return tt.repositoryRes, tt.repositoryErr
				},
			}

			useCase := NewGetBalanceUseCase(repository)
			result, err := useCase.Execute(context.Background(), tt.user)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetBalanceUseCase.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result.User != tt.wantUser {
					t.Errorf("Result.User = %v, want %v", result.User, tt.wantUser)
				}
				if len(result.Balances) != len(tt.wantBalances) {
					t.Errorf("Result.Balances length = %v, want %v", len(result.Balances), len(tt.wantBalances))
				}
				for asset, balance := range tt.wantBalances {
					if result.Balances[asset] != balance {
						t.Errorf("Result.Balances[%v] = %v, want %v", asset, result.Balances[asset], balance)
					}
				}
			}
		})
	}
}
