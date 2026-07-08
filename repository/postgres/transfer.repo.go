package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"github.com/lib/pq"
	"github.com/loopinnovates/wallet-transfer-assignment/internal/domain"
	"github.com/loopinnovates/wallet-transfer-assignment/repository/postgres/sqlcgen"
)

const uniqueViolationCode = "23505"

type TransferRepository struct {
	writeDB      *sql.DB
	readQueries  *sqlcgen.Queries
	writeQueries *sqlcgen.Queries
}

func NewTransferRepository(readDB *sql.DB, writeDB *sql.DB) *TransferRepository {
	return &TransferRepository{
		writeDB:      writeDB,
		readQueries:  sqlcgen.New(readDB),
		writeQueries: sqlcgen.New(writeDB),
	}
}

func (r *TransferRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Transfer, error) {
	t, err := r.readQueries.GetTransferByIdempotencyKey(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toDomainTransfer(t), nil
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
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			fmt.Printf("Error in rolling back transaction: %v", err)
		}
	}()

	qtx := r.writeQueries.WithTx(tx)

	if err := lockWalletPair(ctx, qtx, fromWalletID, toWalletID); err != nil {
		return nil, err
	}
	if err := moveFunds(ctx, qtx, fromWalletID, toWalletID, amount); err != nil {
		return nil, err
	}

	t, err := insertTransferWithLedger(ctx, qtx, idempotencyKey, fromWalletID, toWalletID, amount)
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
func lockWalletPair(ctx context.Context, qtx *sqlcgen.Queries, walletA, walletB string) error {
	sortedIDs := []string{walletA, walletB}
	sort.Strings(sortedIDs)
	if err := lockWallet(ctx, qtx, sortedIDs[0]); err != nil {
		return err
	}
	return lockWallet(ctx, qtx, sortedIDs[1])
}

// lockWallet takes a row lock on the wallet with the given ID.
func lockWallet(ctx context.Context, qtx *sqlcgen.Queries, walletID string) error {
	_, err := qtx.LockWalletByID(ctx, walletID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrWalletNotFound
	}
	return err
}

// moveFunds checks the source balance and applies the debit/credit. Callers
// must hold row locks on both wallets before calling this.
func moveFunds(ctx context.Context, qtx *sqlcgen.Queries, fromWalletID, toWalletID string, amount float64) error {
	fromBalance, err := qtx.GetWalletBalance(ctx, fromWalletID)
	if err != nil {
		return err
	}
	if fromBalance < amount {
		return domain.ErrInsufficientBalance
	}

	if err := qtx.DebitWallet(ctx, sqlcgen.DebitWalletParams{Balance: amount, ID: fromWalletID}); err != nil {
		return err
	}
	if err := qtx.CreditWallet(ctx, sqlcgen.CreditWalletParams{Balance: amount, ID: toWalletID}); err != nil {
		return err
	}
	return nil
}

func insertTransferWithLedger(ctx context.Context, qtx *sqlcgen.Queries, idempotencyKey, fromWalletID, toWalletID string, amount float64) (*domain.Transfer, error) {
	t, err := qtx.InsertTransfer(ctx, sqlcgen.InsertTransferParams{
		IdempotencyKey: idempotencyKey,
		FromWalletID:   fromWalletID,
		ToWalletID:     toWalletID,
		Amount:         amount,
	})
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == uniqueViolationCode {
			return nil, domain.ErrIdempotencyKeyConflict
		}
		return nil, err
	}

	if err := qtx.InsertLedgerEntry(ctx, sqlcgen.InsertLedgerEntryParams{
		TransferID: t.ID,
		WalletID:   fromWalletID,
		EntryType:  string(domain.LedgerDebit),
		Amount:     amount,
	}); err != nil {
		return nil, err
	}
	if err := qtx.InsertLedgerEntry(ctx, sqlcgen.InsertLedgerEntryParams{
		TransferID: t.ID,
		WalletID:   toWalletID,
		EntryType:  string(domain.LedgerCredit),
		Amount:     amount,
	}); err != nil {
		return nil, err
	}

	return toDomainTransfer(t), nil
}

func toDomainTransfer(t sqlcgen.Transfer) *domain.Transfer {
	var failureReason *string
	if t.FailureReason.Valid {
		failureReason = &t.FailureReason.String
	}

	return &domain.Transfer{
		ID:             t.ID,
		IdempotencyKey: t.IdempotencyKey,
		FromWalletID:   t.FromWalletID,
		ToWalletID:     t.ToWalletID,
		Amount:         t.Amount,
		Status:         domain.TransferStatus(t.Status),
		FailureReason:  failureReason,
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
	}
}
