package data

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/names"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/dto"
)

const DefaultDeviceName = "Unknown Device"

func ConnectDevice(ctx context.Context, id, organizationID uuid.UUID, name string) error {
	if device, _ := GetDeviceDTO(ctx, id); device != nil {
		if device.OrganizationID != organizationID {
			return fmt.Errorf("device %q is already connected to another organization", id)
		}

		return nil
	}

	if name == "" {
		name = names.Gen()
	}

	_, err := createDevice(ctx, id, organizationID, name)
	if err != nil {
		return fmt.Errorf("create device: %w", err)
	} else {
		return nil
	}
}

func createDevice(
	ctx context.Context,
	id, orgID uuid.UUID,
	name string,
) (*domain.OrganizationDevice, error) {
	device := domain.OrganizationDevice{
		ID:             id,
		OrganizationID: orgID,
		CreatedAt:      time.Now(),
		Name:           name,
	}

	_, err := db.DB.NewInsert().Model(&device).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("insert device: %w", err)
	}

	return &device, nil
}

func SetName(ctx context.Context, device *domain.OrganizationDevice, name string) error {
	device.Name = name

	_, err := db.DB.NewUpdate().Model(device).Column("name").WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("update device: %w", err)
	}

	return nil
}

func toDTO(device *domain.OrganizationDevice) *dto.DeviceResponse {
	return &dto.DeviceResponse{
		ID:             device.ID,
		OrganizationID: device.OrganizationID,
		CreatedAt:      device.CreatedAt,
		Name:           device.Name,
	}
}

func GetDevice(ctx context.Context, id uuid.UUID) (*domain.OrganizationDevice, error) {
	var device domain.OrganizationDevice

	err := db.DB.NewSelect().
		Model(&device).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("select device: %w", err)
	}

	return &device, nil
}

func GetDeviceDTO(ctx context.Context, id uuid.UUID) (*dto.DeviceResponse, error) {
	device, err := GetDevice(ctx, id)
	if err != nil {
		return nil, err
	}

	return toDTO(device), nil
}

func ListDevices(ctx context.Context, orgID uuid.UUID) ([]domain.OrganizationDevice, error) {
	var devices []domain.OrganizationDevice

	err := db.DB.NewSelect().
		Model(&devices).
		Where("organization_id = ?", orgID).
		Order("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("select devices: %w", err)
	}

	return devices, nil
}

func DisconnectDevice(ctx context.Context, id uuid.UUID) error {
	if device, _ := GetDeviceDTO(ctx, id); device != nil {
		return deleteDevice(ctx, id)
	}

	return nil
}

func deleteDevice(ctx context.Context, id uuid.UUID) error {
	_, err := db.DB.NewDelete().
		Model((*domain.OrganizationDevice)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}

	return nil
}
