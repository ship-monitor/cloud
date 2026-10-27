package handler

import (
	"context"

	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	"github.com/ship-monitor/cloud/internal/v2/domain/user"
)

type TokenService interface {
	ParseToken(ctx context.Context, token string) (*auth.Claims, error)
	CreateTokens(ctx context.Context, user *user.User) (string, string, error)
	RefreshTokens(
		ctx context.Context,
		refreshToken string,
	) (string, string, error)
}
