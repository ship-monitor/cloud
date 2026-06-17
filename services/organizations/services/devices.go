package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/data"
)

var ErrNotMember = fmt.Errorf("access denied: user is not member of organization")

type DevicesService struct {
	orgs *OrganizationsService
}

func NewDevices(orgs *OrganizationsService) *DevicesService {
	return &DevicesService{
		orgs: orgs,
	}
}

// ConnectDevice implements [handlers.DevicesService].
func (d *DevicesService) ConnectDevice(
	ctx context.Context,
	deviceID, organizationID, userID uuid.UUID,
) error {
	panic("unimplemented")
}

// DisconnectDevice implements [handlers.DevicesService].
func (d *DevicesService) DisconnectDevice(ctx context.Context, deviceID, userID uuid.UUID) error {
	dev, err := data.GetDevice(deviceID)
	if err != nil {
		return fmt.Errorf("get device: %w", err)
	}

	if isMember, err := d.orgs.IsMember(ctx, userID, dev.OrganizationID); err != nil {
		return fmt.Errorf("check is member: %w", err)
	} else if !isMember {
		return ErrNotMember
	}

	if err := data.DisconnectDevice(dev.ID); err != nil {
		return fmt.Errorf("disconnect device: %w", err)
	}

	return nil
}

// GetDevice implements [handlers.DevicesService].
func (d *DevicesService) GetDevice(
	ctx context.Context,
	deviceID, userID uuid.UUID,
) (*data.OrganizationDevice, error) {
	dev, err := data.GetDevice(deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}

	if isMember, err := d.orgs.IsMember(ctx, userID, dev.OrganizationID); err != nil {
		return nil, fmt.Errorf("check is member: %w", err)
	} else if !isMember {
		return nil, ErrNotMember
	}

	return dev, nil
}

// GetDevices implements [handlers.DevicesService].
func (d *DevicesService) GetDevices(
	ctx context.Context,
	organizationID, userID uuid.UUID,
) ([]data.OrganizationDevice, error) {
	panic("unimplemented")
}

// RenameDevice implements [handlers.DevicesService].
func (d *DevicesService) RenameDevice(
	ctx context.Context,
	deviceID, userID uuid.UUID,
	name string,
) error {
	if name == "" {
		return fmt.Errorf("empty device name")
	}

	dev, err := data.GetDevice(deviceID)
	if err != nil {
		return fmt.Errorf("get device: %w", err)
	}

	if isMember, err := d.orgs.IsMember(ctx, userID, dev.OrganizationID); err != nil {
		return fmt.Errorf("is member: %w", err)
	} else if !isMember {
		return ErrNotMember
	}

	if err := dev.SetName(ctx, name); err != nil {
		return fmt.Errorf("set device name: %w", err)
	}

	return nil
}

// SendCommand implements [handlers.DevicesService].
func (d *DevicesService) SendCommand(
	ctx context.Context,
	deviceID, userID, command string,
	args map[string]any,
) error {
	panic("unimplemented")
}
