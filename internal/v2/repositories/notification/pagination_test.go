//go:build integration

package notification_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	domain "github.com/ship-monitor/cloud/internal/v2/domain/notification"
	"github.com/ship-monitor/cloud/internal/v2/domain/pagination"
	"github.com/ship-monitor/cloud/internal/v2/repositories/notification"
	"github.com/stretchr/testify/require"
)

type (
	notificationPage    = pagination.Page[domain.Notification, notification.Cursor]
	notificationRequest = pagination.Request[notification.Cursor]
)

//nolint:funlen
func TestRepository_ListByDevicePageForUser(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	repo := notification.New(db)
	ctx := t.Context()
	userID := uuid.New()
	otherUserID := uuid.New()
	first := newNotification()
	second := newNotification()
	third := newNotification()
	second.DeviceID = first.DeviceID
	third.DeviceID = first.DeviceID
	second.OccurredAt = first.OccurredAt
	third.OccurredAt = first.OccurredAt
	first.ID = uuid.UUID{1}
	second.ID = uuid.UUID{2}
	third.ID = uuid.UUID{3}
	otherDevice := newNotification()
	otherDevice.DeviceID = uuid.New()

	for _, item := range []domain.Notification{
		first, second, third, otherDevice,
	} {
		require.NoError(t, repo.InsertNotification(ctx, item))
	}

	require.NoError(t, repo.SetRead(ctx, userID, third.ID, true))
	require.NoError(t, repo.SetRead(ctx, userID, second.ID, false))
	require.NoError(t, repo.SetRead(ctx, otherUserID, second.ID, true))
	require.NoError(t, repo.SetRead(ctx, otherUserID, otherDevice.ID, true))

	page, err := repo.ListByDevicePageForUser(
		ctx, first.DeviceID, userID,
		notificationRequest{After: nil, Limit: 2},
	)
	require.NoError(t, err)
	require.Len(t, page.Items, 2)
	require.Equal(t, third.ID, page.Items[0].ID)
	require.True(t, page.Items[0].IsRead)
	require.Equal(t, second.ID, page.Items[1].ID)
	require.False(t, page.Items[1].IsRead)
	require.NotNil(t, page.Next)
	require.Equal(t, second.ID, page.Next.ID)

	page, err = repo.ListByDevicePageForUser(
		ctx, first.DeviceID, userID,
		notificationRequest{After: page.Next, Limit: 2},
	)
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, first.ID, page.Items[0].ID)
	require.False(t, page.Items[0].IsRead)
	require.Nil(t, page.Next)

	page, err = repo.ListByDevicePageForUser(
		ctx, first.DeviceID, otherUserID,
		notificationRequest{After: nil, Limit: 0},
	)
	require.NoError(t, err)
	require.Len(t, page.Items, 3)
	require.False(t, page.Items[0].IsRead)
	require.True(t, page.Items[1].IsRead)
	require.False(t, page.Items[2].IsRead)
	require.Nil(t, page.Next)

	page, err = repo.ListByDevicePageForUser(
		ctx, uuid.New(), userID, notificationRequest{After: nil, Limit: 0},
	)
	require.NoError(t, err)
	require.NotNil(t, page.Items)
	require.Empty(t, page.Items)
	require.Nil(t, page.Next)

	_, err = repo.ListByDevicePageForUser(
		ctx, first.DeviceID, userID,
		notificationRequest{After: nil, Limit: -1},
	)
	require.ErrorIs(t, err, pagination.ErrInvalidLimit)
	_, err = repo.ListByDevicePageForUser(
		ctx, first.DeviceID, userID,
		notificationRequest{After: nil, Limit: pagination.MaxLimit + 1},
	)
	require.ErrorIs(t, err, pagination.ErrInvalidLimit)
}

func TestRepository_CursorPages(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	repo := notification.New(db)
	ctx := t.Context()
	first := newNotification()
	second := newNotification()
	second.DeviceID = first.DeviceID
	second.OccurredAt = first.OccurredAt
	first.ID = uuid.UUID{1}
	second.ID = uuid.UUID{2}
	other := newNotification()
	other.DeviceID = uuid.New()

	other.Severity = "critical"
	for _, item := range []domain.Notification{first, second, other} {
		require.NoError(t, repo.InsertNotification(ctx, item))
	}

	lists := map[string]func(context.Context, notificationRequest) (notificationPage, error){
		"by device": func(ctx context.Context, req notificationRequest) (notificationPage, error) {
			return repo.ListByDevicePage(ctx, first.DeviceID, req)
		},
		"by severity": func(ctx context.Context, req notificationRequest) (notificationPage, error) {
			return repo.ListBySeverityPage(ctx, first.Severity, req)
		},
	}
	for name, list := range lists {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			page, err := list(ctx, notificationRequest{After: nil, Limit: 1})
			require.NoError(t, err)
			require.Len(t, page.Items, 1)
			require.Equal(t, second.ID, page.Items[0].ID)
			require.NotNil(t, page.Next)

			page, err = list(
				ctx,
				notificationRequest{After: page.Next, Limit: 1},
			)
			require.NoError(t, err)
			require.Len(t, page.Items, 1)
			require.Equal(t, first.ID, page.Items[0].ID)
			require.Nil(t, page.Next)
		})
	}
}
