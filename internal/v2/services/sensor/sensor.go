package sensor

import (
	"context"
	"fmt"

	"github.com/google/uuid"
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

type Service struct {
	repo Repo
}

func New(repo Repo) *Service {
	return &Service{
		repo: repo,
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
	deviceID uuid.UUID,
) ([]sensor.Sensor, error) {
	c, err := s.repo.GetSensors(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get sensors: %w", err)
	}

	return c.Sensors, nil
}
