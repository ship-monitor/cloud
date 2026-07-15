package main

import (
	"context"
	"errors"
	"fmt"
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
	auth "sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/cors"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations"
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
		),
		provideServices(),
		provideHandlers(),
		provideRepos(),
		fx.Provide(workers.NewQueue),
		fx.Provide(auth.NewMiddleware),
		fx.Provide(newHTTPServer),
		fx.Invoke(func(server *gin.Engine) {}),
		fx.Invoke(func(server *workers.QueueWorker) {}),
	).Run()
}

func provideRepos() fx.Option {
	return fx.Options(
		fx.Provide(repository.NewDeviceStatesRepo,
			fx.Annotate(
				repository.NewDeviceStatesRepo,
				fx.As(new(workers.RecordsService)),
			),
		),
		fx.Provide(
			repository.NewOrgs,
			fx.Annotate(
				repository.NewOrgs,
				fx.As(new(pkg.MigrationRepo)),
				fx.ResultTags(`name:"organizations-repo"`),
			),
		),
		fx.Provide(repository.NewUsers,
			fx.Annotate(
				repository.NewUsers,
				fx.As(new(pkg.MigrationRepo)),
				fx.ResultTags(`name:"users-repo"`),
			),
		),
		fx.Provide(repository.NewDevices,
			fx.Annotate(
				repository.NewDevices,
				fx.As(new(pkg.MigrationRepo)),
				fx.ResultTags(`name:"devices-repo"`),
			),
		),
		pkg.ProvideMigrations(),
	)
}

func provideServices() fx.Option {
	return fx.Options(
		fx.Provide(services.NewEmailService,
			fx.Annotate(services.NewEmailService,
				fx.As(new(domain.EmailSender)),
			),
		),
	)
}

func provideHandlers() fx.Option {
	return fx.Options(
		fx.Provide(handlers.NewAuthHandlers,
			fx.Annotate(
				handlers.NewAuthHandlers,
				fx.As(new(pkg.Handler)),
				fx.ResultTags(`name:"auth-handlers"`),
			)),
		fx.Provide(handlers.NewDevice,
			fx.Annotate(
				handlers.NewDevice,
				fx.As(new(pkg.Handler)),
				fx.ResultTags(`name:"device-handlers"`),
			),
		),
	)
}

func newHTTPServer(
	lc fx.Lifecycle,
	config *viper.Viper,
	logger *log.Logger, handlers ...pkg.Handler,
) (*gin.Engine, error) {
	container := di.NewContainer(viper.GetViper(), log.Default())

	gin.SetMode(gin.ReleaseMode)

	engine := gin.Default()

	if config.GetBool("devel") {
		logger.Debug("Setting gin mode to debug")
		gin.SetMode(gin.DebugMode)
	}

	engine.Use(cors.New().Middleware())

	engine.GET("/api/health", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	for _, h := range handlers {
		h.SetupRoutes(engine)
	}

	if err := organizations.SetupRoutes(engine, container); err != nil {
		return nil, fmt.Errorf("setup organizations routes: %w", err)
	}

	server := &http.Server{
		ReadHeaderTimeout: viper.GetDuration("http.server.read-header-timeout"),
		Addr: net.JoinHostPort(
			"",
			viper.GetString("http.server.port"),
		),
		Handler: engine,
	}

	lc.Append(
		fx.Hook{
			OnStart: (func(ctx context.Context) error {
				go func() {
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

	return engine, nil
}
