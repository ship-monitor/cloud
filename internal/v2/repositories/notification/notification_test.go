// Package notification_test contains tests for the notification repository.
package notification_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/ship-monitor/cloud/internal/v2/adapters/db"
	domain "github.com/ship-monitor/cloud/internal/v2/domain/notification"
	"github.com/ship-monitor/cloud/internal/v2/repositories/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestRepository_InsertNotification(t *testing.T) {
	t.Parallel()

	sqlDB := setupDB(t)
	repo := notification.New(sqlDB)

	testCases := []struct {
		name string
		want domain.Notification
	}{
		{
			name: "ok",
			want: newNotification(),
		},
		{
			name: "ok/empty data",
			want: domain.Notification{
				ID:         uuid.New(),
				DeviceID:   uuid.New(),
				SensorID:   nil,
				Type:       "engine_overheat",
				Severity:   "warning",
				OccurredAt: time.Now().UTC(),
				ReportedAt: time.Now().UTC(),
				ReceivedAt: time.Now().UTC(),
				Data:       map[string]any{},
				IsRead:     false,
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			require.NoError(t, repo.InsertNotification(ctx, tt.want))

			var row notification.Row

			err := sqlDB.QueryRowxContext(ctx, `
				SELECT id, device_id, sensor_id, type, severity, occurred_at,
					reported_at, received_at, data
				FROM device_notifications WHERE id = $1`, tt.want.ID).
				Scan(
					&row.ID,
					&row.DeviceID,
					&row.SensorID,
					&row.Type,
					&row.Severity,
					&row.OccurredAt,
					&row.ReportedAt,
					&row.ReceivedAt,
					&row.Data,
				)
			require.NoError(t, err)

			got := row.ToDomain()
			assert.Equal(t, tt.want, got)

			var duplicateErr *pq.Error

			err = repo.InsertNotification(ctx, tt.want)
			require.ErrorAs(t, err, &duplicateErr)
			assert.Equal(t, "unique_violation", duplicateErr.Code.Name())
		})
	}
}

func TestRepository_Get(t *testing.T) {
	t.Parallel()

	sqlDB := setupDB(t)
	repo := notification.New(sqlDB)
	stored := newNotification()

	_, err := sqlDB.ExecContext(t.Context(), `
		INSERT INTO device_notifications
			(id, device_id, sensor_id, type, severity, occurred_at,
			 reported_at, received_at, data)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		stored.ID,
		stored.DeviceID,
		stored.SensorID,
		stored.Type,
		stored.Severity,
		stored.OccurredAt,
		stored.ReportedAt,
		stored.ReceivedAt,
		notification.JSONB(stored.Data),
	)
	require.NoError(t, err)

	//nolint:exhaustruct_v5
	cases := []struct {
		name    string
		id      uuid.UUID
		want    *domain.Notification
		wantErr error
	}{
		{
			name: "existing notification",
			id:   stored.ID,
			want: &stored,
		},
		{
			name:    "missing notification",
			id:      uuid.New(),
			wantErr: domain.ErrNotFound,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := repo.Get(t.Context(), tt.id)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRepository_ListByDevice(t *testing.T) {
	t.Parallel()

	sqlDB := setupDB(t)
	repo := notification.New(sqlDB)
	ctx := t.Context()

	deviceID := uuid.New()
	otherDeviceID := uuid.New()

	now := time.Now().UTC()
	older := now.Add(-time.Hour)
	newer := now.Add(time.Hour)

	first := newNotification()
	first.DeviceID = deviceID
	first.OccurredAt = older

	second := newNotification()
	second.DeviceID = deviceID
	second.OccurredAt = newer

	other := newNotification()
	other.DeviceID = otherDeviceID
	other.OccurredAt = now

	for _, n := range []domain.Notification{first, second, other} {
		require.NoError(t, repo.InsertNotification(ctx, n))
	}

	got, err := repo.ListByDevice(ctx, deviceID, 10, 0)
	require.NoError(t, err)
	require.Len(t, got, 2)

	// Ordered by occurred_at descending: second (newer) first.
	assert.Equal(t, second.ID, got[0].ID)
	assert.Equal(t, first.ID, got[1].ID)
}

func TestRepository_ListBySeverity(t *testing.T) {
	t.Parallel()

	sqlDB := setupDB(t)
	repo := notification.New(sqlDB)
	ctx := t.Context()

	now := time.Now().UTC()

	warning := newNotification()
	warning.Severity = "warning"
	warning.OccurredAt = now.Add(-time.Hour)

	critical := newNotification()
	critical.Severity = "critical"
	critical.OccurredAt = now

	info := newNotification()
	info.Severity = "info"
	info.OccurredAt = now

	for _, n := range []domain.Notification{warning, critical, info} {
		require.NoError(t, repo.InsertNotification(ctx, n))
	}

	got, err := repo.ListBySeverity(ctx, "critical", 10, 0)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, critical.ID, got[0].ID)
}

func TestRepository_Exists(t *testing.T) {
	t.Parallel()

	sqlDB := setupDB(t)
	repo := notification.New(sqlDB)
	ctx := t.Context()

	stored := newNotification()
	require.NoError(t, repo.InsertNotification(ctx, stored))

	exists, err := repo.Exists(ctx, stored.ID)
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.Exists(ctx, uuid.New())
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRepository_Delete(t *testing.T) {
	t.Parallel()

	sqlDB := setupDB(t)
	repo := notification.New(sqlDB)
	ctx := t.Context()

	stored := newNotification()
	require.NoError(t, repo.InsertNotification(ctx, stored))

	require.NoError(t, repo.Delete(ctx, stored.ID))

	exists, err := repo.Exists(ctx, stored.ID)
	require.NoError(t, err)
	assert.False(t, exists)

	err = repo.Delete(ctx, stored.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
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

func newNotification() domain.Notification {
	sensorID := uuid.New()

	return domain.Notification{
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
