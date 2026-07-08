package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/loopinnovates/wallet-transfer-assignment/internal/domain"
)

type WalletRepository struct {
	readDB  *sql.DB
	writeDB *sql.DB
}

func NewWalletRepository(readDB *sql.DB, writeDB *sql.DB) *WalletRepository {
	return &WalletRepository{readDB: readDB, writeDB: writeDB}
}

func (r *WalletRepository) GetByID(ctx context.Context, id string) (*domain.Wallet, error) {
	var w domain.Wallet
	err := r.readDB.QueryRowContext(ctx, queryGetWalletByID, id).Scan(
		&w.ID, &w.OwnerName, &w.Balance, &w.CreatedAt, &w.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrWalletNotFound
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}
