package auth

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	*jwt.RegisteredClaims

	UserID uuid.UUID `json:"userId"`
	Email  string    `json:"email,omitempty"`
}
