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
	auth := router.Group("/api/auth")
	auth.POST("/register", handlers.HandleRegister)
	auth.POST("/login", handlers.HandleLogin)
	auth.POST("/refresh", middleware.WithMiddleware, handlers.HandleRefresh)

	users := router.Group("/api/users", middleware.WithAuthenticationRequired)
	users.GET("/:id", handlers.HandleGetUser)
	users.GET("/", handlers.HandleGetUsersList)
	users.POST("/:id/set-password", handlers.HandleUserSetPassword)
	users.POST("/:id/set-email", handlers.HandleUserSetEmail)
	users.POST("/:id/block", handlers.HandleUserBlock)
}
