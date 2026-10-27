package auth

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	jwt.RegisteredClaims

	UserID uuid.UUID `json:"userId"`
}

func NewClaims(userID uuid.UUID, ttl time.Duration) *Claims {
	var c Claims

	c.ID = uuid.NewString()
	c.IssuedAt = jwt.NewNumericDate(time.Now())
	c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(ttl))

	c.UserID = userID

	return &c
}

func NewRefreshClaims(userID uuid.UUID, ttl time.Duration) *Claims {
	var c Claims

	c.ID = uuid.NewString()
	c.IssuedAt = jwt.NewNumericDate(time.Now())
	c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(ttl))

	c.UserID = userID

	return &c
}

type principalKey struct{}

type ClaimsPrincipal struct {
	claims Claims
}

func NewClaimsPrincipal(claims *Claims) *ClaimsPrincipal {
	return &ClaimsPrincipal{claims: *claims}
}

func (p *ClaimsPrincipal) UserID() uuid.UUID {
	return p.claims.UserID
}

func WithPrincipal(ctx context.Context, p *ClaimsPrincipal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

func GetPrincipal(ctx context.Context) *ClaimsPrincipal {
	val := ctx.Value(principalKey{})

	p, ok := val.(*ClaimsPrincipal)
	if !ok {
		return nil
	}

	return p
}
