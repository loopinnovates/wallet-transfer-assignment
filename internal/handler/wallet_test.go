package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/loopinnovates/wallet-transfer-assignment/internal/dto"
	"github.com/loopinnovates/wallet-transfer-assignment/internal/handler"
	"github.com/loopinnovates/wallet-transfer-assignment/internal/testutils"
)

func TestTransferHandler(t *testing.T) {
	mockWalletSvc := testutils.MockIWalletSvc{}

	ctx := context.Background()
	mockWalletSvc.On("TransferFunds", ctx, "test-idempotency-key", "wallet1", "wallet2", 100.0).Return("txn_123", "success", int64(1234567890), nil)

	body, _ := json.Marshal(dto.TransferRequest{
		IdempotencyKey: "test-idempotency-key",
		FromWalletID:   "wallet1",
		ToWalletID:     "wallet2",
		Amount:         100.0,
	})

	req := httptest.NewRequest(http.MethodPost, "/transfers", bytes.NewReader(body)).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.TransferHandler(&mockWalletSvc).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp dto.TransferResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "txn_123", resp.TransactionID)
	assert.Equal(t, "success", resp.Status)

	mockWalletSvc.AssertExpectations(t)
}

func TestTransferHandler_InvalidRequest_WalletIDCannotBeSame(t *testing.T) {
	mockWalletSvc := testutils.MockIWalletSvc{}

	body, _ := json.Marshal(dto.TransferRequest{
		IdempotencyKey: "test-idempotency-key",
		FromWalletID:   "wallet1",
		ToWalletID:     "wallet1", // Same wallet ID to trigger validation error
		Amount:         100.0,
	})

	req := httptest.NewRequest(http.MethodPost, "/transfers", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.TransferHandler(&mockWalletSvc).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "from_wallet_id and to_wallet_id cannot be the same")
}

func TestTransferHandler_InvalidRequest_NegativeAmount(t *testing.T) {
	mockWalletSvc := testutils.MockIWalletSvc{}

	body, _ := json.Marshal(dto.TransferRequest{
		IdempotencyKey: "test-idempotency-key",
		FromWalletID:   "wallet1",
		ToWalletID:     "wallet2",
		Amount:         -50.0, // Negative amount to trigger validation error
	})

	req := httptest.NewRequest(http.MethodPost, "/transfers", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.TransferHandler(&mockWalletSvc).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "from_wallet_id, to_wallet_id and a positive amount are required")
}

func TestTransferHandler_InvalidRequest_MissingFields(t *testing.T) {
	mockWalletSvc := testutils.MockIWalletSvc{}

	body, _ := json.Marshal(dto.TransferRequest{
		IdempotencyKey: "test-idempotency-key",
		FromWalletID:   "", // Missing from_wallet_id to trigger validation error
		ToWalletID:     "wallet2",
		Amount:         100.0,
	})

	req := httptest.NewRequest(http.MethodPost, "/transfers", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.TransferHandler(&mockWalletSvc).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "from_wallet_id, to_wallet_id and a positive amount are required")
}

func TestTransferHandler_InvalidRequest_BadJSON(t *testing.T) {
	mockWalletSvc := testutils.MockIWalletSvc{}

	// Invalid JSON body
	body := []byte(`{"idempotency_key": "test-idempotency-key", "from_wallet_id": "wallet1", "to_wallet_id": "wallet2",`)

	req := httptest.NewRequest(http.MethodPost, "/transfers", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.TransferHandler(&mockWalletSvc).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid request body")
}

func TestTransferHandler_ServiceError(t *testing.T) {
	mockWalletSvc := testutils.MockIWalletSvc{}

	ctx := context.Background()
	mockWalletSvc.On("TransferFunds", ctx, "test-idempotency-key", "wallet1", "wallet2", 100.0).Return("", "", int64(0), assert.AnError)

	body, _ := json.Marshal(dto.TransferRequest{
		IdempotencyKey: "test-idempotency-key",
		FromWalletID:   "wallet1",
		ToWalletID:     "wallet2",
		Amount:         100.0,
	})

	req := httptest.NewRequest(http.MethodPost, "/transfers", bytes.NewReader(body)).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.TransferHandler(&mockWalletSvc).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to transfer funds")

	mockWalletSvc.AssertExpectations(t)
}
