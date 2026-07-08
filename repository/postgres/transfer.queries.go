package postgres

const (
	queryGetTransferByIdempotencyKey = `
		SELECT id, idempotency_key, from_wallet_id, to_wallet_id, amount, status, failure_reason, created_at, updated_at
		FROM transfers
		WHERE idempotency_key = $1`

	queryInsertTransfer = `
		INSERT INTO transfers (idempotency_key, from_wallet_id, to_wallet_id, amount, status)
		VALUES ($1, $2, $3, $4, 'PROCESSED')
		RETURNING id, idempotency_key, from_wallet_id, to_wallet_id, amount, status, failure_reason, created_at, updated_at`

	queryInsertLedgerEntry = `
		INSERT INTO ledger_entries (transfer_id, wallet_id, entry_type, amount)
		VALUES ($1, $2, $3, $4)`
)
