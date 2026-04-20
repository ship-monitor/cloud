package organizations

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/data"
)

func SetupRoutes(router gin.IRouter) {
	// Запускаем миграции
	data.Migrate()

	middleware := auth.DefaultMiddleware(viper.GetViper())

	// Роуты с проверкой аутентификации
	orgs := router.Group("/api/organizations", middleware.WithAuthenticationRequired)
	orgs.POST("/", HandleCreateOrganization)
	orgs.GET("/my", HandleGetMyOrganizations)
	orgs.GET("/:id", HandleGetOrganization)
	orgs.PATCH("/:id", HandleUpdateOrganization)
	orgs.DELETE("/:id", HandleDeleteOrganization)
}
