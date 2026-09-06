package connection

import (
	"context"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/domain"
	"github.com/ship-monitor/cloud/workers/grpc"
)

type Service struct{}

var _ grpc.Service = (*Service)(nil)
var _ grpc.HubAuthService = (*Service)(nil)

func New() *Service {
	return &Service{}
}

// ConnectDevice implements [grpc.Service].
func (s *Service) ConnectDevice(
	ctx context.Context,
	req domain.ConnectDeviceRequest,
) error {
	panic("unimplemented")
}

// AuthenticateHub implements [grpc.HubAuthService].
func (s *Service) AuthenticateHub(ctx context.Context, hubID uuid.UUID, model string) (*grpc.AuthData, error) {
	panic("unimplemented")
}

// Refresh implements [grpc.HubAuthService].
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*grpc.AuthData, error) {
	panic("unimplemented")
}
