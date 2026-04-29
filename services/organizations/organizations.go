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

	// Роуты приглашений
	api.POST("/:id/invitations", HandleCreateInvitation)
	api.GET("/invitations", HandleListInvitations)
	api.POST("/invitations/:token/accept", HandleAcceptInvitation)
	api.POST("/invitations/:token/decline", HandleDeclineInvitation)
}
