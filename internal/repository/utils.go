package repository

import (
	"database/sql"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func newBunDB(db *sql.DB) *bun.DB {
	return bun.NewDB(db, pgdialect.New())
}
