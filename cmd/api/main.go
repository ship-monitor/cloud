package main

import (
	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/config"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/keyval"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations"
)

func main() {
	config.Setup()
	db.Setup()
	keyval.Setup()
	defer keyval.Close()

	config.Config.RegisterAlias("jwt-security-key", "security-key")
	server := gin.Default()
	
	auth.SetupRoutes(server)
	organizations.SetupRoutes(server)

	if err := server.Run(":8080"); err != nil {
		log.Fatal("failed to start server", "error", err)
	}
}
