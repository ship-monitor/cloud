//go:build integration

package device_test

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	domain "github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/repositories/alive"
	"github.com/ship-monitor/cloud/internal/v2/services/device"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/valkey-io/valkey-go"
)

type heartbeatDeviceRepo struct {
	device.Repository

	stored domain.Device
}

func (r *heartbeatDeviceRepo) Get(
	_ context.Context,
	ownerID, id uuid.UUID,
) (domain.Device, error) {
	if ownerID != r.stored.OwnerID || id != r.stored.ID {
		return domain.Device{}, domain.ErrNotFound
	}

	return r.stored, nil
}

func heartbeatValkeyClient(t *testing.T) (valkey.Client, func()) {
	t.Helper()

	ctx := t.Context()
	request := testcontainers.ContainerRequest{ //nolint:exhaustruct_v5
		Image:        "valkey/valkey:8-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx,
		testcontainers.GenericContainerRequest{ //nolint:exhaustruct_v5
			Started:          true,
			ContainerRequest: request,
		},
	)
	if container != nil {
		testcontainers.CleanupContainer(t, container)
	}

	require.NoError(t, err)
	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "6379/tcp")
	require.NoError(t, err)
	client, err := valkey.NewClient(valkey.ClientOption{ //nolint:exhaustruct_v5
		InitAddress: []string{net.JoinHostPort(host, port.Port())},
	})
	require.NoError(t, err)

	var closeOnce sync.Once

	closeClient := func() { closeOnce.Do(client.Close) }
	t.Cleanup(closeClient)

	return client, closeClient
}

func heartbeatIsMissingKey(err error) bool {
	for err != nil {
		if valkey.IsValkeyNil(err) {
			return true
		}

		err = errors.Unwrap(err)
	}

	return false
}

// The subtests observe successive states of the same key.
//
//nolint:paralleltest,tparallel
func TestService_HeartbeatStatus(t *testing.T) {
	t.Parallel()

	client, _ := heartbeatValkeyClient(t)
	repo := alive.New(client)
	stored := domain.Device{ //nolint:exhaustruct_v5
		ID: uuid.New(), OwnerID: uuid.New(),
	}
	service := device.New(
		&heartbeatDeviceRepo{stored: stored}, //nolint:exhaustruct_v5
		repo, nil, nil,
	)
	actor := auth.Actor{UserID: stored.OwnerID}
	ctx := t.Context()

	checkMissing := func(t *testing.T) {
		t.Helper()

		online, lastSeen, err := service.IsDeviceOnline(ctx, actor, stored.ID)
		require.NoError(t, err, "an absent heartbeat means offline")
		assert.False(t, online)
		assert.True(t, lastSeen.IsZero())
	}

	t.Run("never recorded", checkMissing)

	heartbeat := domain.Heartbeat{
		Timestamp: time.Now().UTC().Truncate(time.Second),
	}
	require.NoError(t, repo.InsertHeartbeat(ctx, stored.ID, heartbeat))
	online, lastSeen, err := service.IsDeviceOnline(ctx, actor, stored.ID)
	require.NoError(t, err)
	assert.True(t, online)
	assert.True(t, heartbeat.Timestamp.Equal(lastSeen))

	stale := domain.Heartbeat{
		Timestamp: heartbeat.Timestamp.Add(-2 * device.PingTimeout),
	}
	require.NoError(t, repo.InsertHeartbeat(ctx, stored.ID, stale))
	online, lastSeen, err = service.IsDeviceOnline(ctx, actor, stored.ID)
	require.NoError(t, err)
	assert.False(t, online)
	assert.True(t, stale.Timestamp.Equal(lastSeen))

	key := "device." + stored.ID.String() + ".heartbeats"
	// Shorten the real key's TTL rather than waiting 48 hours.
	expired, err := client.Do(ctx,
		client.B().Arbitrary("PEXPIRE").Keys(key).Args("10").Build(),
	).ToInt64()
	require.NoError(t, err)
	require.EqualValues(t, 1, expired)
	require.Eventually(t, func() bool {
		err := client.Do(ctx, client.B().Get().Key(key).Build()).Error()

		return valkey.IsValkeyNil(err)
	}, 5*time.Second, 10*time.Millisecond)

	t.Run("expired", checkMissing)
}

func TestService_HeartbeatStoreUnavailable(t *testing.T) {
	t.Parallel()

	client, closeClient := heartbeatValkeyClient(t)
	repo := alive.New(client)
	stored := domain.Device{ //nolint:exhaustruct_v5
		ID: uuid.New(), OwnerID: uuid.New(),
	}
	service := device.New(
		&heartbeatDeviceRepo{stored: stored}, //nolint:exhaustruct_v5
		repo, nil, nil,
	)

	closeClient()

	online, lastSeen, err := service.IsDeviceOnline(
		t.Context(), auth.Actor{UserID: stored.OwnerID}, stored.ID,
	)
	require.Error(t, err)
	require.ErrorContains(t, err, "get last heartbeat")
	assert.False(t, heartbeatIsMissingKey(err))
	assert.False(t, online)
	assert.True(t, lastSeen.IsZero())
}
