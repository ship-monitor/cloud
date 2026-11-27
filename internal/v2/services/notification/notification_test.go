package notification_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	domainnotification "github.com/ship-monitor/cloud/internal/v2/domain/notification"
	"github.com/ship-monitor/cloud/internal/v2/domain/pagination"
	"github.com/ship-monitor/cloud/internal/v2/services/notification"
	"github.com/stretchr/testify/require"
)

type notificationPage = pagination.Page[
	domainnotification.Notification,
	domainnotification.Cursor,
]

type notificationRequest = pagination.Request[domainnotification.Cursor]

type contextKey struct{}

type deviceStub struct {
	actor auth.Actor
	id    uuid.UUID
	value any
	err   error
	calls int
}

func (s *deviceStub) Get(
	ctx context.Context,
	actor auth.Actor,
	id uuid.UUID,
) (device.Device, error) {
	s.value = ctx.Value(contextKey{})
	s.actor = actor
	s.id = id
	s.calls++

	return device.Device{}, s.err
}

type repoStub struct {
	value    any
	deviceID uuid.UUID
	userID   uuid.UUID
	req      notificationRequest
	page     notificationPage
	err      error
	calls    int
}

func (s *repoStub) InsertNotification(
	context.Context,
	domainnotification.Notification,
) error {
	return nil
}

func (s *repoStub) ListByDevicePageForUser(
	ctx context.Context,
	deviceID, userID uuid.UUID,
	req notificationRequest,
) (notificationPage, error) {
	s.value = ctx.Value(contextKey{})
	s.deviceID = deviceID
	s.userID = userID
	s.req = req
	s.calls++

	return s.page, s.err
}

//nolint:exhaustruct_v5,funlen
func TestService_ListByDevicePage(t *testing.T) {
	t.Parallel()

	t.Run("authorizes and forwards actor and cursor", func(t *testing.T) {
		t.Parallel()

		ctx := context.WithValue(t.Context(), contextKey{}, "request")
		actor := auth.Actor{UserID: uuid.New()}
		deviceID := uuid.New()
		cursor := domainnotification.Cursor{
			OccurredAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
			ID:         uuid.New(),
		}
		req := notificationRequest{After: &cursor, Limit: 2}
		want := notificationPage{
			Items: []domainnotification.Notification{{
				ID: uuid.New(), DeviceID: deviceID, IsRead: true,
			}},
			Next: &cursor,
		}
		devices := &deviceStub{}
		repo := &repoStub{page: want}
		service := notification.New(repo, devices)

		got, err := service.ListByDevicePage(ctx, actor, deviceID, req)
		require.NoError(t, err)
		require.Equal(t, want, got)
		require.Equal(t, 1, devices.calls)
		require.Equal(t, "request", devices.value)
		require.Equal(t, actor, devices.actor)
		require.Equal(t, deviceID, devices.id)
		require.Equal(t, 1, repo.calls)
		require.Equal(t, "request", repo.value)
		require.Equal(t, deviceID, repo.deviceID)
		require.Equal(t, actor.UserID, repo.userID)
		require.Equal(t, req, repo.req)
	})

	t.Run("inaccessible device prevents listing", func(t *testing.T) {
		t.Parallel()

		devices := &deviceStub{err: device.ErrNotFound}
		repo := &repoStub{}
		service := notification.New(repo, devices)

		_, err := service.ListByDevicePage(
			t.Context(), auth.Actor{UserID: uuid.New()}, uuid.New(),
			notificationRequest{},
		)
		require.ErrorIs(t, err, device.ErrNotFound)
		require.Equal(t, 1, devices.calls)
		require.Zero(t, repo.calls)
	})

	t.Run("device errors prevent listing", func(t *testing.T) {
		t.Parallel()

		deviceErr := io.EOF
		devices := &deviceStub{err: deviceErr}
		repo := &repoStub{}
		service := notification.New(repo, devices)

		_, err := service.ListByDevicePage(
			t.Context(), auth.Actor{UserID: uuid.New()}, uuid.New(),
			notificationRequest{},
		)
		require.ErrorIs(t, err, deviceErr)
		require.Zero(t, repo.calls)
	})

	t.Run("repository errors propagate", func(t *testing.T) {
		t.Parallel()

		devices := &deviceStub{}
		repo := &repoStub{err: pagination.ErrInvalidLimit}
		service := notification.New(repo, devices)

		_, err := service.ListByDevicePage(
			t.Context(), auth.Actor{UserID: uuid.New()}, uuid.New(),
			notificationRequest{Limit: -1},
		)
		require.ErrorIs(t, err, pagination.ErrInvalidLimit)
		require.Equal(t, 1, devices.calls)
		require.Equal(t, 1, repo.calls)
	})
}
