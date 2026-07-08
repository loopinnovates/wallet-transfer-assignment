package service

import (
	"context"
	"errors"
	"time"

	"github.com/loopinnovates/wallet-transfer-assignment/internal/domain"
)

type IWalletSvc interface {
	TransferFunds(ctx context.Context, idempotencyKey string, fromWalletID string, toWalletID string, amount float64) (TransactionID string, Status string, timestamp int64, err error)
}

type WalletService struct {
	WalletRepo   domain.IWalletRepository
	TransferRepo domain.ITransferRepository
}

func (s *WalletService) TransferFunds(ctx context.Context, idempotencyKey string, fromWalletID string, toWalletID string, amount float64) (string, string, int64, error) {

	timestamp := time.Now().UnixMilli()

	transfer, err := s.TransferRepo.ExecuteTransfer(ctx, idempotencyKey, fromWalletID, toWalletID, amount, timestamp)
	if err != nil {
		if errors.Is(err, domain.ErrIdempotencyKeyConflict) {
			return s.replayIdempotentTransfer(ctx, idempotencyKey, fromWalletID, toWalletID, amount)
		}
		return "", "", 0, err
	}

	if transfer == nil {
		return "", "", 0, domain.ErrSomethingWentWrong
	}

	if transfer.Status == domain.TransferFailed {
		return "", "", 0, domain.ErrTransferFailed
	}

	return transfer.ID, string(transfer.Status), transfer.CreatedAt.UnixMilli(), nil
}

// replayIdempotentTransfer handles a unique-constraint conflict on
// idempotencyKey. If the retried request matches the original transfer's
// parameters, it returns the original result instead of an error. Otherwise
// the key was reused with different parameters and that is a real conflict.
func (s *WalletService) replayIdempotentTransfer(ctx context.Context, idempotencyKey, fromWalletID, toWalletID string, amount float64) (string, string, int64, error) {
	existing, err := s.TransferRepo.GetByIdempotencyKey(ctx, idempotencyKey)
	if err != nil {
		return "", "", 0, err
	}
	if existing == nil {
		return "", "", 0, domain.ErrIdempotencyKeyConflict
	}

	if existing.FromWalletID != fromWalletID || existing.ToWalletID != toWalletID || existing.Amount != amount {
		return "", "", 0, domain.ErrIdempotencyKeyConflict
	}

	if existing.Status == domain.TransferFailed {
		return "", "", 0, domain.ErrTransferFailed
	}

	return existing.ID, string(existing.Status), existing.CreatedAt.UnixMilli(), nil
}
