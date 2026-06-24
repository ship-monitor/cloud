package db

import (
	"database/sql"

	"charm.land/log/v2"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/driver/sqliteshim"
)

var DB *bun.DB

func setupDebugDB() {
	log.Warn("Debug database setup with sqlite")

	sqldb, err := sql.Open(sqliteshim.ShimName, "file:test.db?cache=shared&mode=rwc")
	if err != nil {
		panic(err)
	}

	DB = bun.NewDB(sqldb, sqlitedialect.New())
}

func Setup() {
	if viper.GetBool("devel") {
		setupDebugDB()

		return
	}

	dsn := viper.GetString("database-url")
	if dsn == "" {
		panic("database-url not configured")
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(
		pgdriver.WithDSN(dsn),
	))
	DB = bun.NewDB(sqldb, pgdialect.New())
}
