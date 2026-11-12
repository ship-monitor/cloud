package db

import (
	"context"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/ship-monitor/cloud/migrations/postgres"
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
	provider, err := goose.NewProvider(Driver, db.DB, postgres.Migrations)
	if err != nil {
		return fmt.Errorf("new provider: %w", err)
	}

	if _, err := provider.Up(context.Background()); err != nil {
		return fmt.Errorf("up: %w", err)
	}

	return nil
}
