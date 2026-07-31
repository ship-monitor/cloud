package workers

import (
	"context"
	"errors"
	"net"
	"net/http"

	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/middleware"
)

func NewServerMux(handlers []pkg.Handler) http.Handler {
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

func NewHTTPServer(
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
