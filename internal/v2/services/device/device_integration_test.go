//go:build integration

package device_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/ship-monitor/cloud/internal/v2/adapters/db"
	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	domain "github.com/ship-monitor/cloud/internal/v2/domain/device"
	connectrepo "github.com/ship-monitor/cloud/internal/v2/repositories/connect"
	devicerepo "github.com/ship-monitor/cloud/internal/v2/repositories/device"
	"github.com/ship-monitor/cloud/internal/v2/services/device"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupDeviceDB(t *testing.T) *sqlx.DB {
	t.Helper()

	ctx := t.Context()

	container, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.BasicWaitStrategies(),
	)
	if container != nil {
		testcontainers.CleanupContainer(t, container)
	}

	require.NoError(t, err)
	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	sqlDB, err := db.New(&db.Config{DSN: connStr})
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, sqlDB.Close()) })
	require.NoError(t, db.Migrate(ctx, sqlDB))

	return sqlDB
}

func createDeviceTestUser(t *testing.T, sqlDB *sqlx.DB) uuid.UUID {
	t.Helper()

	id := uuid.New()
	_, err := sqlDB.ExecContext(t.Context(), `
		INSERT INTO users (id, name, email, password_hash, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		id, "test user", id.String()+"@example.test", "test hash", time.Now(),
	)
	require.NoError(t, err)

	return id
}

func TestService_RenameOwnerIsolation(t *testing.T) {
	t.Parallel()

	sqlDB := setupDeviceDB(t)
	repo := devicerepo.New(sqlDB)
	ownerID := createDeviceTestUser(t, sqlDB)
	otherUserID := createDeviceTestUser(t, sqlDB)
	stored := domain.Device{
		ID: uuid.New(), OwnerID: ownerID, Name: "original",
		Manufacturer: "test", Model: "test", Revision: "test",
		Software: "test",
	}
	require.NoError(t, repo.Create(t.Context(), stored))
	service := device.New(repo, nil, nil, nil)

	err := service.Rename(t.Context(), auth.Actor{UserID: otherUserID},
		stored.ID, "foreign rename")
	require.ErrorIs(t, err, domain.ErrNotFound)
	got, err := repo.Get(t.Context(), stored.OwnerID, stored.ID)
	require.NoError(t, err)
	assert.Equal(t, "original", got.Name)

	err = service.Rename(t.Context(), auth.Actor{UserID: stored.OwnerID},
		uuid.New(), "missing")
	require.ErrorIs(t, err, domain.ErrNotFound)
	require.NoError(t, service.Rename(
		t.Context(), auth.Actor{UserID: stored.OwnerID}, stored.ID, "renamed",
	))
	got, err = repo.Get(t.Context(), stored.OwnerID, stored.ID)
	require.NoError(t, err)
	assert.Equal(t, "renamed", got.Name)
}

func TestService_ApproveConnectionIdempotent(t *testing.T) {
	t.Parallel()

	sqlDB := setupDeviceDB(t)
	repo := devicerepo.New(sqlDB)
	requests := connectrepo.New(sqlDB)
	request := domain.ConnectRequest{
		ID: uuid.New(), DeviceID: uuid.New(),
		UserID:    createDeviceTestUser(t, sqlDB),
		CreatedAt: time.Now(),
	}
	ctx := domain.WithDeviceID(t.Context(), request.DeviceID)
	require.NoError(t, requests.Create(ctx, request))
	service := device.New(repo, nil, nil, requests)

	require.NoError(
		t,
		service.ApproveConnection(ctx, request.DeviceID, request.ID),
	)
	created, err := repo.Get(ctx, request.UserID, request.DeviceID)
	require.NoError(t, err)
	gotRequest, err := requests.Get(ctx, request.ID)
	require.NoError(t, err)
	require.True(t, gotRequest.Approved)

	err = service.ApproveConnection(ctx, request.DeviceID, request.ID)
	require.NoError(t, err, "a duplicate approval should be harmless")
	got, err := repo.Get(ctx, request.UserID, request.DeviceID)
	require.NoError(t, err)
	assert.Equal(t, created, got)

	gotRequest, err = requests.Get(ctx, request.ID)
	require.NoError(t, err)
	assert.True(t, gotRequest.Approved)
}

func TestService_ApproveConnectionAtomicOnFailure(t *testing.T) {
	t.Parallel()

	sqlDB := setupDeviceDB(t)
	repo := devicerepo.New(sqlDB)
	requests := connectrepo.New(sqlDB)
	request := domain.ConnectRequest{
		ID: uuid.New(), DeviceID: uuid.New(),
		UserID:    createDeviceTestUser(t, sqlDB),
		CreatedAt: time.Now(),
		Approved:  false,
	}
	ctx := domain.WithDeviceID(t.Context(), request.DeviceID)
	require.NoError(t, requests.Create(ctx, request))
	_, err := sqlDB.ExecContext(ctx, `
		CREATE FUNCTION reject_approval() RETURNS trigger AS $$
		BEGIN
			RAISE EXCEPTION 'injected approval failure';
		END;
		$$ LANGUAGE plpgsql;
		CREATE TRIGGER reject_approval BEFORE UPDATE OF approved
		ON connect_requests FOR EACH ROW EXECUTE FUNCTION reject_approval();
	`)
	require.NoError(t, err)

	service := device.New(repo, nil, nil, requests)

	err = service.ApproveConnection(ctx, request.DeviceID, request.ID)
	require.ErrorContains(t, err, "injected approval failure")
	gotRequest, err := requests.Get(ctx, request.ID)
	require.NoError(t, err)
	assert.False(t, gotRequest.Approved)

	_, err = repo.Get(ctx, request.UserID, request.DeviceID)
	require.ErrorIs(t, err, domain.ErrNotFound,
		"failed approval must not leave a device")

	_, err = sqlDB.ExecContext(ctx,
		`DROP TRIGGER reject_approval ON connect_requests`)
	require.NoError(t, err)
	err = service.ApproveConnection(ctx, request.DeviceID, request.ID)
	require.NoError(t, err, "retry after a storage failure should succeed")
	gotRequest, err = requests.Get(ctx, request.ID)
	require.NoError(t, err)
	assert.True(t, gotRequest.Approved)

	_, err = repo.Get(ctx, request.UserID, request.DeviceID)
	assert.NoError(t, err)
}
