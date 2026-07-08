package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"github.com/lib/pq"
	"github.com/loopinnovates/wallet-transfer-assignment/internal/domain"
)

const uniqueViolationCode = "23505"

type TransferRepository struct {
	readDB  *sql.DB
	writeDB *sql.DB
}

func NewTransferRepository(readDB *sql.DB, writeDB *sql.DB) *TransferRepository {
	return &TransferRepository{readDB: readDB, writeDB: writeDB}
}

func (r *TransferRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Transfer, error) {
	t, err := scanTransfer(r.readDB.QueryRowContext(ctx, queryGetTransferByIdempotencyKey, key))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

// ExecuteTransfer atomically debits fromWalletID, credits toWalletID, and
// records the transfer plus its ledger entries in a single DB transaction.
// Wallet rows are locked in a fixed ID order to avoid deadlocks between
// concurrent transfers that touch the same pair of wallets in reverse order.
func (r *TransferRepository) ExecuteTransfer(ctx context.Context, idempotencyKey, fromWalletID, toWalletID string, amount float64, timestamp int64) (*domain.Transfer, error) {

	tx, err := r.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil {
			fmt.Printf("Error in rolling back transaction: %v", err)
		}
	}()

	if err := lockWalletPair(ctx, tx, fromWalletID, toWalletID); err != nil {
		return nil, err
	}
	if err := moveFunds(ctx, tx, fromWalletID, toWalletID, amount); err != nil {
		return nil, err
	}

	t, err := insertTransferWithLedger(ctx, tx, idempotencyKey, fromWalletID, toWalletID, amount)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return t, nil
}

// lockWalletPair takes row locks on both wallets in a fixed ID order to
// avoid deadlocks between concurrent transfers touching the same pair.
func lockWalletPair(ctx context.Context, tx *sql.Tx, walletA, walletB string) error {
	sortedIDs := []string{walletA, walletB}
	sort.Strings(sortedIDs)
	if err := lockWallet(ctx, tx, sortedIDs[0]); err != nil {
		return err
	}
	return lockWallet(ctx, tx, sortedIDs[1])
}

// lockWallet takes a row lock on the wallet with the given ID.
func lockWallet(ctx context.Context, tx *sql.Tx, walletID string) error {
	var id string
	err := tx.QueryRowContext(ctx, queryLockWalletByID, walletID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrWalletNotFound
	}
	return err
}

// moveFunds checks the source balance and applies the debit/credit. Callers
// must hold row locks on both wallets before calling this.
func moveFunds(ctx context.Context, tx *sql.Tx, fromWalletID, toWalletID string, amount float64) error {
	var fromBalance float64
	if err := tx.QueryRowContext(ctx, queryGetWalletBalance, fromWalletID).Scan(&fromBalance); err != nil {
		return err
	}
	if fromBalance < amount {
		return domain.ErrInsufficientBalance
	}

	if _, err := tx.ExecContext(ctx, queryDebitWallet, amount, fromWalletID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, queryCreditWallet, amount, toWalletID); err != nil {
		return err
	}
	return nil
}

func insertTransferWithLedger(ctx context.Context, tx *sql.Tx, idempotencyKey, fromWalletID, toWalletID string, amount float64) (*domain.Transfer, error) {
	t, err := scanTransfer(tx.QueryRowContext(ctx, queryInsertTransfer, idempotencyKey, fromWalletID, toWalletID, amount))
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == uniqueViolationCode {
			return nil, domain.ErrIdempotencyKeyConflict
		}
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, queryInsertLedgerEntry, t.ID, fromWalletID, domain.LedgerDebit, amount); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, queryInsertLedgerEntry, t.ID, toWalletID, domain.LedgerCredit, amount); err != nil {
		return nil, err
	}
	return t, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTransfer(row rowScanner) (*domain.Transfer, error) {
	var t domain.Transfer
	err := row.Scan(
		&t.ID, &t.IdempotencyKey, &t.FromWalletID, &t.ToWalletID,
		&t.Amount, &t.Status, &t.FailureReason, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
