package devices

import (
	"context"
	"database/sql"
	"fmt"

	"charm.land/log/v2"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/ship-monitor/cloud/internal/domain"
	"github.com/ship-monitor/cloud/internal/repository/common"
	"github.com/ship-monitor/cloud/pkg"
	"github.com/ship-monitor/cloud/pkg/names"
	"github.com/ship-monitor/cloud/pkg/paging"
	"github.com/spf13/viper" //nolint:depguard
	"github.com/uptrace/bun"
)

var _ pkg.MigrationRepo = (*Repository)(nil)

type Repository struct {
	db     *bun.DB
	logger *log.Logger
	redis  *redis.Client
	ch     clickhouse.Conn
}

func New(
	db *sql.DB,
	logger *log.Logger,
	redis *redis.Client,
	ch clickhouse.Conn,
) *Repository {
	return &Repository{
		db:     common.NewBunDB(db),
		logger: logger,
		redis:  redis,
		ch:     ch,
	}
}

// Migrate implements [pkg.MigrationRepo].
func (r *Repository) Migrate(ctx context.Context) error {
	_, err := r.db.NewCreateTable().
		Model((*domain.Device)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to migrate devices: %w", err)
	}

	_, err = r.db.NewCreateIndex().
		Model((*domain.Device)(nil)).
		IfNotExists().
		Index("idx_device_owner_id").
		Column("owner_id").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create device index: %w", err)
	}

	// TODO: remove device migration
	ids := viper.GetStringSlice("builtin-devices")
	r.logger.Info("Migrating devices", "devices", ids)

	for _, id := range ids {
		r.logger.Info("Migrating builtin device", "id", id)

		deviceID := uuid.MustParse(id)
		if err := r.migrateDevice(ctx, deviceID); err != nil {
			return fmt.Errorf("migrate device: %w", err)
		}
	}

	const query = `
		CREATE TABLE IF NOT EXISTS device_states
		(
			device_id String,
			state String,
			timestamp DateTime64(6, 'UTC'),

			value_float Nullable(Float64),
			value_int Nullable(Int64),
			value_bool Nullable(Bool),
			value_string Nullable(String),
			inserted_at DateTime64(6, 'UTC') DEFAULT now64(6)
		)
		ENGINE = MergeTree
		ORDER BY (device_id, state, timestamp)
	`

	if err := r.ch.Exec(ctx, query); err != nil {
		return fmt.Errorf("create %s table: %w", deviceStatesTable, err)
	}

	return nil
}

func (r *Repository) GetDevice(
	ctx context.Context,
	id domain.DeviceID,
) (*domain.Device, error) {
	var device domain.Device

	err := r.db.NewSelect().
		Model(&device).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("select device: %w", err)
	}

	return &device, nil
}

func (r *Repository) GetDevicesByIDs(
	ctx context.Context,
	ids []domain.DeviceID,
) ([]domain.Device, error) {
	if len(ids) == 0 {
		return []domain.Device{}, nil
	}

	devices := make([]domain.Device, 0, len(ids))

	err := r.db.NewSelect().
		Model(&devices).
		Where("id IN (?)", bun.List(ids)).
		Order("id ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("select devices by IDs: %w", err)
	}

	return devices, nil
}

func (r *Repository) DeviceExists(
	ctx context.Context,
	id domain.DeviceID,
) (bool, error) {
	var device domain.Device

	exists, err := r.db.NewSelect().
		Model(&device).
		Where("id = ?", id).
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("check device existence: %w", err)
	}

	return exists, nil
}

func (r *Repository) GetDevices(
	ctx context.Context,
	page paging.Paging,
) ([]domain.Device, error) {
	var devices []domain.Device

	err := r.db.NewSelect().
		Model(&devices).
		Order("id ASC").
		Limit(page.Size).
		Offset(page.Page * page.Size).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("select devices: %w", err)
	}

	return devices, nil
}

func (r *Repository) ConnectDevice(
	ctx context.Context,
	deviceID domain.DeviceID,
	userID uuid.UUID,
	name string,
) (*domain.Device, error) {
	result, err := r.db.NewUpdate().
		Model((*domain.Device)(nil)).
		Set("owner_id = ?", userID).
		Set("name = ?", name).
		Where("id = ?", deviceID).
		Where("owner_id IS NULL").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect device: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("get affected device count: %w", err)
	}

	if rowsAffected == 0 {
		return nil, domain.ErrDeviceAlreadyConnected
	}

	device, err := r.GetDevice(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get connected device: %w", err)
	}

	return device, nil
}

func (r *Repository) DisconnectDevice(
	ctx context.Context,
	deviceID domain.DeviceID,
) error {
	_, err := r.db.NewUpdate().
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

func (r *Repository) InsertDevice(
	ctx context.Context,
	device *domain.Device,
) error {
	_, err := r.db.NewInsert().Model(device).Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert device: %w", err)
	}

	return nil
}

func (r *Repository) RenameDevice(
	ctx context.Context,
	id domain.DeviceID,
	name string,
) error {
	_, err := r.db.NewUpdate().
		Model((*domain.Device)(nil)).
		Set("name = ?", name).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("rename device: %w", err)
	}

	return nil
}

func (r *Repository) migrateDevice(
	ctx context.Context,
	id domain.DeviceID,
) error {
	exists, err := r.DeviceExists(ctx, id)
	if err != nil {
		return fmt.Errorf("failed check device in DB: %w", err)
	}

	if exists {
		return nil
	}

	if err := r.InsertDevice(
		ctx,
		&domain.Device{ //nolint:exhaustruct_v5
			ID:           id,
			Name:         new(names.Gen()),
			PasswordHash: domain.HashPassword("password"),
			Model:        "Ship 0.1 (test)",
		},
	); err != nil {
		return fmt.Errorf("insert device: %w", err)
	}

	return nil
}
