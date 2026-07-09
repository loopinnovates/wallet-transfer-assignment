package service

import (
	"context"
	"errors"
	"time"

	"github.com/loopinnovates/wallet-transfer-assignment/internal/domain"
	"github.com/loopinnovates/wallet-transfer-assignment/pkg/logger"
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

	logger.Debug().
		Str("idempotency_key", idempotencyKey).
		Str("from_wallet_id", fromWalletID).
		Str("to_wallet_id", toWalletID).
		Float64("amount", amount).
		Msg("transfer funds requested")

	transfer, err := s.TransferRepo.ExecuteTransfer(ctx, idempotencyKey, fromWalletID, toWalletID, amount, timestamp)
	if err != nil {
		if errors.Is(err, domain.ErrIdempotencyKeyConflict) {
			logger.Info().Str("idempotency_key", idempotencyKey).Msg("idempotency key conflict, replaying existing transfer")
			return s.replayIdempotentTransfer(ctx, idempotencyKey, fromWalletID, toWalletID, amount)
		}
		logger.Warn().Err(err).Str("idempotency_key", idempotencyKey).Msg("failed to execute transfer")
		return "", "", 0, err
	}

	if transfer == nil {
		logger.Warn().Str("idempotency_key", idempotencyKey).Msg("execute transfer returned nil transfer")
		return "", "", 0, domain.ErrSomethingWentWrong
	}

	if transfer.Status == domain.TransferFailed {
		logger.Warn().Str("transfer_id", transfer.ID).Msg("transfer failed")
		return "", "", 0, failureReasonError(transfer)
	}

	logger.Info().Str("transfer_id", transfer.ID).Str("status", string(transfer.Status)).Msg("transfer funds completed")
	return transfer.ID, string(transfer.Status), transfer.CreatedAt.UnixMilli(), nil
}

func (s *WalletService) replayIdempotentTransfer(ctx context.Context, idempotencyKey, fromWalletID, toWalletID string, amount float64) (string, string, int64, error) {
	existing, err := s.TransferRepo.GetByIdempotencyKey(ctx, idempotencyKey)
	if err != nil {
		logger.Warn().Err(err).Str("idempotency_key", idempotencyKey).Msg("failed to look up existing transfer for replay")
		return "", "", 0, err
	}
	if existing == nil {
		logger.Warn().Str("idempotency_key", idempotencyKey).Msg("no existing transfer found for idempotency key conflict")
		return "", "", 0, domain.ErrIdempotencyKeyConflict
	}

	if existing.FromWalletID != fromWalletID || existing.ToWalletID != toWalletID || existing.Amount != amount {
		logger.Warn().Str("idempotency_key", idempotencyKey).Msg("idempotency key reused with different transfer parameters")
		return "", "", 0, domain.ErrIdempotencyKeyConflict
	}

	if existing.Status == domain.TransferFailed {
		logger.Info().Str("transfer_id", existing.ID).Msg("replaying previously failed transfer")
		return "", "", 0, failureReasonError(existing)
	}

	logger.Debug().Str("transfer_id", existing.ID).Msg("replayed existing transfer")
	return existing.ID, string(existing.Status), existing.CreatedAt.UnixMilli(), nil
}

func failureReasonError(t *domain.Transfer) error {
	if t.FailureReason != nil && *t.FailureReason == domain.ErrInsufficientBalance.Error() {
		return domain.ErrInsufficientBalance
	}
	return domain.ErrTransferFailed
}
