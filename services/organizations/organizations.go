package organizations

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/di"
	repository "sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/repositories"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/commands"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/handlers"
)

var (
	_ services.OrganizationsRepository = (*repository.OrganizationsRepo)(nil)
	_ handlers.OrgDevicesService       = (*services.OrgDevicesService)(nil)
)

func SetupRoutes(ctx context.Context, router gin.IRouter, container *di.Container) error {
	// Запускаем миграции
	err := commands.Connect()
	if err != nil {
		return fmt.Errorf("connect to commands queue: %w", err)
	}

	middleware := auth.DefaultMiddleware(viper.GetViper())

	api := router.Group("/api", middleware.WithAuthenticationRequired)

	// TODO: remove migrations
	orgsRepository := repository.NewOrgs(db.DB.DB)

	err = orgsRepository.Migrate(ctx)
	if err != nil {
		return fmt.Errorf("migrate organizations schema: %w", err)
	}

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
	orgs.POST("/:id/devices/:deviceId/command", handlers.HandleSendCommand)

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
	api.POST("/devices/:id/command", webHandler.HandleSendCommand)

	api.GET(
		"/v2/devices/:id/state/:state",
		middleware.WithAuthenticationRequired,
		devicesHandler.HandleGetState,
	)

	return nil
}
