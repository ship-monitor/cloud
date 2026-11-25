package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	domainnotification "github.com/ship-monitor/cloud/internal/v2/domain/notification"
	"github.com/ship-monitor/cloud/internal/v2/domain/pagination"
	"github.com/ship-monitor/cloud/pkg/connector/asyncapi"
)

type Repo interface {
	InsertNotification(
		ctx context.Context,
		n domainnotification.Notification,
	) error
	ListByDevicePageForUser(
		ctx context.Context,
		deviceID, userID uuid.UUID,
		req pagination.Request[domainnotification.Cursor],
	) (
		pagination.Page[domainnotification.Notification, domainnotification.Cursor],
		error,
	)
}

// DeviceService authorizes access to devices on behalf of the actor.
type DeviceService interface {
	Get(
		ctx context.Context,
		actor auth.Actor,
		id uuid.UUID,
	) (device.Device, error)
}

// Service handles notification business logic.
type Service struct {
	repo    Repo
	devices DeviceService
}

// New creates a new notification service.
func New(repo Repo, devices DeviceService) *Service {
	return &Service{repo: repo, devices: devices}
}

// ListByDevicePage returns notifications and read state for the actor's device.
// Devices owned by other users are reported as not found by the device service.
func (s *Service) ListByDevicePage(
	ctx context.Context,
	actor auth.Actor,
	deviceID uuid.UUID,
	req pagination.Request[domainnotification.Cursor],
) (
	pagination.Page[domainnotification.Notification, domainnotification.Cursor],
	error,
) {
	var page pagination.Page[domainnotification.Notification, domainnotification.Cursor]

	_, err := s.devices.Get(ctx, actor, deviceID)
	if err != nil {
		return page, fmt.Errorf("get device: %w", err)
	}

	page, err = s.repo.ListByDevicePageForUser(ctx, deviceID, actor.UserID, req)
	if err != nil {
		return page, fmt.Errorf("list device notifications: %w", err)
	}

	return page, nil
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
		IsRead:     false,
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
		IsRead:     false,
		Data:       msg.Data.AdditionalProperties,
	}

	err := s.repo.InsertNotification(ctx, n)
	if err != nil {
		return fmt.Errorf("create notification: %w", err)
	}

	return nil
}
