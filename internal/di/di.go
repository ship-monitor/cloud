package di

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/ship-monitor/cloud/internal/config"
	"go.uber.org/fx"
)

func NewRedisClient(lc fx.Lifecycle, config *config.Config) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Redis.URL,
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
	})

	lc.Append(
		fx.StopHook(func(ctx context.Context) error {
			if err := client.Close(); err != nil {
				return fmt.Errorf("close redis: %w", err)
			}

			return nil
		}),
	)

	return client
}

func NewRabbitMQClient(
	lc fx.Lifecycle,
	config *config.Config,
) (*amqp091.Connection, error) {
	client, err := amqp091.Dial(config.RabbitMQURL)
	if err != nil {
		return nil, fmt.Errorf("failed dial rebbitmq: %w", err)
	}

	lc.Append(fx.StopHook(func() error {
		if err := client.Close(); err != nil {
			return fmt.Errorf("close amqp connection: %w", err)
		}

		return nil
	}))

	return client, nil
}

func NewDatabaseClient(
	lc fx.Lifecycle,
	config *config.Config,
) (*sql.DB, error) {
	db, err := ConnectDB(config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	lc.Append(fx.StopHook(func() error {
		if err := db.Close(); err != nil {
			return fmt.Errorf("close db: %w", err)
		}

		return nil
	}))

	return db, nil
}
