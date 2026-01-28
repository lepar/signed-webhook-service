package port

import (
	"context"

	"kii.com/internal/domain/entity"
)

// LedgerRepository is the port for ledger operations
type LedgerRepository interface {
	AddEntry(ctx context.Context, entry entity.LedgerEntry) error
	GetBalance(ctx context.Context, user string) (*entity.BalanceResponse, error)
}
