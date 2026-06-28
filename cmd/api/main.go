package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"charm.land/log/v2"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/config"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/di"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/workers"
)

const maxAge = 12 * time.Hour

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	config.Setup()

	if viper.GetBool("devel") {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	log.SetReportCaller(true)

	container := di.NewContainer(viper.GetViper(), log.Default())

	err := migrate(ctx, container)
	if err != nil {
		log.Error("Failed migrate database", "error", err)

		return
	}

	group, groupCtx := errgroup.WithContext(ctx)

	group.Go(func() error {
		return runServer(groupCtx, container)
	})

	group.Go(func() error {
		q := workers.NewQueue(
			container.RabbitMQ(),
			container.Redis(),
			container.DeviceStates(),
			container.Logger(),
		)

		return q.Serve(groupCtx)
	})

	err = group.Wait()
	if err != nil {
		log.Error("Failed start", "error", err)
	}
}

func migrate(ctx context.Context, c *di.Container) error {
	if err := c.OrganizationsRepo().Migrate(ctx); err != nil {
		return fmt.Errorf("failed migrate organizations: %w", err)
	}

	if err := c.UsersRepo().Migrate(ctx); err != nil {
		return fmt.Errorf("failed migrate users: %w", err)
	}

	return nil
}

func runServer(ctx context.Context, container *di.Container) error {
	server := gin.Default()

	if viper.GetBool("devel") {
		gin.SetMode(gin.DebugMode)
	}

	server.Use(func(ctx *gin.Context) {
		logger := container.Logger().WithPrefix("HTTP Middleware")

		ctx.Next()

		if ctx.Writer.Status() >= http.StatusBadRequest {
			logger.Error("Handled error",
				"status", ctx.Writer.Status(),
				"path", ctx.Request.URL.Path,
				"method", ctx.Request.Method,
			)
		}
	})
	server.Use(cors.New(cors.Config{
		AllowOrigins: viper.GetStringSlice("cors.allow-origins"),
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
			"OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
		},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           maxAge,
	}))

	server.GET("/api/health", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	err := auth.SetupRoutes(ctx, server, container)
	if err != nil {
		return fmt.Errorf("setup auth routes: %w", err)
	}

	err = organizations.SetupRoutes(ctx, server, container)
	if err != nil {
		return fmt.Errorf("setup organizations routes: %w", err)
	}

	if err := server.Run(":8080"); err != nil {
		return fmt.Errorf("run server: %w", err)
	}

	return nil
}
