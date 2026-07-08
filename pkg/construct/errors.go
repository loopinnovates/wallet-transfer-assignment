package construct

import (
	"errors"
	"net/http"

	"github.com/loopinnovates/wallet-transfer-assignment/internal/domain"
	"github.com/loopinnovates/wallet-transfer-assignment/pkg/utils"
)

var (
	// router errors
	ErrRouteNotFound = utils.NewCustomError(http.StatusNotFound, "route not found")

	// handler errors
	ErrInternal            = utils.NewCustomError(http.StatusInternalServerError, "something went wrong")
	ErrInvalidRequestBody  = utils.NewCustomError(http.StatusBadRequest, "invalid request body")
	ErrMissingTransferData = utils.NewCustomError(http.StatusBadRequest, "from_wallet_id, to_wallet_id and a positive amount are required")

	// service errors
	ErrSameWallet             = utils.NewCustomError(http.StatusBadRequest, domain.ErrSameWallet.Error())
	ErrTransferFailed         = utils.NewCustomError(http.StatusInternalServerError, domain.ErrTransferFailed.Error())
	ErrInsufficientBalance    = utils.NewCustomError(http.StatusUnprocessableEntity, domain.ErrInsufficientBalance.Error())
	ErrWalletNotFound         = utils.NewCustomError(http.StatusNotFound, domain.ErrWalletNotFound.Error())
	ErrIdempotencyKeyConflict = utils.NewCustomError(http.StatusConflict, domain.ErrIdempotencyKeyConflict.Error())
)

func MapTransferError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInsufficientBalance):
		return ErrInsufficientBalance
	case errors.Is(err, domain.ErrWalletNotFound):
		return ErrWalletNotFound
	case errors.Is(err, domain.ErrSameWallet):
		return ErrSameWallet
	case errors.Is(err, domain.ErrIdempotencyKeyConflict):
		return ErrIdempotencyKeyConflict
	default:
		return ErrTransferFailed
	}
}
