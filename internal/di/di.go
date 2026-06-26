package di

import (
	"database/sql"

	"charm.land/log/v2"
	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/handlers"
	repository "sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/repositories"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
)

type Container struct {
	config *viper.Viper
	logger *log.Logger

	db *sql.DB

	rabbitMQ             *amqp091.Connection
	redis                *redis.Client
	organizationsService *services.OrganizationsService
	deviceStates         *services.DeviceStatesService
	devicesService       *services.DevicesService
	orgDevicesService    *services.OrgDevicesService

	devicesHandlers *handlers.DevicesHandlers
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

func (c *Container) DB() *sql.DB {
	if c.db == nil {
		db, err := db.Connect()
		if err != nil {
			panic(err)
		}

		c.db = db
	}

	return c.db
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

func (c *Container) DeviceStates() *services.DeviceStatesService {
	if c.deviceStates == nil {
		c.deviceStates = services.NewDeviceStatesCache(
			c.redis,
			services.DeviceStatesConfig{
				MaxHistoryLength: MaxHistoryLength,
			},
			c.logger,
		)
	}

	return c.deviceStates
}

func (c *Container) OrganizationsService() *services.OrganizationsService {
	if c.organizationsService == nil {
		c.organizationsService = services.NewOrganizations(repository.NewOrgs(c.DB()))
	}

	return c.organizationsService
}

func (c *Container) OrgDevicesService() *services.OrgDevicesService {
	if c.orgDevicesService == nil {
		c.orgDevicesService = services.NewOrgDevices(
			repository.NewOrgDevices(c.DB()),
			c.OrganizationsService(),
		)
	}

	return c.orgDevicesService
}

func (c *Container) DevicesService() *services.DevicesService {
	if c.devicesService == nil {
		c.devicesService = services.NewDevices(
			c.DeviceStates(),
			c.OrgDevicesService(),
			c.Logger(),
			c.OrganizationsService(),
		)
	}

	return c.devicesService
}

func (c *Container) DevicesHandlers() *handlers.DevicesHandlers {
	if c.devicesHandlers == nil {
		c.devicesHandlers = handlers.NewDevicesHandlers(c.DevicesService())
	}

	return c.devicesHandlers
}
