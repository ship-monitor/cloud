package security

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	"github.com/ship-monitor/cloud/internal/v2/gen/api"
	"github.com/ship-monitor/cloud/pkg/problem"
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

var errUnauthorized = problem.New(
	"unauthorized",
	http.StatusUnauthorized,
	"User not authorized",
)

// HandleBearerToken implements [api.SecurityHandler].
func (h *Handler) HandleBearerToken(
	ctx context.Context,
	operationName api.OperationName,
	token api.BearerToken,
) (context.Context, error) {
	claims, err := h.tokenService.ParseToken(ctx, token.Token)
	if err != nil {
		return ctx, errUnauthorized.WithDetail(
			fmt.Sprintf("parse token: %v", err),
		)
	}

	return auth.WithActor(ctx, auth.Actor{
		UserID: claims.UserID,
		Roles:  token.Roles,
	}), nil
}
