package notification

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/ship-monitor/cloud/internal/v2/domain/notification"
)

// Repository reads and writes device_notifications in the database.
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

// Row mirrors the columns of the device_notifications table.
type Row struct {
	ID         uuid.UUID  `db:"id"`
	DeviceID   uuid.UUID  `db:"device_id"`
	SensorID   *uuid.UUID `db:"sensor_id"`
	Type       string     `db:"type"`
	Severity   string     `db:"severity"`
	OccurredAt time.Time  `db:"occurred_at"`
	ReportedAt time.Time  `db:"reported_at"`
	ReceivedAt time.Time  `db:"received_at"`
	Data       JSONB      `db:"data"`
}

// ToDomain converts a row to the domain Notification type.
func (r Row) ToDomain() notification.Notification {
	return notification.Notification{
		ID:         r.ID,
		DeviceID:   r.DeviceID,
		SensorID:   r.SensorID,
		Type:       r.Type,
		Severity:   r.Severity,
		OccurredAt: r.OccurredAt,
		ReportedAt: r.ReportedAt,
		ReceivedAt: r.ReceivedAt,
		Data:       map[string]any(r.Data),
		IsRead:     false,
	}
}

// columns lists the device_notifications columns.
const columns = "id, device_id, sensor_id, type, severity, occurred_at, reported_at, received_at, data"

// InsertNotification inserts a new notification.
//
//nolint:varnamelen
func (r *Repository) InsertNotification(
	ctx context.Context,
	n notification.Notification,
) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO device_notifications (`+columns+`)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		n.ID,
		n.DeviceID,
		n.SensorID,
		n.Type,
		n.Severity,
		n.OccurredAt,
		n.ReportedAt,
		n.ReceivedAt,
		JSONB(n.Data),
	)
	if err != nil {
		return fmt.Errorf("execute statement: %w", err)
	}

	return nil
}

// Get returns the notification with the given id.
//
//nolint:varnamelen
func (r *Repository) Get(
	ctx context.Context,
	id uuid.UUID,
) (*notification.Notification, error) {
	var rw Row

	err := r.db.QueryRowxContext(
		ctx,
		`SELECT `+columns+` FROM device_notifications WHERE id = $1`,
		id,
	).StructScan(&rw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, notification.ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf(
			"execute query: %w",
			err,
		)
	}

	return new(rw.ToDomain()), nil
}

// ListByDevice returns notifications for a device, ordered by occurred_at
// descending.
func (r *Repository) ListByDevice(
	ctx context.Context,
	deviceID uuid.UUID,
	limit, offset int,
) ([]notification.Notification, error) {
	var rows []Row

	err := sqlx.SelectContext(ctx, r.db, &rows,
		`SELECT `+columns+`
		 FROM device_notifications WHERE device_id = $1
		 ORDER BY occurred_at DESC LIMIT $2 OFFSET $3`,
		deviceID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("execute select: %w", err)
	}

	notifications := make([]notification.Notification, len(rows))
	for i, rw := range rows {
		notifications[i] = rw.ToDomain()
	}

	return notifications, nil
}

// ListBySeverity returns notifications filtered by severity, ordered by
// occurred_at descending.
func (r *Repository) ListBySeverity(
	ctx context.Context,
	severity string,
	limit, offset int,
) ([]notification.Notification, error) {
	var rows []Row

	err := sqlx.SelectContext(ctx, r.db, &rows,
		`SELECT `+columns+`
		 FROM device_notifications WHERE severity = $1
		 ORDER BY occurred_at DESC LIMIT $2 OFFSET $3`,
		severity, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("execute select: %w", err)
	}

	notifications := make([]notification.Notification, len(rows))
	for i, rw := range rows {
		notifications[i] = rw.ToDomain()
	}

	return notifications, nil
}

// Exists reports whether a notification with the given id exists.
func (r *Repository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool

	err := r.db.QueryRowxContext(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM device_notifications WHERE id = $1)`,
		id,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check notification exists: %w", err)
	}

	return exists, nil
}

// Delete removes the notification with the given id.
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM device_notifications WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("delete notification: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete notification: %w", err)
	}

	if affected == 0 {
		return notification.ErrNotFound
	}

	return nil
}
