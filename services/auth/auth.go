package auth

import (
	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	intservices "sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/auth/data"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/auth/handlers"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/auth/services"
)

var _ handlers.AuthService = (*services.AuthService)(nil)

func SetupRoutes(router gin.IRouter) {
	data.Migrate()

	middleware := auth.DefaultMiddleware(viper.GetViper())

	rdb := redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})

	email, err := intservices.NewEmailService(intservices.EmailServiceConfig{
		SMTPHost:     viper.GetString("email.smtp-host"),
		SMTPPort:     viper.GetUint("email.smtp-port"),
		SenderName:   viper.GetString("email.sender-name"),
		AuthEmail:    viper.GetString("email.email"),
		AuthPassword: viper.GetString("email.password"),
	})
	if err != nil {
		panic(err)
	}

	authService := services.NewAuthService(log.Default(), rdb, email)
	h := handlers.NewAuthHandlers(authService)

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
	users.POST("/start-email-confirmation", h.HandleStartEmailConfirmation)
	users.POST("/confirm-email/:token", h.HandleConfirmEmail)
}
