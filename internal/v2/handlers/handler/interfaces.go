package handler

import (
	"context"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	devicedomain "github.com/ship-monitor/cloud/internal/v2/domain/device"
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

type UserService interface {
	Register(
		ctx context.Context,
		email, password, name string,
	) (user.User, error)
	Login(ctx context.Context, email, password string) (user.User, error)
}

type DeviceService interface {
	Get(ctx context.Context, ownerID, id uuid.UUID) (devicedomain.Device, error)
	ListByOwner(
		ctx context.Context,
		ownerID uuid.UUID,
		paging devicedomain.Paging,
	) ([]devicedomain.Device, error)
	Add(ctx context.Context, ownerID, id uuid.UUID, securityKey string) error
	Remove(ctx context.Context, id, userID uuid.UUID) error
	Rename(ctx context.Context, ownerID, id uuid.UUID, name string) error
}
