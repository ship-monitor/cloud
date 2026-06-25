package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
)

type OrgDevicesRepo struct {
	db *bun.DB
}

func NewOrgDevices(db *bun.DB) *OrgDevicesRepo {
	return &OrgDevicesRepo{
		db: db,
	}
}

func (r *OrgDevicesRepo) ListDevices(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]domain.OrganizationDevice, error) {
	var devices []domain.OrganizationDevice

	err := r.db.NewSelect().
		Model(&devices).
		Where("organization_id = ?", organizationID).
		Order("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("select devices: %w", err)
	}

	return devices, nil
}

func (r *OrgDevicesRepo) GetDevice(
	ctx context.Context,
	deviceID uuid.UUID,
) (*domain.OrganizationDevice, error) {
	var device domain.OrganizationDevice

	err := r.db.NewSelect().
		Model(&device).
		Where("id = ?", deviceID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("select device: %w", err)
	}

	return &device, nil
}

func (r *OrgDevicesRepo) CreateDevice(
	ctx context.Context,
	device *domain.OrganizationDevice,
) error {
	_, err := r.db.NewInsert().Model(&device).Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert device: %w", err)
	}

	return nil
}

func (r *OrgDevicesRepo) DeleteDevice(ctx context.Context, deviceID uuid.UUID) error {
	_, err := r.db.NewDelete().
		Model(&domain.OrganizationDevice{ID: deviceID}).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}

	return nil
}

func (r *OrgDevicesRepo) SetName(ctx context.Context, deviceID uuid.UUID, name string) error {
	_, err := r.db.NewUpdate().
		Model(&domain.OrganizationDevice{ID: deviceID, Name: name}).
		Column("name").
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("update device: %w", err)
	}

	return nil
}

func (r *OrgDevicesRepo) DeviceExists(ctx context.Context, deviceID uuid.UUID) (bool, error) {
	exists, err := r.db.NewSelect().
		Model(&domain.OrganizationDevice{}).
		Where("id = ?", deviceID).
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("select device: %w", err)
	}

	return exists, nil
}
