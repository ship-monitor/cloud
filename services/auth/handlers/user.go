package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/requests"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/auth/data"
)

func (a *AuthHandlers) HandleGetUser(ctx *gin.Context) {
	id := requests.MustGetParamUUID(ctx, "id")

	session := auth.GetSession(ctx)
	if session.UserID != id {
		ctx.AbortWithStatus(http.StatusNotFound)

		return
	}

	user, err := data.GetUser(id)
	if err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)

		return
	}

	ctx.JSON(http.StatusOK, gin.H{"user": user})
}

func (a *AuthHandlers) HandleGetUsersList(ctx *gin.Context) {
	page := 0

	pageStr, ok := ctx.GetQuery("page")
	if ok || pageStr != "" {
		var err error
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 0 {
			ctx.AbortWithStatusJSON(
				http.StatusBadRequest,
				gin.H{"details": "invalid page query parameter"},
			)

			return
		}
	}

	users, err := data.GetUsersList(page)
	if err != nil {
		ctx.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	ctx.JSON(http.StatusOK, gin.H{"users": users})
}

func (a *AuthHandlers) HandleUserSetPassword(ctx *gin.Context) {
	id := requests.MustGetParamUUID(ctx, "id")

	session := auth.GetSession(ctx)
	if session.UserID != id {
		ctx.AbortWithStatus(http.StatusNotFound)

		return
	}

	var request struct {
		Password string `binding:"required" json:"password"`
	}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"details": err.Error()})

		return
	}

	user, err := data.GetUser(id)
	if err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)

		return
	}

	err = user.SetPassword(request.Password)
	if err != nil {
		ctx.AbortWithStatus(http.StatusInternalServerError)

		return
	}
	ctx.JSON(http.StatusOK, gin.H{})
}

func (a *AuthHandlers) HandleUserSetEmail(ctx *gin.Context) {
	id := requests.MustGetParamUUID(ctx, "id")

	session := auth.GetSession(ctx)
	if session.UserID != id {
		ctx.AbortWithStatus(http.StatusNotFound)

		return
	}

	var request struct {
		Email string `binding:"required" json:"email"`
	}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"details": err.Error()})

		return
	}

	user, err := data.GetUser(id)
	if err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)

		return
	}

	err = user.SetEmail(request.Email)
	if err != nil {
		ctx.AbortWithStatus(http.StatusInternalServerError)

		return
	}
	ctx.JSON(http.StatusOK, gin.H{})
}

func (a *AuthHandlers) HandleUserBlock(ctx *gin.Context) {
	id := requests.MustGetParamUUID(ctx, "id")

	session := auth.GetSession(ctx)
	if session.UserID != id {
		ctx.AbortWithStatus(http.StatusNotFound)

		return
	}

	user, err := data.GetUser(id)
	if err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)

		return
	}

	err = user.Block()
	if err != nil {
		ctx.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	ctx.JSON(http.StatusOK, gin.H{})
}
