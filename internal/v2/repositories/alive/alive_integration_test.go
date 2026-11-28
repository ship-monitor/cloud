//go:build integration

package alive_test

import (
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	domain "github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/repositories/alive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/valkey-io/valkey-go"
)

func valkeyClient(t *testing.T) (valkey.Client, func()) {
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

func isMissingKey(err error) bool {
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
func TestHeartbeatMissingAndExpired(t *testing.T) {
	t.Parallel()

	client, _ := valkeyClient(t)
	repo := alive.New(client)
	deviceID := uuid.New()
	ctx := t.Context()

	checkMissing := func(t *testing.T) {
		t.Helper()

		got, err := repo.GetLastHeartbeat(ctx, deviceID)
		require.ErrorIs(t, err, domain.ErrNoHeartbeat)
		assert.Nil(t, got)
	}

	t.Run("never recorded", checkMissing)

	heartbeat := domain.Heartbeat{
		Timestamp: time.Now().UTC().Truncate(time.Second),
	}
	require.NoError(t, repo.InsertHeartbeat(ctx, deviceID, heartbeat))
	got, err := repo.GetLastHeartbeat(ctx, deviceID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, heartbeat.Timestamp.Equal(got.Timestamp))

	key := "device." + deviceID.String() + ".heartbeats"
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
	deviceID := uuid.New()

	closeClient()

	got, err := repo.GetLastHeartbeat(t.Context(), deviceID)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.False(t, isMissingKey(err))
}
