package di

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
)

func NewRedisClient(lc fx.Lifecycle, config *viper.Viper) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     config.GetString("redis.addr"),
		Password: config.GetString("redis.password"),
		DB:       config.GetInt("redis.db"),
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
	config *viper.Viper,
) (*amqp091.Connection, error) {
	client, err := amqp091.Dial(config.GetString("rabbitmq-url"))
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

func NewDatabaseClient(lc fx.Lifecycle, config *viper.Viper) (*sql.DB, error) {
	db, err := db.Connect()
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
