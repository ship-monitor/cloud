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
