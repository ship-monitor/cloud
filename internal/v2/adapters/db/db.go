package db

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	"github.com/ship-monitor/cloud/migrations"
	// _ "github.com/lib/pq".
	_ "modernc.org/sqlite"
)

type Config struct {
	Driver string `validate:"required,oneof=sqlite postgres"`
	DSN    string `validate:"required"`
}

func New(config *Config) (*sqlx.DB, error) {
	err := validator.New().Struct(config)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	db, err := sqlx.Connect(config.Driver, config.DSN)
	if err != nil {
		return nil, fmt.Errorf("connect to %s database: %w", config.Driver, err)
	}

	err = migrate(config, db)
	if err != nil {
		return nil, fmt.Errorf("migrate db: %w", err)
	}

	return db, nil
}

func migrate(config *Config, db *sqlx.DB) error {
	goose.SetBaseFS(migrations.Migrations)

	if err := goose.SetDialect(config.Driver); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}

	if err := goose.Up(db.DB, "."); err != nil {
		return fmt.Errorf("up: %w", err)
	}

	return nil
}
