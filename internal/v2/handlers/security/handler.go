package security

import (
	"context"
	"fmt"

	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	"github.com/ship-monitor/cloud/internal/v2/gen/api"
)

var _ api.SecurityHandler = (*Handler)(nil)

type TokenService interface {
	ParseToken(ctx context.Context, token string) (*auth.Claims, error)
}

type Handler struct {
	tokenService TokenService
}

func New(
	tokenService TokenService,
) *Handler {
	return &Handler{
		tokenService: tokenService,
	}
}

// HandleBearerToken implements [api.SecurityHandler].
func (h *Handler) HandleBearerToken(
	ctx context.Context,
	operationName api.OperationName,
	t api.BearerToken,
) (context.Context, error) {
	claims, err := h.tokenService.ParseToken(ctx, t.Token)
	if err != nil {
		return ctx, fmt.Errorf("parse token: %w", err)
	}

	p := auth.NewClaimsPrincipal(claims)

	ctx = auth.WithPrincipal(ctx, p)

	return ctx, nil
}
