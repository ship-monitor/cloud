package main

import (
	"context"
	"errors"
	"net"
	"net/http"

	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
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
			newHTTPServer,
			fx.Annotate(newServerMux, fx.ParamTags(`group:"handlers"`)),
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

func newServerMux(handlers []pkg.Handler) http.Handler {
	gin.SetMode(gin.ReleaseMode)

	engine := gin.Default()
	engine.Use(middleware.NewCORS().Middleware())
	engine.GET("/api/health", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	for _, h := range handlers {
		h.SetupRoutes(engine)
	}

	return engine
}

func newHTTPServer(
	lc fx.Lifecycle,
	config *viper.Viper,
	logger *log.Logger, handler http.Handler,
) *http.Server {
	server := &http.Server{
		ReadHeaderTimeout: viper.GetDuration("http.server.read-header-timeout"),
		Addr: net.JoinHostPort(
			"",
			viper.GetString("http.server.port"),
		),
		Handler: handler,
	}

	lc.Append(
		fx.Hook{
			OnStart: (func(ctx context.Context) error {
				go func() {
					logger.Info("Starting HTTP server")

					err := server.ListenAndServe()
					if err != nil && !errors.Is(err, http.ErrServerClosed) {
						panic(err)
					}
				}()

				return nil
			}),
			OnStop: func(ctx context.Context) error {
				return server.Shutdown(ctx)
			},
		})

	return server
}
