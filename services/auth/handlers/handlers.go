package handlers

import (
	"context"
	"net/http"

	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type AuthService interface {
	StartEmailConfirmation(ctx context.Context, userID uuid.UUID) error
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
	if err := ctx.ShouldBindJSON(data); err != nil {
		if vErrors, ok := err.(validator.ValidationErrors); ok {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"details":          "body validation fails",
				"validationErrors": mapErrors(vErrors),
			})
			// panic("validation fails")
			a.logger.Error("Validation fails", "error", err)
		}
		a.logger.Error("Failed to bind JSON", "error", err)
		ctx.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{"details": "failed get JSON body: " + err.Error()},
		)

		return
	}
}

func mapErrors(vErrors validator.ValidationErrors) []gin.H {
	res := []gin.H{}

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
