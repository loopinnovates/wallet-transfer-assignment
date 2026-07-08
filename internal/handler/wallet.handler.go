package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/loopinnovates/wallet-transfer-assignment/internal/dto"
	"github.com/loopinnovates/wallet-transfer-assignment/internal/service"
	"github.com/loopinnovates/wallet-transfer-assignment/pkg/construct"
	"github.com/loopinnovates/wallet-transfer-assignment/pkg/utils"
)

func TransferHandler(walletSvc service.IWalletSvc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req dto.TransferRequest

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			fmt.Println("Error decoding request body:", err)
			utils.WriteError(w, construct.ErrInvalidRequestBody)
			return
		}

		if req.FromWalletID == "" || req.ToWalletID == "" || req.Amount <= 0 {
			utils.WriteError(w, construct.ErrMissingTransferData)
			return
		}

		if req.FromWalletID == req.ToWalletID {
			utils.WriteError(w, construct.ErrSameWallet)
			return
		}

		transactionID, status, timestamp, err := walletSvc.TransferFunds(ctx, req.IdempotencyKey, req.FromWalletID, req.ToWalletID, req.Amount)
		if err != nil {
			utils.WriteError(w, construct.MapTransferError(err))
			return
		}

		resp := dto.TransferResponse{
			TransactionID: transactionID,
			FromWalletID:  req.FromWalletID,
			ToWalletID:    req.ToWalletID,
			Amount:        req.Amount,
			Status:        status,
			Timestamp:     timestamp,
		}

		utils.WriteSuccess(w, http.StatusOK, resp)
	})
}
