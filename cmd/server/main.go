package main

import (
	"log"
	"net/http"

	"github.com/loopinnovates/wallet-transfer-assignment/internal/config"
	"github.com/loopinnovates/wallet-transfer-assignment/internal/router"
)

func main() {
	appConfig := config.AppConfig()

	routes := router.Routes()
	log.Printf("listening on :%s", appConfig.Port)
	if err := http.ListenAndServe(":"+appConfig.Port, routes); err != nil {
		log.Fatal(err)
	}
}
