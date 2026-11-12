// Package subscriber subscribes to device events from the broker and
// forwards them to handlers.
//
//nolint:varnamelen
package subscriber

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/domain/metrics"
	"github.com/ship-monitor/cloud/internal/v2/domain/sensor"
	"github.com/ship-monitor/cloud/pkg/connector/asyncapi"
	"go.uber.org/zap"
)

type SecurityHandler interface {
	HandleMessage(
		ctx context.Context,
		header asyncapi.IdentificationHeadersSchema,
	) (context.Context, error)
}

type Handler interface {
	HandleHeartbeat(
		ctx context.Context,
		msg asyncapi.HeartbeatMessage,
	) error
	HandleConnectResponse(
		ctx context.Context,
		response *asyncapi.ConnectResponseMessage,
	) error
}

type SensorService interface {
	DeclareSensors(
		ctx context.Context,
		deviceID uuid.UUID,
		sensors []sensor.SensorDecl,
	) error
}

type MetricsService interface {
	IngestDeviceMetrics(
		ctx context.Context,
		deviceID uuid.UUID,
		ms []metrics.Metric,
	) error
	IngestSensorMetrics(
		ctx context.Context,
		deviceID uuid.UUID,
		sensorID string,
		ms []metrics.Metric,
	) error
}

type S struct {
	controller *asyncapi.AppController
	logger     *zap.Logger
	handler    Handler
	security   SecurityHandler
	sensor     SensorService
	metrics    MetricsService
}

func New(
	controller *asyncapi.AppController,
	logger *zap.Logger,
	handler Handler,
	security SecurityHandler,
	sensor SensorService,
	metrics MetricsService,
) *S {
	return &S{
		controller: controller,
		logger:     logger.Named("Subscriber"),
		handler:    handler,
		security:   security,
		sensor:     sensor,
		metrics:    metrics,
	}
}

func (s *S) Start(ctx context.Context) error {
	err := errors.Join(
		s.controller.SubscribeToOnDeviceHeartbeatOperation(
			ctx,
			asyncapi.DevicesAliveChannelParameters{DeviceId: "*"},
			s.HandleHeartbeat,
		),
		s.controller.SubscribeToOnConnectResponseOperation(
			ctx,
			asyncapi.ConnectResponsesChannelParameters{DeviceId: "*"},
			s.HandleConnectResponse,
		),
		s.controller.SubscribeToMetricsOperationForMetrics(
			ctx,
			asyncapi.MetricsChannelParameters{DeviceId: "*"},
			s.HandleMetrics,
		),
		s.controller.SubscribeToMetricsOperationForSensorMetrics(
			ctx,
			asyncapi.MetricsChannelParameters{DeviceId: "*"},
			s.HandleSensorMetrics,
		),
		s.controller.SubscribeToSensorDeclarationOperation(
			ctx,
			asyncapi.SensorDeclarationsChannelParameters{DeviceId: "*"},
			s.HandleSensorDeclaration,
		),
	)
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	return nil
}

func (s *S) Stop(ctx context.Context) error {
	s.controller.UnsubscribeFromAllChannels(ctx)

	return nil
}

func (s *S) HandleHeartbeat(
	ctx context.Context,
	message asyncapi.HeartbeatMessage,
) error {
	s.logger.Info("Handle heartbeat")

	ctx, err := s.security.HandleMessage(ctx, message.Headers)
	if err != nil {
		return fmt.Errorf("handle security: %w", err)
	}

	err = s.handler.HandleHeartbeat(ctx, message)
	if err != nil {
		return fmt.Errorf("handle heartbeat: %w", err)
	}

	return nil
}

func (s *S) HandleConnectResponse(
	ctx context.Context,
	message asyncapi.ConnectResponseMessage,
) error {
	s.logger.Debug("Handling connect response")

	ctx, err := s.security.HandleMessage(ctx, message.Headers)
	if err != nil {
		s.logger.Error("Handle security", zap.Error(err))

		return fmt.Errorf("handle security: %w", err)
	}

	err = s.handler.HandleConnectResponse(ctx, &message)
	if err != nil {
		s.logger.Error("Handle connect response", zap.Error(err))

		return fmt.Errorf("handle connect response: %w", err)
	}

	s.logger.Info("Handled connect response")

	return nil
}

func (s *S) HandleMetrics(
	ctx context.Context,
	msg asyncapi.MetricsMessage,
) error {
	s.logger.Debug("Handling device metrics")

	ctx, err := s.security.HandleMessage(ctx, msg.Headers)
	if err != nil {
		return fmt.Errorf("handle security: %w", err)
	}

	deviceID, ok := device.DeviceIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("deviceID: %w", device.ErrNoIDInContext)
	}

	ms := toDomainMetrics(msg.Payload.Metrics)

	err = s.metrics.IngestDeviceMetrics(ctx, deviceID, ms)
	if err != nil {
		return fmt.Errorf("service: ingest device metrics: %w", err)
	}

	return nil
}

func (s *S) HandleSensorMetrics(
	ctx context.Context,
	msg asyncapi.SensorMetricsMessage,
) error {
	s.logger.Debug("Handling sensor metrics")

	ctx, err := s.security.HandleMessage(ctx, msg.Headers)
	if err != nil {
		return fmt.Errorf("handle security: %w", err)
	}

	deviceID, ok := device.DeviceIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("deviceID: %w", device.ErrNoIDInContext)
	}

	ms := toDomainMetrics(msg.Payload.Metrics)

	err = s.metrics.IngestSensorMetrics(
		ctx,
		deviceID,
		string(msg.Payload.SensorID),
		ms,
	)
	if err != nil {
		return fmt.Errorf("service: ingest sensor metrics: %w", err)
	}

	return nil
}

// toDomainMetrics converts generated AsyncAPI metric schemas to domain
// metrics.
func toDomainMetrics(
	in []asyncapi.MetricSchema,
) []metrics.Metric {
	ms := make([]metrics.Metric, 0, len(in))
	for _, m := range in {
		ms = append(ms, metrics.Metric{
			Name:      m.Name,
			Type:      metrics.Type(m.Type),
			Unit:      m.Unit,
			Value:     m.Value,
			Timestamp: m.Timestamp,
		})
	}

	return ms
}

func (s *S) HandleSensorDeclaration(
	ctx context.Context,
	msg asyncapi.SensorDeclarationMessage,
) error {
	ctx, err := s.security.HandleMessage(ctx, msg.Headers)
	if err != nil {
		return fmt.Errorf("handle security: %w", err)
	}

	deviceID, ok := device.DeviceIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("deviceID: %w", device.ErrNoIDInContext)
	}

	sensors := make([]sensor.SensorDecl, 0, len(msg.Payload.Sensors))
	for _, s := range msg.Payload.Sensors {
		sensors = append(
			sensors,
			sensor.SensorDecl{ID: string(s.Id), Name: s.Name},
		)
	}

	err = s.sensor.DeclareSensors(ctx, deviceID, sensors)
	if err != nil {
		return fmt.Errorf("service: declare sensors: %w", err)
	}

	return nil
}
