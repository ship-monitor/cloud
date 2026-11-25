package notification

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// SetRead records whether a notification has been read by a particular user.
// Callers are responsible for authorizing access to the notification.
func (r *Repository) SetRead(
	ctx context.Context,
	userID, notificationID uuid.UUID,
	isRead bool,
) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO user_notification_states (user_id, notification_id, is_read)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, notification_id)
		 DO UPDATE SET is_read = EXCLUDED.is_read`,
		userID,
		notificationID,
		isRead,
	)
	if err != nil {
		return fmt.Errorf("set notification read state: %w", err)
	}

	return nil
}
