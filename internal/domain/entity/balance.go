package entity

// BalanceResponse represents the balance response for a user
type BalanceResponse struct {
	User     string            `json:"user"`
	Balances map[string]string `json:"balances"`
}

// LedgerEntry represents a single ledger entry
type LedgerEntry struct {
	User   string
	Asset  string
	Amount string
}
