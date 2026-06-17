package organizations

import (
	"context"

	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/commands"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/data"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/handlers"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/repository"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/services"
)

var (
	_ handlers.OrganizationService     = (*services.OrganizationsService)(nil)
	_ services.OrganizationsRepository = (*repository.OrganizationsRepo)(nil)
	_ handlers.DevicesService          = (*services.DevicesService)(nil)
)

func SetupRoutes(router gin.IRouter) {
	// Запускаем миграции
	data.Migrate()

	if err := commands.Connect(); err != nil {
		log.Fatal("Failed connect to commands queue", "error", err)
	}

	middleware := auth.DefaultMiddleware(viper.GetViper())

	api := router.Group("/api", middleware.WithAuthenticationRequired)

	orgsRepository := repository.New(db.DB.DB)
	if err := orgsRepository.Migrate(context.Background()); err != nil {
		log.Fatal("Failed migrate organizations schema")
	}

	orgsService := services.NewOrganizations(orgsRepository)
	devicesService := services.NewDevices(orgsService)

	webHandler := handlers.New(orgsService, devicesService)

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
	orgs.POST("/:id/devices", handlers.HandleConnectDevice)
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
}
