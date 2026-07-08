package dto

type TransferRequest struct {
	IdempotencyKey string  `json:"key"`
	FromWalletID   string  `json:"from_wallet_id"`
	ToWalletID     string  `json:"to_wallet_id"`
	Amount         float64 `json:"amount"`
}

type TransferResponse struct {
	TransactionID string  `json:"transaction_id"`
	FromWalletID  string  `json:"from_wallet_id"`
	ToWalletID    string  `json:"to_wallet_id"`
	Amount        float64 `json:"amount"`
	Status        string  `json:"status"`
	Timestamp     int64   `json:"timestamp"`
}
