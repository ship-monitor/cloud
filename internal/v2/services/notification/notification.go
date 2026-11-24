package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	domainnotification "github.com/ship-monitor/cloud/internal/v2/domain/notification"
	"github.com/ship-monitor/cloud/pkg/connector/asyncapi"
)

type Repo interface {
	InsertNotification(
		ctx context.Context,
		n domainnotification.Notification,
	) error
}

// Service handles notification business logic.
type Service struct {
	repo Repo
}

// New creates a new notification service.
func New(repo Repo) *Service {
	return &Service{repo: repo}
}

var (
	errInvalidSource = errors.New("invalid source type of notification")
	errInvalidDevice = fmt.Errorf(
		"%w: device try to send not own notification",
		errInvalidSource,
	)
)

// IngestNotification processes an incoming notification message and stores it.
//
//nolint:wrapcheck
func (s *Service) IngestNotification(
	ctx context.Context,
	deviceID uuid.UUID,
	msg asyncapi.NotificationSchema,
) error {
	if msg.Source.Type == asyncapi.NotificationSourceSchemaDevice {
		return s.IngestDeviceNotification(ctx, deviceID, msg)
	}

	if msg.Source.Type == asyncapi.NotificationSourceSchemaSensor {
		return s.IngestSensorNotification(ctx, deviceID, msg)
	}

	return fmt.Errorf(
		"%w: source %q",
		errInvalidSource,
		msg.Source.Type,
	)
}

//nolint:varnamelen
func (s *Service) IngestSensorNotification(
	ctx context.Context,
	deviceID uuid.UUID,
	msg asyncapi.NotificationSchema,
) error {
	n := domainnotification.Notification{
		ID:         uuid.New(),
		DeviceID:   deviceID,
		SensorID:   &msg.Source.Id,
		Type:       msg.Type,
		Severity:   string(msg.Severity),
		OccurredAt: msg.OccurredAt,
		ReportedAt: msg.ReportedAt,
		ReceivedAt: time.Now().UTC(),
		Data:       msg.Data.AdditionalProperties,
	}

	err := s.repo.InsertNotification(ctx, n)
	if err != nil {
		return fmt.Errorf("create notification: %w", err)
	}

	return nil
}

//nolint:varnamelen
func (s *Service) IngestDeviceNotification(
	ctx context.Context,
	deviceID uuid.UUID,
	msg asyncapi.NotificationSchema,
) error {
	if msg.Source.Type == asyncapi.NotificationSourceSchemaDevice &&
		deviceID != msg.Source.Id {
		return fmt.Errorf(
			"%w: device id = %q, source id = %q ",
			errInvalidDevice,
			deviceID.String(),
			msg.Source.Id,
		)
	}
	// Convert asyncapi.NotificationMessage to domain.Notification
	n := domainnotification.Notification{
		ID:         uuid.New(),
		DeviceID:   deviceID,
		SensorID:   nil,
		Type:       msg.Type,
		Severity:   string(msg.Severity),
		OccurredAt: msg.OccurredAt,
		ReportedAt: msg.ReportedAt,
		ReceivedAt: time.Now().UTC(),
		Data:       msg.Data.AdditionalProperties,
	}

	err := s.repo.InsertNotification(ctx, n)
	if err != nil {
		return fmt.Errorf("create notification: %w", err)
	}

	return nil
}
