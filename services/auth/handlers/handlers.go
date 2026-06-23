package handlers

import (
	"context"
	"errors"
	"net/http"

	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/requests"
)

type AuthService interface {
	StartEmailConfirmation(ctx context.Context, userID uuid.UUID) error
	ConfirmEmail(ctx context.Context, userID uuid.UUID, token string) error
}

type AuthHandlers struct {
	logger      *log.Logger
	authService AuthService
}

func NewAuthHandlers(authService AuthService) *AuthHandlers {
	return &AuthHandlers{
		authService: authService,
		logger:      log.WithPrefix("Auth handlers"),
	}
}

func (a *AuthHandlers) bindJSON(ctx *gin.Context, data any) {
	err := ctx.ShouldBindJSON(data)
	if err != nil {
		if vErrors, ok := errors.AsType[validator.ValidationErrors](err); ok {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"details":          "body validation fails",
				"validationErrors": mapErrors(vErrors),
			})
			a.logger.Error("Validation fails", "error", err)
		}

		a.logger.Error("Failed to bind JSON", "error", err)
		ctx.AbortWithStatusJSON(
			http.StatusBadRequest,
			requests.BadResponse{Details: err.Error()},
		)

		return
	}
}

func mapErrors(vErrors validator.ValidationErrors) []gin.H {
	res := make([]gin.H, 0, len(vErrors))

	for _, err := range vErrors {
		res = append(res, gin.H{
			"field":       err.Field(),
			"error":       err.Error(),
			"actualTag":   err.ActualTag(),
			"tag":         err.Tag(),
			"structField": err.StructField(),
		})
	}

	return res
}
