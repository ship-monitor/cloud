package clickhousedb

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/go-playground/validator/v10"
	"github.com/pressly/goose/v3"
	clickMigrations "github.com/ship-monitor/cloud/migrations/clickhouse"
	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
	"go.uber.org/zap/zapcore"
)

const Driver = "clickhouse"

type Config struct {
	DSN string `validate:"required"`
}

type Clickhouse struct {
	options *clickhouse.Options
	conn    driver.Conn
	logger  *zap.Logger
}

func New(config *Config, logger *zap.Logger) (*Clickhouse, error) {
	log := logger.Named("Clickhouse")

	err := validator.New().Struct(config)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	options, err := clickhouse.ParseDSN(config.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse DSN: %w", err)
	}

	options.Logger = slog.New(
		zapslog.NewHandler(
			log.WithOptions(zap.IncreaseLevel(zapcore.WarnLevel)).Core(),
		),
	)
	log = log.With(zap.Strings("addr", options.Addr))

	db, err := clickhouse.Open(options)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	return &Clickhouse{
		options: options,
		conn:    db,
		logger:  log,
	}, nil
}

func (c *Clickhouse) Conn() driver.Conn {
	return c.conn
}

func (c *Clickhouse) Migrate(ctx context.Context) error {
	db := clickhouse.OpenDB(c.options)
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("open db: ping db: %w", err)
	}

	defer func() {
		err := db.Close()
		if err != nil {
			c.logger.Named("Migration").
				Error("Close migration DB", zap.Error(err))
		}
	}()

	provider, err := goose.NewProvider(Driver, db, clickMigrations.Migrations)
	if err != nil {
		return fmt.Errorf("new provider: %w", err)
	}

	_, err = provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("up: %w", err)
	}

	c.logger.Info("Up migrations")

	return nil
}
