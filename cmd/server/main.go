package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/loopinnovates/wallet-transfer-assignment/internal/config"
	"github.com/loopinnovates/wallet-transfer-assignment/internal/router"
	"github.com/loopinnovates/wallet-transfer-assignment/internal/service"
	"github.com/loopinnovates/wallet-transfer-assignment/internal/worker"
	"github.com/loopinnovates/wallet-transfer-assignment/pkg/factory"
	"github.com/loopinnovates/wallet-transfer-assignment/repository/postgres"
)

const shutdownTimeout = 10 * time.Second

func main() {
	appConfig := config.AppConfig()

	appFactory, err := factory.NewFactory(appConfig)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer appFactory.Close()

	transferRepo := postgres.NewTransferRepository(appFactory.ReadPGReplica, appFactory.WritePGReplica)

	RouterContext := &router.RouterContext{
		WalletSvc: &service.WalletService{
			WalletRepo:   postgres.NewWalletRepository(appFactory.ReadPGReplica, appFactory.WritePGReplica),
			TransferRepo: transferRepo,
		},
	}

	srv := &http.Server{
		Addr:    ":" + appConfig.Port,
		Handler: router.New(RouterContext),
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Printf("listening on :%s", appConfig.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	reconCtx, cancelRecon := context.WithCancel(context.Background())
	defer cancelRecon()
	recon := &worker.PendingTransferRecon{TransferRepo: transferRepo}
	go recon.Run(reconCtx)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serveErr:
		cancelRecon()
		if err != nil {
			log.Fatalf("server failed: %v", err)
		}
	case sig := <-stop:
		log.Printf("received %s, shutting down gracefully", sig)
		cancelRecon()

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("graceful shutdown failed: %v", err)
		}
		log.Println("server shut down cleanly")
	}
}
