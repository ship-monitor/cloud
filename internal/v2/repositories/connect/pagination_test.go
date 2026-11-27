//go:build integration

package connect_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/domain/pagination"
	"github.com/ship-monitor/cloud/internal/v2/repositories/connect"
	"github.com/stretchr/testify/require"
)

func TestRepository_CursorPages(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	requests := connect.New(db)
	firstID, foreignID := uuid.UUID{1}, uuid.UUID{2}
	stamp := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	firstRequest := device.ConnectRequest{
		ID: uuid.UUID{1}, DeviceID: firstID, CreatedAt: stamp,
		UserID: uuid.New(), Approved: false,
	}
	lastRequest := firstRequest
	lastRequest.ID = uuid.UUID{3}
	other := firstRequest
	other.ID = uuid.UUID{2}
	other.DeviceID = foreignID

	for _, item := range []device.ConnectRequest{
		lastRequest, other, firstRequest,
	} {
		require.NoError(t, requests.Create(t.Context(), item))
	}

	list := func(ctx context.Context, req pagination.Request[uuid.UUID]) (
		pagination.Page[device.ConnectRequest, uuid.UUID], error,
	) {
		return requests.ListByDevicePage(ctx, firstID, req)
	}
	checkPages(
		t,
		list,
		[]device.ConnectRequest{firstRequest, lastRequest},
		func(r device.ConnectRequest) uuid.UUID { return r.ID },
	)
}
