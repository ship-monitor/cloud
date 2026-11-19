// Package device provides a sqlx-based repository for the devices table.
//
//nolint:varnamelen
package device

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	domain "github.com/ship-monitor/cloud/internal/v2/domain/device"
)

// ErrNotFound is returned when no device matches the given constraints.
var ErrNotFound = errors.New("device not found")

// Repository reads and writes devices in the database.
//
// It depends only on sqlx.ExtContext, so it works with both *sqlx.DB and
// *sqlx.Tx, letting callers run multiple operations in one transaction.
type Repository struct {
	db sqlx.ExtContext
}

// New returns a Repository backed by the given database.
func New(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// WithTx returns a Repository that runs all its queries within tx.
func (r *Repository) WithTx(tx *sqlx.Tx) *Repository {
	return &Repository{db: tx}
}

// row mirrors the columns of the devices table.
type row struct {
	ID           uuid.UUID  `db:"id"`
	OwnerID      uuid.UUID  `db:"owner_id"`
	Name         string     `db:"name"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    *time.Time `db:"updated_at"`
	Manufacturer string     `db:"manufacturer"`
	Model        string     `db:"model"`
	Revision     string     `db:"revision"`
	Software     string     `db:"software"`
}

func (r row) toDomain() domain.Device {
	return domain.Device{
		ID:           r.ID,
		OwnerID:      r.OwnerID,
		Name:         r.Name,
		Manufacturer: r.Manufacturer,
		Model:        r.Model,
		Revision:     r.Revision,
		Software:     r.Software,
	}
}

const columns = "id, owner_id, name, created_at, updated_at, manufacturer, model, revision, software"

// Create inserts a new device.
func (r *Repository) Create(ctx context.Context, d domain.Device) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO devices (`+columns+`)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		d.ID, d.OwnerID, d.Name, time.Now().UTC(),
		d.Manufacturer, d.Model, d.Revision, d.Software,
	)
	if err != nil {
		return fmt.Errorf("insert device: %w", err)
	}

	return nil
}

// Get returns the device with the given id if it is owned by ownerID,
// or ErrNotFound. Foreign devices are reported as not found,
// so they cannot be probed.
func (r *Repository) Get(
	ctx context.Context,
	ownerID, id uuid.UUID,
) (domain.Device, error) {
	var rw row

	err := r.db.
		QueryRowxContext(
			ctx,
			`SELECT `+columns+` FROM devices WHERE id = $1 AND owner_id = $2`,
			id, ownerID,
		).
		StructScan(&rw)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Device{}, ErrNotFound
	}

	if err != nil {
		return domain.Device{}, fmt.Errorf("select device: %w", err)
	}

	return rw.toDomain(), nil
}

// ListByOwner returns a page of devices owned by the given user,
// ordered by id.
func (r *Repository) ListByOwner(
	ctx context.Context,
	ownerID uuid.UUID,
	paging domain.Paging,
) ([]domain.Device, error) {
	var rows []row

	err := sqlx.SelectContext(ctx, r.db, &rows,
		`SELECT `+columns+
			` FROM devices WHERE owner_id = $1 ORDER BY id LIMIT $2 OFFSET $3`,
		ownerID, paging.Size, paging.Offset(),
	)
	if err != nil {
		return nil, fmt.Errorf("select devices by owner: %w", err)
	}

	return toDomainAll(rows), nil
}

// Exists reports whether a device with the given id exists,
// regardless of the owner.
func (r *Repository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool

	err := r.db.
		QueryRowxContext(
			ctx,
			`SELECT EXISTS (SELECT 1 FROM devices WHERE id = $1)`,
			id,
		).
		Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check device exists: %w", err)
	}

	return exists, nil
}

// List returns all devices ordered by id.
func (r *Repository) List(ctx context.Context) ([]domain.Device, error) {
	var rows []row

	err := sqlx.SelectContext(ctx, r.db, &rows,
		`SELECT `+columns+` FROM devices ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("select devices: %w", err)
	}

	return toDomainAll(rows), nil
}

// Rename updates the name of the device with the given id if it is
// owned by ownerID, or ErrNotFound.
func (r *Repository) Rename(
	ctx context.Context,
	ownerID, id uuid.UUID,
	name string,
) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE devices SET name = $1 WHERE id = $2 AND owner_id = $3`,
		name, id, ownerID,
	)
	if err != nil {
		return fmt.Errorf("execute update: %w", err)
	}

	return nil
}

// Delete removes the device with the given id if it is owned by ownerID,
// or ErrNotFound.
func (r *Repository) Delete(ctx context.Context, ownerID, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM devices WHERE id = $1 AND owner_id = $2`,
		id, ownerID,
	)
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}

	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Repository) UpdateFirmware(
	ctx context.Context,
	device domain.Device,
) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE devices
		SET manufacturer = $1, model = $2, revision = $3, software = $4
		WHERE id = $5`, device.Manufacturer, device.Model, device.Revision, device.Software, device.ID)
	if err != nil {
		return fmt.Errorf("execute update: %w", err)
	}

	return nil
}

func toDomainAll(rows []row) []domain.Device {
	devices := make([]domain.Device, len(rows))
	for i, rw := range rows {
		devices[i] = rw.toDomain()
	}

	return devices
}
