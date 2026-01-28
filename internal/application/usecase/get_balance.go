package usecase

import (
	"context"

	"kii.com/internal/domain/entity"
	"kii.com/internal/domain/port"
)

// GetBalanceUseCase handles balance retrieval
type GetBalanceUseCase struct {
	repository port.LedgerRepository
}

// NewGetBalanceUseCase creates a new GetBalanceUseCase
func NewGetBalanceUseCase(repository port.LedgerRepository) *GetBalanceUseCase {
	return &GetBalanceUseCase{
		repository: repository,
	}
}

// Execute retrieves the balance for a user
func (uc *GetBalanceUseCase) Execute(ctx context.Context, user string) (*entity.BalanceResponse, error) {
	return uc.repository.GetBalance(ctx, user)
}
