package router

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/loopinnovates/wallet-transfer-assignment/internal/handler"
	"github.com/loopinnovates/wallet-transfer-assignment/internal/middleware"
	"github.com/loopinnovates/wallet-transfer-assignment/internal/service"
	"github.com/loopinnovates/wallet-transfer-assignment/pkg/construct"
	"github.com/loopinnovates/wallet-transfer-assignment/pkg/utils"
)

type RouterContext struct {
	WalletSvc service.IWalletSvc
}

func New(rc *RouterContext) *mux.Router {

	transferHandler := utils.Wrap(handler.TransferHandler(rc.WalletSvc))

	router := mux.NewRouter()
	router.Use(middleware.Recover)
	router.HandleFunc("/health", handler.Health).Methods("GET")
	router.Handle("/transfers", transferHandler).Methods("POST")

	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Route not found: %s %s", r.Method, r.URL.Path)
		utils.WriteError(w, construct.ErrRouteNotFound)
	})
	return router
}
