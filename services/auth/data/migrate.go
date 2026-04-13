package data

import (
	"context"

	"charm.land/log/v2"
	migrate "github.com/rubenv/sql-migrate"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
)

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

func Migrate() {
	ctx := context.TODO()
	models := []interface{}{
		(*User)(nil),
	}

	for _, model := range models {
		_, err := db.DB.NewCreateTable().
			Model(model).
			IfNotExists().
			Exec(ctx)
		panicIfErr(err)
	}
}

const (
	DialectPostgres = "postgres"
	DialectSqlite   = "sqlite"
)

func Migrate2() {
	migrations := getMigrationSource()
	sqlDb := db.DB.DB
	count, err := migrate.Exec(sqlDb, DialectPostgres, migrations, migrate.Up)

	panicIfErr(err)

	log.Info("Executed migrations", "count", count)
}

func getMigrationSource() migrate.MigrationSource {
	return &migrate.FileMigrationSource{
		Dir: "migrations",
	}
}
