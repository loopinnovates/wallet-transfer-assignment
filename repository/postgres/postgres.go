package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func NewDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		go func() {
			err := db.Close()
			if err != nil {
				fmt.Printf("Error in closing db: %v", err)
			}
		}()
		return nil, err
	}
	return db, nil
}
