package router

import (
	"net/http"

	"github.com/loopinnovates/wallet-transfer-assignment/internal/handler"
)

func Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handler.Health)
	return mux
}
