package di

import (
	"net/http"

	"github.com/ship-monitor/cloud/internal/config"
	"github.com/ship-monitor/cloud/internal/domain"
	"github.com/ship-monitor/cloud/internal/handlers"
	"github.com/ship-monitor/cloud/internal/repository"
	devicesRepo "github.com/ship-monitor/cloud/internal/repository/devices"
	"github.com/ship-monitor/cloud/internal/services"
	"github.com/ship-monitor/cloud/internal/services/device"
	"github.com/ship-monitor/cloud/logger"
	"github.com/ship-monitor/cloud/pkg"
	"github.com/ship-monitor/cloud/pkg/middleware"
	"github.com/ship-monitor/cloud/workers"
	"github.com/ship-monitor/cloud/workers/server"
	"go.uber.org/fx"
)

func NewApp() *fx.App {
	return fx.New(
		fx.Provide(config.NewConfig),
		fx.Provide(logger.NewLogger),
		fx.Provide(
			NewRabbitMQClient,
			NewRedisClient,
			NewDatabaseClient,
			NewSpiceDB,
		),
		provideRepos(),
		provideServices(),
		provideHandlers(),
		fx.Provide(workers.NewQueue),
		fx.Provide(
			fx.Annotate(
				middleware.NewAuthMiddleware,
				fx.As(new(handlers.AuthMiddleware)),
			),
			middleware.NewAuthCookieManager,
			fx.Annotate(
				middleware.NewAuthCookieManager,
				fx.As(new(handlers.CookieManager)),
			),
		),
		fx.Provide(
			server.NewHTTPServer,
			fx.Annotate(server.NewServerMux, fx.ParamTags(`group:"handlers"`)),
		),
		fx.Invoke(func(server *http.Server) {}),
		fx.Invoke(func(server *workers.QueueWorker) {}),
	)
}

func provideRepos() fx.Option {
	return fx.Options(
		fx.Provide(
			fx.Annotate(
				devicesRepo.New,
				fx.As(
					new(device.StatesRepository),
				),
				fx.As(
					new(workers.RecordsService),
				),
			),
			fx.Annotate(
				repository.NewUsers,
				fx.As(new(services.UsersRepo)),
			),
			fx.Annotate(
				devicesRepo.New,
				fx.As(new(device.DeviceRepository)),
			),
			fx.Annotate(repository.NewRedisSessionStore,
				fx.As(new(services.SessionStore)),
			),
			pkg.AsMigrationRepo(repository.NewUsers),
			pkg.AsMigrationRepo(devicesRepo.New),
		),

		pkg.ProvideMigrations(),
	)
}

func provideServices() fx.Option {
	return fx.Options(
		fx.Provide(
			services.NewTopicPublisher,
			fx.Annotate(services.NewEmailService,
				fx.As(new(domain.EmailSender)),
			),
			fx.Annotate(
				services.NewAuthService,
				fx.As(new(handlers.AuthService)),
				fx.As(new(handlers.UserService)),
			),
			fx.Annotate(
				device.NewService,
				fx.As(new(handlers.DevicesService)),
			),
			services.NewSessions,
			fx.Annotate(
				services.NewSessions,
				fx.As(new(middleware.AuthService)),
				fx.As(new(handlers.SessionsService)),
			),
		),
	)
}

func provideHandlers() fx.Option {
	return fx.Options(
		fx.Provide(handlers.NewAuthHandlers,
			AsHandlers(handlers.NewAuthHandlers),
			AsHandlers(handlers.NewDevice),
			AsHandlers(handlers.NewUser),
		),
	)
}

func AsHandlers(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(pkg.Handler)),
		fx.ResultTags(`group:"handlers"`),
	)
}
