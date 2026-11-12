package di

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/cors"
	"github.com/ship-monitor/cloud/internal/v2/adapters/server"
	"github.com/ship-monitor/cloud/internal/v2/config"
	"github.com/ship-monitor/cloud/internal/v2/gen/api"
	"github.com/ship-monitor/cloud/internal/v2/handlers/handler"
	"github.com/ship-monitor/cloud/internal/v2/handlers/queue"
	queuesecurity "github.com/ship-monitor/cloud/internal/v2/handlers/queue_security"
	"github.com/ship-monitor/cloud/internal/v2/handlers/security"
	aliverepo "github.com/ship-monitor/cloud/internal/v2/repositories/alive"
	"github.com/ship-monitor/cloud/internal/v2/repositories/connect"
	devicerepo "github.com/ship-monitor/cloud/internal/v2/repositories/device"
	devicemetricsrepo "github.com/ship-monitor/cloud/internal/v2/repositories/devicemetrics"
	sensorrepo "github.com/ship-monitor/cloud/internal/v2/repositories/sensor"
	sensormetricsrepo "github.com/ship-monitor/cloud/internal/v2/repositories/sensormetrics"
	userrepo "github.com/ship-monitor/cloud/internal/v2/repositories/user"
	"github.com/ship-monitor/cloud/internal/v2/services/device"
	"github.com/ship-monitor/cloud/internal/v2/services/sensor"
	"github.com/ship-monitor/cloud/internal/v2/services/token"
	"github.com/ship-monitor/cloud/internal/v2/services/user"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewApp(config *config.Config) *fx.App {
	return fx.New(
		logger(config),
		fx.Supply(config),
		fx.Provide(newDB),
		fx.Provide(newClickHouse),
		fx.Provide(newNATS),
		fx.Provide(newValkey),
		fx.Provide(userrepo.New),
		fx.Provide(devicerepo.New),
		fx.Provide(aliverepo.New),
		fx.Provide(connect.New),
		fx.Provide(sensorrepo.New),
		fx.Provide(devicemetricsrepo.New),
		fx.Provide(sensormetricsrepo.New),
		fx.Provide(newTokenSerice),
		fx.Provide(newUserService),
		fx.Provide(newDeviceService),
		fx.Provide(newSensorsService),
		fx.Provide(newHandler),
		fx.Provide(newSecurityHandler),
		fx.Provide(newQueueHandler),
		fx.Provide(newQueueSecurityHandler),
		fx.Provide(newServer),
		fx.Provide(newHTTPServer),
		fx.Provide(newPublisher),
		provideConnector(),
		deprecations(config),
		fx.Invoke(func(*http.Server) {}),
	)
}

func logger(config *config.Config) fx.Option {
	return fx.Module("logger",
		fx.Provide(func() (*zap.Logger, error) {
			logConf := zap.NewDevelopmentConfig()

			logConf.DisableStacktrace = true

			logConf.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
			if config.IsDevelopment() {
				logConf.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
			}

			logger, err := logConf.Build()
			if err != nil {
				return nil, fmt.Errorf("build logger from config: %w", err)
			}

			return logger.Named("SHIP"), nil
		}),
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{
				Logger: log,
			}
		}),
	)
}

func newHandler(
	tokens *token.Service,
	users *user.Service,
	devices *device.Service,
	logger *zap.Logger,
	sensors *sensor.Service,
) *handler.Handler {
	return handler.New(tokens, users, devices, sensors, logger)
}

func newSecurityHandler(t *token.Service) *security.Handler {
	return security.New(t)
}

func newServer(
	logger *zap.Logger,
	handler *handler.Handler,
	security *security.Handler,
) (*api.Server, error) {
	server, err := server.New(logger, handler, security)
	if err != nil {
		return nil, fmt.Errorf("new server: %w", err)
	}

	return server, nil
}

const ReadHeaderTimeout = time.Second * 5

func newHTTPServer(
	lc fx.Lifecycle,
	config *config.Config,
	handler *api.Server,
	logger *zap.Logger,
) *http.Server {
	logger = logger.Named("HTTP")

	srvHandler := http.Handler(handler)

	if config.IsDevelopment() {
		c := getCORS(config, logger)
		srvHandler = c.Handler(srvHandler)
	}

	server := http.Server{ //nolint:exhaustruct_v5
		ReadHeaderTimeout: ReadHeaderTimeout,
		Handler:           srvHandler,
		Addr:              net.JoinHostPort(config.Host, config.Port),
	}

	url := url.URL{ //nolint:exhaustruct_v5
		Scheme: "http",
		Host:   server.Addr,
	}

	lc.Append(fx.StartHook(func() {
		go func() {
			logger.Info("Starting", zap.String("addr", url.String()))

			err := server.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Error("Failed serve", zap.Error(err))
			} else {
				logger.Info("Stopped")
			}
		}()
	}))

	lc.Append(fx.StopHook(func(ctx context.Context) error {
		err := server.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}

		return nil
	}))

	return &server
}

func getCORS(config *config.Config, logger *zap.Logger) *cors.Cors {
	if len(config.CorsOrigins) == 0 {
		logger.Warn("No CORS origins specified")
	}

	return cors.New(cors.Options{ //nolint:exhaustruct_v5
		AllowedOrigins: config.CorsOrigins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	})
}

func newQueueHandler(
	repo *aliverepo.Repo,
	device *device.Service,
	logger *zap.Logger,
) *queue.Handler {
	return queue.New(repo, device, logger)
}

func newQueueSecurityHandler(logger *zap.Logger) *queuesecurity.Handler {
	return queuesecurity.New(logger)
}
