package config

import (
	"github.com/loopinnovates/wallet-transfer-assignment/pkg/utils"
)

type Config struct {
	Port       string
	ReadPGURI  string
	WritePGURI string
}

func AppConfig() *Config {
	port := utils.GetEnvStr(PortKey, "8080")

	readPGURI := utils.GetEnvStr(ReadPGURIKey, "postgres://wallet:wallet@localhost:5432/wallet_transfer?sslmode=disable")
	writePGURI := utils.GetEnvStr(WritePGURIKey, "postgres://wallet:wallet@localhost:5432/wallet_transfer?sslmode=disable")

	configObj := Config{
		Port:       port,
		ReadPGURI:  readPGURI,
		WritePGURI: writePGURI,
	}

	return &configObj
}
