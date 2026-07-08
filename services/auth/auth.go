package auth

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/di"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/handlers"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/auth"
)

var _ handlers.AuthService = (*services.AuthService)(nil)

func SetupRoutes(
	ctx context.Context,
	router gin.IRouter,
	container *di.Container,
) error {
	middleware := auth.DefaultMiddleware(viper.GetViper())

	authService := container.AuthService()
	h := handlers.NewAuthHandlers(authService)

	auth := router.Group("/api/auth")
	auth.POST("/register", h.HandleRegister)
	auth.POST("/login", h.HandleLogin)
	auth.POST("/refresh", middleware.WithMiddleware, h.HandleRefresh)

	users := router.Group("/api/users", middleware.WithAuthenticationRequired)
	users.GET("/me", h.HandleGetUser)
	users.GET("/:id", h.HandleGetUser)
	users.POST("/:id/set-password", h.HandleUserSetPassword)
	users.POST("/:id/set-email", h.HandleUserSetEmail)
	users.POST("/start-email-confirmation", h.HandleStartEmailConfirmation)
	users.POST("/confirm-email/:token", h.HandleConfirmEmail)

	return nil
}
