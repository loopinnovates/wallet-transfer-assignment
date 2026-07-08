package service_test

import (
	"context"
	"testing"

	"github.com/loopinnovates/wallet-transfer-assignment/internal/domain"
	"github.com/loopinnovates/wallet-transfer-assignment/internal/service"
	"github.com/loopinnovates/wallet-transfer-assignment/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestWalletService_TransferFunds(t *testing.T) {
	ctx := context.Background()

	mockWalletRepo := &testutils.MockWalletRepository{}
	mockTransferRepo := &testutils.MockTransferRepository{}

	mockWalletRepo.On("GetByID", ctx, "wallet1").Return(&domain.Wallet{ID: "wallet1", Balance: 200.0}, nil)

	mockWalletRepo.On("GetByID", ctx, "wallet2").Return(&domain.Wallet{ID: "wallet2", Balance: 50.0}, nil)

	mockTransferRepo.On("GetByIdempotencyKey", ctx, "unique-key-123").Return(nil, nil)

	mockTransferRepo.On("ExecuteTransfer", ctx, "unique-key-123", "wallet1", "wallet2", 10000.0, mock.AnythingOfType("int64")).Return(&domain.Transfer{ID: "txn_123", Status: "success"}, nil)

	walletService := service.WalletService{
		WalletRepo:   mockWalletRepo,
		TransferRepo: mockTransferRepo,
	}

	fromWalletID := "wallet1"
	toWalletID := "wallet2"
	amount := 100.0
	idempotencyKey := "unique-key-123"

	transactionID, status, timestamp, err := walletService.TransferFunds(ctx, idempotencyKey, fromWalletID, toWalletID, amount)
	assert.NoError(t, err)
	assert.NotNil(t, transactionID)
	assert.Equal(t, "success", status)
	assert.NotZero(t, timestamp)
}

func TestWalletService_TransferFunds_InsufficientBalance(t *testing.T) {
	ctx := context.Background()

	mockWalletRepo := &testutils.MockWalletRepository{}
	mockTransferRepo := &testutils.MockTransferRepository{}

	mockWalletRepo.On("GetByID", ctx, "wallet1").Return(&domain.Wallet{ID: "wallet1", Balance: 50.0}, nil)

	walletService := service.WalletService{
		WalletRepo:   mockWalletRepo,
		TransferRepo: mockTransferRepo,
	}

	fromWalletID := "wallet1"
	toWalletID := "wallet2"
	amount := 100.0
	idempotencyKey := "unique-key-123"

	transactionID, status, timestamp, err := walletService.TransferFunds(ctx, idempotencyKey, fromWalletID, toWalletID, amount)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrInsufficientBalance, err)
	assert.Empty(t, transactionID)
	assert.Empty(t, status)
	assert.Zero(t, timestamp)
}

func TestWalletService_TransferFunds_SameWallet(t *testing.T) {
	ctx := context.Background()

	mockWalletRepo := &testutils.MockWalletRepository{}
	mockTransferRepo := &testutils.MockTransferRepository{}

	mockWalletRepo.On("GetByID", ctx, "wallet1").Return(&domain.Wallet{ID: "wallet1", Balance: 200.0}, nil)

	walletService := service.WalletService{
		WalletRepo:   mockWalletRepo,
		TransferRepo: mockTransferRepo,
	}

	fromWalletID := "wallet1"
	toWalletID := "wallet1" // Same wallet ID to trigger validation error
	amount := 100.0
	idempotencyKey := "unique-key-123"

	transactionID, status, timestamp, err := walletService.TransferFunds(ctx, idempotencyKey, fromWalletID, toWalletID, amount)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrSameWallet, err)
	assert.Empty(t, transactionID)
	assert.Empty(t, status)
	assert.Zero(t, timestamp)
}

func TestWalletService_TransferFunds_IdempotencyKeyConflict(t *testing.T) {
	ctx := context.Background()

	mockWalletRepo := &testutils.MockWalletRepository{}
	mockTransferRepo := &testutils.MockTransferRepository{}

	mockWalletRepo.On("GetByID", ctx, "wallet1").Return(&domain.Wallet{ID: "wallet1", Balance: 200.0}, nil)

	mockWalletRepo.On("GetByID", ctx, "wallet2").Return(&domain.Wallet{ID: "wallet2", Balance: 50.0}, nil)

	mockTransferRepo.On("GetByIdempotencyKey", ctx, "unique-key-123").Return(&domain.Transfer{ID: "txn_123", Status: "success"}, nil)

	walletService := service.WalletService{
		WalletRepo:   mockWalletRepo,
		TransferRepo: mockTransferRepo,
	}

	fromWalletID := "wallet1"
	toWalletID := "wallet2"
	amount := 100.0
	idempotencyKey := "unique-key-123"

	transactionID, status, timestamp, err := walletService.TransferFunds(ctx, idempotencyKey, fromWalletID, toWalletID, amount)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrIdempotencyKeyConflict, err)
	assert.Empty(t, transactionID)
	assert.Empty(t, status)
	assert.Zero(t, timestamp)
}

func TestWalletService_TransferFunds_TransferExecutionError(t *testing.T) {
	ctx := context.Background()

	mockWalletRepo := &testutils.MockWalletRepository{}
	mockTransferRepo := &testutils.MockTransferRepository{}

	mockWalletRepo.On("GetByID", ctx, "wallet1").Return(&domain.Wallet{ID: "wallet1", Balance: 200.0}, nil)

	mockWalletRepo.On("GetByID", ctx, "wallet2").Return(&domain.Wallet{ID: "wallet2", Balance: 50.0}, nil)

	mockTransferRepo.On("GetByIdempotencyKey", ctx, "unique-key-123").Return(nil, nil)

	mockTransferRepo.On("ExecuteTransfer", ctx, "unique-key-123", "wallet1", "wallet2", 10000.0, mock.AnythingOfType("int64")).Return(nil, domain.ErrSomethingWentWrong)

	walletService := service.WalletService{
		WalletRepo:   mockWalletRepo,
		TransferRepo: mockTransferRepo,
	}

	fromWalletID := "wallet1"
	toWalletID := "wallet2"
	amount := 100.0
	idempotencyKey := "unique-key-123"

	transactionID, status, timestamp, err := walletService.TransferFunds(ctx, idempotencyKey, fromWalletID, toWalletID, amount)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrSomethingWentWrong, err)
	assert.Empty(t, transactionID)
	assert.Empty(t, status)
	assert.Zero(t, timestamp)
}

func TestWalletService_TransferFunds_WalletRepoError(t *testing.T) {
	ctx := context.Background()

	mockWalletRepo := &testutils.MockWalletRepository{}
	mockTransferRepo := &testutils.MockTransferRepository{}

	mockWalletRepo.On("GetByID", ctx, "wallet1").Return(nil, domain.ErrSomethingWentWrong)

	walletService := service.WalletService{
		WalletRepo:   mockWalletRepo,
		TransferRepo: mockTransferRepo,
	}

	fromWalletID := "wallet1"
	toWalletID := "wallet2"
	amount := 100.0
	idempotencyKey := "unique-key-123"

	transactionID, status, timestamp, err := walletService.TransferFunds(ctx, idempotencyKey, fromWalletID, toWalletID, amount)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrSomethingWentWrong, err)
	assert.Empty(t, transactionID)
	assert.Empty(t, status)
	assert.Zero(t, timestamp)
}

func TestWalletService_TransferFunds_TransferRepoError(t *testing.T) {
	ctx := context.Background()

	mockWalletRepo := &testutils.MockWalletRepository{}
	mockTransferRepo := &testutils.MockTransferRepository{}

	mockWalletRepo.On("GetByID", ctx, "wallet1").Return(&domain.Wallet{ID: "wallet1", Balance: 200.0}, nil)

	mockWalletRepo.On("GetByID", ctx, "wallet2").Return(&domain.Wallet{ID: "wallet2", Balance: 50.0}, nil)

	mockTransferRepo.On("GetByIdempotencyKey", ctx, "unique-key-123").Return(nil, domain.ErrSomethingWentWrong)

	walletService := service.WalletService{
		WalletRepo:   mockWalletRepo,
		TransferRepo: mockTransferRepo,
	}

	fromWalletID := "wallet1"
	toWalletID := "wallet2"
	amount := 100.0
	idempotencyKey := "unique-key-123"

	transactionID, status, timestamp, err := walletService.TransferFunds(ctx, idempotencyKey, fromWalletID, toWalletID, amount)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrSomethingWentWrong, err)
	assert.Empty(t, transactionID)
	assert.Empty(t, status)
	assert.Zero(t, timestamp)
}

func TestWalletService_TransferFunds_ExecuteTransferReturnsNil(t *testing.T) {
	ctx := context.Background()

	mockWalletRepo := &testutils.MockWalletRepository{}
	mockTransferRepo := &testutils.MockTransferRepository{}

	mockWalletRepo.On("GetByID", ctx, "wallet1").Return(&domain.Wallet{ID: "wallet1", Balance: 200.0}, nil)

	mockWalletRepo.On("GetByID", ctx, "wallet2").Return(&domain.Wallet{ID: "wallet2", Balance: 50.0}, nil)

	mockTransferRepo.On("GetByIdempotencyKey", ctx, "unique-key-123").Return(nil, nil)

	mockTransferRepo.On("ExecuteTransfer", ctx, "unique-key-123", "wallet1", "wallet2", 10000.0, mock.AnythingOfType("int64")).Return(nil, nil)

	walletService := service.WalletService{
		WalletRepo:   mockWalletRepo,
		TransferRepo: mockTransferRepo,
	}

	fromWalletID := "wallet1"
	toWalletID := "wallet2"
	amount := 100.0
	idempotencyKey := "unique-key-123"

	transactionID, status, timestamp, err := walletService.TransferFunds(ctx, idempotencyKey, fromWalletID, toWalletID, amount)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrSomethingWentWrong, err)
	assert.Empty(t, transactionID)
	assert.Empty(t, status)
	assert.Zero(t, timestamp)
}

func TestWalletService_TransferFunds_ExecuteTransferStatusFailed(t *testing.T) {
	ctx := context.Background()

	mockWalletRepo := &testutils.MockWalletRepository{}
	mockTransferRepo := &testutils.MockTransferRepository{}

	mockWalletRepo.On("GetByID", ctx, "wallet1").Return(&domain.Wallet{ID: "wallet1", Balance: 200.0}, nil)

	mockWalletRepo.On("GetByID", ctx, "wallet2").Return(&domain.Wallet{ID: "wallet2", Balance: 50.0}, nil)

	mockTransferRepo.On("GetByIdempotencyKey", ctx, "unique-key-123").Return(nil, nil)

	mockTransferRepo.On("ExecuteTransfer", ctx, "unique-key-123", "wallet1", "wallet2", 10000.0, mock.AnythingOfType("int64")).Return(&domain.Transfer{ID: "txn_123", Status: "failed"}, nil)

	walletService := service.WalletService{
		WalletRepo:   mockWalletRepo,
		TransferRepo: mockTransferRepo,
	}

	fromWalletID := "wallet1"
	toWalletID := "wallet2"
	amount := 100.0
	idempotencyKey := "unique-key-123"

	transactionID, status, timestamp, err := walletService.TransferFunds(ctx, idempotencyKey, fromWalletID, toWalletID, amount)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrTransferFailed, err)
	assert.Empty(t, transactionID)
	assert.Empty(t, status)
	assert.Zero(t, timestamp)
}

func TestWalletService_TransferFunds_ExecuteTransferStatusPending(t *testing.T) {
	ctx := context.Background()

	mockWalletRepo := &testutils.MockWalletRepository{}
	mockTransferRepo := &testutils.MockTransferRepository{}

	mockWalletRepo.On("GetByID", ctx, "wallet1").Return(&domain.Wallet{ID: "wallet1", Balance: 200.0}, nil)

	mockWalletRepo.On("GetByID", ctx, "wallet2").Return(&domain.Wallet{ID: "wallet2", Balance: 50.0}, nil)

	mockTransferRepo.On("GetByIdempotencyKey", ctx, "unique-key-123").Return(nil, nil)

	mockTransferRepo.On("ExecuteTransfer", ctx, "unique-key-123", "wallet1", "wallet2", 10000.0, mock.AnythingOfType("int64")).Return(&domain.Transfer{ID: "txn_unique-key-123", Status: "pending"}, nil)

	walletService := service.WalletService{
		WalletRepo:   mockWalletRepo,
		TransferRepo: mockTransferRepo,
	}

	fromWalletID := "wallet1"
	toWalletID := "wallet2"
	amount := 100.0
	idempotencyKey := "unique-key-123"

	transactionID, status, timestamp, err := walletService.TransferFunds(ctx, idempotencyKey, fromWalletID, toWalletID, amount)
	assert.NoError(t, err)
	assert.Equal(t, "txn_unique-key-123", transactionID)
	assert.Equal(t, "pending", status)
	assert.NotZero(t, timestamp)
}

func TestWalletService_TransferFunds_GetByIdToWalletError(t *testing.T) {
	ctx := context.Background()

	mockWalletRepo := &testutils.MockWalletRepository{}
	mockTransferRepo := &testutils.MockTransferRepository{}

	mockWalletRepo.On("GetByID", ctx, "wallet1").Return(&domain.Wallet{ID: "wallet1", Balance: 200.0}, nil)

	mockWalletRepo.On("GetByID", ctx, "wallet2").Return(nil, domain.ErrSomethingWentWrong)

	walletService := service.WalletService{
		WalletRepo:   mockWalletRepo,
		TransferRepo: mockTransferRepo,
	}

	fromWalletID := "wallet1"
	toWalletID := "wallet2"
	amount := 100.0
	idempotencyKey := "unique-key-123"

	transactionID, status, timestamp, err := walletService.TransferFunds(ctx, idempotencyKey, fromWalletID, toWalletID, amount)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrSomethingWentWrong, err)
	assert.Empty(t, transactionID)
	assert.Empty(t, status)
	assert.Zero(t, timestamp)
}
