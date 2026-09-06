package grpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/domain"
)

type Service interface {
	ConnectDevice(ctx context.Context, req domain.ConnectDeviceRequest) error
}

type AuthData struct {
	AccessToken  string
	RefreshToken string
}

type HubAuthService interface {
	AuthenticateHub(
		ctx context.Context,
		hubID uuid.UUID,
		model string,
	) (*AuthData, error)
	Refresh(ctx context.Context, refreshToken string) (*AuthData, error)
}

// ConnectDevice implements [Service].
func (h *HubAuthService) ConnectDevice(ctx context.Context, req domain.ConnectDeviceRequest) error {
	panic("unimplemented")
}

// ConnectDevice implements [Service].
func (h *HubAuthService) ConnectDevice(ctx context.Context, req domain.ConnectDeviceRequest) error {
	panic("unimplemented")
}
