package di

import (
	"database/sql"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
)

func NewRedisClient(config *viper.Viper) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     config.GetString("redis.addr"),
		Password: config.GetString("redis.password"),
		DB:       config.GetInt("redis.db"),
	})
}

func NewRabbitMQClient(config *viper.Viper) (*amqp091.Connection, error) {
	client, err := amqp091.Dial(config.GetString("rabbitmq-url"))
	if err != nil {
		return nil, fmt.Errorf("failed dial rebbitmq: %w", err)
	}

	return client, nil
}

func NewDatabaseClient(config *viper.Viper) (*sql.DB, error) {
	db, err := db.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}
