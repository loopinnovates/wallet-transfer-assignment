package config

import (
	"github.com/loopinnovates/wallet-transfer-assignment/pkg/utils"
)

type config struct {
	Port string
}

func AppConfig() *config {
	port := utils.GetEnvStr(PortKey, "8080")
	configObj := config{
		Port: port,
	}

	return &configObj
}
