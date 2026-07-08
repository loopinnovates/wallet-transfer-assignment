package domain

import "time"

// Wallet.Balance is stored in the smallest currency unit (cents) to avoid
// float precision issues, matching the DOUBLE PRECISION column in the wallets table.
type Wallet struct {
	ID        string
	OwnerName string
	Balance   float64
	CreatedAt time.Time
	UpdatedAt time.Time
}
