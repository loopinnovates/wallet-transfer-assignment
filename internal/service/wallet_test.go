package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/loopinnovates/wallet-transfer-assignment/internal/domain"
	"github.com/loopinnovates/wallet-transfer-assignment/internal/service"
	"github.com/loopinnovates/wallet-transfer-assignment/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newWalletService() (*service.WalletService, *testutils.MockWalletRepository, *testutils.MockTransferRepository) {
	mockWalletRepo := &testutils.MockWalletRepository{}
	mockTransferRepo := &testutils.MockTransferRepository{}
	return &service.WalletService{
		WalletRepo:   mockWalletRepo,
		TransferRepo: mockTransferRepo,
	}, mockWalletRepo, mockTransferRepo
}

func TestWalletService_TransferFunds_Success(t *testing.T) {
	ctx := context.Background()
	walletService, _, mockTransferRepo := newWalletService()

	mockTransferRepo.On("ExecuteTransfer", ctx, "unique-key-123", "wallet1", "wallet2", 100.0, mock.AnythingOfType("int64")).
		Return(&domain.Transfer{ID: "txn_123", Status: domain.TransferProcessed}, nil)

	transactionID, status, timestamp, err := walletService.TransferFunds(ctx, "unique-key-123", "wallet1", "wallet2", 100.0)

	assert.NoError(t, err)
	assert.Equal(t, "txn_123", transactionID)
	assert.Equal(t, string(domain.TransferProcessed), status)
	assert.NotZero(t, timestamp)
	mockTransferRepo.AssertExpectations(t)
}

func TestWalletService_TransferFunds_Pending(t *testing.T) {
	ctx := context.Background()
	walletService, _, mockTransferRepo := newWalletService()

	mockTransferRepo.On("ExecuteTransfer", ctx, "unique-key-123", "wallet1", "wallet2", 100.0, mock.AnythingOfType("int64")).
		Return(&domain.Transfer{ID: "txn_123", Status: domain.TransferPending}, nil)

	transactionID, status, timestamp, err := walletService.TransferFunds(ctx, "unique-key-123", "wallet1", "wallet2", 100.0)

	assert.NoError(t, err)
	assert.Equal(t, "txn_123", transactionID)
	assert.Equal(t, string(domain.TransferPending), status)
	assert.NotZero(t, timestamp)
}

func TestWalletService_TransferFunds_ExecuteTransferStatusFailed(t *testing.T) {
	ctx := context.Background()
	walletService, _, mockTransferRepo := newWalletService()

	mockTransferRepo.On("ExecuteTransfer", ctx, "unique-key-123", "wallet1", "wallet2", 100.0, mock.AnythingOfType("int64")).
		Return(&domain.Transfer{ID: "txn_123", Status: domain.TransferFailed}, nil)

	transactionID, status, timestamp, err := walletService.TransferFunds(ctx, "unique-key-123", "wallet1", "wallet2", 100.0)

	assert.ErrorIs(t, err, domain.ErrTransferFailed)
	assert.Empty(t, transactionID)
	assert.Empty(t, status)
	assert.Zero(t, timestamp)
}

func TestWalletService_TransferFunds_ExecuteTransferReturnsNil(t *testing.T) {
	ctx := context.Background()
	walletService, _, mockTransferRepo := newWalletService()

	mockTransferRepo.On("ExecuteTransfer", ctx, "unique-key-123", "wallet1", "wallet2", 100.0, mock.AnythingOfType("int64")).
		Return(nil, nil)

	transactionID, status, timestamp, err := walletService.TransferFunds(ctx, "unique-key-123", "wallet1", "wallet2", 100.0)

	assert.ErrorIs(t, err, domain.ErrSomethingWentWrong)
	assert.Empty(t, transactionID)
	assert.Empty(t, status)
	assert.Zero(t, timestamp)
}

func TestWalletService_TransferFunds_ExecuteTransferError(t *testing.T) {
	ctx := context.Background()
	walletService, _, mockTransferRepo := newWalletService()

	mockTransferRepo.On("ExecuteTransfer", ctx, "unique-key-123", "wallet1", "wallet2", 100.0, mock.AnythingOfType("int64")).
		Return(nil, domain.ErrInsufficientBalance)

	transactionID, status, timestamp, err := walletService.TransferFunds(ctx, "unique-key-123", "wallet1", "wallet2", 100.0)

	assert.ErrorIs(t, err, domain.ErrInsufficientBalance)
	assert.Empty(t, transactionID)
	assert.Empty(t, status)
	assert.Zero(t, timestamp)
}

// A retry with the exact same parameters as the original transfer must
// return the original result instead of erroring.
func TestWalletService_TransferFunds_IdempotentReplay_SameParams(t *testing.T) {
	ctx := context.Background()
	walletService, _, mockTransferRepo := newWalletService()

	createdAt := time.Now().Add(-time.Minute)
	existing := &domain.Transfer{
		ID:           "txn_123",
		FromWalletID: "wallet1",
		ToWalletID:   "wallet2",
		Amount:       100.0,
		Status:       domain.TransferProcessed,
		CreatedAt:    createdAt,
	}

	mockTransferRepo.On("ExecuteTransfer", ctx, "unique-key-123", "wallet1", "wallet2", 100.0, mock.AnythingOfType("int64")).
		Return(nil, domain.ErrIdempotencyKeyConflict)
	mockTransferRepo.On("GetByIdempotencyKey", ctx, "unique-key-123").Return(existing, nil)

	transactionID, status, timestamp, err := walletService.TransferFunds(ctx, "unique-key-123", "wallet1", "wallet2", 100.0)

	assert.NoError(t, err)
	assert.Equal(t, "txn_123", transactionID)
	assert.Equal(t, string(domain.TransferProcessed), status)
	assert.Equal(t, createdAt.UnixMilli(), timestamp)
}

// A retry with the same key but different parameters is a genuine conflict.
func TestWalletService_TransferFunds_IdempotentReplay_DifferentParams(t *testing.T) {
	ctx := context.Background()
	walletService, _, mockTransferRepo := newWalletService()

	existing := &domain.Transfer{
		ID:           "txn_123",
		FromWalletID: "wallet1",
		ToWalletID:   "wallet2",
		Amount:       50.0, // different amount than the retried request
		Status:       domain.TransferProcessed,
		CreatedAt:    time.Now(),
	}

	mockTransferRepo.On("ExecuteTransfer", ctx, "unique-key-123", "wallet1", "wallet2", 100.0, mock.AnythingOfType("int64")).
		Return(nil, domain.ErrIdempotencyKeyConflict)
	mockTransferRepo.On("GetByIdempotencyKey", ctx, "unique-key-123").Return(existing, nil)

	transactionID, status, timestamp, err := walletService.TransferFunds(ctx, "unique-key-123", "wallet1", "wallet2", 100.0)

	assert.ErrorIs(t, err, domain.ErrIdempotencyKeyConflict)
	assert.Empty(t, transactionID)
	assert.Empty(t, status)
	assert.Zero(t, timestamp)
}

// A replayed original transfer that had failed should still surface as a failure.
func TestWalletService_TransferFunds_IdempotentReplay_OriginalFailed(t *testing.T) {
	ctx := context.Background()
	walletService, _, mockTransferRepo := newWalletService()

	existing := &domain.Transfer{
		ID:           "txn_123",
		FromWalletID: "wallet1",
		ToWalletID:   "wallet2",
		Amount:       100.0,
		Status:       domain.TransferFailed,
		CreatedAt:    time.Now(),
	}

	mockTransferRepo.On("ExecuteTransfer", ctx, "unique-key-123", "wallet1", "wallet2", 100.0, mock.AnythingOfType("int64")).
		Return(nil, domain.ErrIdempotencyKeyConflict)
	mockTransferRepo.On("GetByIdempotencyKey", ctx, "unique-key-123").Return(existing, nil)

	transactionID, status, timestamp, err := walletService.TransferFunds(ctx, "unique-key-123", "wallet1", "wallet2", 100.0)

	assert.ErrorIs(t, err, domain.ErrTransferFailed)
	assert.Empty(t, transactionID)
	assert.Empty(t, status)
	assert.Zero(t, timestamp)
}

func TestWalletService_TransferFunds_IdempotentReplay_LookupError(t *testing.T) {
	ctx := context.Background()
	walletService, _, mockTransferRepo := newWalletService()

	mockTransferRepo.On("ExecuteTransfer", ctx, "unique-key-123", "wallet1", "wallet2", 100.0, mock.AnythingOfType("int64")).
		Return(nil, domain.ErrIdempotencyKeyConflict)
	mockTransferRepo.On("GetByIdempotencyKey", ctx, "unique-key-123").Return(nil, domain.ErrSomethingWentWrong)

	transactionID, status, timestamp, err := walletService.TransferFunds(ctx, "unique-key-123", "wallet1", "wallet2", 100.0)

	assert.ErrorIs(t, err, domain.ErrSomethingWentWrong)
	assert.Empty(t, transactionID)
	assert.Empty(t, status)
	assert.Zero(t, timestamp)
}

func TestWalletService_TransferFunds_IdempotentReplay_NotFound(t *testing.T) {
	ctx := context.Background()
	walletService, _, mockTransferRepo := newWalletService()

	mockTransferRepo.On("ExecuteTransfer", ctx, "unique-key-123", "wallet1", "wallet2", 100.0, mock.AnythingOfType("int64")).
		Return(nil, domain.ErrIdempotencyKeyConflict)
	mockTransferRepo.On("GetByIdempotencyKey", ctx, "unique-key-123").Return(nil, nil)

	transactionID, status, timestamp, err := walletService.TransferFunds(ctx, "unique-key-123", "wallet1", "wallet2", 100.0)

	assert.ErrorIs(t, err, domain.ErrIdempotencyKeyConflict)
	assert.Empty(t, transactionID)
	assert.Empty(t, status)
	assert.Zero(t, timestamp)
}
