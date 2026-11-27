//go:build integration

package alive_test

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
	devicerepo "github.com/ship-monitor/cloud/internal/v2/repositories/device"
	"github.com/ship-monitor/cloud/internal/v2/services/device"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/valkey-io/valkey-go"
)

type ownedDeviceRepo struct {
	device.Repository

	stored domain.Device
}

func (r *ownedDeviceRepo) Get(
	_ context.Context,
	ownerID, id uuid.UUID,
) (domain.Device, error) {
	if ownerID != r.stored.OwnerID || id != r.stored.ID {
		return domain.Device{}, devicerepo.ErrNotFound
	}

	return r.stored, nil
}

func valkeyClient(t *testing.T) (valkey.Client, func()) {
	t.Helper()

	ctx := t.Context()

	request := testcontainers.ContainerRequest{
		Image:        "valkey/valkey:8-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx,
		testcontainers.GenericContainerRequest{
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
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{net.JoinHostPort(host, port.Port())},
	})
	require.NoError(t, err)

	var closeOnce sync.Once

	closeClient := func() { closeOnce.Do(client.Close) }
	t.Cleanup(closeClient)

	return client, closeClient
}

func isMissingKey(err error) bool {
	for err != nil {
		if valkey.IsValkeyNil(err) {
			return true
		}

		err = errors.Unwrap(err)
	}

	return false
}

func TestHeartbeatMissingAndExpired(t *testing.T) {
	t.Parallel()

	client, _ := valkeyClient(t)
	repo := alive.New(client)
	stored := domain.Device{ID: uuid.New(), OwnerID: uuid.New()}
	service := device.New(&ownedDeviceRepo{stored: stored}, repo, nil, nil)
	actor := auth.Actor{UserID: stored.OwnerID}
	ctx := t.Context()

	checkMissing := func(t *testing.T) {
		t.Helper()
		t.Run("repository", func(t *testing.T) {
			got, err := repo.GetLastHeartbeat(ctx, stored.ID)
			require.NoError(
				t,
				err,
				"an absent heartbeat is not a store failure",
			)
			assert.Nil(t, got)
		})
		t.Run("service", func(t *testing.T) {
			online, lastSeen, err := service.IsDeviceOnline(
				ctx,
				actor,
				stored.ID,
			)
			require.NoError(t, err, "an absent heartbeat means offline")
			assert.False(t, online)
			assert.True(t, lastSeen.IsZero())
		})
	}

	t.Run("never recorded", checkMissing)

	heartbeat := domain.Heartbeat{
		Timestamp: time.Now().UTC().Truncate(time.Second),
	}
	require.NoError(t, repo.InsertHeartbeat(ctx, stored.ID, heartbeat))
	got, err := repo.GetLastHeartbeat(ctx, stored.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, heartbeat.Timestamp.Equal(got.Timestamp))

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
	ttl, err := client.Do(ctx,
		client.B().Arbitrary("TTL").Keys(key).Build(),
	).ToInt64()
	require.NoError(t, err)
	assert.Positive(t, ttl, "inserted heartbeats must expire")
	assert.LessOrEqual(t, ttl, int64(alive.ExpirationDuration.Seconds()))

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

func TestHeartbeatStoreUnavailable(t *testing.T) {
	t.Parallel()

	client, closeClient := valkeyClient(t)
	repo := alive.New(client)
	stored := domain.Device{ID: uuid.New(), OwnerID: uuid.New()}
	service := device.New(&ownedDeviceRepo{stored: stored}, repo, nil, nil)

	closeClient()

	got, err := repo.GetLastHeartbeat(t.Context(), stored.ID)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.False(t, isMissingKey(err))

	online, lastSeen, err := service.IsDeviceOnline(
		t.Context(), auth.Actor{UserID: stored.OwnerID}, stored.ID,
	)
	require.Error(t, err)
	require.ErrorContains(t, err, "get last heartbeat")
	assert.False(t, isMissingKey(err))
	assert.False(t, online)
	assert.True(t, lastSeen.IsZero())
}
