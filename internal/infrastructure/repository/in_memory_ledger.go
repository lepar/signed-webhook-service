package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/shopspring/decimal"

	"kii.com/internal/domain/entity"
	"kii.com/internal/domain/port"
	"kii.com/internal/infrastructure/logger"
)

// InMemoryLedger implements the LedgerRepository port
type InMemoryLedger struct {
	mu       sync.RWMutex
	balances map[string]map[string]string 
	entries  []entity.LedgerEntry         
	logger   logger.Logger
}

// NewInMemoryLedger creates a new in-memory ledger
func NewInMemoryLedger(logger logger.Logger) port.LedgerRepository {
	return &InMemoryLedger{
		balances: make(map[string]map[string]string),
		entries:  make([]entity.LedgerEntry, 0),
		logger:   logger,
	}
}

// AddEntry adds a ledger entry and updates the balance
func (l *InMemoryLedger) AddEntry(ctx context.Context, entry entity.LedgerEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Initialize user balance map if it doesn't exist
	if l.balances[entry.User] == nil {
		l.balances[entry.User] = make(map[string]string)
	}

	// Get current balance (default to "0")
	currentBalance := l.balances[entry.User][entry.Asset]
	if currentBalance == "" {
		currentBalance = "0"
	}
 
	// Parse and add amounts as strings to maintain precision
	newBalance, err := addDecimalStrings(currentBalance, entry.Amount)
	if err != nil {
		l.logger.LogError(ctx, "Failed to add balance", err,
			"user", entry.User,
			"asset", entry.Asset,
			"current", currentBalance,
			"amount", entry.Amount)
		return fmt.Errorf("invalid amount format: %w", err)
	}

	// Update balance
	l.balances[entry.User][entry.Asset] = newBalance

	// Add to audit trail
	l.entries = append(l.entries, entry)

	l.logger.LogInfo(ctx, "Balance updated",
		"user", entry.User,
		"asset", entry.Asset,
		"amount", entry.Amount,
		"new_balance", newBalance)

	return nil
}

// GetBalance returns the balance for a specific user
func (l *InMemoryLedger) GetBalance(ctx context.Context, user string) (*entity.BalanceResponse, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	userBalances := l.balances[user]
	if userBalances == nil {
		userBalances = make(map[string]string)
	}

	// Create a copy to avoid race conditions
	balancesCopy := make(map[string]string)
	for asset, balance := range userBalances {
		balancesCopy[asset] = balance
	}

	return &entity.BalanceResponse{
		User:     user,
		Balances: balancesCopy,
	}, nil
}

// addDecimalStrings adds two decimal strings while maintaining precision
// using the shopspring/decimal library to avoid floating point rounding issues.
func addDecimalStrings(a, b string) (string, error) {
	if a == "" {
		a = "0"
	}
	if b == "" {
		b = "0"
	}

	aDec, err := decimal.NewFromString(a)
	if err != nil {
		return "", fmt.Errorf("invalid decimal string: %s", a)
	}

	bDec, err := decimal.NewFromString(b)
	if err != nil {
		return "", fmt.Errorf("invalid decimal string: %s", b)
	}

	result := aDec.Add(bDec)

	return result.StringFixed(8), nil
}
