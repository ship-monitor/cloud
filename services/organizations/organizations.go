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
	_ handlers.OrgDevicesService = (*services.OrgDevicesService)(
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
	orgDevicesService := container.OrgDevicesService()

	webHandler := handlers.New(
		orgsService,
		orgsService,
		orgDevicesService,
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

	// Device routes
	orgs.POST("/:id/devices", webHandler.HandleConnectDevice)
	orgs.GET("/:id/devices", webHandler.HandleListDevices)
	orgs.GET("/:id/devices/:deviceId", webHandler.HandleGetDevice)
	orgs.PATCH("/:id/devices/:deviceId", webHandler.HandlePatchDevice)
	orgs.DELETE("/:id/devices/:deviceId", webHandler.HandleDisconnectDevice)

	// Devices separate routes
	api.GET("/devices/:id", webHandler.HandleGetDevice)
	api.PATCH("/devices/:id", webHandler.HandlePatchDevice)
	api.DELETE("/devices/:id", webHandler.HandleDisconnectDevice)

	return nil
}
