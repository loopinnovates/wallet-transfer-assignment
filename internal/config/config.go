package config

import (
	"fmt"
	"time"

	"github.com/loopinnovates/wallet-transfer-assignment/pkg/utils"
)

type Config struct {
	Port                  string
	ReadPGURI             string
	WritePGURI            string
	MaxConcurrentRequests int
	LogLevel              string

	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
	DBConnMaxIdleTime time.Duration
}

func AppConfig() *Config {
	port := utils.GetEnvStr(PortKey, "8080")

	maxConcurrentRequests := utils.GetEnvInt(MaxConcurrentRequestsKey, 200)
	logLevel := utils.GetEnvStr(LogLevelKey, "info")

	// DB Config
	dbUsername := utils.GetEnvStr(usernameKey, "wallet")
	dbPassword := utils.GetEnvStr(passwordKey, "wallet")
	readPGURI := fmt.Sprintf(utils.GetEnvStr(ReadPGURIKey, "postgres://%s:%s@localhost:5432/wallet_transfer?sslmode=disable"), dbUsername, dbPassword)
	writePGURI := fmt.Sprintf(utils.GetEnvStr(WritePGURIKey, "postgres://%s:%s@localhost:5432/wallet_transfer?sslmode=disable"), dbUsername, dbPassword)

	dbMaxOpenConns := utils.GetEnvInt(DBMaxOpenConnsKey, 25)
	dbMaxIdleConns := utils.GetEnvInt(DBMaxIdleConnsKey, 25)
	dbConnMaxLifetime := time.Duration(utils.GetEnvInt(DBConnMaxLifetimeMinutesKey, 5)) * time.Minute
	dbConnMaxIdleTime := time.Duration(utils.GetEnvInt(DBConnMaxIdleTimeMinutesKey, 2)) * time.Minute

	configObj := Config{
		Port:                  port,
		ReadPGURI:             readPGURI,
		WritePGURI:            writePGURI,
		MaxConcurrentRequests: maxConcurrentRequests,
		LogLevel:              logLevel,
		DBMaxOpenConns:        dbMaxOpenConns,
		DBMaxIdleConns:        dbMaxIdleConns,
		DBConnMaxLifetime:     dbConnMaxLifetime,
		DBConnMaxIdleTime:     dbConnMaxIdleTime,
	}

	return &configObj
}
