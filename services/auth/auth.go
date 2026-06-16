package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/auth/data"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/auth/handlers"
)

func SetupRoutes(router gin.IRouter) {
	data.Migrate()
	middleware := auth.DefaultMiddleware(viper.GetViper())

	h := handlers.NewAuthHandlers()
	auth := router.Group("/api/auth")
	auth.POST("/register", h.HandleRegister)
	auth.POST("/login", h.HandleLogin)
	auth.POST("/refresh", middleware.WithMiddleware, h.HandleRefresh)

	users := router.Group("/api/users", middleware.WithAuthenticationRequired)
	users.GET("/:id", h.HandleGetUser)
	users.GET("/", h.HandleGetUsersList)
	users.POST("/:id/set-password", h.HandleUserSetPassword)
	users.POST("/:id/set-email", h.HandleUserSetEmail)
	users.POST("/:id/block", h.HandleUserBlock)
}
