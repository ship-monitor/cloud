package organizations

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/di"
	repository "sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/repository"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/handlers"
)

var (
	_ services.OrganizationsRepository = (*repository.OrganizationsRepo)(
		nil,
	)
	_ handlers.OrganizationService = (*services.OrganizationsService)(
		nil,
	)
	_ handlers.OrganizationMembersService = (*services.OrganizationsService)(
		nil,
	)
)

func SetupRoutes(
	router gin.IRouter,
	container *di.Container,
) error {
	sessionStore := auth.NewRedisSessionStore(
		container.Redis(),
		auth.SessionTTL(viper.GetViper()),
	)
	middleware := auth.NewMiddleware(sessionStore, container.Config())

	api := router.Group("/api", middleware.WithAuthenticationRequired)

	orgsService := container.OrganizationsService()

	webHandler := handlers.New(
		orgsService,
		orgsService,
	)

	// Роуты с проверкой аутентификации
	orgs := api.Group("/organizations")
	orgs.POST("/", webHandler.HandleCreateOrganization)
	orgs.GET("/my", webHandler.HandleGetMyOrganizations)
	orgs.GET("/:id", webHandler.HandleGetOrganization)
	orgs.PATCH("/:id", webHandler.HandleUpdateOrganization)
	orgs.DELETE("/:id", webHandler.HandleDeleteOrganization)

	// Роуты участников организации
	orgs.GET("/:id/members", webHandler.HandleGetMembers)
	orgs.POST("/:id/members", webHandler.HandleAddMember)
	orgs.PATCH("/:id/members/:userId", webHandler.HandleUpdateMemberRole)
	orgs.DELETE("/:id/members/:userId", webHandler.HandleRemoveMember)

	return nil
}
