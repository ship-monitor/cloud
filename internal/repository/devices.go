package repository

import (
	"context"
	"database/sql"
	"fmt"

	"charm.land/log/v2"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/names"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/paging"
)

var _ pkg.MigrationRepo = (*DeviceRepo)(nil)

type DeviceRepo struct {
	db     *bun.DB
	logger *log.Logger
}

func NewDevices(db *sql.DB, logger *log.Logger) *DeviceRepo {
	return &DeviceRepo{
		db:     newBunDB(db),
		logger: logger,
	}
}

// Migrate implements [pkg.MigrationRepo].
func (d *DeviceRepo) Migrate(ctx context.Context) error {
	_, err := d.db.NewCreateTable().
		Model((*domain.Device)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to migrate devices: %w", err)
	}

	_, err = d.db.NewCreateIndex().
		Model((*domain.Device)(nil)).
		IfNotExists().
		Index("idx_device_owner_id").
		Column("owner_id").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create device index: %w", err)
	}

	ids := viper.GetStringSlice("builtin-devices")
	d.logger.Info("Migrating devices", "devices", ids)

	for _, id := range ids {
		d.logger.Info("Migrating builtin device", "id", id)

		deviceID := uuid.MustParse(id)
		if err := d.migrateDevice(ctx, deviceID); err != nil {
			return fmt.Errorf("migrate device: %w", err)
		}
	}

	return nil
}

func (d *DeviceRepo) GetDevice(
	ctx context.Context,
	id domain.DeviceID,
) (*domain.Device, error) {
	var device domain.Device

	err := d.db.NewSelect().
		Model(&device).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("select device: %w", err)
	}

	return &device, nil
}

func (d *DeviceRepo) DeviceExists(
	ctx context.Context,
	id domain.DeviceID,
) (bool, error) {
	var device domain.Device

	exists, err := d.db.NewSelect().
		Model(&device).
		Where("id = ?", id).
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("check device existence: %w", err)
	}

	return exists, nil
}

func (d *DeviceRepo) GetDevices(
	ctx context.Context,
	p paging.Paging,
) ([]domain.Device, error) {
	var devices []domain.Device

	err := d.db.NewSelect().
		Model(&devices).
		Order("id ASC").
		Limit(p.Size).
		Offset(p.Page * p.Size).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("select devices: %w", err)
	}

	return devices, nil
}

func (d *DeviceRepo) ConnectDevice(
	ctx context.Context,
	deviceID domain.DeviceID,
	userID uuid.UUID,
	name string,
) (*domain.Device, error) {
	_, err := d.db.NewUpdate().
		Model((*domain.Device)(nil)).
		Set("owner_id = ?", userID).
		Set("name = ?", name).
		Where("id = ?", deviceID).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect device: %w", err)
	}

	device, err := d.GetDevice(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get connected device: %w", err)
	}

	return device, nil
}

func (d *DeviceRepo) DisconnectDevice(
	ctx context.Context,
	deviceID domain.DeviceID,
) error {
	_, err := d.db.NewUpdate().
		Model((*domain.Device)(nil)).
		Set("owner_id = NULL").
		Set("name = NULL").
		Where("id = ?", deviceID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("disconnect device: %w", err)
	}

	return nil
}

func (d *DeviceRepo) InsertDevice(
	ctx context.Context,
	device *domain.Device,
) error {
	_, err := d.db.NewInsert().Model(device).Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert device: %w", err)
	}

	return nil
}

func (d *DeviceRepo) RenameDevice(
	ctx context.Context,
	id domain.DeviceID,
	name string,
) error {
	_, err := d.db.NewUpdate().
		Model((*domain.Device)(nil)).
		Set("name = ?", name).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("rename device: %w", err)
	}

	return nil
}

func (d *DeviceRepo) migrateDevice(
	ctx context.Context,
	id domain.DeviceID,
) error {
	exists, err := d.DeviceExists(ctx, id)
	if err != nil {
		return fmt.Errorf("failed check device in DB: %w", err)
	}

	if exists {
		return nil
	}

	if err := d.InsertDevice(ctx, &domain.Device{
		ID:           id,
		Name:         new(names.Gen()),
		PasswordHash: domain.HashPassword("password"),
		Model:        "Ship 0.1 (test)",
	}); err != nil {
		return fmt.Errorf("insert device: %w", err)
	}

	return nil
}
