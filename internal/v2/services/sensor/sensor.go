package sensor

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/domain/sensor"
	"go.uber.org/zap"
)

// errDuplicateSensor is returned when a declaration declares the same
// sensor ID more than once.
var errDuplicateSensor = errors.New("duplicate sensor declaration")

type Repo interface {
	InsertSensor(
		ctx context.Context,
		s *sensor.Sensor,
	) error
	GetSensors(
		ctx context.Context,
		deviceID uuid.UUID,
	) ([]sensor.Sensor, error)
	GetSensor(
		ctx context.Context,
		sensorID string,
		deviceID uuid.UUID,
	) (*sensor.Sensor, error)
	SensorExists(
		ctx context.Context,
		sensorID string,
		deviceID uuid.UUID,
	) (bool, error)
	DeleteSensor(
		ctx context.Context,
		sensorID string,
		deviceID uuid.UUID,
	) error
	RenameSensor(
		ctx context.Context,
		sensorID string,
		deviceID uuid.UUID,
		displayName string,
	) error
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
	logger  *zap.Logger
}

func New(
	repo Repo,
	devices DeviceService,
	logger *zap.Logger,
) *Service {
	return &Service{
		repo:    repo,
		devices: devices,
		logger:  logger,
	}
}

// DeclareSensors syncs the device's stored sensor list with the declared
// one: sensors missing from the database are inserted, stored sensors
// missing from the declaration are deleted, and unchanged sensors are left
// as is.
// It returns ErrDuplicateSensor if the declaration contains the same ID
// more than once, and ErrSensorConflict if it changes an already declared
// sensor's name or display name.
func (s *Service) DeclareSensors(
	ctx context.Context,
	deviceID uuid.UUID,
	sensors []sensor.SensorDecl,
) error {
	stored, err := s.repo.GetSensors(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("get sensors: %w", err)
	}

	storedByID, declaredByID, err := checkCollisions(stored, sensors)
	if err != nil {
		return fmt.Errorf("check collisions: %w", err)
	}

	if err := s.insertDeclared(ctx, deviceID, sensors, storedByID); err != nil {
		return fmt.Errorf("insert declared sensors: %w", err)
	}

	if err := s.deleteUndeclared(
		ctx,
		deviceID,
		stored,
		declaredByID,
	); err != nil {
		return fmt.Errorf("delete undeclared sensors: %w", err)
	}

	return nil
}

// checkCollisions indexes both sensor lists by ID and rejects declarations
// that contain the same ID more than once or that redefine an already
// stored sensor. It returns the indexes for the insert and delete steps.
func checkCollisions(
	stored []sensor.Sensor,
	sensors []sensor.SensorDecl,
) (map[string]sensor.Sensor, map[string]sensor.SensorDecl, error) {
	storedByID := make(map[string]sensor.Sensor, len(stored))
	for _, item := range stored {
		storedByID[item.ID] = item
	}

	declaredByID := make(map[string]sensor.SensorDecl, len(sensors))
	for _, item := range sensors {
		if _, dup := declaredByID[item.ID]; dup {
			return nil, nil, fmt.Errorf(
				"%w: sensor %q declared more than once",
				errDuplicateSensor,
				item.ID,
			)
		}

		declaredByID[item.ID] = item
	}

	return storedByID, declaredByID, nil
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

	sensors, err := s.repo.GetSensors(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get sensors: %w", err)
	}

	return sensors, nil
}

func (s *Service) GetSensor(
	ctx context.Context,
	actor auth.Actor,
	sensorID string,
	deviceID uuid.UUID,
) (*sensor.Sensor, error) {
	_, err := s.devices.Get(ctx, actor, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}

	sensor, err := s.repo.GetSensor(ctx, sensorID, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get sensor: %w", err)
	}

	return sensor, nil
}

func (s *Service) Rename(
	ctx context.Context,
	actor auth.Actor,
	id string,
	deviceID uuid.UUID,
	displayName string,
) error {
	_, err := s.devices.Get(ctx, actor, deviceID)
	if err != nil {
		return fmt.Errorf("get device: %w", err)
	}

	err = s.repo.RenameSensor(ctx, id, deviceID, displayName)
	if err != nil {
		return fmt.Errorf("rename sensor: %w", err)
	}

	return nil
}

// insertDeclared stores declared sensors that are not present in storedByID.
func (s *Service) insertDeclared(
	ctx context.Context,
	deviceID uuid.UUID,
	sensors []sensor.SensorDecl,
	storedByID map[string]sensor.Sensor,
) error {
	for _, item := range sensors {
		if _, found := storedByID[item.ID]; found {
			continue
		}

		newSensor := sensor.Sensor{
			ID:          item.ID,
			Name:        item.Name,
			DisplayName: item.Name,
			DeviceID:    deviceID,
		}
		if err := s.repo.InsertSensor(ctx, &newSensor); err != nil {
			return fmt.Errorf("insert sensor %q: %w", item.ID, err)
		}

		s.logger.Debug(
			"New sensor added", zap.Any("sensor", item),
		)
	}

	return nil
}

// deleteUndeclared removes stored sensors that are missing from declaredByID.
func (s *Service) deleteUndeclared(
	ctx context.Context,
	deviceID uuid.UUID,
	stored []sensor.Sensor,
	declaredByID map[string]sensor.SensorDecl,
) error {
	for _, item := range stored {
		if _, declared := declaredByID[item.ID]; declared {
			continue
		}

		if err := s.repo.DeleteSensor(ctx, item.ID, deviceID); err != nil {
			return fmt.Errorf("delete sensor %q: %w", item.ID, err)
		}

		s.logger.Debug(
			"Sensor removed", zap.Any("sensor", item),
		)
	}

	return nil
}
