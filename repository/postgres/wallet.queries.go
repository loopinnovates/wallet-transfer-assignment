package postgres

const (
	queryGetWalletByID = `
		SELECT id, owner_name, balance, created_at, updated_at
		FROM wallets
		WHERE id = $1`

	queryLockWalletByID = `SELECT id FROM wallets WHERE id = $1 FOR UPDATE`

	queryGetWalletBalance = `SELECT balance FROM wallets WHERE id = $1`

	queryDebitWallet = `UPDATE wallets SET balance = balance - $1, updated_at = now() WHERE id = $2`

	queryCreditWallet = `UPDATE wallets SET balance = balance + $1, updated_at = now() WHERE id = $2`
)
