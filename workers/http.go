package workers

import (
	"context"
	"errors"
	"net"
	"net/http"

	"charm.land/log/v2"
	"github.com/labstack/echo/v5"
	echoMiddleware "github.com/labstack/echo/v5/middleware"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg"
)

func NewServerMux(handlers []pkg.Handler, config *viper.Viper) http.Handler {
	mux := echo.New()
	mux.Use(echoMiddleware.RequestLogger())
	mux.Use(echoMiddleware.Recover())
	mux.Use(echoMiddleware.CORS(config.GetStringSlice("cors.allow-origins")...))

	mux.GET("/api/health", func(ctx *echo.Context) error {
		return ctx.JSON(http.StatusOK, map[string]string{
			"ok": "ok",
		})
	})

	for _, h := range handlers {
		h.SetupRoutes(mux.Group("/"))
	}

	return mux
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
