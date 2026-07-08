package service

import (
	"context"
	"fmt"
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
	fromWallet, err := s.WalletRepo.GetByID(ctx, fromWalletID)
	if err != nil {
		return "", "", 0, err
	}

	if fromWallet.Balance < amount {
		return "", "", 0, domain.ErrInsufficientBalance
	}

	toWallet, err := s.WalletRepo.GetByID(ctx, toWalletID)
	if err != nil {
		fmt.Println("Error retrieving toWallet:", err)
		return "", "", 0, err
	}

	if toWallet.ID == fromWallet.ID {
		return "", "", 0, domain.ErrSameWallet
	}

	existingTransaction, err := s.TransferRepo.GetByIdempotencyKey(ctx, idempotencyKey)
	if err != nil {
		return "", "", 0, err
	}

	if existingTransaction != nil {
		return "", "", 0, domain.ErrIdempotencyKeyConflict
	}

	timestamp := time.Now().UnixMilli()

	transfer, err := s.TransferRepo.ExecuteTransfer(ctx, idempotencyKey, fromWalletID, toWalletID, amount*100, timestamp)
	if err != nil {
		return "", "", 0, err
	}

	if transfer == nil {
		return "", "", 0, domain.ErrSomethingWentWrong
	}

	if transfer.Status == "failed" {
		return "", "", 0, domain.ErrTransferFailed
	}

	return transfer.ID, string(transfer.Status), timestamp, nil
}
