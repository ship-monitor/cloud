// Package device provides a sqlx-based repository for the devices table.
//
//nolint:varnamelen
package device

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

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
	ID          uuid.UUID `db:"id"`
	OwnerID     uuid.UUID `db:"owner_id"`
	Name        string    `db:"name"`
	SecurityKey string    `db:"security_key"`
}

func (r row) toDomain() domain.Device {
	return domain.Device{
		ID:          r.ID,
		OwnerID:     r.OwnerID,
		Name:        r.Name,
		SecurityKey: r.SecurityKey,
	}
}

const columns = "id, owner_id, name, security_key"

// Create inserts a new device.
func (r *Repository) Create(ctx context.Context, d domain.Device) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO devices (id, owner_id, name, security_key)
		 VALUES (?, ?, ?, ?)`,
		d.ID, d.OwnerID, d.Name,
	)
	if err != nil {
		return fmt.Errorf("insert device: %w", err)
	}

	return nil
}

// Get returns the device with the given id, or ErrNotFound.
func (r *Repository) Get(
	ctx context.Context,
	id uuid.UUID,
) (domain.Device, error) {
	var rw row

	err := r.db.
		QueryRowxContext(
			ctx,
			`SELECT `+columns+` FROM devices WHERE id = ?`,
			id,
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
			` FROM devices WHERE owner_id = ? ORDER BY id LIMIT ? OFFSET ?`,
		ownerID, paging.Size, paging.Offset(),
	)
	if err != nil {
		return nil, fmt.Errorf("select devices by owner: %w", err)
	}

	return toDomainAll(rows), nil
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

// Rename updates the name of the device with the given id, or ErrNotFound.
func (r *Repository) Rename(
	ctx context.Context,
	id uuid.UUID,
	name string,
) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE devices SET name = ? WHERE id = ?`,
		name, id,
	)
	if err != nil {
		return fmt.Errorf("rename device: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rename device: %w", err)
	}

	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

// Delete removes the device with the given id, or ErrNotFound.
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM devices WHERE id = ?`, id)
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

func toDomainAll(rows []row) []domain.Device {
	devices := make([]domain.Device, len(rows))
	for i, rw := range rows {
		devices[i] = rw.toDomain()
	}

	return devices
}
