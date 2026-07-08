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
		readReplica.Close()
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
		f.ReadPGReplica.Close()
	}
	if f.WritePGReplica != nil {
		f.WritePGReplica.Close()
	}
}
