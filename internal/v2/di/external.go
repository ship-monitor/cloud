package di

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/jmoiron/sqlx"
	"github.com/nats-io/nats.go"
	"github.com/ship-monitor/cloud/internal/v2/adapters/clickhousedb"
	"github.com/ship-monitor/cloud/internal/v2/adapters/db"
	"github.com/ship-monitor/cloud/internal/v2/config"
	"github.com/valkey-io/valkey-go"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func newNATS(lc fx.Lifecycle, conf *config.Config) (*nats.Conn, error) {
	conn, err := nats.Connect(conf.NatsURL)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}

	lc.Append(fx.StopHook(conn.Close))

	return conn, nil
}

func newDB(lc fx.Lifecycle, conf *config.Config) (*sqlx.DB, error) {
	db, err := db.New(&conf.DB)
	if err != nil {
		return nil, fmt.Errorf("new db: %w", err)
	}

	lc.Append(fx.StopHook(func() error {
		err := db.Close()
		if err != nil {
			return fmt.Errorf("close db: %w", err)
		}

		return nil
	}))

	return db, nil
}

// Named "clickhouse" to avoid fx conflict with the postgres *sqlx.DB provider.
func newClickHouse(
	lc fx.Lifecycle,
	conf *config.Config,
	logger *zap.Logger,
) (driver.Conn, error) {
	ch, err := clickhousedb.New(&conf.ClickHouse, logger)
	if err != nil {
		return nil, fmt.Errorf("new clickhouse: %w", err)
	}

	conn := ch.Conn()

	lc.Append(fx.StartStopHook(func(ctx context.Context) error {
		err := ch.Migrate(ctx)
		if err != nil {
			return fmt.Errorf("migrate clickhouse: %w", err)
		}

		return nil
	}, func() error {
		err := conn.Close()
		if err != nil {
			return fmt.Errorf("close clickhouse: %w", err)
		}

		return nil
	}))

	return conn, nil
}

func newValkey(lc fx.Lifecycle, conf *config.Config) (valkey.Client, error) {
	client, err := valkey.NewClient(valkey.ClientOption{ //nolint:exhaustruct_v5
		InitAddress: []string{conf.ValkeyURL},
	})
	if err != nil {
		return nil, fmt.Errorf("new vakey client: %w", err)
	}

	lc.Append(fx.StopHook(client.Close))

	return client, nil
}
