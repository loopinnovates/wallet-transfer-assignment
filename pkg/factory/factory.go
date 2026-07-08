package factory

import (
	"database/sql"
	"fmt"

	"github.com/loopinnovates/wallet-transfer-assignment/internal/config"
	"github.com/loopinnovates/wallet-transfer-assignment/repository/postgres"
)

type Factory struct {
	ReadPGReplica  *sql.DB
	WritePGReplica *sql.DB
}

func NewFactory(appConfig *config.Config) (*Factory, error) {
	readReplica, err := getReadReplica(appConfig.ReadPGURI)
	if err != nil {
		return nil, fmt.Errorf("connecting to read replica: %w", err)
	}

	writeReplica, err := getWriteReplica(appConfig.WritePGURI)
	if err != nil {
		err2 := readReplica.Close()
		if err2 != nil {
			fmt.Printf("Error closing read replica: %v", err2)
		}
		return nil, fmt.Errorf("connecting to write replica: %w", err)
	}

	return &Factory{
		ReadPGReplica:  readReplica,
		WritePGReplica: writeReplica,
	}, nil
}

func getReadReplica(uri string) (*sql.DB, error) {
	return postgres.NewDB(uri)
}

func getWriteReplica(uri string) (*sql.DB, error) {
	return postgres.NewDB(uri)
}

func (f *Factory) Close() {
	if f.ReadPGReplica != nil {
		err := f.ReadPGReplica.Close()
		if err != nil {
			fmt.Printf("Error closing read replica: %v", err)
		}
	}
	if f.WritePGReplica != nil {
		err := f.WritePGReplica.Close()
		if err != nil {
			fmt.Printf("Error closing write replica: %v", err)
		}
	}
}
