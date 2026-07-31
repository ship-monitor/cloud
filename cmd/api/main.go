package main

import (
	"net/http"

	"go.uber.org/fx"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/config"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/di"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/handlers"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/repository"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/logger"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/middleware"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/workers"
)

func main() {
	fx.New(
		fx.Provide(config.NewConfig),
		fx.Provide(logger.NewLogger),
		fx.Provide(
			di.NewRabbitMQClient,
			di.NewRedisClient,
			di.NewDatabaseClient,
			di.NewSpiceDB,
		),
		provideRepos(),
		provideServices(),
		provideHandlers(),
		fx.Provide(workers.NewQueue),
		fx.Provide(
			middleware.NewAuthMiddleware,
			middleware.NewAuthCookieManager,
		),
		fx.Provide(
			workers.NewHTTPServer,
			fx.Annotate(workers.NewServerMux, fx.ParamTags(`group:"handlers"`)),
		),
		fx.Invoke(func(server *http.Server) {}),
		fx.Invoke(func(server *workers.QueueWorker) {}),
	).Run()
}

func provideRepos() fx.Option {
	return fx.Options(
		fx.Provide(
			fx.Annotate(
				repository.NewDeviceStatesRepo,
				fx.As(
					new(services.StatesRepository),
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
				repository.NewDevices,
				fx.As(new(services.DeviceRepository)),
			),
			fx.Annotate(repository.NewRedisSessionStore,
				fx.As(new(services.SessionStore)),
			),
			pkg.AsMigrationRepo(repository.NewUsers),
			pkg.AsMigrationRepo(repository.NewDevices),
			pkg.AsMigrationRepo(repository.NewDevices),
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
				services.NewDevices,
				fx.As(new(handlers.DevicesService)),
			),
			services.NewSessions,
			fx.Annotate(
				services.NewSessions,
				fx.As(new(middleware.AuthService)),
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
		fx.ResultTags(`group:"handlers"`))
}
