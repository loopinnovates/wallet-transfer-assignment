package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/loopinnovates/wallet-transfer-assignment/internal/domain"
	"github.com/loopinnovates/wallet-transfer-assignment/repository/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A bad wallet ID must fail before any transfer row is committed - not
// leave a stuck PENDING row behind. lockWalletPair inside
// insertPendingTransfer's own transaction maps the missing row to
// ErrWalletNotFound and rolls back before the INSERT ever runs.
func TestExecuteTransfer_UnknownWallet_LeavesNoTransferRow(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewTransferRepository(db, db)
	ctx := context.Background()

	realWallet := createTestWallet(t, db, 100)
	const missingWallet = "00000000-0000-0000-0000-000000000000"
	idempotencyKey := fmt.Sprintf("unknown-wallet-%s", t.Name())

	_, err := repo.ExecuteTransfer(ctx, idempotencyKey, realWallet, missingWallet, 10, 1)
	assert.ErrorIs(t, err, domain.ErrWalletNotFound)

	persisted, err := repo.GetByIdempotencyKey(ctx, idempotencyKey)
	require.NoError(t, err)
	assert.Nil(t, persisted, "no transfer row should exist for a request that never got past wallet verification")
}

// Simulates a process crash between phase 1 (PENDING committed) and phase 2
// (finalize) by inserting a PENDING row directly, bypassing
// executeTransferOnce entirely. A retry with the same idempotency key must
// not silently move funds a second time or treat the stuck row as done -
// it should see the same PENDING row via the unique constraint conflict,
// and a caller replaying it (see wallet.svc.go) gets back status: PENDING
// rather than a false success or a lost request.
func TestExecuteTransfer_StuckPendingRow_IsDedupedNotReprocessed(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewTransferRepository(db, db)
	ctx := context.Background()

	const startBalance = 1000.0
	const transferAmount = 100.0

	fromWallet := createTestWallet(t, db, startBalance)
	toWallet := createTestWallet(t, db, 0)
	idempotencyKey := fmt.Sprintf("stuck-pending-%s", t.Name())

	var transferID string
	err := db.QueryRow(
		`INSERT INTO transfers (idempotency_key, from_wallet_id, to_wallet_id, amount, status)
		 VALUES ($1, $2, $3, $4, 'PENDING') RETURNING id`,
		idempotencyKey, fromWallet, toWallet, transferAmount,
	).Scan(&transferID)
	require.NoError(t, err)

	// A retry with the same key must hit the unique constraint, not
	// silently proceed to debit/credit as if this were a fresh request.
	_, retryErr := repo.ExecuteTransfer(ctx, idempotencyKey, fromWallet, toWallet, transferAmount, 1)
	assert.ErrorIs(t, retryErr, domain.ErrIdempotencyKeyConflict)

	// The stuck row is still exactly PENDING - not silently finalized by
	// the retry, and not something a caller should mistake for success.
	persisted, err := repo.GetByIdempotencyKey(ctx, idempotencyKey)
	require.NoError(t, err)
	require.NotNil(t, persisted)
	assert.Equal(t, domain.TransferPending, persisted.Status)

	// Funds were never moved - the finalize step that debits/credits never
	// ran for this stuck row.
	assert.Equal(t, startBalance, getWalletBalance(t, db, fromWallet))
	assert.Equal(t, 0.0, getWalletBalance(t, db, toWallet))
}
