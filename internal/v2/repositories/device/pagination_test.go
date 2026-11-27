//go:build integration

package device_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	domaindevice "github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/domain/pagination"
	"github.com/ship-monitor/cloud/internal/v2/repositories/device"
	"github.com/stretchr/testify/require"
)

func TestRepository_CursorPages(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	ctx := t.Context()
	devices := device.New(db)
	first, foreign, last := newDevice(), newDevice(), newDevice()
	first.ID, foreign.ID, last.ID = uuid.UUID{1}, uuid.UUID{2}, uuid.UUID{3}

	last.OwnerID = first.OwnerID
	for _, item := range []domaindevice.Device{last, first, foreign} {
		require.NoError(t, devices.Create(ctx, item))
	}

	t.Run("all devices", func(t *testing.T) {
		t.Parallel()

		checkPages(
			t,
			devices.ListPage,
			[]domaindevice.Device{first, foreign, last},
			func(d domaindevice.Device) uuid.UUID { return d.ID },
		)
	})

	t.Run("devices by owner", func(t *testing.T) {
		t.Parallel()

		list := func(ctx context.Context, req pagination.Request[uuid.UUID]) (
			pagination.Page[domaindevice.Device, uuid.UUID], error,
		) {
			return devices.ListByOwnerPage(ctx, first.OwnerID, req)
		}
		checkPages(t, list, []domaindevice.Device{first, last},
			func(d domaindevice.Device) uuid.UUID { return d.ID })
	})
}

func checkPages[T any, C comparable](
	t *testing.T,
	list func(
		context.Context,
		pagination.Request[C],
	) (pagination.Page[T, C], error),
	want []T,
	key func(T) C,
) {
	t.Helper()

	req := pagination.Request[C]{After: nil, Limit: 1}
	for index, item := range want {
		page, err := list(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []T{item}, page.Items)

		cursor := key(item)
		if index+1 < len(want) {
			require.Equal(t, &cursor, page.Next)
			req.After = page.Next
		} else {
			require.Nil(t, page.Next)

			req.After = &cursor
		}
	}

	page, err := list(t.Context(), req)
	require.NoError(t, err)
	require.Empty(t, page.Items)
	require.Nil(t, page.Next)
}
