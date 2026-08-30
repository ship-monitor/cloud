package device

import (
	"context"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/domain"
)

// GetDeviceSettings will be implemented in next version.
func (s *Service) GetDeviceSettings(ctx context.Context, applicant *domain.Principal, deviceID uuid.UUID) (any, error)

// SetDeviceSettings will be implemented in next version.
func (s *Service) SetDeviceSettings(ctx context.Context, applicant *domain.Principal, deviceID uuid.UUID, settings any) error
