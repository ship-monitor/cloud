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

	rabbitMQ *amqp091.Connection
	redis    *redis.Client

	deviceStates   *repository.DeviceStatesRepository
	orgsRepository *repository.OrganizationsRepo
	orgDevicesRepo *repository.OrgDevicesRepo
	usersRepo      *repository.UsersRepo

	organizationsService *services.OrganizationsService
	devicesService       *services.DevicesService
	orgDevicesService    *services.OrgDevicesService
	authService          *services.AuthService
	emailService         *services.EmailService

	topicPublisher *services.TopicPublisher

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

func (c *Container) TopicPublisher() *services.TopicPublisher {
	if c.topicPublisher == nil {
		c.topicPublisher = services.NewTopicPublisher(c.RabbitMQ(), c.Logger())
	}

	return c.topicPublisher
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

func (c *Container) DeviceStates() *repository.DeviceStatesRepository {
	if c.deviceStates == nil {
		c.deviceStates = repository.NewDeviceStatesRepo(
			c.redis,
			repository.DeviceStatesConfig{
				MaxHistoryLength: MaxHistoryLength,
			},
			c.logger,
		)
	}

	return c.deviceStates
}

func (c *Container) OrganizationsRepo() *repository.OrganizationsRepo {
	if c.orgsRepository == nil {
		c.orgsRepository = repository.NewOrgs(c.DB())
	}

	return c.orgsRepository
}

func (c *Container) OrgDevicesRepo() *repository.OrgDevicesRepo {
	if c.orgDevicesRepo == nil {
		c.orgDevicesRepo = repository.NewOrgDevices(c.DB())
	}

	return c.orgDevicesRepo
}

func (c *Container) OrganizationsService() *services.OrganizationsService {
	if c.organizationsService == nil {
		c.organizationsService = services.NewOrganizations(
			c.OrganizationsRepo(),
		)
	}

	return c.organizationsService
}

func (c *Container) EmailService() *services.EmailService {
	if c.emailService == nil {
		email, err := services.NewEmailService(services.EmailServiceConfig{
			SMTPHost:     viper.GetString("email.smtp-host"),
			SMTPPort:     viper.GetUint("email.smtp-port"),
			SenderName:   viper.GetString("email.sender-name"),
			AuthEmail:    viper.GetString("email.email"),
			AuthPassword: viper.GetString("email.password"),
		})
		if err != nil {
			panic(err)
		}

		c.emailService = email
	}

	return c.emailService
}

func (c *Container) AuthService() *services.AuthService {
	if c.authService == nil {
		c.authService = services.NewAuthService(
			c.Logger(),
			c.Redis(),
			c.EmailService(),
			c.UsersRepo())
	}

	return c.authService
}

func (c *Container) OrgDevicesService() *services.OrgDevicesService {
	if c.orgDevicesService == nil {
		c.orgDevicesService = services.NewOrgDevices(
			c.OrgDevicesRepo(),
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
			c.TopicPublisher(),
		)
	}

	return c.devicesService
}

func (c *Container) UsersRepo() *repository.UsersRepo {
	if c.usersRepo == nil {
		c.usersRepo = repository.NewUsers(c.DB())
	}

	return c.usersRepo
}

func (c *Container) DevicesHandlers() *handlers.DevicesHandlers {
	if c.devicesHandlers == nil {
		c.devicesHandlers = handlers.NewDevicesHandlers(c.DevicesService())
	}

	return c.devicesHandlers
}
