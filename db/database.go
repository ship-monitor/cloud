package db

import (
	"database/sql"
	"errors"

	"github.com/spf13/viper"
	"github.com/uptrace/bun/driver/pgdriver"
)

var ErrNoConnectionString = errors.New("connection string not specified (key database-url)")

func Connect() (*sql.DB, error) {
	dsn := viper.GetString("database-url")
	if dsn == "" {
		return nil, ErrNoConnectionString
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(
		pgdriver.WithDSN(dsn),
	))

	return sqldb, nil
}
