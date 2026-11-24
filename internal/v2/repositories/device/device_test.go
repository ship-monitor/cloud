// Package device_test contains tests for the device repository.
package device_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/ship-monitor/cloud/internal/v2/adapters/db"
	domain "github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/repositories/device"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestRepository_Create(t *testing.T) {
	t.Parallel()

	sqlDB := setupDB(t)
	repo := device.New(sqlDB)
	ctx := t.Context()
	want := newDevice()

	require.NoError(t, repo.Create(ctx, want))

	var got domain.Device

	err := sqlDB.QueryRowxContext(ctx, `
		SELECT id, owner_id, name, manufacturer, model, revision, software
		FROM devices WHERE id = $1`, want.ID).
		Scan(
			&got.ID,
			&got.OwnerID,
			&got.Name,
			&got.Manufacturer,
			&got.Model,
			&got.Revision,
			&got.Software,
		)
	require.NoError(t, err)
	assert.Equal(t, want, got)

	var duplicateErr *pq.Error

	err = repo.Create(ctx, want)
	require.ErrorAs(t, err, &duplicateErr)
	assert.Equal(t, "unique_violation", duplicateErr.Code.Name())
}

//nolint:funlen
func TestRepository_Get(t *testing.T) {
	t.Parallel()

	sqlDB := setupDB(t)
	repo := device.New(sqlDB)
	stored := newDevice()

	_, err := sqlDB.ExecContext(t.Context(), `
		INSERT INTO devices
			(id, owner_id, name, created_at, manufacturer, model,
			 revision, software)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, $4, $5, $6, $7)`,
		stored.ID,
		stored.OwnerID,
		stored.Name,
		stored.Manufacturer,
		stored.Model,
		stored.Revision,
		stored.Software,
	)
	require.NoError(t, err)

	//nolint:exhaustruct_v5
	cases := []struct {
		name    string
		ownerID uuid.UUID
		id      uuid.UUID
		want    domain.Device
		wantErr error
	}{
		{
			name:    "owned device",
			ownerID: stored.OwnerID,
			id:      stored.ID,
			want:    stored,
		},
		{
			name:    "missing device",
			ownerID: stored.OwnerID,
			id:      uuid.New(),
			wantErr: device.ErrNotFound,
		},
		{
			name:    "device owned by another user",
			ownerID: uuid.New(),
			id:      stored.ID,
			wantErr: device.ErrNotFound,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := repo.Get(t.Context(), tt.ownerID, tt.id)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
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

func newDevice() domain.Device {
	return domain.Device{
		ID:           uuid.New(),
		OwnerID:      uuid.New(),
		Name:         "Engine monitor",
		Manufacturer: "Ship Monitor",
		Model:        "SM-100",
		Revision:     "rev-a",
		Software:     "1.2.3",
	}
}
