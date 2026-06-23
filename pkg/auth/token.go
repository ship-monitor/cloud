package auth

import (
	"errors"
	"fmt"

	"charm.land/log/v2"
	"github.com/golang-jwt/jwt/v5"
)

// ParseToken returns error if token is invalid or expired.
func (m *Middleware) ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, m.keyFunc)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			log.Error("JWT expired", "error", err)

			return nil, fmt.Errorf("parse token: %w", err)
		}

		log.Error("Failed parse JWT", "error", err)

		return nil, fmt.Errorf("parse token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("token invalid")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid claims type")
	}

	return claims, nil
}
