//go:build integration

package notification_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/repositories/notification"
	"github.com/stretchr/testify/require"
)

//nolint:funlen
func TestRepository_SetRead(t *testing.T) {
	t.Parallel()
	db := setupDB(t)
	repo := notification.New(db)
	ctx := t.Context()
	userID, otherUser, notificationID := uuid.New(), uuid.New(), uuid.New()

	require.NoError(t, repo.SetRead(ctx, userID, notificationID, true))
	require.NoError(t, repo.SetRead(ctx, userID, notificationID, false))
	require.NoError(t, repo.SetRead(ctx, otherUser, notificationID, true))

	var count int

	err := db.GetContext(
		ctx,
		&count,
		`SELECT count(*) FROM user_notification_states
		WHERE user_id = $1 AND notification_id = $2`,
		userID,
		notificationID,
	)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	var isRead bool

	err = db.GetContext(
		ctx,
		&isRead,
		`SELECT is_read FROM user_notification_states
		WHERE user_id = $1 AND notification_id = $2`,
		userID,
		notificationID,
	)
	require.NoError(t, err)
	require.False(t, isRead)
	err = db.GetContext(
		ctx,
		&isRead,
		`SELECT is_read FROM user_notification_states
		WHERE user_id = $1 AND notification_id = $2`,
		otherUser,
		notificationID,
	)
	require.NoError(t, err)
	require.True(t, isRead)

	tx, err := db.BeginTxx(ctx, nil)
	require.NoError(t, err)
	require.NoError(
		t,
		repo.WithTx(tx).SetRead(ctx, userID, notificationID, true),
	)
	require.NoError(t, tx.Rollback())

	err = db.GetContext(
		ctx,
		&isRead,
		`SELECT is_read FROM user_notification_states
		WHERE user_id = $1 AND notification_id = $2`,
		userID,
		notificationID,
	)
	require.NoError(t, err)
	require.False(t, isRead)
}
