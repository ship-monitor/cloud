package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/names"
)

var (
	ErrNotMember = errors.New(
		"access denied: user is not member of organization",
	)
	ErrEmptyDeviceName  = errors.New("empty device name")
	ErrAlreadyConnected = errors.New("device already connected to organization")
)

type OrgDevicesRepo interface {
	ListDevices(
		ctx context.Context,
		organizationID uuid.UUID,
	) ([]domain.OrganizationDevice, error)
	GetDevice(
		ctx context.Context,
		deviceID uuid.UUID,
	) (*domain.OrganizationDevice, error)
	CreateDevice(ctx context.Context, device *domain.OrganizationDevice) error
	DeleteDevice(ctx context.Context, deviceID uuid.UUID) error
	SetName(ctx context.Context, deviceID uuid.UUID, name string) error
	DeviceExists(ctx context.Context, deviceID uuid.UUID) (bool, error)
}

type OrgDevicesService struct {
	orgs *OrganizationsService
	repo OrgDevicesRepo
}

func NewOrgDevices(
	devRepo OrgDevicesRepo,
	orgs *OrganizationsService,
) *OrgDevicesService {
	return &OrgDevicesService{
		orgs: orgs,
		repo: devRepo,
	}
}

// ConnectDevice implements [handlers.DevicesService].
func (d *OrgDevicesService) ConnectDevice(
	ctx context.Context,
	deviceID, organizationID, userID uuid.UUID,
	name string,
) error {
	if isMember, err := d.orgs.IsMember(
		ctx,
		userID,
		organizationID,
	); err != nil {
		return fmt.Errorf("check is member: %w", err)
	} else if !isMember {
		return ErrNotMember
	}

	if exists, err := d.repo.DeviceExists(ctx, deviceID); err != nil {
		return fmt.Errorf("check already connected: %w", err)
	} else if exists {
		return ErrAlreadyConnected
	}

	if name == "" {
		name = names.Gen()
	}

	err := d.repo.CreateDevice(ctx, &domain.OrganizationDevice{
		ID:             deviceID,
		Name:           name,
		OrganizationID: organizationID,
		CreatedAt:      time.Now(),
	})
	if err != nil {
		return fmt.Errorf("create device: %w", err)
	} else {
		return nil
	}
}

// DisconnectDevice implements [handlers.DevicesService].
func (d *OrgDevicesService) DisconnectDevice(
	ctx context.Context,
	deviceID, userID uuid.UUID,
) error {
	dev, err := d.repo.GetDevice(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("get device: %w", err)
	}

	if isMember, err := d.orgs.IsMember(
		ctx,
		userID,
		dev.OrganizationID,
	); err != nil {
		return fmt.Errorf("check is member: %w", err)
	} else if !isMember {
		return ErrNotMember
	}

	if err := d.repo.DeleteDevice(ctx, deviceID); err != nil {
		return fmt.Errorf("disconnect device: %w", err)
	}

	return nil
}

// GetDevice implements [handlers.DevicesService].
func (d *OrgDevicesService) GetDevice(
	ctx context.Context,
	deviceID, userID uuid.UUID,
) (*domain.OrganizationDevice, error) {
	dev, err := d.repo.GetDevice(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}

	if isMember, err := d.orgs.IsMember(
		ctx,
		userID,
		dev.OrganizationID,
	); err != nil {
		return nil, fmt.Errorf("check is member: %w", err)
	} else if !isMember {
		return nil, ErrNotMember
	}

	return dev, nil
}

// GetDevices implements [handlers.DevicesService].
func (d *OrgDevicesService) GetDevices(
	ctx context.Context,
	organizationID, userID uuid.UUID,
) ([]domain.OrganizationDevice, error) {
	if isMember, err := d.orgs.IsMember(
		ctx,
		userID,
		organizationID,
	); err != nil {
		return nil, fmt.Errorf("check is member: %w", err)
	} else if !isMember {
		return nil, ErrNotMember
	}

	devs, err := d.repo.ListDevices(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}

	return devs, nil
}

// RenameDevice implements [handlers.DevicesService].
func (d *OrgDevicesService) RenameDevice(
	ctx context.Context,
	deviceID, userID uuid.UUID,
	name string,
) error {
	if name == "" {
		return ErrEmptyDeviceName
	}

	dev, err := d.repo.GetDevice(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("get device: %w", err)
	}

	isMember, err := d.orgs.IsMember(ctx, userID, dev.OrganizationID)
	if err != nil {
		return fmt.Errorf("is member: %w", err)
	} else if !isMember {
		return ErrNotMember
	}

	if err := d.repo.SetName(ctx, deviceID, name); err != nil {
		return fmt.Errorf("set device name: %w", err)
	}

	return nil
}

func (d *OrgDevicesService) UserCanGetState(
	ctx context.Context,
	userID, deviceID uuid.UUID,
) (bool, error) {
	dev, err := d.repo.GetDevice(ctx, deviceID)
	if err != nil {
		return false, fmt.Errorf("get device: %w", err)
	}

	if isMember, err := d.orgs.IsMember(
		ctx,
		userID,
		dev.OrganizationID,
	); err != nil {
		return false, fmt.Errorf("is member: %w", err)
	} else if !isMember {
		return false, ErrNotMember
	}

	return true, nil
}

// UserCanSendCommand returns whether the user can send a command to the device.
// Currently, all members can send commands, same as
// [OrgDevicesService.UserCanGetState].
func (d *OrgDevicesService) UserCanSendCommand(
	ctx context.Context,
	userID, deviceID uuid.UUID,
) (bool, error) {
	dev, err := d.repo.GetDevice(ctx, deviceID)
	if err != nil {
		return false, fmt.Errorf("get device: %w", err)
	}

	if isMember, err := d.orgs.IsMember(
		ctx,
		userID,
		dev.OrganizationID,
	); err != nil {
		return false, fmt.Errorf("is member: %w", err)
	} else if !isMember {
		return false, ErrNotMember
	}

	return true, nil
}
