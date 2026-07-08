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
	return withRetry(ctx, func() (*domain.Transfer, error) {
		t, err := r.readQueries.GetTransferByIdempotencyKey(ctx, key)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		return toDomainTransfer(t), nil
	})
}

func (r *TransferRepository) ListPendingTransfers(ctx context.Context) ([]*domain.Transfer, error) {
	return withRetry(ctx, func() ([]*domain.Transfer, error) {
		rows, err := r.readQueries.ListPendingTransfers(ctx)
		if err != nil {
			return nil, err
		}
		transfers := make([]*domain.Transfer, len(rows))
		for i, row := range rows {
			transfers[i] = toDomainTransfer(row)
		}
		return transfers, nil
	})
}

func (r *TransferRepository) ResolvePendingTransfer(ctx context.Context, transfer *domain.Transfer) (*domain.Transfer, error) {
	return withRetry(ctx, func() (*domain.Transfer, error) {
		return r.finalizePendingTransfer(ctx, transfer.ID, transfer.FromWalletID, transfer.ToWalletID, transfer.Amount)
	})
}

func (r *TransferRepository) ExecuteTransfer(ctx context.Context, idempotencyKey, fromWalletID, toWalletID string, amount float64, timestamp int64) (*domain.Transfer, error) {
	return withRetry(ctx, func() (*domain.Transfer, error) {
		return r.executeTransferOnce(ctx, idempotencyKey, fromWalletID, toWalletID, amount, timestamp)
	})
}

func (r *TransferRepository) executeTransferOnce(ctx context.Context, idempotencyKey, fromWalletID, toWalletID string, amount float64, timestamp int64) (*domain.Transfer, error) {
	pending, err := insertPendingTransfer(ctx, r.writeDB, r.writeQueries, idempotencyKey, fromWalletID, toWalletID, amount)
	if err != nil {
		return nil, err
	}
	return r.finalizePendingTransfer(ctx, pending.ID, fromWalletID, toWalletID, amount)
}

func (r *TransferRepository) finalizePendingTransfer(ctx context.Context, transferID, fromWalletID, toWalletID string, amount float64) (*domain.Transfer, error) {
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

	current, err := qtx.GetTransferByID(ctx, transferID)
	if err != nil {
		return nil, err
	}
	if current.Status != string(domain.TransferPending) {
		return toDomainTransfer(current), nil
	}

	result, err := r.moveAndFinalize(ctx, qtx, transferID, fromWalletID, toWalletID, amount)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *TransferRepository) moveAndFinalize(ctx context.Context, qtx *sqlcgen.Queries, transferID, fromWalletID, toWalletID string, amount float64) (*domain.Transfer, error) {
	moveErr := moveFunds(ctx, qtx, fromWalletID, toWalletID, amount)
	if moveErr == nil {
		processed, err := qtx.MarkTransferProcessed(ctx, transferID)
		if err != nil {
			return nil, err
		}
		if err := insertLedgerEntries(ctx, qtx, processed.ID, fromWalletID, toWalletID, amount); err != nil {
			return nil, err
		}
		return toDomainTransfer(processed), nil
	}
	if !errors.Is(moveErr, domain.ErrInsufficientBalance) {
		return nil, moveErr
	}

	failed, err := qtx.MarkTransferFailed(ctx, sqlcgen.MarkTransferFailedParams{
		ID:            transferID,
		FailureReason: sql.NullString{String: moveErr.Error(), Valid: true},
	})
	if err != nil {
		return nil, err
	}
	return toDomainTransfer(failed), nil
}

func insertPendingTransfer(ctx context.Context, db *sql.DB, q *sqlcgen.Queries, idempotencyKey, fromWalletID, toWalletID string, amount float64) (sqlcgen.Transfer, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return sqlcgen.Transfer{}, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			fmt.Printf("Error in rolling back transaction: %v", err)
		}
	}()

	qtx := q.WithTx(tx)

	if err := lockWalletPair(ctx, qtx, fromWalletID, toWalletID); err != nil {
		return sqlcgen.Transfer{}, err
	}

	t, err := qtx.InsertPendingTransfer(ctx, sqlcgen.InsertPendingTransferParams{
		IdempotencyKey: idempotencyKey,
		FromWalletID:   fromWalletID,
		ToWalletID:     toWalletID,
		Amount:         amount,
	})
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == uniqueViolationCode {
			return sqlcgen.Transfer{}, domain.ErrIdempotencyKeyConflict
		}
		return sqlcgen.Transfer{}, err
	}

	if err := tx.Commit(); err != nil {
		return sqlcgen.Transfer{}, err
	}
	return t, nil
}

func lockWalletPair(ctx context.Context, qtx *sqlcgen.Queries, walletA, walletB string) error {
	sortedIDs := []string{walletA, walletB}
	sort.Strings(sortedIDs)
	if err := lockWallet(ctx, qtx, sortedIDs[0]); err != nil {
		return err
	}
	return lockWallet(ctx, qtx, sortedIDs[1])
}

func lockWallet(ctx context.Context, qtx *sqlcgen.Queries, walletID string) error {
	_, err := qtx.LockWalletByID(ctx, walletID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrWalletNotFound
	}
	return err
}

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

func insertLedgerEntries(ctx context.Context, qtx *sqlcgen.Queries, transferID, fromWalletID, toWalletID string, amount float64) error {
	if err := qtx.InsertLedgerEntry(ctx, sqlcgen.InsertLedgerEntryParams{
		TransferID: transferID,
		WalletID:   fromWalletID,
		EntryType:  string(domain.LedgerDebit),
		Amount:     amount,
	}); err != nil {
		return err
	}
	return qtx.InsertLedgerEntry(ctx, sqlcgen.InsertLedgerEntryParams{
		TransferID: transferID,
		WalletID:   toWalletID,
		EntryType:  string(domain.LedgerCredit),
		Amount:     amount,
	})
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
