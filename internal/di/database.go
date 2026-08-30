package di

import (
	"database/sql"
	"errors"

	"github.com/uptrace/bun/driver/pgdriver"
)

var ErrNoConnectionString = errors.New(
	"connection string not specified (key database-url)",
)

func ConnectDB(url string) (*sql.DB, error) {
	if url == "" {
		return nil, ErrNoConnectionString
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(
		pgdriver.WithDSN(url),
	))

	return sqldb, nil
}
