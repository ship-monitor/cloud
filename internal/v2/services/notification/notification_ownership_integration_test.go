//go:build integration

package notification_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/ship-monitor/cloud/internal/v2/adapters/db"
	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	devicedomain "github.com/ship-monitor/cloud/internal/v2/domain/device"
	notificationdomain "github.com/ship-monitor/cloud/internal/v2/domain/notification"
	"github.com/ship-monitor/cloud/internal/v2/domain/pagination"
	connectrepo "github.com/ship-monitor/cloud/internal/v2/repositories/connect"
	devicerepo "github.com/ship-monitor/cloud/internal/v2/repositories/device"
	notificationrepo "github.com/ship-monitor/cloud/internal/v2/repositories/notification"
	deviceservice "github.com/ship-monitor/cloud/internal/v2/services/device"
	notificationservice "github.com/ship-monitor/cloud/internal/v2/services/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

//nolint:funlen // Keep the complete ownership transition in one scenario.
func TestNotificationHistoryIsolatedAfterDeviceReconnect(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	ctx := t.Context()
	devices := devicerepo.New(db)
	requests := connectrepo.New(db)
	notifications := notificationrepo.New(db)
	deviceService := deviceservice.New(devices, nil, nil, requests)
	notificationService := notificationservice.New(notifications, deviceService)

	oldOwner := auth.Actor{UserID: uuid.New()}
	newOwner := auth.Actor{UserID: uuid.New()}
	deviceID := uuid.New()
	require.NoError(t, devices.Create(ctx, devicedomain.Device{
		ID: deviceID, OwnerID: oldOwner.UserID, Name: "device",
		Manufacturer: deviceservice.Unknown, Model: deviceservice.Unknown,
		Revision: deviceservice.Unknown, Software: deviceservice.Unknown,
	}))

	oldNotification := newNotification()
	oldNotification.DeviceID = deviceID
	oldNotification.SensorID = nil
	require.NoError(t, notifications.InsertNotification(ctx, oldNotification))

	pageRequest := pagination.Request[notificationdomain.Cursor]{
		After: nil, Limit: 10,
	}
	page, err := notificationService.ListByDevicePage(
		ctx, oldOwner, deviceID, pageRequest,
	)
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, oldNotification.ID, page.Items[0].ID)
	require.Nil(t, page.Next, "fixture history must fit in one page")

	_, err = notificationService.ListByDevicePage(
		ctx, newOwner, deviceID, pageRequest,
	)
	require.ErrorIs(t, err, devicedomain.ErrNotFound)

	require.NoError(t, deviceService.Remove(ctx, oldOwner, deviceID))
	request := devicedomain.ConnectRequest{
		ID: uuid.New(), DeviceID: deviceID, UserID: newOwner.UserID,
		CreatedAt: time.Now(), Approved: false,
	}
	require.NoError(t, requests.Create(ctx, request))
	deviceCtx := devicedomain.WithDeviceID(ctx, deviceID)
	require.NoError(
		t,
		deviceService.ApproveConnection(
			deviceCtx,
			request.DeviceID,
			request.ID,
		),
	)

	_, err = notificationService.ListByDevicePage(
		ctx, oldOwner, deviceID, pageRequest,
	)
	require.ErrorIs(t, err, devicedomain.ErrNotFound)

	page, err = notificationService.ListByDevicePage(
		ctx, newOwner, deviceID, pageRequest,
	)
	require.NoError(t, err)
	assert.Empty(t, page.Items, "new owner must not see previous history")
	assert.Nil(t, page.Next, "fixture history must fit in one page")

	freshNotification := newNotification()
	freshNotification.DeviceID = deviceID
	freshNotification.SensorID = nil
	require.NoError(t, notifications.InsertNotification(ctx, freshNotification))

	page, err = notificationService.ListByDevicePage(
		ctx, newOwner, deviceID, pageRequest,
	)
	require.NoError(t, err)

	ids := make([]uuid.UUID, 0, len(page.Items))
	for _, item := range page.Items {
		ids = append(ids, item.ID)
	}

	assert.Nil(t, page.Next, "fixture history must fit in one page")
	assert.Contains(t, ids, freshNotification.ID, "new history must be visible")
	assert.NotContains(t, ids, oldNotification.ID,
		"previous owner's history must remain inaccessible")
}

func setupDB(t *testing.T) *sqlx.DB {
	t.Helper()

	ctx := t.Context()
	container, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.BasicWaitStrategies(),
	)
	testcontainers.CleanupContainer(t, container)
	require.NoError(t, err)

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	sqlDB, err := db.New(&db.Config{DSN: connStr})
	require.NoError(t, err)

	err = db.Migrate(t.Context(), sqlDB)
	require.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, sqlDB.Close())
	})

	return sqlDB
}

func newNotification() notificationdomain.Notification {
	sensorID := uuid.New()

	return notificationdomain.Notification{
		ID:         uuid.New(),
		DeviceID:   uuid.New(),
		SensorID:   &sensorID,
		Type:       "engine_overheat",
		Severity:   "warning",
		OccurredAt: time.Now().UTC(),
		ReportedAt: time.Now().UTC(),
		ReceivedAt: time.Now().UTC(),
		Data: map[string]any{
			"temperature": 95.5,
			"threshold":   90.0,
			"str":         "strval",
			"map": map[string]any{
				"some": 123.0,
			},
		},
		IsRead: false,
	}
}
