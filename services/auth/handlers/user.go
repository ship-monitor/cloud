package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/katy248/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/auth/data"
)

func HandleGetUser(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"details": "invalid id"})
		return
	}

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

func HandleGetUsersList(ctx *gin.Context) {
	page := 0

	pageStr, ok := ctx.GetQuery("page")
	if ok || pageStr != "" {
		var err error
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 0 {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"details": "invalid page query parameter"})
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
func HandleUserSetPassword(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"details": "invalid id"})
		return
	}

	session := auth.GetSession(ctx)
	if session.UserID != id {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	var request struct {
		Password string `json:"password" binding:"required"`
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
func HandleUserSetEmail(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"details": "invalid id"})
		return
	}

	session := auth.GetSession(ctx)
	if session.UserID != id {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	var request struct {
		Email string `json:"email" binding:"required"`
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

func HandleUserBlock(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"details": "invalid id"})
		return
	}

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
