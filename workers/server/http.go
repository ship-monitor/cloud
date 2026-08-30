package server

import (
	"context"
	"errors"
	"net/http"

	"charm.land/log/v2"
	"github.com/labstack/echo/v5"
	echoMiddleware "github.com/labstack/echo/v5/middleware"
	"github.com/ship-monitor/cloud/pkg"
	"go.uber.org/fx"
)

func NewServerMux(handlers []pkg.Handler, config *Config) http.Handler {
	mux := echo.New()
	mux.Use(echoMiddleware.RequestLogger())
	mux.Use(echoMiddleware.Recover())
	mux.Use(echoMiddleware.CORS(config.CORS.AllowedOrigins...))

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
	config *Config,
	logger *log.Logger, handler http.Handler,
) *http.Server {
	server := &http.Server{ //nolint:exhaustruct_v5
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		Addr:              config.Host(),
		Handler:           handler,
	}

	lc.Append(
		fx.Hook{
			OnStart: func(ctx context.Context) error {
				go func() {
					logger.Info("Starting HTTP server")

					err := server.ListenAndServe()
					if err != nil && !errors.Is(err, http.ErrServerClosed) {
						panic(err)
					}
				}()

				return nil
			},
			OnStop: func(ctx context.Context) error {
				return server.Shutdown(ctx)
			},
		},
	)

	return server
}
