package connect

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/domain/pagination"
)

// ListByDevicePage returns connect requests in ascending ID order.
func (r *Repo) ListByDevicePage(
	ctx context.Context,
	deviceID uuid.UUID,
	req pagination.Request[uuid.UUID],
) (pagination.Page[device.ConnectRequest, uuid.UUID], error) {
	var page pagination.Page[device.ConnectRequest, uuid.UUID]

	size, err := req.Size()
	if err != nil {
		return page, fmt.Errorf("connect request page size: %w", err)
	}

	var rows []row

	err = sqlx.SelectContext(ctx, r.db, &rows,
		`SELECT `+columns+` FROM connect_requests
		 WHERE device_id = $1 AND ($2::uuid IS NULL OR id > $2)
		 ORDER BY id LIMIT $3`, deviceID, req.After, size+1)
	if err != nil {
		return page, fmt.Errorf("select connect requests page: %w", err)
	}

	items := make([]device.ConnectRequest, len(rows))
	for i, row := range rows {
		items[i] = row.toDomain()
	}

	return pagination.FromRows(items, size,
		func(item device.ConnectRequest) uuid.UUID { return item.ID }), nil
}
