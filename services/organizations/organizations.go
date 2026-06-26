package organizations

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/di"
	repository "sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/repositories"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/handlers"
)

var (
	_ services.OrganizationsRepository = (*repository.OrganizationsRepo)(nil)
	_ handlers.OrgDevicesService       = (*services.OrgDevicesService)(nil)
)

func SetupRoutes(ctx context.Context, router gin.IRouter, container *di.Container) error {
	middleware := auth.DefaultMiddleware(viper.GetViper())

	api := router.Group("/api", middleware.WithAuthenticationRequired)

	orgsService := container.OrganizationsService()
	orgDevicesService := container.OrgDevicesService()

	webHandler := handlers.New(orgsService, orgDevicesService)
	devicesHandler := container.DevicesHandlers()

	// Роуты с проверкой аутентификации
	orgs := api.Group("/organizations")
	orgs.POST("/", webHandler.HandleCreateOrganization)
	orgs.GET("/my", handlers.HandleGetMyOrganizations)
	orgs.GET("/:id", webHandler.HandleGetOrganization)
	orgs.PATCH("/:id", handlers.HandleUpdateOrganization)
	orgs.DELETE("/:id", handlers.HandleDeleteOrganization)

	// Роуты участников организации
	orgs.GET("/:id/members", handlers.HandleGetMembers)
	orgs.POST("/:id/members", handlers.HandleAddMember)
	orgs.PATCH("/:id/members/:userId", handlers.HandleUpdateMemberRole)
	orgs.DELETE("/:id/members/:userId", handlers.HandleRemoveMember)

	// Device routes
	orgs.POST("/:id/devices", webHandler.HandleConnectDevice)
	orgs.GET("/:id/devices", handlers.HandleListDevices)
	orgs.GET("/:id/devices/:deviceId", handlers.HandleGetDevice)
	orgs.PATCH("/:id/devices/:deviceId", handlers.HandlePatchDevice)
	orgs.DELETE("/:id/devices/:deviceId", handlers.HandleDisconnectDevice)

	// Invitation routes
	orgs.POST("/:id/invitations", handlers.HandleCreateInvitation)
	orgs.GET("/:id/invitations", handlers.HandleListOrgInvitations)
	api.GET("/invitations", handlers.HandleListMyInvitations)
	api.POST("/invitations/:id/accept", handlers.HandleAcceptInvitation)
	api.POST("/invitations/:id/decline", handlers.HandleDeclineInvitation)

	// Devices separate routes
	api.GET("/devices/:id", webHandler.HandleGetDevice)
	api.PATCH("/devices/:id", webHandler.HandlePatchDevice)
	api.DELETE("/devices/:id", webHandler.HandleDisconnectDevice)

	api.GET(
		"/v2/devices/:id/state/:state",
		middleware.WithAuthenticationRequired,
		devicesHandler.HandleGetState,
	)

	return nil
}
