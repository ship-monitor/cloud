package common

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
)

func NewBunDB(db *sql.DB) *bun.DB {
	driverType := strings.ToLower(fmt.Sprintf("%T", db.Driver()))
	if strings.Contains(driverType, "sqlite") {
		return bun.NewDB(db, sqlitedialect.New())
	}

	return bun.NewDB(db, pgdialect.New())
}
