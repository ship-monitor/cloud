package di

import (
	"context"
	"fmt"
	"log/slog"

	"charm.land/log/v2"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

func NewClickhouse(
	lc fx.Lifecycle,
	logger *log.Logger,
) (clickhouse.Conn, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{viper.GetString("clickhouse.addr")},
		Auth: clickhouse.Auth{
			Database: viper.GetString("clickhouse.database"),
			Username: viper.GetString("clickhouse.user"),
			Password: viper.GetString("clickhouse.password"),
		},
		Logger: slog.New(logger),
		// TLS:    &tls.Config{InsecureSkipVerify: true},
	})
	if err != nil {
		return nil, fmt.Errorf("open clickhouse connection: %w", err)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := conn.Ping(ctx); err != nil {
				return fmt.Errorf("ping clickhouse: %w", err)
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			if err := conn.Close(); err != nil {
				return fmt.Errorf("close clickhouse connection: %w", err)
			}

			return nil
		},
	})

	return conn, nil
}
