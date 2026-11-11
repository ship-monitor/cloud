package db

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/ship-monitor/cloud/migrations"
)

const Driver = "postgres"

type Config struct {
	DSN string `validate:"required"`
}

func New(config *Config) (*sqlx.DB, error) {
	err := validator.New().Struct(config)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	db, err := sqlx.Connect(Driver, config.DSN)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres database: %w", err)
	}

	err = migrate(db)
	if err != nil {
		return nil, fmt.Errorf("migrate db: %w", err)
	}

	return db, nil
}

func migrate(db *sqlx.DB) error {
	goose.SetBaseFS(migrations.Migrations)

	if err := goose.SetDialect(Driver); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}

	if err := goose.Up(db.DB, "."); err != nil {
		return fmt.Errorf("up: %w", err)
	}

	return nil
}
