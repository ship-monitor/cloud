package device

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	domain "github.com/ship-monitor/cloud/internal/v2/domain/device"
)

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
		return domain.ErrNotFound
	}

	return nil
}
