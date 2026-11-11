package sensor

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/domain/sensor"
)

type Repo interface {
	SetSensors(
		ctx context.Context,
		deviceID uuid.UUID,
		container *sensor.Container,
	) error
	GetSensors(
		ctx context.Context,
		deviceID uuid.UUID,
	) (*sensor.Container, error)
}

type DeviceService interface {
	Get(
		ctx context.Context,
		actor auth.Actor,
		id uuid.UUID,
	) (device.Device, error)
}

type Service struct {
	repo    Repo
	devices DeviceService
}

func New(
	repo Repo,
	devices DeviceService,
) *Service {
	return &Service{
		repo:    repo,
		devices: devices,
	}
}

func (s *Service) DeclareSensors(
	ctx context.Context,
	deviceID uuid.UUID,
	sensors []sensor.Sensor,
) error {
	container := sensor.NewContainer(sensors)

	err := s.repo.SetSensors(ctx, deviceID, container)
	if err != nil {
		return fmt.Errorf("set sensors: %w", err)
	}

	return nil
}

func (s *Service) GetSensors(
	ctx context.Context,
	actor auth.Actor,
	deviceID uuid.UUID,
) ([]sensor.Sensor, error) {
	_, err := s.devices.Get(ctx, actor, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}

	c, err := s.repo.GetSensors(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get sensors: %w", err)
	}

	return c.Sensors, nil
}
