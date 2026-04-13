package organizations

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/auth"
)

func SetupRoutes(router gin.IRouter) {
	middleware := auth.DefaultMiddleware(viper.GetViper())
	orgs := router.Group("/api/organizations", middleware.WithMiddleware)
	orgs.POST("/")
	orgs.GET("/:id")
	orgs.PATCH("/:id")
	orgs.DELETE("/:id")

}
