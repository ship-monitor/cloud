package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Session struct {
	ID     string    `json:"-"`
	UserID uuid.UUID `json:"userId"`
	Email  string    `json:"email"`
	c      *gin.Context
}
