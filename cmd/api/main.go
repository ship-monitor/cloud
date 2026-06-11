package main

import (
	"time"

	"charm.land/log/v2"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/config"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/keyval"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/connector"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations"
)

func main() {
	config.Setup()

	db.Setup()
	keyval.Setup()
	defer keyval.Close()

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
		MaxAge:           12 * time.Hour,
	}))

	auth.SetupRoutes(server)
	organizations.SetupRoutes(server)
	if viper.GetBool("services.connector.enable") {
		log.Info("Connector service enabled")
		connector.Setup(server)
	} else {
		log.Warn("Connector service disabled")
	}

	if err := server.Run(":8080"); err != nil {
		log.Fatal("failed to start server", "error", err)
	}
}
