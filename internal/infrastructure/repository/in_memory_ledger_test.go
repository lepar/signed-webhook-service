package repository

import (
	"context"
	"testing"

	"kii.com/internal/domain/entity"
	"kii.com/internal/infrastructure/logger"
)

func TestInMemoryLedger_AddEntry(t *testing.T) {
	logger := logger.NewLogger()
	ledger := NewInMemoryLedger(logger).(*InMemoryLedger)
	ctx := context.Background()

	tests := []struct {
		name      string
		entry     entity.LedgerEntry
		wantErr   bool
		checkFunc func(*testing.T, *InMemoryLedger)
	}{
		{
			name: "add first entry",
			entry: entity.LedgerEntry{
				User:   "user1",
				Asset:  "BTC",
				Amount: "100.5",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, l *InMemoryLedger) {
				balance, err := l.GetBalance(ctx, "user1")
				if err != nil {
					t.Fatalf("GetBalance() error = %v", err)
				}
				if balance.Balances["BTC"] != "100.50000000" {
					t.Errorf("Balance = %v, want 100.50000000", balance.Balances["BTC"])
				}
			},
		},
		{
			name: "add to existing balance",
			entry: entity.LedgerEntry{
				User:   "user1",
				Asset:  "BTC",
				Amount: "50.25",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, l *InMemoryLedger) {
				balance, err := l.GetBalance(ctx, "user1")
				if err != nil {
					t.Fatalf("GetBalance() error = %v", err)
				}
				if balance.Balances["BTC"] != "150.75000000" {
					t.Errorf("Balance = %v, want 150.75000000", balance.Balances["BTC"])
				}
			},
		},
		{
			name: "add different asset",
			entry: entity.LedgerEntry{
				User:   "user1",
				Asset:  "ETH",
				Amount: "200.75",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, l *InMemoryLedger) {
				balance, err := l.GetBalance(ctx, "user1")
				if err != nil {
					t.Fatalf("GetBalance() error = %v", err)
				}
				if balance.Balances["ETH"] != "200.75000000" {
					t.Errorf("Balance = %v, want 200.75000000", balance.Balances["ETH"])
				}
				// BTC balance should still exist
				if balance.Balances["BTC"] != "150.75000000" {
					t.Errorf("BTC Balance = %v, want 150.75000000", balance.Balances["BTC"])
				}
			},
		},
		{
			name: "add to different user",
			entry: entity.LedgerEntry{
				User:   "user2",
				Asset:  "BTC",
				Amount: "75.0",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, l *InMemoryLedger) {
				balance, err := l.GetBalance(ctx, "user2")
				if err != nil {
					t.Fatalf("GetBalance() error = %v", err)
				}
				if balance.Balances["BTC"] != "75.00000000" {
					t.Errorf("Balance = %v, want 75.00000000", balance.Balances["BTC"])
				}
				// user1 balances should be unchanged
				balance1, _ := l.GetBalance(ctx, "user1")
				if balance1.Balances["BTC"] != "150.75000000" {
					t.Errorf("user1 BTC Balance = %v, want 150.75000000", balance1.Balances["BTC"])
				}
			},
		},
		{
			name: "invalid amount format",
			entry: entity.LedgerEntry{
				User:   "user1",
				Asset:  "BTC",
				Amount: "invalid",
			},
			wantErr: true,
		},
		{
			name: "negative amount",
			entry: entity.LedgerEntry{
				User:   "user1",
				Asset:  "BTC",
				Amount: "-50.0",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, l *InMemoryLedger) {
				balance, err := l.GetBalance(ctx, "user1")
				if err != nil {
					t.Fatalf("GetBalance() error = %v", err)
				}
				// Should handle negative amounts (subtraction)
				if balance.Balances["BTC"] != "100.75000000" {
					t.Errorf("Balance = %v, want 100.75000000", balance.Balances["BTC"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ledger.AddEntry(ctx, tt.entry)
			if (err != nil) != tt.wantErr {
				t.Errorf("InMemoryLedger.AddEntry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkFunc != nil {
				tt.checkFunc(t, ledger)
			}
		})
	}
}

func TestInMemoryLedger_GetBalance(t *testing.T) {
	logger := logger.NewLogger()
	ledger := NewInMemoryLedger(logger).(*InMemoryLedger)
	ctx := context.Background()

	// Add some entries
	ledger.AddEntry(ctx, entity.LedgerEntry{User: "user1", Asset: "BTC", Amount: "100.5"})
	ledger.AddEntry(ctx, entity.LedgerEntry{User: "user1", Asset: "ETH", Amount: "50.25"})

	tests := []struct {
		name     string
		user     string
		wantUser string
		wantLen  int
	}{
		{
			name:     "existing user",
			user:     "user1",
			wantUser: "user1",
			wantLen:  2,
		},
		{
			name:     "non-existent user",
			user:     "user999",
			wantUser: "user999",
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balance, err := ledger.GetBalance(ctx, tt.user)
			if err != nil {
				t.Errorf("InMemoryLedger.GetBalance() error = %v", err)
				return
			}
			if balance.User != tt.wantUser {
				t.Errorf("Balance.User = %v, want %v", balance.User, tt.wantUser)
			}
			if len(balance.Balances) != tt.wantLen {
				t.Errorf("Balance.Balances length = %v, want %v", len(balance.Balances), tt.wantLen)
			}
		})
	}
}

func TestInMemoryLedger_DecimalPrecision(t *testing.T) {
	logger := logger.NewLogger()
	ledger := NewInMemoryLedger(logger).(*InMemoryLedger)
	ctx := context.Background()

	tests := []struct {
		name     string
		entries  []entity.LedgerEntry
		expected string
	}{
		{
			name: "small decimal amounts",
			entries: []entity.LedgerEntry{
				{User: "user1", Asset: "BTC", Amount: "0.00000001"},
				{User: "user1", Asset: "BTC", Amount: "0.00000002"},
			},
			expected: "0.00000003",
		},
		{
			name: "large amounts with decimals",
			entries: []entity.LedgerEntry{
				{User: "user2", Asset: "BTC", Amount: "999999.99999999"},
				{User: "user2", Asset: "BTC", Amount: "0.00000001"},
			},
			expected: "1000000.00000000",
		},
		{
			name: "multiple decimal places",
			entries: []entity.LedgerEntry{
				{User: "user3", Asset: "BTC", Amount: "1.23456789"},
				{User: "user3", Asset: "BTC", Amount: "2.34567890"},
			},
			expected: "3.58024679", // Actual result due to float precision
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset ledger for each test
			ledger = NewInMemoryLedger(logger).(*InMemoryLedger)

			for _, entry := range tt.entries {
				if err := ledger.AddEntry(ctx, entry); err != nil {
					t.Fatalf("AddEntry() error = %v", err)
				}
			}

			balance, err := ledger.GetBalance(ctx, tt.entries[0].User)
			if err != nil {
				t.Fatalf("GetBalance() error = %v", err)
			}

			actual := balance.Balances[tt.entries[0].Asset]
			if actual != tt.expected {
				t.Errorf("Balance = %v, want %v", actual, tt.expected)
			}
		})
	}
}

func TestInMemoryLedger_ConcurrentAccess(t *testing.T) {
	logger := logger.NewLogger()
	ledger := NewInMemoryLedger(logger).(*InMemoryLedger)
	ctx := context.Background()

	// Test concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			entry := entity.LedgerEntry{
				User:   "user1",
				Asset:  "BTC",
				Amount: "1.0",
			}
			ledger.AddEntry(ctx, entry)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final balance
	balance, err := ledger.GetBalance(ctx, "user1")
	if err != nil {
		t.Fatalf("GetBalance() error = %v", err)
	}

	expected := "10.00000000"
	if balance.Balances["BTC"] != expected {
		t.Errorf("Balance = %v, want %v", balance.Balances["BTC"], expected)
	}
}
