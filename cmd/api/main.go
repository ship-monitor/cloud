package main

import (
	"context"
	"net/http"
	"time"

	"charm.land/log/v2"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/config"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/connector"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations"
)

const maxAge = 12 * time.Hour

func main() {
	config.Setup()

	db.Setup()

	if viper.GetBool("devel") {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	config.Config.RegisterAlias("jwt-security-key", "security-key")

	server := gin.Default()

	if viper.GetBool("devel") {
		gin.SetMode(gin.DebugMode)
	}

	server.Use(cors.New(cors.Config{
		AllowOrigins: viper.GetStringSlice("cors.allow-origins"),
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
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
		ctx.JSON(http.StatusOK, gin.H{
			"healthy": true,
		})
	})

	auth.SetupRoutes(server)
	organizations.SetupRoutes(server)

	if viper.GetBool("services.connector.enable") {
		log.Info("Connector service enabled")
		connector.Setup(context.Background(), server)
	} else {
		log.Warn("Connector service disabled")
	}

	err := server.Run(":8080")
	if err != nil {
		log.Error("failed to start server", "error", err)
	}
}
