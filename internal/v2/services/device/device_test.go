package device_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	domain "github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/services/device"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type onlineDeviceRepo struct {
	device.Repository

	stored domain.Device
}

func (r *onlineDeviceRepo) Get(
	_ context.Context,
	ownerID, id uuid.UUID,
) (domain.Device, error) {
	if ownerID != r.stored.OwnerID || id != r.stored.ID {
		return domain.Device{}, domain.ErrNotFound
	}

	return r.stored, nil
}

type heartbeatRepo struct {
	heartbeat *domain.Heartbeat
	err       error
}

func (r *heartbeatRepo) GetLastHeartbeat(
	_ context.Context,
	_ uuid.UUID,
) (*domain.Heartbeat, error) {
	return r.heartbeat, r.err
}

func TestService_IsDeviceOnline(t *testing.T) {
	t.Parallel()

	stored := domain.Device{ID: uuid.New(), OwnerID: uuid.New()}
	heartbeatErr := errors.New("heartbeat store unavailable")
	now := time.Now()
	tests := []struct {
		name      string
		heartbeat *domain.Heartbeat
		err       error
		online    bool
		lastSeen  time.Time
	}{
		{
			name:      "recent heartbeat",
			heartbeat: &domain.Heartbeat{Timestamp: now},
			online:    true, lastSeen: now,
		},
		{
			name: "old heartbeat",
			heartbeat: &domain.Heartbeat{
				Timestamp: now.Add(-2 * device.PingTimeout),
			},
			lastSeen: now.Add(-2 * device.PingTimeout),
		},
		{name: "repository error", err: heartbeatErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := device.New(
				&onlineDeviceRepo{stored: stored},
				&heartbeatRepo{heartbeat: tt.heartbeat, err: tt.err},
				nil, nil,
			)

			online, lastSeen, err := service.IsDeviceOnline(
				t.Context(), auth.Actor{UserID: stored.OwnerID}, stored.ID,
			)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.online, online)
			assert.Equal(t, tt.lastSeen, lastSeen)
		})
	}
}
