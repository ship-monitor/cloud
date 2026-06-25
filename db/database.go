package db

import (
	"database/sql"
	"errors"

	"charm.land/log/v2"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/driver/sqliteshim"
)

var DB *bun.DB

// setupDebugDB connects to local sqlite database.
//
// Deprecated: for development use docker-compose.yml file.
func setupDebugDB() {
	log.Warn("Debug database setup with sqlite")

	sqldb, err := sql.Open(sqliteshim.ShimName, "file:test.db?cache=shared&mode=rwc")
	if err != nil {
		panic(err)
	}

	DB = bun.NewDB(sqldb, sqlitedialect.New())
}

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

// Setup initializes [DB] global variable.
//
// Deprecated: should not use global variables. Use [Connect] instead.
func Setup() {
	if viper.GetBool("devel") {
		setupDebugDB()

		return
	}

	sqldb, err := Connect()
	if err != nil {
		panic(err)
	}

	DB = bun.NewDB(sqldb, pgdialect.New())
}
