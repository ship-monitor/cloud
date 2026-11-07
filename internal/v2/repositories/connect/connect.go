// Package connect provides a sqlx-based repository for connect requests.
package connect

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
)

const TimeFormat = time.RFC3339

// ErrNotFound is returned when no connect request matches the given
// constraints.
var ErrNotFound = errors.New("connect request not found")

// Repo reads and writes connect requests in the database.
type Repo struct {
	db sqlx.ExtContext
}

// New returns a repository backed by db.
func New(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

// WithTx returns a repository that runs its queries within tx.
func (r *Repo) WithTx(tx *sqlx.Tx) *Repo {
	return &Repo{db: tx}
}

type row struct {
	ID        uuid.UUID `db:"id"`
	DeviceID  uuid.UUID `db:"device_id"`
	UserID    uuid.UUID `db:"user_id"`
	CreatedAt time.Time `db:"created_at"`
	Approved  bool      `db:"approved"`
}

func (r row) toDomain() device.ConnectRequest {
	return device.ConnectRequest{
		ID:        r.ID,
		DeviceID:  r.DeviceID,
		CreatedAt: r.CreatedAt,
		Approved:  r.Approved,
		UserID:    r.UserID,
	}
}

const columns = "id, device_id, created_at, approved, user_id"

// Create inserts a connect request with the supplied ID and state.
func (r *Repo) Create(
	ctx context.Context,
	request device.ConnectRequest,
) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO connect_requests (id, device_id, created_at, approved, user_id)
		 VALUES (?, ?, ?, ?, ?)`,
		request.ID,
		request.DeviceID,
		request.CreatedAt.Format(TimeFormat),
		request.Approved,
		request.UserID,
	)
	if err != nil {
		return fmt.Errorf("insert connect request: %w", err)
	}

	return nil
}

// Get returns the connect request with the given ID, or ErrNotFound.
func (r *Repo) Get(
	ctx context.Context,
	id uuid.UUID,
) (device.ConnectRequest, error) {
	var record row

	err := r.db.QueryRowxContext(ctx,
		`SELECT `+columns+` FROM connect_requests WHERE id = ?`, id,
	).StructScan(&record)
	if errors.Is(err, sql.ErrNoRows) {
		return device.ConnectRequest{}, ErrNotFound
	}

	if err != nil {
		return device.ConnectRequest{}, fmt.Errorf(
			"select connect request: %w",
			err,
		)
	}

	return record.toDomain(), nil
}

// ListByDevice returns the device's connect requests ordered by ID.
func (r *Repo) ListByDevice(
	ctx context.Context,
	deviceID uuid.UUID,
) ([]device.ConnectRequest, error) {
	var rows []row

	err := sqlx.SelectContext(
		ctx,
		r.db,
		&rows,
		`SELECT `+columns+` FROM connect_requests WHERE device_id = ? ORDER BY id`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("select connect requests by device: %w", err)
	}

	requests := make([]device.ConnectRequest, len(rows))
	for i, rw := range rows {
		requests[i] = rw.toDomain()
	}

	return requests, nil
}

// SetApproved updates the approval state only if the request belongs to
// deviceID.
// It returns ErrNotFound if either identifier does not match.
func (r *Repo) SetApproved(
	ctx context.Context,
	id uuid.UUID,
	approved bool,
) error {
	res, err := r.db.ExecContext(
		ctx,
		`UPDATE connect_requests SET approved = ? WHERE id = ?`,
		approved,
		id,
	)
	if err != nil {
		return fmt.Errorf("update connect request approval: %w", err)
	}

	if err := requireAffected(res); err != nil {
		return fmt.Errorf("update connect request approval: %w", err)
	}

	return nil
}

// Delete removes the connect request with the given ID, or returns ErrNotFound.
func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(
		ctx,
		`DELETE FROM connect_requests WHERE id = ?`,
		id,
	)
	if err != nil {
		return fmt.Errorf("delete connect request: %w", err)
	}

	if err := requireAffected(res); err != nil {
		return fmt.Errorf("delete connect request: %w", err)
	}

	return nil
}

func requireAffected(res sql.Result) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if affected == 0 {
		return ErrNotFound
	}

	return nil
}
