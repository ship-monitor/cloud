// Package metrics implements business logic for device and sensor
// metrics: ingestion from the queue and owner-scoped queries for the API.
//
//nolint:varnamelen
package metrics

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/domain/metrics"
	"go.uber.org/zap"
)

// DeviceRepo reads and writes device metrics.
type DeviceRepo interface {
	InsertBatch(
		ctx context.Context,
		deviceID uuid.UUID,
		ms []metrics.Metric,
	) error
	List(
		ctx context.Context,
		deviceID uuid.UUID,
		query metrics.Query,
	) ([]metrics.Metric, error)
}

// SensorRepo reads and writes sensor metrics.
type SensorRepo interface {
	InsertBatch(
		ctx context.Context,
		deviceID uuid.UUID,
		sensorID string,
		ms []metrics.Metric,
	) error
	List(
		ctx context.Context,
		deviceID uuid.UUID,
		sensorID string,
		query metrics.Query,
	) ([]metrics.Metric, error)
}

// DeviceService authorizes queries on behalf of the given actor;
// devices of other users are reported as not found.
type DeviceService interface {
	Get(
		ctx context.Context,
		actor auth.Actor,
		id uuid.UUID,
	) (device.Device, error)
}

type Service struct {
	devices    DeviceService
	deviceRepo DeviceRepo
	sensorRepo SensorRepo
	logger     *zap.Logger
}

func New(
	devices DeviceService,
	deviceRepo DeviceRepo,
	sensorRepo SensorRepo,
	logger *zap.Logger,
) *Service {
	return &Service{
		devices:    devices,
		deviceRepo: deviceRepo,
		sensorRepo: sensorRepo,
		logger:     logger,
	}
}

// IngestDeviceMetrics stores a batch of device metrics received from a
// device. The device is already authenticated by the queue security
// handler, so no ownership check is needed here.
func (s *Service) IngestDeviceMetrics(
	ctx context.Context,
	deviceID uuid.UUID,
	ms []metrics.Metric,
) error {
	err := s.deviceRepo.InsertBatch(ctx, deviceID, ms)
	if err != nil {
		return fmt.Errorf("insert device metrics: %w", err)
	}

	s.logger.Debug(
		"Ingested device metrics",
		zap.String("deviceId", deviceID.String()),
		zap.Int("count", len(ms)),
	)

	return nil
}

// IngestSensorMetrics stores a batch of metrics for a sensor of the given
// device. The device is already authenticated by the queue security
// handler, so no ownership check is needed here.
func (s *Service) IngestSensorMetrics(
	ctx context.Context,
	deviceID uuid.UUID,
	sensorID string,
	ms []metrics.Metric,
) error {
	err := s.sensorRepo.InsertBatch(ctx, deviceID, sensorID, ms)
	if err != nil {
		return fmt.Errorf("insert sensor metrics: %w", err)
	}

	s.logger.Debug(
		"Ingested sensor metrics",
		zap.String("deviceId", deviceID.String()),
		zap.String("sensorId", sensorID),
		zap.Int("count", len(ms)),
	)

	return nil
}

// GetDeviceMetrics returns device metrics for the given actor; devices of
// other users are reported as not found.
func (s *Service) GetDeviceMetrics(
	ctx context.Context,
	actor auth.Actor,
	deviceID uuid.UUID,
	query metrics.Query,
) ([]metrics.Metric, error) {
	_, err := s.devices.Get(ctx, actor, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}

	ms, err := s.deviceRepo.List(ctx, deviceID, query)
	if err != nil {
		return nil, fmt.Errorf("list device metrics: %w", err)
	}

	return ms, nil
}

// GetSensorMetrics returns sensor metrics for the given actor; devices of
// other users are reported as not found.
func (s *Service) GetSensorMetrics(
	ctx context.Context,
	actor auth.Actor,
	deviceID uuid.UUID,
	sensorID string,
	query metrics.Query,
) ([]metrics.Metric, error) {
	_, err := s.devices.Get(ctx, actor, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}

	ms, err := s.sensorRepo.List(ctx, deviceID, sensorID, query)
	if err != nil {
		return nil, fmt.Errorf("list sensor metrics: %w", err)
	}

	return ms, nil
}
