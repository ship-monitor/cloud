package device_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	domaindevice "github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/domain/pagination"
	domainsensor "github.com/ship-monitor/cloud/internal/v2/domain/sensor"
	domainuser "github.com/ship-monitor/cloud/internal/v2/domain/user"
	"github.com/ship-monitor/cloud/internal/v2/repositories/connect"
	"github.com/ship-monitor/cloud/internal/v2/repositories/device"
	"github.com/ship-monitor/cloud/internal/v2/repositories/sensor"
	"github.com/ship-monitor/cloud/internal/v2/repositories/user"
	"github.com/stretchr/testify/require"
)

// Exercise list operations against the same migrated PostgreSQL DB.
//
//nolint:funlen // Keep the shared database fixture and related cases together.
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

	t.Run("connect requests by device", func(t *testing.T) {
		t.Parallel()

		requests := connect.New(db)
		stamp := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		firstRequest := domaindevice.ConnectRequest{
			ID: uuid.UUID{1}, DeviceID: first.ID, CreatedAt: stamp,
			UserID: uuid.New(), Approved: false,
		}
		lastRequest := firstRequest
		lastRequest.ID = uuid.UUID{3}
		other := firstRequest
		other.ID = uuid.UUID{2}

		other.DeviceID = foreign.ID
		for _, item := range []domaindevice.ConnectRequest{lastRequest, other, firstRequest} {
			require.NoError(t, requests.Create(t.Context(), item))
		}

		list := func(ctx context.Context, req pagination.Request[uuid.UUID]) (
			pagination.Page[domaindevice.ConnectRequest, uuid.UUID], error,
		) {
			return requests.ListByDevicePage(ctx, first.ID, req)
		}
		checkPages(
			t,
			list,
			[]domaindevice.ConnectRequest{firstRequest, lastRequest},
			func(r domaindevice.ConnectRequest) uuid.UUID { return r.ID },
		)
	})

	t.Run("users by email", func(t *testing.T) {
		t.Parallel()

		users := user.New(db)
		firstUser := domainuser.User{
			ID: last.ID, Email: "a@example.org", Name: "", PasswordHash: "",
		}
		lastUser := domainuser.User{
			ID: first.ID, Email: "b@example.org", Name: "", PasswordHash: "",
		}
		require.NoError(t, users.Create(t.Context(), lastUser))
		require.NoError(t, users.Create(t.Context(), firstUser))
		checkPages(t, users.ListPage, []domainuser.User{firstUser, lastUser},
			func(u domainuser.User) string { return u.Email })
	})

	t.Run("sensors by device", func(t *testing.T) {
		t.Parallel()

		sensors := sensor.New(db)
		firstSensor := domainsensor.Sensor{
			ID: uuid.UUID{1}, DeviceID: first.ID, Name: "", DisplayName: "",
			Manufacturer: "", Model: "", Revision: "", Software: "",
		}
		lastSensor := firstSensor
		lastSensor.ID = uuid.UUID{3}

		other := firstSensor
		other.ID = uuid.UUID{2}

		other.DeviceID = foreign.ID
		for _, item := range []domainsensor.Sensor{lastSensor, other, firstSensor} {
			require.NoError(t, sensors.InsertSensor(t.Context(), &item))
		}

		list := func(ctx context.Context, req pagination.Request[domainsensor.ID]) (
			pagination.Page[domainsensor.Sensor, domainsensor.ID], error,
		) {
			return sensors.GetSensorsPage(ctx, first.ID, req)
		}
		checkPages(t, list, []domainsensor.Sensor{firstSensor, lastSensor},
			func(s domainsensor.Sensor) domainsensor.ID { return s.ID })
	})
}

func checkPages[T any, C comparable](
	t *testing.T,
	list func(context.Context, pagination.Request[C]) (pagination.Page[T, C], error),
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
