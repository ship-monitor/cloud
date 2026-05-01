package organizations

import (
	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/commands"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/data"
)

func SetupRoutes(router gin.IRouter) {
	// Запускаем миграции
	data.Migrate()

	if err := commands.Connect(); err != nil {
		log.Fatal("Failed connect to commands queue", "error", err)
	}

	middleware := auth.DefaultMiddleware(viper.GetViper())

	api := router.Group("/api", middleware.WithAuthenticationRequired)

	// Роуты с проверкой аутентификаци
	orgs := api.Group("/organizations")
	orgs.POST("/", HandleCreateOrganization)
	orgs.GET("/my", HandleGetMyOrganizations)
	orgs.GET("/:id", HandleGetOrganization)
	orgs.PATCH("/:id", HandleUpdateOrganization)
	orgs.DELETE("/:id", HandleDeleteOrganization)

	// Роуты участников организации
	orgs.GET("/:id/members", HandleGetMembers)
	orgs.POST("/:id/members", HandleAddMember)
	orgs.PATCH("/:id/members/:userId", HandleUpdateMemberRole)
	orgs.DELETE("/:id/members/:userId", HandleRemoveMember)

	// Invitation routes
	orgs.POST("/:id/invitations", HandleCreateInvitation)
	orgs.GET("/:id/invitations", HandleListOrgInvitations)
	api.GET("/invitations", HandleListMyInvitations)
	api.POST("/invitations/:id/accept", HandleAcceptInvitation)
	api.POST("/invitations/:id/decline", HandleDeclineInvitation)
}
