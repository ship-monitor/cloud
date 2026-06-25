package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/requests"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/auth/data"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/auth/services"
)

func (a *AuthHandlers) HandleStartEmailConfirmation(ctx *gin.Context) {
	session := auth.GetSession(ctx)

	err := a.authService.StartEmailConfirmation(ctx.Request.Context(), session.UserID)
	if err != nil {
		if errors.Is(err, services.ErrEmailAlreadyConfirmed) {
			ctx.AbortWithStatusJSON(http.StatusNotModified, requests.BadResponse{
				Details: err.Error(),
			})

			return
		}

		ctx.AbortWithStatusJSON(
			http.StatusInternalServerError,
			requests.BadResponse{Details: err.Error()},
		)

		return
	}

	ctx.Status(http.StatusOK)
}

func (a *AuthHandlers) HandleConfirmEmail(ctx *gin.Context) {
	session := auth.GetSession(ctx)

	token := ctx.Param("token")

	err := a.authService.ConfirmEmail(ctx.Request.Context(), session.UserID, token)
	if err != nil {
		if errors.Is(err, services.ErrEmailAlreadyConfirmed) {
			ctx.AbortWithStatusJSON(http.StatusNotModified, requests.BadResponse{
				Details: err.Error(),
			})

			return
		}

		ctx.AbortWithStatusJSON(
			http.StatusInternalServerError,
			requests.BadResponse{Details: err.Error()},
		)

		return
	}

	ctx.Status(http.StatusOK)
}

func (a *AuthHandlers) HandleGetUser(ctx *gin.Context) {
	id := requests.MustGetParamUUID(ctx, "id")

	session := auth.GetSession(ctx)
	if session.UserID != id {
		ctx.AbortWithStatus(http.StatusNotFound)

		return
	}

	user, err := data.GetUser(ctx.Request.Context(), id)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusNotFound, requests.BadResponse{
			Details: err.Error(),
		})

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
				requests.BadResponse{
					Details: "invalid page query parameters",
				},
			)

			return
		}
	}

	users, err := data.GetUsersList(page)
	if err != nil {
		ctx.AbortWithStatusJSON(
			http.StatusInternalServerError,
			requests.BadResponse{Details: err.Error()},
		)

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
		ctx.AbortWithStatusJSON(http.StatusBadRequest, requests.BadResponse{
			Details: err.Error(),
		})

		return
	}

	user, err := data.GetUser(ctx.Request.Context(), id)
	if err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)

		return
	}

	err = data.SetPassword(user, request.Password)
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
		ctx.AbortWithStatusJSON(
			http.StatusForbidden,
			requests.BadResponse{Details: "not allowed to set email for this user"},
		)

		return
	}

	var request struct {
		Email string `binding:"required" json:"email"`
	}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, requests.BadResponse{Details: err.Error()})

		return
	}

	user, err := data.GetUser(ctx.Request.Context(), id)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusNotFound, requests.BadResponse{Details: err.Error()})

		return
	}

	err = data.SetEmail(user, request.Email)
	if err != nil {
		ctx.AbortWithStatusJSON(
			http.StatusInternalServerError,
			requests.BadResponse{Details: err.Error()},
		)

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

	user, err := data.GetUser(ctx.Request.Context(), id)
	if err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)

		return
	}

	err = data.Block(user)
	if err != nil {
		ctx.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	ctx.JSON(http.StatusOK, gin.H{})
}
