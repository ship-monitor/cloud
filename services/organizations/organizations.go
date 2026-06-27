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
	_ services.OrganizationsRepository        = (*repository.OrganizationsRepo)(nil)
	_ handlers.OrganizationService            = (*services.OrganizationsService)(nil)
	_ handlers.OrganizationMembersService     = (*services.OrganizationsService)(nil)
	_ handlers.OrganizationInvitationsService = (*services.OrganizationsService)(nil)
	_ handlers.OrgDevicesService              = (*services.OrgDevicesService)(nil)
)

func SetupRoutes(ctx context.Context, router gin.IRouter, container *di.Container) error {
	middleware := auth.DefaultMiddleware(viper.GetViper())

	api := router.Group("/api", middleware.WithAuthenticationRequired)

	orgsService := container.OrganizationsService()
	orgDevicesService := container.OrgDevicesService()

	webHandler := handlers.New(orgsService, orgsService, orgsService, orgDevicesService)
	devicesHandler := container.DevicesHandlers()

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

	// Invitation routes
	orgs.POST("/:id/invitations", webHandler.HandleCreateInvitation)
	orgs.GET("/:id/invitations", webHandler.HandleListOrgInvitations)
	api.GET("/invitations", webHandler.HandleListMyInvitations)
	api.POST("/invitations/:id/accept", webHandler.HandleAcceptInvitation)
	api.POST("/invitations/:id/decline", webHandler.HandleDeclineInvitation)

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
