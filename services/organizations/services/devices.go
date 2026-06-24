package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/commands"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/data"
)

var (
	ErrNotMember       = errors.New("access denied: user is not member of organization")
	ErrEmptyDeviceName = errors.New("empty device name")
)

type DevicesService struct {
	orgs *services.OrganizationsService
}

func NewDevices(orgs *services.OrganizationsService) *DevicesService {
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
	dev, err := data.GetDevice(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("get device: %w", err)
	}

	if isMember, err := d.orgs.IsMember(ctx, userID, dev.OrganizationID); err != nil {
		return fmt.Errorf("check is member: %w", err)
	} else if !isMember {
		return ErrNotMember
	}

	if err := data.DisconnectDevice(ctx, dev.ID); err != nil {
		return fmt.Errorf("disconnect device: %w", err)
	}

	return nil
}

// GetDevice implements [handlers.DevicesService].
func (d *DevicesService) GetDevice(
	ctx context.Context,
	deviceID, userID uuid.UUID,
) (*data.OrganizationDevice, error) {
	dev, err := data.GetDevice(ctx, deviceID)
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
		return ErrEmptyDeviceName
	}

	dev, err := data.GetDevice(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("get device: %w", err)
	}

	isMember, err := d.orgs.IsMember(ctx, userID, dev.OrganizationID)
	if err != nil {
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
	deviceID, userID uuid.UUID, command string,
	args map[string]any,
) (*commands.CommandResponse, error) {
	dev, err := data.GetDevice(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}

	if isMember, err := d.orgs.IsMember(ctx, userID, dev.OrganizationID); err != nil {
		return nil, fmt.Errorf("is member: %w", err)
	} else if !isMember {
		return nil, ErrNotMember
	}

	cmd := commands.NewCommand(deviceID.String(), command, args)
	result := commands.SendCommand(ctx, cmd)

	return &result, nil
}
