package di

import (
	"charm.land/log/v2"
	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
)

type Container struct {
	config *viper.Viper
	logger *log.Logger

	rabbitMQ     *amqp091.Connection
	redis        *redis.Client
	deviceStates *services.DeviceStatesCache
}

func NewContainer(config *viper.Viper, logger *log.Logger) *Container {
	return &Container{
		config: config,
		logger: logger,
	}
}

func (c *Container) Logger() *log.Logger {
	return c.logger
}

func (c *Container) Redis() *redis.Client {
	if c.redis == nil {
		c.redis = redis.NewClient(&redis.Options{
			Addr:     c.config.GetString("redis.url"),
			Password: c.config.GetString("redis.password"),
		})
	}

	return c.redis
}

func (c *Container) RabbitMQ() *amqp091.Connection {
	if c.rabbitMQ == nil {
		conn, err := amqp091.Dial(c.config.GetString("rabbitmq-url"))
		if err != nil {
			panic(err)
		}

		c.rabbitMQ = conn
	}

	return c.rabbitMQ
}

const MaxHistoryLength = 100

func (c *Container) DeviceStates() *services.DeviceStatesCache {
	if c.deviceStates == nil {
		c.deviceStates = services.NewDeviceStatesCache(
			c.redis,
			services.DeviceStateCacheConfig{
				MaxHistoryLength: MaxHistoryLength,
			},
			c.logger,
		)
	}

	return c.deviceStates
}
