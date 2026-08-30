package device

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"charm.land/log/v2"
	"github.com/authzed/authzed-go/v1"
	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/domain"
	"github.com/ship-monitor/cloud/internal/handlers"
	"github.com/ship-monitor/cloud/internal/services"
)

type Service struct {
	devices        DeviceRepository
	states         StatesRepository
	logger         *log.Logger
	topicPublisher *services.TopicPublisher
	spicedb        *authzed.Client
}

var _ handlers.DevicesService = (*Service)(nil)

func NewService(
	states StatesRepository,
	devices DeviceRepository,
	logger *log.Logger,
	topicPublisher *services.TopicPublisher,
	spicedb *authzed.Client,
) *Service {
	return &Service{
		devices:        devices,
		states:         states,
		logger:         logger,
		topicPublisher: topicPublisher,
		spicedb:        spicedb,
	}
}

var (
	ErrForbidden = fmt.Errorf(
		"%w: this action forbidden",
		domain.ErrForbidden,
	)
	ErrInvalidHistoryLength = errors.New(
		"invalid history length specified",
	)
	errAccessRelationshipMissingSubject = errors.New(
		"device access relationship has no subject",
	)
)

func (s *Service) GetDevice(
	ctx context.Context,
	applicant *domain.Principal,
	deviceID domain.DeviceID,
) (*domain.Device, error) {
	if err := s.checkPermissions(
		ctx,
		deviceID,
		applicant.UserID,
		DevicePermissionView,
	); err != nil {
		return nil, fmt.Errorf("check permission: %w", err)
	}

	device, err := s.devices.GetDevice(ctx, deviceID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, fmt.Errorf("get device: %w", handlers.ErrDeviceNotFound)
	case err != nil:
		return nil, fmt.Errorf("get device: %w", err)
	default:
		return device, nil
	}
}

func (s *Service) GetDevices(
	ctx context.Context,
	applicant *domain.Principal,
) ([]domain.Device, error) {
	deviceIDs, err := s.lookupAccessibleDeviceIDs(ctx, applicant.UserID)
	if err != nil {
		return nil, fmt.Errorf("lookup accessible devices: %w", err)
	}

	devices, err := s.devices.GetDevicesByIDs(ctx, deviceIDs)
	if err != nil {
		return nil, fmt.Errorf("get devices by IDs: %w", err)
	}

	return devices, nil
}

func (s *Service) RenameDevice(
	ctx context.Context,
	applicant *domain.Principal,
	deviceID uuid.UUID,
	name string,
) error {
	if err := s.checkPermissions(
		ctx,
		deviceID,
		applicant.UserID,
		DevicePermissionRename,
	); err != nil {
		return fmt.Errorf("check permissions: %w", err)
	}

	err := s.devices.RenameDevice(ctx, deviceID, name)
	if err != nil {
		return fmt.Errorf("rename device: %w", err)
	}

	return nil
}
