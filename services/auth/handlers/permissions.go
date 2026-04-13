package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/auth/data"
)

func HandleGetPermissions(ctx *gin.Context) {
	permissions := data.GetAllPermissions()
	ctx.JSON(http.StatusOK, gin.H{"permissions": permissions})
}
