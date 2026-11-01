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

// Actor is the authenticated user on whose behalf the request is performed.
type Actor struct {
	UserID uuid.UUID

	// Roles comes from the token, and is empty unless the token carries them.
	Roles []string
}

type actorKey struct{}

// WithActor stores the actor in ctx. Called by the security handler
// after successful authentication.
func WithActor(ctx context.Context, a Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, a)
}

// ActorFromContext returns the actor from ctx, or ok=false
// when the request was not authenticated.
//
//nolint:nonamedreturns
func ActorFromContext(ctx context.Context) (a Actor, ok bool) {
	a, ok = ctx.Value(actorKey{}).(Actor)

	return a, ok
}
