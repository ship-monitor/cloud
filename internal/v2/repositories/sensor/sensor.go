package sensor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/sensor"
	"github.com/valkey-io/valkey-go"
)

type Repo struct {
	valkey valkey.Client
}

func New(
	valkey valkey.Client,
) *Repo {
	return &Repo{
		valkey: valkey,
	}
}

func (s *Repo) SetSensors(
	ctx context.Context,
	deviceID uuid.UUID,
	container *sensor.Container,
) error {
	data, err := json.Marshal(container)
	if err != nil {
		return fmt.Errorf("marshal sensors: %w", err)
	}

	err = s.valkey.Do(
		ctx,
		s.valkey.B().Set().Key(getKey(deviceID)).Value(string(data)).Build(),
	).Error()
	if err != nil {
		return fmt.Errorf("set value: %w", err)
	}

	return nil
}

func (s *Repo) GetSensors(
	ctx context.Context,
	deviceID uuid.UUID,
) (*sensor.Container, error) {
	data, err := s.valkey.Do(
		ctx,
		s.valkey.B().Get().Key(getKey(deviceID)).Build(),
	).AsBytes()
	if err != nil {
		return nil, fmt.Errorf("get value: %w", err)
	}

	var container sensor.Container

	err = json.Unmarshal(data, &container)
	if err != nil {
		return nil, fmt.Errorf("unmarshal sensors: %w", err)
	}

	return &container, nil
}

func getKey(deviceID uuid.UUID) string {
	return "device." + deviceID.String() + ".sensors"
}
