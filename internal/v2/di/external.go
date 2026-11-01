package di

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/nats-io/nats.go"
	"github.com/ship-monitor/cloud/internal/v2/adapters/db"
	"github.com/ship-monitor/cloud/internal/v2/config"
	"github.com/valkey-io/valkey-go"
	"go.uber.org/fx"
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
