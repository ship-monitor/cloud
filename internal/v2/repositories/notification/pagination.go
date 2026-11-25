package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	domain "github.com/ship-monitor/cloud/internal/v2/domain/notification"
	"github.com/ship-monitor/cloud/internal/v2/domain/pagination"
)

// Cursor identifies a position in descending occurrence time and ID order.
type Cursor struct {
	OccurredAt time.Time
	ID         uuid.UUID
}

// ListByDevicePage returns a page of notifications for a device.
func (r *Repository) ListByDevicePage(
	ctx context.Context,
	deviceID uuid.UUID,
	req pagination.Request[Cursor],
) (pagination.Page[domain.Notification, Cursor], error) {
	page, err := r.listPage(ctx, "device_id = $1", deviceID, req)
	if err != nil {
		return page, fmt.Errorf(
			"list page: %w",
			err,
		)
	}

	return page, nil
}

// ListBySeverityPage returns a page of notifications of a severity.
func (r *Repository) ListBySeverityPage(
	ctx context.Context,
	severity string,
	req pagination.Request[Cursor],
) (pagination.Page[domain.Notification, Cursor], error) {
	page, err := r.listPage(ctx, "severity = $1", severity, req)
	if err != nil {
		return page, fmt.Errorf(
			"list page: %w",
			err,
		)
	}

	return page, nil
}

func (r *Repository) listPage(
	ctx context.Context,
	filter string,
	value any,
	req pagination.Request[Cursor],
) (pagination.Page[domain.Notification, Cursor], error) {
	var page pagination.Page[domain.Notification, Cursor]

	size, err := req.Size()
	if err != nil {
		return page, fmt.Errorf("notification page size: %w", err)
	}

	var (
		timestamp any
		id        any
	)

	if req.After != nil {
		timestamp = req.After.OccurredAt
		id = req.After.ID
	}

	var rows []Row

	err = sqlx.SelectContext(ctx, r.db, &rows,
		`SELECT `+columns+` FROM device_notifications WHERE `+filter+`
		 AND ($2::timestamptz IS NULL OR (occurred_at, id) < ($2, $3))
		 ORDER BY occurred_at DESC, id DESC LIMIT $4`,
		value, timestamp, id, size+1)
	if err != nil {
		return page, fmt.Errorf("select notifications page: %w", err)
	}

	items := make([]domain.Notification, len(rows))
	for i, row := range rows {
		items[i] = row.ToDomain()
	}

	return pagination.FromRows(
		items,
		size,
		func(item domain.Notification) Cursor {
			return Cursor{OccurredAt: item.OccurredAt, ID: item.ID}
		},
	), nil
}
